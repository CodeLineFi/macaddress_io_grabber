package models

type OUIRedis struct {
	Assignment          MAC            `json:"oui"`
	IsPrivate           bool           `json:"isPrivate"`
	CompanyID           uint64         `json:"companyId"`
	CompanyName         string         `json:"companyName"`
	CompanyAddress      string         `json:"companyAddress"`
	CountryCode         string         `json:"countryCode"`
	AssignmentBlockSize string         `json:"assignmentBlockSize"`
	DateCreated         string         `json:"dateCreated"`
	DateUpdated         string         `json:"dateUpdated"`
	History             []OUIHistRedis `json:"history"`
}

type OUIHistRedis struct {
	Date           string `json:"date"`
	CompanyID      uint64 `json:"companyId"`
	CompanyName    string `json:"companyName"`
	CompanyAddress string `json:"companyAddress"`
	CountryCode    string `json:"countryCode"`
}

func (oui *OUIRedis) BlockSize() uint64 {

	l := oui.Assignment.Length()

	switch l {
	case 0:
		return 0
	case 6:
		return 16777216
	case 7:
		return 1048576
	case 9:
		return 4096
	}

	res := uint64(1)
	for i := 12 - l; i > 0; i-- {
		res *= 16
	}

	return res
}

type OUIJson struct {
	Assignment          MAC    `json:"oui"`
	IsPrivate           bool   `json:"isPrivate"`
	CompanyName         string `json:"companyName"`
	CompanyAddress      string `json:"companyAddress"`
	CountryCode         string `json:"countryCode"`
	AssignmentBlockSize string `json:"assignmentBlockSize"`
	DateCreated         string `json:"dateCreated"`
	DateUpdated         string `json:"dateUpdated"`
}
