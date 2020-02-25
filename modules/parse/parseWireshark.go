package parse

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	m "macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/download"
	"macaddress_io_grabber/utils/database"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

func ParseWireshark(dir string, d time.Time) error {

	getLast := func(old, new *m.WiresharkOUI) bool { return true }
	wireshark, err := parseWireShark(dir, download.WiresharkFile)
	if err != nil {
		return err
	}
	wireshark.Unique(getLast)
	sort.Sort(&wireshark)

	var wg = sync.WaitGroup{}
	var maxTasks = runtime.GOMAXPROCS(0)
	var tasks = 0

	var eChan = make(chan error, maxTasks)

	for _, r := range wireshark {
		if tasks >= maxTasks {
			if err := <-eChan; err != nil {
				wg.Wait()
				return err
			}
			tasks--
		}

		wg.Add(1)
		go func(r *m.WiresharkOUI) {
			defer wg.Done()

			wiresharkNote := database.WiresharkNote{
				Assignment: r.Assignment,
				Note:       r.Note,
			}

			eChan <- wiresharkNote.Save()
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

func parseWireShark(dir, filename string) (m.WiresharkRegistry, error) {
	var r m.WiresharkRegistry = make([]*m.WiresharkOUI, 0, 2048)

	f, err := os.Open(filepath.Join(dir, filename))
	if err != nil {
		return nil, errors.New("cannot parse file " + filename + ": " + err.Error())
	}

	defer f.Close()

	reader := csv.NewReader(bufio.NewReader(f))
	reader.Comma = '\t'
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	for {
		line, err := reader.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		oui := strings.Replace(line[0], ":", "", -1)
		comment := strings.Join(line[2:], " ")
		comment = strings.Trim(comment, " ")

		if comment == "" {
			continue
		}

		wiresharkOUI := &m.WiresharkOUI{
			Note: comment,
		}

		wiresharkOUI.Assignment = oui
		wiresharkOUI.Note = comment

		r = append(r, wiresharkOUI)
	}

	return r, nil
}
