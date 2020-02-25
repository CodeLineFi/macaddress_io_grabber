package auto

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"io"
	"io/ioutil"
	"macaddress_io_grabber/config"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/download"
	"macaddress_io_grabber/utils/database"
	"macaddress_io_grabber/utils/mail"
	"macaddress_io_grabber/utils/redispool"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

func Check(config *config.Config, d time.Time) (err error) {

	defer func() {
		if err != nil {
			mail.SendMail(config, err.Error())
			err = nil
		}
		if i := recover(); i != nil {

			if e, ok := i.(error); ok {
				err = e
			} else {
				err = errors.New(fmt.Sprint(i))
			}

			mail.SendMail(config, "Panicked with: "+err.Error()+"\n"+string(debug.Stack()))
		}
	}()

	var today = d.Format("2006-01-02")

	if config.AutoUpdate.DirSource == "" {
		return errors.New("source directory in config is not specified")
	}
	sDir := filepath.Join(config.AutoUpdate.DirSource, today)

	var records = make([]database.OUI, 0)
	var recordsMap = make(map[string]*database.OUI)
	var vmRanges = make([]database.VMRange, 0)
	var appRanges = make([]database.ApplicationRange, 0)

	if db := database.Instance.Where("id = ref").Find(&records); db.Error != nil {
		return db.Error
	}

	if db := database.Instance.Find(&vmRanges); db.Error != nil {
		return db.Error
	}

	if db := database.Instance.Find(&appRanges); db.Error != nil {
		return db.Error
	}

	warnings := make([]string, 0)

	var isAnyUpdate bool
	var cannotFindOrgs = 0

	for i := range records {

		if err = records[i].FindOrg(); err != nil {
			if cannotFindOrgs < 10 {
				warnings = append(warnings, "Cannot find organization for "+records[i].Assignment)
			}
			if cannotFindOrgs == 10 {
				warnings = append(warnings, "Cannot find more than 10 organizations")
			}
			cannotFindOrgs++
			continue
		}

		if err = records[i].FindOrg(); err != nil {
			return err
		}

		recordsMap[records[i].Assignment] = &records[i]

		if !isAnyUpdate && records[i].CreatedAt != nil && records[i].CreatedAt.Format("2006-01-02") == today {
			isAnyUpdate = true
		}
	}

	if !isAnyUpdate {
		warnings = append(warnings, "There's no updated records today")
	}

	files := []string{
		download.MalFile,
		download.MamFile,
		download.MasFile,
		download.IabFile,
		download.CidFile,
	}

	for _, f := range files {
		if stat, err := os.Stat(filepath.Join(sDir, f)); err != nil {
			warnings = append(warnings, err.Error())
		} else {
			if stat.Size() == 0 {
				warnings = append(warnings, "Source file "+f+" is empty")
			}
		}
	}

	if config.AutoUpdate.ExportToRedis {
		warnings = append(warnings, testRedis(
			config,
			records,
			vmRanges,
			appRanges,
		)...)
	}

	files = []string{
		config.AutoUpdate.ResultJSON,
		config.AutoUpdate.ResultXML,
		config.AutoUpdate.ResultCisco,
		config.AutoUpdate.ResultCSV,
	}

	for _, f := range files {
		if f == "" {
			continue
		}
		if stat, err := os.Stat(f); err != nil {
			warnings = append(warnings, err.Error())
		} else {
			if stat.Size() == 0 {
				warnings = append(warnings, "File "+f+" is empty")
			}
		}
	}

	if config.AutoUpdate.ResultJSON != "" {
		if warns := testJSON(config.AutoUpdate.ResultJSON, recordsMap); len(warns) > 0 {
			warnings = append(warnings, warns...)
		}
	}
	if config.AutoUpdate.ResultCSV != "" {
		if warns := testCSV(config.AutoUpdate.ResultCSV, recordsMap); len(warns) > 0 {
			warnings = append(warnings, warns...)
		}
	}
	if config.AutoUpdate.ResultXML != "" {
		if warns := testXML(config.AutoUpdate.ResultXML, recordsMap); len(warns) > 0 {
			warnings = append(warnings, warns...)
		}
	}
	if config.AutoUpdate.ResultCisco != "" {
		if warns := testCisco(config.AutoUpdate.ResultCisco, recordsMap); len(warns) > 0 {
			warnings = append(warnings, warns...)
		}
	}

	if len(warnings) != 0 {
		return errors.New(strings.Join(warnings, "\n"))
	}

	if err := removeOldFiles(config.AutoUpdate.DirSource, d, config.AutoUpdate.StoreDays); err != nil {
		warnings = append(warnings, "Cannot remove old files: "+err.Error())
	}

	if config.AutoUpdate.DirServer != "" {
		if config.AutoUpdate.ResultJSON != "" {
			_, file := filepath.Split(config.AutoUpdate.ResultJSON)
			err := moveFile(config.AutoUpdate.ResultJSON, filepath.Join(config.AutoUpdate.DirServer, file))
			if err != nil {
				warnings = append(warnings, err.Error())
			}
		}
		if config.AutoUpdate.ResultCSV != "" {
			_, file := filepath.Split(config.AutoUpdate.ResultCSV)
			err := moveFile(config.AutoUpdate.ResultCSV, filepath.Join(config.AutoUpdate.DirServer, file))
			if err != nil {
				warnings = append(warnings, err.Error())
			}
		}
		if config.AutoUpdate.ResultXML != "" {
			_, file := filepath.Split(config.AutoUpdate.ResultXML)
			err := moveFile(config.AutoUpdate.ResultXML, filepath.Join(config.AutoUpdate.DirServer, file))
			if err != nil {
				warnings = append(warnings, err.Error())
			}
		}
		if config.AutoUpdate.ResultCisco != "" {
			_, file := filepath.Split(config.AutoUpdate.ResultCisco)
			err := moveFile(config.AutoUpdate.ResultCisco, filepath.Join(config.AutoUpdate.DirServer, file))
			if err != nil {
				warnings = append(warnings, err.Error())
			}
		}
	}

	if len(warnings) != 0 {
		return errors.New(strings.Join(warnings, "\n"))
	}

	return nil
}

