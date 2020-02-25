package parse

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	m "macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/download"
	"macaddress_io_grabber/utils/database"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

func Parse(dir string, d time.Time) error {

	getLast := func(old, new *m.OUI) bool { return true }
	getFirst := (func(old, new *m.OUI) bool)(nil)

	mal, err := parse("MA-L", dir, download.MalFile)
	if err != nil {
		return err
	}
	mal.Unique(getLast)
	sort.Sort(&mal)

	mam, err := parse("MA-M", dir, download.MamFile)
	if err != nil {
		return err
	}
	mam.Unique(getLast)
	sort.Sort(&mam)

	mas, err := parse("MA-S", dir, download.MasFile)
	if err != nil {
		return err
	}
	mas.Unique(getLast)
	sort.Sort(&mas)

	// Legacy MA-S
	iab, err := parse("IAB", dir, download.IabFile)
	if err != nil {
		return err
	}
	iab.Unique(getLast)
	sort.Sort(&iab)

	cid, err := parse("CID", dir, download.CidFile)
	if err != nil {
		return err
	}
	cid.Unique(getLast)
	sort.Sort(&cid)

	merged := merge(mal, mam, mas, iab, cid)
	merged.Unique(getFirst)
	sort.Sort(&merged)

	var wg = sync.WaitGroup{}
	var maxTasks = runtime.GOMAXPROCS(0)
	var tasks = 0

	var eChan = make(chan error, maxTasks)

	for _, r := range merged {
		if tasks >= maxTasks {
			if err := <-eChan; err != nil {
				wg.Wait()
				return err
			}
			tasks--
		}

		wg.Add(1)
		go func(r *m.OUI) {
			defer wg.Done()

			oui := database.OUI{
				Assignment:    r.Assignment,
				RawOrgName:    r.OrgName,
				RawOrgAddress: cleanAddr(r.OrgAddress),
				Type:          r.Type,
				CreatedAt:     &d,
			}

			eChan <- oui.Save()
		}(r)
		tasks++
	}

	for tasks > 0 {
		tasks--
		if err := <-eChan; err != nil {
			wg.Wait()
			return err
		}
	}

	return nil
}

var reSpaces = regexp.MustCompile(`[\s]{2,}`)

func cleanAddr(s string) string {
	return reSpaces.ReplaceAllString(strings.TrimSpace(s), " ")
}

func parse(recordType, dir, filename string) (m.Registry, error) {
	var r m.Registry = make([]*m.OUI, 0, 2048)

	f, err := os.Open(filepath.Join(dir, filename))
	if err != nil {
		return nil, errors.New("cannot parse file " + filename + ": " + err.Error())
	}

	defer f.Close()

	var header = true

	reader := bufio.NewScanner(f)

	var next = true

	for reader.Err() == nil && next {
		var hexAndOrg, b16AndOrg, address string

		var block []string

		block, next, err = readBlock(reader, header)
		if err != nil {
			return nil, err
		}

		if header {
			header = false
		}

		if len(block) == 0 {
			continue
		}

		for i := range block {
			block[i] = strings.Trim(block[i], "\t ")
		}

		if len(block) > 1 {
			hexAndOrg = block[0]
			b16AndOrg = block[1]
		} else {
			return nil, fmt.Errorf("cannot parse record: %#v", block)
		}
		if len(block) > 2 {
			address = strings.Join(block[2:], " ")
		}

		oui, org, err := parseOuiAndOrg(hexAndOrg, b16AndOrg)
		if err != nil {
			return nil, err
		}

		r = append(r, &m.OUI{
			Assignment: oui,
			OrgName:    org,
			OrgAddress: address,
			Type:       recordType,
		})
	}

	if err := reader.Err(); err != nil {
		if err != io.EOF {
			return nil, err
		}
	}

	return r, nil
}

var permuteScan = false

