package export

import (
	"encoding/csv"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/stats"
	"macaddress_io_grabber/utils/database"
	"os"
	"sort"
	"strings"
)

func ToCSV(filename string) (err error) {

	var records map[string]map[int64]*database.OUI

	records, err = stats.FetchRecordsMapWithReserve()

	var writer *csv.Writer

	if filename != "" {
		file, err := createFile(filename)
		if err != nil {
			return err
		}
		writer = csv.NewWriter(file)
	} else {
		writer = csv.NewWriter(os.Stdout)
	}

	err = writeCsv(writer, "oui", "isPrivate", "companyName", "companyAddress", "countryCode", "assignmentBlockSize", "dateCreated", "dateUpdated")
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			writer.Flush()
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

		var isPrivate = "0"
		if strings.ToLower(last.Org.NName) == "private" {
			isPrivate = "1"
		}

		var dateCreated string
		if first.CreatedAt != nil {
			dateCreated = first.CreatedAt.Format("2006-01-02")
		}

		var dateUpdated string
		if last.CreatedAt != nil {
			dateUpdated = last.CreatedAt.Format("2006-01-02")
		}

		err = writeCsv(writer, mac.Format(), isPrivate, last.Org.NName,
			last.RawOrgAddress, last.Country,
			last.Type, dateCreated, dateUpdated)
		if err != nil {
			return
		}
	}

	writer.Flush()

	return
}

func writeCsv(writer *csv.Writer, oui, isPrivate, companyName, companyAddress, countryCode, assignmentBlockSize, dateCreated, dateUpdated string) error {
	return writer.Write([]string{oui, isPrivate, companyName, companyAddress, countryCode, assignmentBlockSize, dateCreated, dateUpdated})
}
