package normalize

import (
	"macaddress_io_grabber/utils/database"
	"regexp"
	"strings"
)

var reWord = regexp.MustCompile(`(?i)([\w]+)`)
var (
	reInc  = regexp.MustCompile(`(?i)\b(inc|incorporated)(\.|\b)`)
	reLtd  = regexp.MustCompile(`(?i)\b(ltd|limited)(\.|\b)`)
	reLLC  = regexp.MustCompile(`(?i)\b(llc)(\.|\b)`)
	reComp = regexp.MustCompile(`(?i)\b(co|comp|company)(\.|\b)`)
	reCorp = regexp.MustCompile(`(?i)\b(corp|corporation|corporate)(\.|\b)`)
	reTech = regexp.MustCompile(`(?i)\b(tech|technology|technologies)(\.|\b)`)
	reGmbH = regexp.MustCompile(`(?i)\b(GmbH)(\.|\b)`)

	reSpaces  = regexp.MustCompile(`\s+`)
	reNoSpace = regexp.MustCompile(`(?i)[a-z](,)[a-z]`)
)

func All(redo bool) error {
	if err := OrganizationsRelax(); err != nil {
		return err
	}
	if err := Organizations(redo); err != nil {
		return err
	}
	if err := Addresses(redo); err != nil {
		return err
	}

	return nil
}

func OrganizationsRelax() error {
	var records = make([]database.OUI, 0)
	if db := database.Instance.Where("id = ref").Find(&records); db.Error != nil {
		return db.Error
	}

	for i := range records {
		id := records[i].OrgID
		if err := records[i].FindOrgByText(); err != nil {
			return err
		}
		if err := records[i].FindOrg(); err != nil {
			return err
		}
		if id != records[i].OrgID {
			if err := database.Instance.Model(&records[i]).Update("org_id", records[i].OrgID).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func Organizations(redo bool) error {
	var orgs = make([]database.Org, 0)
	if redo {
		if db := database.Instance.Find(&orgs); db.Error != nil {
			return db.Error
		}
	} else {
		if db := database.Instance.Where("n_name = ?", "").Find(&orgs); db.Error != nil {
			return db.Error
		}
	}

	var org *database.Org

	for i := range orgs {
		org = &orgs[i]

		orgName := Name(org.Name)

		if err := database.Instance.Model(&org).Update("n_name", orgName).Error; err != nil {
			return err
		}
	}

	return nil
}

var reCountryCode = regexp.MustCompile(`(?i) ([A-Z]{2})$`)

func Addresses(redo bool) error {
	var records = make([]database.OUI, 0)
	if redo {
		if db := database.Instance.Find(&records); db.Error != nil {
			return db.Error
		}
	} else {
		if err := database.Instance.Where("address = '' and country = '' and postal_code = ''").
			Find(&records).Error; err != nil {

			return err
		}
	}

	var oui *database.OUI

	for i := range records {
		oui = &records[i]

		if oui.RawOrgAddress == "" {
			continue
		}

		country := reCountryCode.FindStringSubmatch(oui.RawOrgAddress)

		if len(country) > 0 {
			oui.Country = country[1]
		}

		if err := database.Instance.Model(&oui).Updates(
			map[string]string{
				"address":     oui.Address,
				"country":     oui.Country,
				"postal_code": oui.PostalCode,
			}).Error; err != nil {

			return err
		}
	}

	return nil
}

func Name(orgName string) string {

	orgName = strings.TrimSpace(orgName)

	orgName = reInc.ReplaceAllString(orgName, "Inc")
	orgName = reLtd.ReplaceAllString(orgName, "Ltd")
	orgName = reLLC.ReplaceAllString(orgName, "Llc")
	orgName = reComp.ReplaceAllString(orgName, "Co")
	orgName = reCorp.ReplaceAllString(orgName, "Corp")
	orgName = reTech.ReplaceAllString(orgName, "Tech")
	orgName = reGmbH.ReplaceAllString(orgName, "GmbH")
	orgName = reSpaces.ReplaceAllString(orgName, " ")
	orgName = reNoSpace.ReplaceAllStringFunc(orgName, func(ss string) string {
		return ss[0:2] + " " + ss[2:3]
	})

	orgName = reWord.ReplaceAllStringFunc(orgName, func(ss string) string {

		if ss == strings.ToUpper(ss) && len(ss) > 2 {
			ss = ss[0:1] + strings.ToLower(ss[1:])
		}

		return ss
	})

	return orgName
}
