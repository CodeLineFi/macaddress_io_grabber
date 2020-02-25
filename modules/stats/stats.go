package stats

import (
	"fmt"
	"log"
	"macaddress_io_grabber/models"
	"macaddress_io_grabber/utils/database"
	"runtime"
	"sort"
	"time"
)

func Calculate() error {

	records, err := FetchRecordsMap()
	if err != nil {
		return err
	}

	BreakdownsAll(records)

	return nil
}

func usedSpace(s string) uint64 {
	switch len(s) {
	case 6:
		return 1099511627776
	case 7:
		return 68719476736
	case 9:
		return 268435456
	default:
		return 0
	}
}

func FetchRecords() ([]database.OUI, error) {

	var records = make([]database.OUI, 0)

	if err := database.Instance.Where("ref = id").Find(&records).Error; err != nil {
		return nil, err
	}

	for i := range records {
		if err := records[i].FindOrg(); err != nil {
			return nil, err
		}
	}

	return records, nil
}

func FetchRecordsMap() (map[string]map[int64]*database.OUI, error) {

	var recordsMap, err = FetchRecordsMapWithReserve()

	if err != nil {
		return recordsMap, err
	}

	for assignment := range recordsMap {
		if len(assignment) == 9 {
			delete(recordsMap, assignment[0:7])
			delete(recordsMap, assignment[0:6])
		}
	}
	for assignment := range recordsMap {
		if len(assignment) == 7 {
			delete(recordsMap, assignment[0:6])
		}
	}

	return recordsMap, nil
}

func FetchRecordsMapWithReserve() (map[string]map[int64]*database.OUI, error) {

	var records = make([]database.OUI, 0)

	if err := database.Instance.Find(&records).Error; err != nil {
		return nil, err
	}

	for i := range records {
		if err := records[i].FindOrg(); err != nil {
			return nil, err
		}
	}

	var recordsMap = map[string]map[int64]*database.OUI{}

	for i := range records {
		if _, ok := recordsMap[records[i].Assignment]; !ok {
			recordsMap[records[i].Assignment] = map[int64]*database.OUI{}
		}

		var date int64

		if records[i].CreatedAt != nil {
			y, m, d := records[i].CreatedAt.Date()
			date = time.Date(y, m, d, 0, 0, 0, 0, records[i].CreatedAt.Location()).Unix()
		}

		recordsMap[records[i].Assignment][date] = &records[i]
	}

	return recordsMap, nil
}

func BreakdownsAll(recordsMap map[string]map[int64]*database.OUI) {

	var GlobalBreakdowns = GlobalBreakdown(recordsMap)
	var CountryBreakdowns = CountryBreakdown(recordsMap)
	var OrganizationBreakdowns = OrganizationBreakdown(recordsMap)

	log.Println("total orgs:", GlobalBreakdowns.TotalOrganizations)
	log.Println("total blocks:", GlobalBreakdowns.TotalMacBlocks)
	log.Println("new blocks:", GlobalBreakdowns.NewMacBlocks)
	log.Println("private blocks:", GlobalBreakdowns.PrivateBlocks)
	log.Println("used space:", GlobalBreakdowns.UsedSpace)
	log.Println("used space:", fmt.Sprintf("%.2f", float64(GlobalBreakdowns.UsedSpace)/float64(1<<64)*100))

	var keys []int64
	for k := range GlobalBreakdowns.Dynamics {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		log.Println(k/10000, k%10000/100, k%100, GlobalBreakdowns.Dynamics[k].Created, GlobalBreakdowns.Dynamics[k].Changed)
	}

	log.Println("Countries:", len(CountryBreakdowns))

	for id, c := range CountryBreakdowns {
		if id != "US" && id != "RU" {
			continue
		}

		log.Println(id, "total orgs:", c.TotalOrganizations)
		for k := range c.Organizations {
			if k < 100 || k > 105 {
				continue
			}
			log.Println(id, k, c.Organizations[k].ID, c.Organizations[k].Name, c.Organizations[k].Blocks)
		}
		log.Println(id, "total blocks:", c.TotalMacBlocks)
		for k := range c.Dynamics {
			log.Println(id, k, c.Dynamics[k].Date, c.Dynamics[k].Created, c.Dynamics[k].Removed, c.Dynamics[k].Changed)
		}
	}

	log.Println("Organiazation:", len(OrganizationBreakdowns))

	for id, c := range OrganizationBreakdowns {
		if id < 100 || id > 105 {
			continue
		}

		log.Println(id, "total countries:", c.TotalCountries)
		for k := range c.Countries {
			if c.Countries[k].ID != "US" && c.Countries[k].ID != "RU" {
				continue
			}
			log.Println(id, k, c.Countries[k].ID, c.Countries[k].Blocks)
		}
		log.Println(id, "total blocks:", c.TotalMacBlocks)
		for k := range c.Dynamics {
			log.Println(id, k, c.Dynamics[k].Date, c.Dynamics[k].Created, c.Dynamics[k].Removed, c.Dynamics[k].Changed)
		}
	}

	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	log.Println(mem.Alloc/1024/1024, "Mb")
}

