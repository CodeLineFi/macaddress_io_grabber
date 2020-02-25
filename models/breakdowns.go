package models

import (
	"sort"
)

type Breakdown struct {
	TotalMacBlocks uint64
	NewMacBlocks   uint64
	PrivateBlocks  uint64
	UsedSpace      uint64

	Countries     []CountryInfo
	Organizations []OrgInfo
	Types         []TypeInfo

	TotalOrganizations uint64
	TotalCountries     uint64

	Dynamics map[int64]*DateValue
}

type CountryBreakdown struct {
	TotalMacBlocks uint64
	NewMacBlocks   uint64
	PrivateBlocks  uint64
	UsedSpace      uint64

	Dynamics map[int64]*DateValue

	Types []TypeInfo

	Organizations      []OrgInfo
	TotalOrganizations uint64
}

type OrgBreakdown struct {
	Name      string
	Addresses []string

	TotalMacBlocks uint64
	NewMacBlocks   uint64
	PrivateBlocks  uint64
	UsedSpace      uint64

	Dynamics map[int64]*DateValue

	Types []TypeInfo

	Countries      []CountryInfo
	TotalCountries uint64
}

type DateBreakdown struct {
	ID      int64        `json:"id"`
	Changes []ChangeInfo `json:"changes"`
}

type ChangeInfo struct {
	Old *OuiInfo `json:"old,omitempty"`
	New *OuiInfo `json:"new,omitempty"`
}

type OuiInfo struct {
	Assignment     string `json:"assignment"`
	Type           string `json:"type"`
	CompanyID      uint64 `json:"companyId"`
	CompanyName    string `json:"companyName"`
	CompanyAddress string `json:"companyAddress"`
	CountryCode    string `json:"countryCode"`
}

type OrgInfo struct {
	ID     uint64  `json:"id"`
	Name   string  `json:"name"`
	Blocks uint64  `json:"blocks"`
	Share  float64 `json:"share"`
}

type CountryInfo struct {
	ID     string  `json:"id"`
	Name   string  `json:"name,omitempty"`
	Blocks uint64  `json:"blocks"`
	Share  float64 `json:"share"`
}

type TypeInfo struct {
	ID     string `json:"id"`
	Blocks uint64 `json:"blocks"`
}

type TopCountry []CountryInfo

func (a TopCountry) Len() int           { return len(a) }
func (a TopCountry) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TopCountry) Less(i, j int) bool { return a[i].Blocks > a[j].Blocks }

type TopOrg []OrgInfo

func (a TopOrg) Len() int           { return len(a) }
func (a TopOrg) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TopOrg) Less(i, j int) bool { return a[i].Blocks > a[j].Blocks }

type TopType []TypeInfo

func (a TopType) Len() int           { return len(a) }
func (a TopType) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TopType) Less(i, j int) bool { return a[i].Blocks > a[j].Blocks }

type DateValue struct {
	Date    int64 `json:"date"`
	Total   int   `json:"total,omitempty"`
	Created int   `json:"created,omitempty"`
	Changed int   `json:"changed,omitempty"`
	Removed int   `json:"removed,omitempty"`
}

func CalcDateValueTotal(m map[int64]*DateValue) []DateValue {
	keys := make([]int64, 0, len(m))
	for d := range m {
		keys = append(keys, d)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	total := 0
	res := make([]DateValue, 0, len(m))

	for _, key := range keys {
		total += m[key].Created - m[key].Removed
		m[key].Total = total

		res = append(res, *m[key])
	}

	return res
}