func moveFile(old, new string) error {
	c := exec.Command("mv", old, new)

	var stderr io.ReadCloser
	if se, err := c.StderrPipe(); err == nil {
		stderr = se
	} else {
		return err
	}

	c.Stdout = nil

	if err := c.Start(); err != nil {
		return err
	}

	slurp, _ := ioutil.ReadAll(stderr)

	if err := c.Wait(); err != nil {
		return errors.New(err.Error() + ": " + string(slurp))
	}

	return nil
}

func removeOldFiles(dir string, d time.Time, storeDays int) error {

	matchDate := regexp.MustCompile(`(\d\d\d\d-\d\d-\d\d)`)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if str := matchDate.FindString(file.Name()); str != "" {
			t, _ := time.ParseInLocation("2006-01-02", str, time.UTC)
			if d.AddDate(0, 0, -storeDays).After(t) {
				if err := os.RemoveAll(path.Join(dir, file.Name())); err != nil {
					return err
				}
			}
			continue
		}
	}

	return nil
}

func testRedis(config *config.Config, records []database.OUI, vmRanges []database.VMRange, appRanges []database.ApplicationRange) []string {

	warnings := make([]string, 0)

	if len(config.Redis.Databases) != 2 || config.Redis.Databases[0] == config.Redis.Databases[1] {
		warnings = append(warnings,
			"list of redis databases has to contain 2 unique elements, have "+fmt.Sprint(config.Redis.Databases))
		return warnings
	}

	var redisPool = redispool.New(config.Redis.ConnString, redispool.TagTarget, "mac_api", 10, 10, config.Redis.Databases)

	var client = redisPool.Get()
	defer client.Close()

	dbInfoList, err := redisPool.GetDatabasesInfo()
	if err != nil {
		warnings = append(warnings, err.Error())
		return warnings
	}

	if dbInfoList.Candidate < 0 {
		warnings = append(warnings, "There's no release candidate in redis database")
		return warnings
	}

	if err := selectDB(client, dbInfoList.Candidate); err != nil {
		warnings = append(warnings, err.Error())
		return warnings
	}

	var findOUI = func(conn redis.Conn, s string) (*models.OUIRedis, error) {
		str, err := redis.String(conn.Do("HGET", "records", s))
		if err != nil {
			return nil, err
		}
		if len(str) == 0 {
			return nil, nil
		}

		oui := &models.OUIRedis{}

		err = json.Unmarshal([]byte(str), oui)
		if err != nil {
			return nil, err
		}

		return oui, nil
	}

	for i := range records {

		if oui, err := findOUI(client, records[i].Assignment); err != nil {
			if len(warnings) > 10 {
				break
			}
			warnings = append(warnings, "Cannot find "+records[i].Assignment+" "+err.Error())
		} else {

			var isPrivate bool
			if strings.ToLower(records[i].Org.NName) == "private" {
				isPrivate = true
			}

			var dateUpdated string
			if records[i].CreatedAt != nil {
				dateUpdated = records[i].CreatedAt.Format("2006-01-02")
			}

			var mismatch = true

			const mismatchRecords = "Mismatch records from redis and database for %s where %s != %s"

			switch {
			case oui.CountryCode != records[i].Country:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.CountryCode, records[i].Country))
			case oui.CompanyName != records[i].Org.NName:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.CompanyName, records[i].Org.NName))
			case oui.IsPrivate != isPrivate:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), strconv.FormatBool(oui.IsPrivate), strconv.FormatBool(isPrivate)))
			case oui.DateUpdated != dateUpdated:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.DateUpdated, dateUpdated))
			case oui.CompanyAddress != records[i].RawOrgAddress:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.CompanyAddress, records[i].RawOrgAddress))
			case oui.AssignmentBlockSize != records[i].Type:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.AssignmentBlockSize, records[i].Type))
			case oui.Assignment.String() != records[i].Assignment:
				warnings = append(warnings,
					fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.Assignment.String(), records[i].Assignment))
			default:
				mismatch = false
			}

			if mismatch {
				break
			}
		}
	}

	vmRangesWarnings := testVmRanges(client, vmRanges)
	appRangesWarnings := testAppRanges(client, appRanges)
	warnings = append(warnings, vmRangesWarnings...)
	warnings = append(warnings, appRangesWarnings...)

	if len(warnings) == 0 {
		err := redisPool.UpdateTags(dbInfoList.Candidate, -1)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
	}

	return warnings
}