func GlobalBreakdown(recordsMap map[string]map[int64]*database.OUI) models.Breakdown {

	var breakdowns = models.Breakdown{Dynamics: map[int64]*models.DateValue{}}
	var organizations = map[uint64]struct{}{}
	var countries = map[string]struct{}{}

	var weekAgo = time.Now().AddDate(0, 0, -7)
	weekAgo = time.Date(weekAgo.Year(), weekAgo.Month(), weekAgo.Day(), 0, 0, 0, 0, time.UTC)

	for _, recordByDates := range recordsMap {

		var keys = make([]int64, 0, len(recordByDates))
		for k := range recordByDates {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		var prev *database.OUI

		for i, date := range keys {

			r := recordByDates[date]

			if _, ok := breakdowns.Dynamics[date]; !ok {
				breakdowns.Dynamics[date] = &models.DateValue{Date: date}
			}
			// Appeared first time
			if i == 0 {
				breakdowns.Dynamics[date].Created++
				if r.CreatedAt != nil && r.CreatedAt.After(weekAgo) {
					breakdowns.NewMacBlocks++
				}
			} else {
				breakdowns.Dynamics[date].Changed++
			}

			organizations[r.OrgID] = struct{}{}
			countries[r.Country] = struct{}{}

			prev = r
		}

		if prev.OrgText == "private" {
			breakdowns.PrivateBlocks++
		}

		if u := usedSpace(prev.Assignment); u != 0 {
			breakdowns.UsedSpace += u
		} else {
			log.Println(prev.Assignment)
		}

		breakdowns.Types = addTypeToList(breakdowns.Types, prev.Type)

		breakdowns.TotalMacBlocks++
	}

	sort.Sort(models.TopType(breakdowns.Types))

	delete(countries, "")

	breakdowns.TotalOrganizations = uint64(len(organizations))
	breakdowns.TotalCountries = uint64(len(countries))

	return breakdowns
}

func CountryBreakdown(recordsMap map[string]map[int64]*database.OUI) map[string]*models.CountryBreakdown {

	var breakdowns = map[string]*models.CountryBreakdown{}
	var organizations = map[string]map[uint64]*models.OrgInfo{}

	for _, recordByDates := range recordsMap {

		var keys = make([]int64, 0, len(recordByDates))
		for k := range recordByDates {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		var prev *database.OUI

		for _, date := range keys {

			r := recordByDates[date]

			if _, ok := breakdowns[r.Country]; !ok {
				breakdowns[r.Country] = &models.CountryBreakdown{
					Dynamics: map[int64]*models.DateValue{},
				}
				organizations[r.Country] = map[uint64]*models.OrgInfo{}
			}

			if prev != nil {
				if _, ok := breakdowns[r.Country].Dynamics[date]; !ok {
					breakdowns[r.Country].Dynamics[date] = &models.DateValue{Date: date}
				}
				// If block has been moved from one country to another
				if prev.Country != r.Country {
					if _, ok := breakdowns[prev.Country].Dynamics[date]; !ok {
						breakdowns[prev.Country].Dynamics[date] = &models.DateValue{Date: date}
					}
					breakdowns[prev.Country].Dynamics[date].Removed++
					breakdowns[r.Country].Dynamics[date].Created++
				} else {
					breakdowns[r.Country].Dynamics[date].Changed++
				}
			} else {
				if _, ok := breakdowns[r.Country].Dynamics[date]; !ok {
					breakdowns[r.Country].Dynamics[date] = &models.DateValue{Date: date}
				}
				breakdowns[r.Country].Dynamics[date].Created++
			}

			prev = r
		}

		// Use last record as current info
		breakdowns[prev.Country].TotalMacBlocks++
		if _, ok := organizations[prev.Country][prev.OrgID]; !ok {
			organizations[prev.Country][prev.OrgID] = &models.OrgInfo{
				ID:     prev.OrgID,
				Name:   prev.Org.Name,
				Blocks: 0,
			}
		}
		organizations[prev.Country][prev.OrgID].Blocks++
		breakdowns[prev.Country].UsedSpace += usedSpace(prev.Assignment)

		breakdowns[prev.Country].Types = addTypeToList(breakdowns[prev.Country].Types, prev.Type)
	}

	delete(breakdowns, "")
	delete(organizations, "")

	for code, b := range breakdowns {
		b.Organizations = make(models.TopOrg, len(organizations[code]))
		i := 0

		for _, c := range organizations[code] {
			b.Organizations[i] = *c
			i++
		}
		sort.Sort(models.TopOrg(b.Organizations))
		b.TotalOrganizations = uint64(len(b.Organizations))

		sort.Sort(models.TopType(b.Types))
	}

	return breakdowns
}

func OrganizationBreakdown(recordsMap map[string]map[int64]*database.OUI) map[uint64]*models.OrgBreakdown {

	var breakdowns = map[uint64]*models.OrgBreakdown{}
	var countries = map[uint64]map[string]*models.CountryInfo{}

	for _, recordByDates := range recordsMap {

		var keys = make([]int64, 0, len(recordByDates))
		for k := range recordByDates {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		var prev *database.OUI

		for _, date := range keys {

			r := recordByDates[date]

			if _, ok := breakdowns[r.OrgID]; !ok {
				breakdowns[r.OrgID] = &models.OrgBreakdown{
					Name:     r.Org.NName,
					Dynamics: map[int64]*models.DateValue{},
				}
				countries[r.OrgID] = map[string]*models.CountryInfo{}
			}

			if prev != nil {
				if _, ok := breakdowns[r.OrgID].Dynamics[date]; !ok {
					breakdowns[r.OrgID].Dynamics[date] = &models.DateValue{Date: date}
				}
				if prev.OrgID != r.OrgID {
					if _, ok := breakdowns[r.OrgID].Dynamics[date]; !ok {
						breakdowns[r.OrgID].Dynamics[date] = &models.DateValue{Date: date}
					}
					breakdowns[r.OrgID].Dynamics[date].Removed++
					breakdowns[r.OrgID].Dynamics[date].Created++
				} else {
					breakdowns[r.OrgID].Dynamics[date].Changed++
				}
			} else {
				if _, ok := breakdowns[r.OrgID].Dynamics[date]; !ok {
					breakdowns[r.OrgID].Dynamics[date] = &models.DateValue{Date: date}
				}
				breakdowns[r.OrgID].Dynamics[date].Created++
			}

			prev = r
		}

		// Use last record as current info
		if prev.Country != "" {
			if _, ok := countries[prev.OrgID][prev.Country]; !ok {
				countries[prev.OrgID][prev.Country] = &models.CountryInfo{
					ID:     prev.Country,
					Blocks: 0,
				}
			}
			countries[prev.OrgID][prev.Country].Blocks++
		}
		breakdowns[prev.OrgID].TotalMacBlocks++
		breakdowns[prev.OrgID].UsedSpace += usedSpace(prev.Assignment)

		newAddress := prev.RawOrgAddress
		for _, a := range breakdowns[prev.OrgID].Addresses {
			if a == newAddress {
				newAddress = ""
				break
			}
		}
		if newAddress != "" {
			breakdowns[prev.OrgID].Addresses = append(breakdowns[prev.OrgID].Addresses, newAddress)
		}

		breakdowns[prev.OrgID].Types = addTypeToList(breakdowns[prev.OrgID].Types, prev.Type)
	}

	for code, b := range breakdowns {
		b.Countries = make(models.TopCountry, len(countries[code]))
		i := 0

		for _, c := range countries[code] {
			b.Countries[i] = *c
			i++
		}
		sort.Sort(models.TopCountry(b.Countries))
		b.TotalCountries = uint64(len(b.Countries))

		sort.Sort(models.TopType(b.Types))
	}

	return breakdowns
}

func DateBreakdown(recordsMap map[string]map[int64]*database.OUI) map[int64]*models.DateBreakdown {

	var breakdowns = map[int64]*models.DateBreakdown{}

	for _, recordByDates := range recordsMap {

		var keys = make([]int64, 0, len(recordByDates))
		for k := range recordByDates {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		var prev *database.OUI

		for _, date := range keys {

			r := recordByDates[date]

			if r.CreatedAt != nil {
				if _, ok := breakdowns[date]; !ok {
					breakdowns[date] = &models.DateBreakdown{
						ID:      date,
						Changes: []models.ChangeInfo{},
					}
				}

				macStr := ""
				if mac, err := models.NewMac(r.Assignment); err == nil {
					macStr = mac.Format()
				}

				breakdowns[date].Changes = append(breakdowns[date].Changes,
					models.ChangeInfo{
						New: &models.OuiInfo{
							Assignment:     macStr,
							Type:           r.Type,
							CompanyID:      r.OrgID,
							CompanyName:    r.Org.NName,
							CompanyAddress: r.RawOrgAddress,
							CountryCode:    r.Country,
						},
						Old: nil,
					},
				)

				if prev != nil {
					breakdowns[date].Changes[len(breakdowns[date].Changes)-1].Old = &models.OuiInfo{
						Assignment:     macStr,
						Type:           prev.Type,
						CompanyID:      prev.OrgID,
						CompanyName:    prev.Org.NName,
						CompanyAddress: prev.RawOrgAddress,
						CountryCode:    prev.Country,
					}
				}
			}

			prev = r
		}
	}

	return breakdowns
}

func addTypeToList(types []models.TypeInfo, t string) []models.TypeInfo {
	for i := range types {
		if types[i].ID == t {
			types[i].Blocks++
			return types
		}
	}
	types = append(types, models.TypeInfo{
		ID:     t,
		Blocks: 1,
	})

	return types
}