func readBlock(reader *bufio.Scanner, header bool) ([]string, bool, error) {

	block := make([]string, 0, 5)

	for permuteScan || reader.Scan() {

		if permuteScan {
			permuteScan = false
		}

		t := reader.Text()

		if t == "" {
			return block, true, nil
		}

		tr := strings.Trim(t, "\t ")

		if r1.MatchString(tr) {
			if len(block) == 0 {
				block = append(block, t)
				continue
			} else {
				permuteScan = true
				return block, true, nil
			}
		}

		if len(block) == 0 {
			if !header {
				end, skipped := skipLines(reader)
				return block, end, errors.New(strings.Join(skipped, "\n"))
			}
			continue
		}

		if r2.MatchString(tr) {
			if len(block) == 1 {
				block = append(block, t)
				continue
			} else {
				end, skipped := skipLines(reader)
				return block, end, errors.New(strings.Join(skipped, "\n"))
			}
		}

		if len(block) == 1 {
			end, skipped := skipLines(reader)
			return block, end, errors.New(strings.Join(skipped, "\n"))
		}

		if len(block) < 2 {
			return block, true, nil
		}

		if tr != "" {
			block = append(block, t)
		}
	}

	return block, false, reader.Err()
}

func skipLines(reader *bufio.Scanner) (bool, []string) {

	t := reader.Text()

	skippedLines := make([]string, 0, 10)
	skippedLines = append(skippedLines, t)

	count := 0

	for reader.Scan() {
		t := reader.Text()
		if r1.MatchString(strings.Trim(t, "\t ")) {
			permuteScan = true
			return true, skippedLines
		}
		skippedLines = append(skippedLines, t)
		count++
	}

	return false, skippedLines
}

var r1 = regexp.MustCompile(`^([0-9A-F]{2}-[0-9A-F]{2}-[0-9A-F]{2})\s+\(hex\)\s+(.+)$`)
var r2 = regexp.MustCompile(`^([0-9A-F]+)(-([0-9A-F]+))?\s+\(base 16\)(\s+(.+))?$`)

func parseOuiAndOrg(hexAndOrg, b16AndOrg string) (string, string, error) {

	matches := r1.FindStringSubmatch(hexAndOrg)

	var oui, org string

	if len(matches) > 0 {
		oui = matches[1]
		org = matches[2]
	}

	matches = r2.FindStringSubmatch(b16AndOrg)

	if len(matches) > 0 {
		if len(matches[3]) != 0 {
			b1 := matches[1]
			b2 := matches[3]

			if len(b1) != len(b2) {
				return "", "", errors.New("base 16 is not correct: " + b16AndOrg)
			}
			for i := 0; i < len(b1); i++ {
				if b1[i] != b2[i] {
					oui += b1[0:i]
					break
				}
			}
		}

		if matches[5] != org && !(strings.ToLower(org) == "private" && matches[5] == "") {
			return "", "", errors.New("Organizations should be the same: " + org + " :: " + matches[3])
		}
	}

	mac, err := m.NewMac(oui)
	if err != nil {
		return "", "", err
	}

	return mac.String(), org, nil
}

func merge(rr ...m.Registry) m.Registry {
	if len(rr) == 0 {
		return nil
	}
	if len(rr) == 1 {
		return rr[0]
	}

	result := rr[0]
	rr = rr[1:]

	for _, r2 := range rr {
		r1 := result
		result = make([]*m.OUI, len(r1)+len(r2))
		for i, k := 0, 0; ; {
			if len(r1) == i {
				for len(r2) > k {
					result[i+k] = r2[k]
					k++
				}
				break
			}
			if len(r2) == k {
				for len(r1) > i {
					result[i+k] = r1[i]
					i++
				}
				break
			}
			if r1[i].Assignment <= r2[k].Assignment {
				result[i+k] = r1[i]
				i++
			} else {
				result[i+k] = r2[k]
				k++
			}
		}
	}

	return result
}