func testVmRanges(client redis.Conn, ranges []database.VMRange) []string {
	warnings := make([]string, 0)

	const mismatchRecords = "Mismatch records from redis and database for %s where %s != %s"

	var stringInSlice = func(needle string, haystack []string) bool {
		for _, str := range haystack {
			if strings.Contains(str, needle) {
				return true
			}
		}
		return false
	}

	leftBordersRedis, err :=
		redis.Strings(client.Do("HGETALL", models.RedisVmRangeSearchTable))

	if err != nil {
		warnings = append(warnings,
			fmt.Sprintf("Can't retrieve hash %s from Redis", models.RedisVmRangeSearchTable))
		return warnings
	}

	for _, vmRangeDb := range ranges {
		dbRightBorder, err := models.NewMac(vmRangeDb.RightBorder)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Can't parse vm range %s in DB: ", vmRangeDb.RightBorder))
			break
		}
		dbLeftBorder, err := models.NewMac(vmRangeDb.LeftBorder)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Can't parse vm range %s in DB: ", vmRangeDb.LeftBorder))
			break
		}

		slice, err :=
			redis.String(client.Do("HGET", models.RedisVmRangeHash, strconv.FormatUint(vmRangeDb.ID, 10)))

		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("VM range with ID %s not found. Error: %s", strconv.FormatUint(vmRangeDb.ID, 10), err.Error()))
			break
		}

		vmRangeRedis := &models.VMRangeRedis{}
		err = json.Unmarshal([]byte(slice), vmRangeRedis)

		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Can't unmarshal %v", vmRangeRedis))
			break
		}

		if vmRangeRedis.RBorder.String() != dbRightBorder.String() {
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, string(vmRangeDb.ID), vmRangeRedis.RBorder, vmRangeDb.RightBorder))
			break
		}
		if vmRangeRedis.LBorder.String() != dbLeftBorder.String() {
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, string(vmRangeDb.ID), vmRangeRedis.LBorder, vmRangeDb.LeftBorder))
			break
		}
		if vmRangeRedis.VirtualMachineName != vmRangeDb.VirtualMachineName {
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, string(vmRangeDb.ID), vmRangeRedis.VirtualMachineName, vmRangeDb.VirtualMachineName))
			break
		}

		if !stringInSlice(vmRangeRedis.LBorder.String(), leftBordersRedis) {
			warnings = append(warnings,
				fmt.Sprintf("Left border %s not found in search hash", vmRangeRedis.LBorder.String()))
			break
		}
	}

	return warnings
}

