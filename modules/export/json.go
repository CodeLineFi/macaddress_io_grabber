package export

import (
	"bufio"
	"encoding/json"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/stats"
	"macaddress_io_grabber/utils/database"
	"os"
	"sort"
	"strings"
)

func ToJSON(filename string) (err error) {

	var records map[string]map[int64]*database.OUI

	records, err = stats.FetchRecordsMapWithReserve()

	var writer *bufio.Writer

	if filename != "" {
		file, err := createFile(filename)
		if err != nil {
			return err
		}
		writer = bufio.NewWriterSize(file, 1024*16)
	} else {
		writer = bufio.NewWriterSize(os.Stdout, 1024*16)
	}

	defer func() {
		if err == nil {
			err = writer.Flush()
		}
	}()

	for _, recordByDates := range records {

		var keys = make([]int64, 0, len(recordByDates))
		for k := range recordByDates {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		if len(keys) == 0 {
			continue
		}

		var (
			first = recordByDates[keys[0]]
			last  = recordByDates[keys[len(keys)-1]]
		)

		var mac models.MAC
		mac, err = models.NewMac(last.Assignment)
		if err != nil {
			return
		}

		var isPrivate bool
		if strings.ToLower(last.Org.NName) == "private" {
			isPrivate = true
		}

		var dateCreated string
		if first.CreatedAt != nil {
			dateCreated = first.CreatedAt.Format("2006-01-02")
		}

		var dateUpdated string
		if last.CreatedAt != nil {
			dateUpdated = last.CreatedAt.Format("2006-01-02")
		}

		bytes, err := json.Marshal(models.OUIJson{
			Assignment:          mac,
			IsPrivate:           isPrivate,
			CompanyName:         last.Org.NName,
			CompanyAddress:      last.RawOrgAddress,
			AssignmentBlockSize: last.Type,
			CountryCode:         last.Country,
			DateCreated:         dateCreated,
			DateUpdated:         dateUpdated,
		})
		if err != nil {
			return err
		}

		if _, err := writer.WriteString(string(bytes) + "\n"); err != nil {
			return err
		}
	}

	return nil
}
