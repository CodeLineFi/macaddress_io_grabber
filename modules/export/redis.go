package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/modules/stats"
	"macaddress_io_grabber/utils/database"
	"macaddress_io_grabber/utils/redispool"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	sadd  = "SADD"
	hmset = "HMSET"
)

func ToRedis(redisURL string, redisDBs []int) error {

	if len(redisDBs) != 2 || redisDBs[0] == redisDBs[1] {
		return errors.New("list of redis databases has to contain 2 unique elements, have " + fmt.Sprint(redisDBs))
	}

	pool := redispool.New(redisURL, redispool.TagTarget, "mac_api", 10, 10, redisDBs)

	if err := flushDB(pool.Pool); err != nil {
		return err
	}

	info, err := pool.GetDatabasesInfo()
	if err != nil {
		return err
	}

	err = pool.UpdateTags(info.Production, -1)
	if err != nil {
		return err
	}

	client := pool.Get()
	defer client.Close()

	for _, f := range []func(redis.Conn) error{
		breakdowns,
		vmRanges,
		appRanges,
		wireshark,
		customNotes,
	} {
		err := f(client)
		if err != nil {
			return err
		}
	}

	err = pool.UpdateTags(info.Production, info.Target)
	if err != nil {
		return err
	}

	return nil
}