func testAppRanges(client redis.Conn, ranges []database.ApplicationRange) []string {
	warnings := make([]string, 0)

	const mismatchRecords = "Mismatch records from redis and database for %s where %s != %s"

	var stringInSlice = func(needle string, haystack []string) bool {
		for _, str := range haystack {
			if strings.Contains(str, needle) {
				return true
			}
		}
		return false
	}

	leftBordersRedis, err :=
		redis.Strings(client.Do("HGETALL", models.RedisAppRangeSearchTable))

	if err != nil {
		warnings = append(warnings,
			fmt.Sprintf("Can't retrieve hash %s from Redis", models.RedisAppRangeSearchTable))
		return warnings
	}

	for _, appRangeDb := range ranges {
		dbRightBorder, err := models.NewMac(appRangeDb.RightBorder)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Can't parse vm range %s in DB: ", appRangeDb.RightBorder))
			break
		}
		dbLeftBorder, err := models.NewMac(appRangeDb.LeftBorder)
		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Can't parse application range %s in DB: ", appRangeDb.LeftBorder))
			break
		}

		slice, err :=
			redis.String(client.Do("HGET", models.RedisAppRangeHash, strconv.FormatUint(appRangeDb.ID, 10)))

		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Application range with ID %s not found. Error: %s", strconv.FormatUint(appRangeDb.ID, 10), err.Error()))
			break
		}

		appRangeRedis := &models.ApplicationRangeRedis{}
		err = json.Unmarshal([]byte(slice), appRangeRedis)

		if err != nil {
			warnings = append(warnings,
				fmt.Sprintf("Can't unmarshal %v", appRangeRedis))
			break
		}

		if appRangeRedis.RBorder.String() != dbRightBorder.String() {
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, string(appRangeDb.ID), appRangeRedis.RBorder, appRangeDb.RightBorder))
			break
		}
		if appRangeRedis.LBorder.String() != dbLeftBorder.String() {
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, string(appRangeDb.ID), appRangeRedis.LBorder, appRangeDb.LeftBorder))
			break
		}
		if appRangeRedis.Application != appRangeDb.Application {
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, string(appRangeDb.ID), appRangeRedis.Application, appRangeDb.Application))
			break
		}

		if !stringInSlice(appRangeRedis.LBorder.String(), leftBordersRedis) {
			warnings = append(warnings,
				fmt.Sprintf("Left border %s not found in search hash", appRangeRedis.LBorder.String()))
			break
		}
	}

	return warnings
}

func testJSON(filename string, recordsMap map[string]*database.OUI) []string {

	warnings := make([]string, 0)

	f, err := os.Open(filename)
	if err != nil {
		warnings = append(warnings, err.Error())
		return warnings
	}

	defer f.Close()

	var str = ""

	reader := bufio.NewScanner(f)

	counter := 0

	for reader.Scan() {

		counter++

		str = reader.Text()

		oui := models.OUIJson{}

		if err := json.Unmarshal([]byte(str), &oui); err != nil {
			warnings = append(warnings, "Cannot parse JSON:"+err.Error())
			break
		}
		if oui.Assignment.String() == "" {
			warnings = append(warnings, "OUI Assignment is empty string")
			break
		}

		record := recordsMap[oui.Assignment.String()]

		var isPrivate bool
		if strings.ToLower(record.Org.NName) == "private" {
			isPrivate = true
		}

		var dateUpdated string
		if record.CreatedAt != nil {
			dateUpdated = record.CreatedAt.Format("2006-01-02")
		}

		var mismatch = true

		const mismatchRecords = "Mismatch records from json and database for %s where %s != %s"

		switch {
		case oui.CountryCode != record.Country:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.CountryCode, record.Country))
		case oui.CompanyName != record.Org.NName:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.CompanyName, record.Org.NName))
		case oui.IsPrivate != isPrivate:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), strconv.FormatBool(oui.IsPrivate), strconv.FormatBool(isPrivate)))
		case oui.DateUpdated != dateUpdated:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.DateUpdated, dateUpdated))
		case oui.CompanyAddress != record.RawOrgAddress:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.CompanyAddress, record.RawOrgAddress))
		case oui.AssignmentBlockSize != record.Type:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.AssignmentBlockSize, record.Type))
		case oui.Assignment.String() != record.Assignment:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, oui.Assignment.String(), oui.Assignment.String(), record.Assignment))
		default:
			mismatch = false
		}

		if mismatch {
			break
		}
	}

	if counter != len(recordsMap) {
		warnings = append(warnings, "Count of records in json doesn't match database")
	}

	return warnings
}

