package export

import (
	"encoding/xml"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/stats"
	"macaddress_io_grabber/utils/database"
	"os"
	"sort"
	"strings"
)

func ToXML(filename string) (err error) {

	var records map[string]map[int64]*database.OUI

	records, err = stats.FetchRecordsMapWithReserve()

	var writer *xml.Encoder

	if filename != "" {
		file, err := createFile(filename)
		if err != nil {
			return err
		}
		writer = xml.NewEncoder(file)
	} else {
		writer = xml.NewEncoder(os.Stdout)
	}

	defer writer.Flush()

	headerName := xml.Name{Local: "records"}
	elementName := xml.Name{Local: "record"}

	writer.Indent("", "  ")

	err = writer.EncodeToken(xml.StartElement{
		Name: headerName,
	})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = writer.Flush()
		}
	}()

	var encodeElement = func(i interface{}, name string) error {
		return writer.EncodeElement(i, xml.StartElement{Name: xml.Name{Local: name}})
	}

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

		var isPrivate uint
		if strings.ToLower(last.Org.NName) == "private" {
			isPrivate = 1
		}

		var dateCreated string
		if first.CreatedAt != nil {
			dateCreated = first.CreatedAt.Format("2006-01-02")
		}

		var dateUpdated string
		if last.CreatedAt != nil {
			dateUpdated = last.CreatedAt.Format("2006-01-02")
		}

		err = writer.EncodeToken(xml.StartElement{
			Name: elementName,
		})
		if err != nil {
			return
		}

		err = encodeElement(mac.Format(), "oui")
		if err != nil {
			return
		}
		err = encodeElement(isPrivate, "isPrivate")
		if err != nil {
			return
		}
		err = encodeElement(last.Org.NName, "companyName")
		if err != nil {
			return
		}
		err = encodeElement(last.RawOrgAddress, "companyAddress")
		if err != nil {
			return
		}
		err = encodeElement(last.Country, "countryCode")
		if err != nil {
			return
		}
		err = encodeElement(last.Type, "assignmentBlockSize")
		if err != nil {
			return
		}
		err = encodeElement(dateCreated, "dateCreated")
		if err != nil {
			return
		}
		err = encodeElement(dateUpdated, "dateUpdated")
		if err != nil {
			return
		}

		err = writer.EncodeToken(xml.EndElement{Name: elementName})
	}

	err = writer.EncodeToken(xml.EndElement{Name: headerName})

	return
}

func ToVendorMacXML(filename string) (err error) {

	var records = make([]database.OUI, 0)

	if db := database.Instance.Where("id = ref").Find(&records); db.Error != nil {
		return db.Error
	}

	var writer *xml.Encoder

	if filename != "" {
		file, err := createFile(filename)
		if err != nil {
			return err
		}
		writer = xml.NewEncoder(file)
	} else {
		writer = xml.NewEncoder(os.Stdout)
	}

	defer writer.Flush()

	headerName := xml.Name{Local: "MacAddressVendorMappings", Space: "http://www.cisco.com/server/spt"}
	elementName := xml.Name{Local: "VendorMapping"}

	writer.Indent("", "  ")

	err = writer.EncodeToken(xml.StartElement{
		Name: headerName,
	})
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = writer.Flush()
		}
	}()

	for i := range records {

		if err = records[i].FindOrg(); err != nil {
			return err
		}

		var mac models.MAC

		mac, err = models.NewMac(records[i].Assignment)
		if err != nil {
			return err
		}

		err = writer.EncodeElement("", xml.StartElement{
			Name: elementName,
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "mac_prefix"}, Value: mac.Format()},
				{Name: xml.Name{Local: "vendor_name"}, Value: records[i].Org.NName},
			},
		})
		if err != nil {
			return err
		}
	}

	err = writer.EncodeToken(xml.EndElement{Name: headerName})

	return
}