func brAssignments(recordsMap map[string]map[int64]*database.OUI, conn redis.Conn) (err error) {

	for _, recordByDates := range recordsMap {

		var keys = make([]int64, 0, len(recordByDates))
		for k := range recordByDates {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

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

		history := make([]models.OUIHistRedis, 0, len(keys))

		for _, date := range keys {

			actionDate := ""

			if recordByDates[date].CreatedAt != nil {
				actionDate = recordByDates[date].CreatedAt.Format("2006-01-02")
			}

			history = append(history, models.OUIHistRedis{
				Date:           actionDate,
				CompanyID:      recordByDates[date].Org.ID,
				CompanyName:    recordByDates[date].Org.NName,
				CompanyAddress: recordByDates[date].RawOrgAddress,
				CountryCode:    recordByDates[date].Country,
			})
		}

		var bb []byte
		bb, err = json.Marshal(models.OUIRedis{
			Assignment:          mac,
			IsPrivate:           isPrivate,
			CompanyID:           last.Org.ID,
			CompanyName:         last.Org.NName,
			CompanyAddress:      last.RawOrgAddress,
			AssignmentBlockSize: last.Type,
			CountryCode:         last.Country,
			DateCreated:         dateCreated,
			DateUpdated:         dateUpdated,
			History:             history,
		})

		if err != nil {
			return
		}

		if _, err = conn.Do(
			hmset, "records", last.Assignment, string(bb),
		); err != nil {
			return
		}
	}

	return
}

func brAssignmentsSet(recordsMap map[string]map[int64]*database.OUI, conn redis.Conn) (err error) {

	for oui := range recordsMap {
		if _, err = conn.Do(sadd, "br:glb:ouis", oui); err != nil {
			return
		}
	}

	return
}

func brBasic(global models.Breakdown, conn redis.Conn) error {
	args := redis.Args{}.Add(
		"br:glb:orgs", global.TotalOrganizations,
		"br:glb:countries", global.TotalCountries,
		"br:glb:blocks:total", global.TotalMacBlocks,
		"br:glb:blocks:new", global.NewMacBlocks,
		"br:glb:blocks:private", global.PrivateBlocks,
		"br:glb:usedSpace", math.Round(float64(global.UsedSpace)/float64(1<<64)*10000)/100,
	)

	if str, err := redis.String(conn.Do("MSET", args...)); err != nil || str != "OK" {
		return errors.New(err.Error() + ":" + str)
	}

	return nil
}

func brBasicDates(global models.Breakdown, conn redis.Conn) error {
	args := redis.Args{"br:glb:dates"}

	for _, d := range models.CalcDateValueTotal(global.Dynamics) {
		jsonDate, err := json.Marshal(d)
		if err != nil {
			return err
		}

		args = args.Add(jsonDate)
	}

	_, err := conn.Do("RPUSH", args...)

	return err
}

func brTopCountries(global models.Breakdown, conn redis.Conn) error {
	if len(global.Countries) == 0 {
		return nil
	}

	args := make(redis.Args, 1, len(global.Countries))
	args[0] = "br:glb:top:countries"

	for _, i := range global.Countries {
		jsonDate, err := json.Marshal(i)
		if err != nil {
			return err
		}

		args = append(args, jsonDate)
	}

	_, err := conn.Do("RPUSH", args...)

	return err
}

func brTopOrgs(global models.Breakdown, conn redis.Conn) error {
	args := redis.Args{"br:glb:top:orgs"}

	for _, i := range global.Organizations {
		jsonDate, err := json.Marshal(i)
		if err != nil {
			return err
		}

		args = args.Add(jsonDate)
	}

	_, err := conn.Do("RPUSH", args...)

	return err
}

func brTopTypes(global models.Breakdown, conn redis.Conn) error {
	args := redis.Args{"br:glb:top:types"}

	for _, i := range global.Types {
		jsonDate, err := json.Marshal(i)
		if err != nil {
			return err
		}

		args = args.Add(jsonDate)
	}

	_, err := conn.Do("RPUSH", args...)

	return err
}

func brOrgs(organizations map[uint64]*models.OrgBreakdown, conn redis.Conn) error {
	if len(organizations) == 0 {
		return nil
	}

	args := make(redis.Args, 0, len(organizations))

	for id, org := range organizations {

		s := "br:org:" + strconv.FormatUint(id, 10)

		addresses, err := json.Marshal(org.Addresses)
		if err != nil {
			return err
		}

		args = args.Add(
			s+":name", org.Name,
			s+":blocks:total", org.TotalMacBlocks,
			s+":addresses", addresses,
			s+":countries", org.TotalCountries,
		)
	}

	if _, err := conn.Do("MSET", args...); err != nil {
		return err
	}

	return nil
}

func brOrgsCountries(organizations map[uint64]*models.OrgBreakdown, conn redis.Conn) error {
	for id, org := range organizations {

		if len(org.Countries) == 0 {
			continue
		}

		s := "br:org:" + strconv.FormatUint(id, 10) + ":top:countries"

		args := redis.Args{s}

		for _, c := range org.Countries {
			jsonCountry, err := json.Marshal(c)
			if err != nil {
				return err
			}
			args = args.Add(string(jsonCountry))
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brOrgsDates(organizations map[uint64]*models.OrgBreakdown, conn redis.Conn) error {
	for id, org := range organizations {

		s := "br:org:" + strconv.FormatUint(id, 10) + ":dates"

		args := redis.Args{s}

		for _, v := range models.CalcDateValueTotal(org.Dynamics) {
			jsonDate, err := json.Marshal(v)
			if err != nil {
				return err
			}
			args = args.Add(jsonDate)
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brOrgsTypes(organizations map[uint64]*models.OrgBreakdown, conn redis.Conn) error {
	for id, org := range organizations {

		if len(org.Types) == 0 {
			continue
		}

		args := redis.Args{
			"br:org:" + strconv.FormatUint(id, 10) + ":top:types",
		}

		for _, v := range org.Types {
			jsonDate, err := json.Marshal(v)
			if err != nil {
				return err
			}
			args = args.Add(jsonDate)
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brOrgsRecords(organizations map[uint64]*models.OrgBreakdown, conn redis.Conn) error {

	records, err := stats.FetchRecords()
	if err != nil {
		return err
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CreatedAt == nil {
			if records[j].CreatedAt != nil {
				return false
			}
			return false
		}

		if records[j].CreatedAt == nil {
			return false
		}

		return records[i].CreatedAt.Before(*records[j].CreatedAt)
	})

	l := make([]string, 0, 1000)

	for id, org := range organizations {

		if len(org.Types) == 0 {
			continue
		}

		l = l[:0]

		for i := range records {
			oui := &records[i]

			if oui.OrgID == id {
				l = append(l, oui.Assignment)
			}
		}

		if len(l) == 0 {
			continue
		}

		args := make(redis.Args, 1, len(l)+1)
		args[0] = "br:org:" + strconv.FormatUint(id, 10) + ":records"

		for i := range l {
			args = append(args, l[i])
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brCountries(countries map[string]*models.CountryBreakdown, conn redis.Conn) error {
	if len(countries) == 0 {
		return nil
	}

	args := make(redis.Args, 0, len(countries))

	for id, c := range countries {

		s := "br:cnt:" + id

		args = args.Add(
			s+":blocks:total", c.TotalMacBlocks,
			s+":organizations", c.TotalOrganizations,
		)
	}

	if _, err := conn.Do("MSET", args...); err != nil {
		return err
	}

	return nil
}

func brCountriesOrgs(countries map[string]*models.CountryBreakdown, conn redis.Conn) error {
	for id, c := range countries {

		if len(c.Organizations) == 0 {
			continue
		}

		s := "br:cnt:" + id + ":top:organizations"

		args := redis.Args{s}

		for _, org := range c.Organizations {
			jsonOrg, err := json.Marshal(org)
			if err != nil {
				return err
			}
			args = args.Add(jsonOrg)
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brCountriesDates(countries map[string]*models.CountryBreakdown, conn redis.Conn) error {
	for id, c := range countries {

		s := "br:cnt:" + id + ":dates"

		args := redis.Args{s}

		for _, v := range models.CalcDateValueTotal(c.Dynamics) {
			jsonDate, err := json.Marshal(v)
			if err != nil {
				return err
			}
			args = args.Add(jsonDate)
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brCountriesTypes(countries map[string]*models.CountryBreakdown, conn redis.Conn) error {
	for id, c := range countries {

		if len(c.Types) == 0 {
			continue
		}

		args := redis.Args{
			"br:cnt:" + id + ":top:types",
		}

		for _, v := range c.Types {
			jsonDate, err := json.Marshal(v)
			if err != nil {
				return err
			}
			args = args.Add(jsonDate)
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func brDates(dates map[int64]*models.DateBreakdown, conn redis.Conn) error {
	for id, c := range dates {

		s := "br:dat:" + strconv.FormatInt(id, 10) + ":list"

		args := redis.Args{s}

		for i := range c.Changes {

			oldBytes, err := json.Marshal(c.Changes[i].Old)
			newBytes, err := json.Marshal(c.Changes[i].New)

			if err != nil {
				return err
			}

			args = args.Add(oldBytes, newBytes)
		}

		if _, err := conn.Do("RPUSH", args...); err != nil {
			return err
		}
	}

	return nil
}

func breakdowns(conn redis.Conn) error {

	records, err := stats.FetchRecordsMapWithReserve()
	if err != nil {
		return err
	}

	if err := brAssignments(records, conn); err != nil {
		return errors.New("assignments breakdowns:" + err.Error())
	}

	records, err = stats.FetchRecordsMap()
	if err != nil {
		return err
	}

	if err := brAssignmentsSet(records, conn); err != nil {
		return errors.New("assignments breakdowns:" + err.Error())
	}

	var (
		global        = stats.GlobalBreakdown(records)
		organizations = stats.OrganizationBreakdown(records)
		countries     = stats.CountryBreakdown(records)
		dates         = stats.DateBreakdown(records)
	)

	global.Organizations = make([]models.OrgInfo, 0, global.TotalOrganizations)
	for id, org := range organizations {
		global.Organizations = append(global.Organizations, models.OrgInfo{
			ID:     id,
			Name:   org.Name,
			Blocks: org.TotalMacBlocks,
		})
	}
	sort.Sort(models.TopOrg(global.Organizations))

	global.Countries = make([]models.CountryInfo, 0, global.TotalCountries)
	for id, org := range countries {
		global.Countries = append(global.Countries, models.CountryInfo{
			ID:     id,
			Blocks: org.TotalMacBlocks,
		})
	}
	sort.Sort(models.TopCountry(global.Countries))

	if err := brBasic(global, conn); err != nil {
		return errors.New("basic breakdowns:" + err.Error())
	}
	if err := brBasicDates(global, conn); err != nil {
		return errors.New("dates breakdowns:" + err.Error())
	}

	if err := brTopCountries(global, conn); err != nil {
		return errors.New("dates countries breakdowns:" + err.Error())
	}
	if err := brTopOrgs(global, conn); err != nil {
		return errors.New("dates organizations breakdowns:" + err.Error())
	}
	if err := brTopTypes(global, conn); err != nil {
		return errors.New("dates types breakdowns:" + err.Error())
	}

	if err := brOrgs(organizations, conn); err != nil {
		return errors.New("organizations breakdowns:" + err.Error())
	}
	if err := brOrgsCountries(organizations, conn); err != nil {
		return errors.New("organizations-countries breakdowns:" + err.Error())
	}
	if err := brOrgsDates(organizations, conn); err != nil {
		return errors.New("organizations dates breakdowns:" + err.Error())
	}
	if err := brOrgsTypes(organizations, conn); err != nil {
		return errors.New("organizations types breakdowns:" + err.Error())
	}
	if err := brOrgsRecords(organizations, conn); err != nil {
		return errors.New("organizations records breakdowns:" + err.Error())
	}

	if err := brCountries(countries, conn); err != nil {
		return errors.New("countries breakdowns:" + err.Error())
	}
	if err := brCountriesOrgs(countries, conn); err != nil {
		return errors.New("countries-organizations breakdowns:" + err.Error())
	}
	if err := brCountriesDates(countries, conn); err != nil {
		return errors.New("countries dates breakdowns:" + err.Error())
	}
	if err := brCountriesTypes(countries, conn); err != nil {
		return errors.New("countries types breakdowns:" + err.Error())
	}

	if err := brDates(dates, conn); err != nil {
		return errors.New("dates breakdowns:" + err.Error())
	}

	return nil
}

func flushDB(pool *redis.Pool) error {

	client := pool.Get()
	defer client.Close()

	err := client.Send("FLUSHDB")
	if err != nil {
		return errors.New("cannot flush database: " + err.Error())
	}

	return nil
}

func wireshark(client redis.Conn) error {
	var wiresharkNotes []database.WiresharkNote

	database.Instance.Find(&wiresharkNotes)

	for _, wiresharkNote := range wiresharkNotes {
		if _, err := client.Do(
			hmset, models.RedisWiresharkHash, wiresharkNote.Assignment, wiresharkNote.Note,
		); err != nil {
			return err
		}
	}

	return nil
}

func customNotes(client redis.Conn) error {
	var customNotes []database.CustomNote

	database.Instance.Find(&customNotes)

	for _, customNote := range customNotes {
		if _, err := client.Do(
			hmset, models.RedisCustomNoteskHash, customNote.Assignment, customNote.Note,
		); err != nil {
			return err
		}
	}

	return nil
}