func testCSV(filename string, recordsMap map[string]*database.OUI) []string {

	warnings := make([]string, 0)

	f, err := os.Open(filename)
	if err != nil {
		warnings = append(warnings, err.Error())
		return warnings
	}

	defer f.Close()

	var str []string

	reader := csv.NewReader(f)

	var skipHeader = true
	var counter = 0

	for {
		if str, err = reader.Read(); err != nil {
			if err != io.EOF {
				warnings = append(warnings, err.Error())
			}
			break
		}

		if skipHeader {
			skipHeader = false
			continue
		}

		counter++

		mac, err := models.NewMac(str[0])
		if err != nil {
			warnings = append(warnings, err.Error())
			break
		}

		if len(str) != 8 {
			warnings = append(warnings, "CSV fields isn't correct for:"+str[0])
		}

		record := recordsMap[mac.String()]

		var isPrivate = "0"
		if strings.ToLower(record.Org.NName) == "private" {
			isPrivate = "1"
		}

		var dateUpdated string
		if record.CreatedAt != nil {
			dateUpdated = record.CreatedAt.Format("2006-01-02")
		}

		var mismatch = true

		const mismatchRecords = "Mismatch records from csv and database for %s where %s != %s"

		switch {
		case str[0] != mac.Format():
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[0], mac.Format()))
		case str[1] != isPrivate:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[1], isPrivate))
		case str[2] != record.Org.NName:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[2], record.Org.NName))
		case str[3] != record.RawOrgAddress:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[3], record.RawOrgAddress))
		case str[4] != record.Country:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[4], record.Country))
		case str[5] != record.Type:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[5], record.Type))
		case str[7] != dateUpdated:
			warnings = append(warnings,
				fmt.Sprintf(mismatchRecords, mac.String(), str[6], dateUpdated))
		default:
			mismatch = false
		}

		if mismatch {
			break
		}
	}

	if counter != len(recordsMap) {
		warnings = append(warnings, "Count of records in csv doesn't match database")
	}

	return warnings
}

func testXML(filename string, recordsMap map[string]*database.OUI) []string {

	warnings := make([]string, 0)

	f, err := os.Open(filename)
	if err != nil {
		warnings = append(warnings, err.Error())
		return warnings
	}

	defer f.Close()

	var token xml.Token

	reader := xml.NewDecoder(f)

	reader.Strict = true

	var depth = 0
	var counter = 0
	var oui = map[string]string{}

	for {

		token, err = reader.Token()
		if err != nil {
			if err != io.EOF {
				warnings = append(warnings, "Cannot parse xml: "+err.Error())
			}
			break
		}

		switch e := token.(type) {
		case xml.StartElement:
			switch depth {
			case 0:
				t := xml.StartElement{
					Name: xml.Name{Local: "records"},
				}
				if !reflect.DeepEqual(e.Name, t.Name) {
					warnings = append(warnings, "depth = 0 "+fmt.Sprint(e.Name, " != ", t.Name))
					return warnings
				}
				if len(e.Attr)+len(t.Attr) != 0 && !reflect.DeepEqual(e.Attr, t.Attr) {
					warnings = append(warnings, "depth = 0 "+fmt.Sprint(e.Attr, " != ", t.Attr))
					return warnings
				}
				depth++
			case 1:
				t := xml.StartElement{
					Name: xml.Name{Local: "record"},
				}
				if !reflect.DeepEqual(e.Name, t.Name) {
					warnings = append(warnings, "depth = 1 "+fmt.Sprint(e.Name, " != ", t.Name))
					return warnings
				}
				if len(e.Attr)+len(t.Attr) != 0 && !reflect.DeepEqual(e.Attr, t.Attr) {
					warnings = append(warnings, "depth = 1 "+fmt.Sprint(e.Attr, " != ", t.Attr))
					return warnings
				}
				oui = map[string]string{}
				depth++
			case 2:
				var str string
				err := reader.DecodeElement(&str, &e)
				if err != nil {
					warnings = append(warnings, "depth = 2: "+err.Error())
					return warnings
				}
				oui[e.Name.Local] = str
			}
		case xml.EndElement:
			if depth == 2 {
				mac, err := models.NewMac(oui["oui"])
				if err != nil {
					warnings = append(warnings, "Cannot parse oui in xml: "+err.Error())
					return warnings
				}

				record := recordsMap[mac.String()]

				var isPrivate = "0"
				if strings.ToLower(record.Org.NName) == "private" {
					isPrivate = "1"
				}

				var dateUpdated string
				if record.CreatedAt != nil {
					dateUpdated = record.CreatedAt.Format("2006-01-02")
				}

				var mismatch = true

				const mismatchRecords = "Mismatch records from xml and database for %s where %s != %s"

				switch {
				case oui["oui"] != mac.Format():
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["oui"], mac.Format()))
				case oui["isPrivate"] != isPrivate:
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["isPrivate"], isPrivate))
				case oui["companyName"] != record.Org.NName:
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["companyName"], record.Org.NName))
				case oui["companyAddress"] != record.RawOrgAddress:
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["companyAddress"], record.RawOrgAddress))
				case oui["countryCode"] != record.Country:
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["countryCode"], record.Country))
				case oui["assignmentBlockSize"] != record.Type:
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["assignmentBlockSize"], record.Type))
				case oui["dateUpdated"] != dateUpdated:
					warnings = append(warnings,
						fmt.Sprintf(mismatchRecords, mac.String(), oui["dateUpdated"], dateUpdated))
				default:
					mismatch = false
				}

				if mismatch {
					return warnings
				}
				counter++
			}
			depth--
		}
	}

	if depth != 0 {
		warnings = append(warnings, "Wrong xml structure")
	}

	if counter != len(recordsMap) {
		warnings = append(warnings, "Count of records in xml doesn't match database")
	}

	return warnings
}

func testCisco(filename string, recordsMap map[string]*database.OUI) []string {

	warnings := make([]string, 0)

	f, err := os.Open(filename)
	if err != nil {
		warnings = append(warnings, err.Error())
		return warnings
	}

	defer f.Close()

	var token xml.Token

	reader := xml.NewDecoder(f)

	reader.Strict = true

	var depth = 0
	var counter = 0

	for {

		token, err = reader.Token()
		if err != nil {
			if err != io.EOF {
				warnings = append(warnings, "Cannot parse xml: "+err.Error())
			}
			break
		}

		const ns = "http://www.cisco.com/server/spt"

		switch e := token.(type) {
		case xml.StartElement:
			switch depth {
			case 0:
				t := xml.StartElement{
					Name: xml.Name{Local: "MacAddressVendorMappings", Space: ns},
					Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: ns}},
				}
				if !reflect.DeepEqual(e.Name, t.Name) {
					warnings = append(warnings, "depth = 0 "+fmt.Sprint(e.Name, " != ", t.Name))
					return warnings
				}
				if len(e.Attr)+len(t.Attr) != 0 && !reflect.DeepEqual(e.Attr, t.Attr) {
					warnings = append(warnings, "depth = 0 "+fmt.Sprint(e.Attr, " != ", t.Attr))
					return warnings
				}
			case 1:
				oui := ""
				if len(e.Attr) == 0 {
					warnings = append(warnings, "Vendor mapping should has attributes: "+err.Error())
					return warnings
				}
				oui = e.Attr[0].Value
				mac, err := models.NewMac(oui)
				if err != nil {
					warnings = append(warnings, "Cannot parse mac in xml: "+err.Error())
					return warnings
				}

				record := recordsMap[mac.String()]

				t := xml.StartElement{
					Name: xml.Name{Local: "VendorMapping", Space: ns},
					Attr: []xml.Attr{
						{Name: xml.Name{Local: "mac_prefix"}, Value: mac.Format()},
						{Name: xml.Name{Local: "vendor_name"}, Value: record.Org.NName},
					},
				}

				if !reflect.DeepEqual(e.Name, t.Name) {
					warnings = append(warnings, "depth = 1 "+fmt.Sprint(e.Name, " != ", t.Name))
					return warnings
				}
				if len(e.Attr)+len(t.Attr) != 0 && !reflect.DeepEqual(e.Attr, t.Attr) {
					warnings = append(warnings, "depth = 1 "+fmt.Sprint(e.Attr, " != ", t.Attr))
					return warnings
				}
				counter++
			}
			depth++
		case xml.EndElement:
			depth--
		}
	}

	if depth != 0 {
		warnings = append(warnings, "Wrong xml structure")
	}

	if counter != len(recordsMap) {
		warnings = append(warnings, "Count of records in xml doesn't match database")
	}

	return warnings
}
