package database

import (
	"errors"
	"github.com/jinzhu/gorm"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type OUI struct {
	ID            uint64 `gorm:"primary_key"`
	CreatedAt     *time.Time
	Assignment    string `gorm:"column:assignment;type:varchar(100);not null;index:assign_idx"`
	Type          string `gorm:"column:type;type:varchar(500);not null"`
	RawOrgName    string `gorm:"column:raw_org_name;type:varchar(500);not null"`
	RawOrgAddress string `gorm:"column:raw_org_addr;type:varchar(500);not null"`
	Address       string `gorm:"column:address;type:varchar(500);not null"`
	Country       string `gorm:"column:country;type:varchar(2);not null"`
	PostalCode    string `gorm:"column:postal_code;type:varchar(50);not null"`
	Org           *Org
	OrgText       string `gorm:"column:org_text;type:varbinary(255);not null;index:org_text_idx"`
	OrgID         uint64 `gorm:"column:org_id;"`
	Reference     uint64 `gorm:"column:ref;not null;index:ref_idx"`
	Created       bool   `gorm:"column:created;not null;default:0"`
}

func (oui OUI) TableName() string {
	return "records"
}

func (oui *OUI) FindOrgByText() error {

	var org = Org{}

	if err := Instance.Where(Org{Shorten: oui.OrgText}).FirstOrCreate(&org).Error; err != nil {
		return errors.New("cannot find organization by '" + oui.OrgText + "': " + err.Error())
	}

	if org.ID == 0 {
		return errors.New("cannot find organization by '" + oui.OrgText + "'")
	}

	oui.Org = &org
	oui.OrgID = org.ID

	return nil
}

func (oui *OUI) FindOrg() error {

	for oui.Org == nil || oui.Org.ID != oui.Org.Reference {
		var org = Org{}
		if err := Instance.Where("id = ?", oui.OrgID).Find(&org).Error; err != nil {
			return err
		}

		oui.Org = &org
		if oui.Org.ID == 0 && oui.Org.Reference == 0 {
			return errors.New("cannot find organization for " + oui.Assignment + " by " + strconv.FormatUint(oui.OrgID, 10))
		}
		oui.OrgID = org.Reference
	}

	return nil
}

type Org struct {
	ID        uint64 `gorm:"primary_key"`
	CreatedAt *time.Time
	Shorten   string `gorm:"column:shorten;type:varbinary(255);not null;unique_index:shorten_idx"`
	Name      string `gorm:"column:name;type:varchar(255);not null"`
	NName     string `gorm:"column:n_name;type:varchar(255);not null"`
	Reference uint64 `gorm:"column:ref;not null;index:ref_idx"`
}

func (org Org) TableName() string {
	return "organizations"
}

var reNonLetter = regexp.MustCompile(`\W+`)

func shorten(s string) string {
	return strings.ToLower(reNonLetter.ReplaceAllString(s, ""))
}

func (oui *OUI) Save() (e error) {

	db := Instance

	var old = new(OUI)
	var org = new(Org)

	if err := db.Where("assignment = ? AND id = ref", oui.Assignment).Last(old).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}

	oui.OrgText = shorten(oui.RawOrgName)
	org.Shorten = oui.OrgText

	// Check if record with same assignment exists
	if old.ID != 0 {
		if old.OrgText != oui.OrgText || old.RawOrgAddress != oui.RawOrgAddress || old.Type != oui.Type {

			org.Name = oui.RawOrgName

			if err := saveOrg(org, db); err != nil {
				return errors.New("cannot save organization '" + org.Shorten + "': " + err.Error())
			}

			oui.OrgID = org.ID

			if err := db.Create(&oui).Error; err != nil {
				return err
			}

			// Moved it after oui record creation, as otherwise gorm tries to update organization
			oui.Org = org

			if err := db.Model(OUI{}).Where("assignment = ?", oui.Assignment).Update("ref", oui.ID).Error; err != nil {
				return err
			}
		}
		return nil
	}

	org.Name = oui.RawOrgName

	if err := saveOrg(org, db); err != nil {
		return errors.New("cannot save organization '" + org.Shorten + "': " + err.Error())
	}

	oui.OrgID = org.ID
	oui.Created = true

	if err := db.Create(&oui).Error; err != nil {
		return err
	}

	if err := db.Model(&oui).Update("ref", oui.ID).Error; err != nil {
		return err
	}

	return nil
}

var createOrgMutex = sync.Mutex{}

func saveOrg(org *Org, db *gorm.DB) (e error) {
	createOrgMutex.Lock()

	defer createOrgMutex.Unlock()

	db = db.Begin()
	if err := db.Error; err != nil {
		return err
	}

	defer func() {
		if e != nil {
			db = db.Rollback()
		} else {
			db = db.Commit()
			if db.Error != nil {
				e = db.Error
			}
		}
	}()

	if err := db.Where(Org{Shorten: org.Shorten}).FirstOrCreate(&org).Error; err != nil {
		return err
	}

	if org.Reference == 0 {
		if err := db.Model(Org{}).Where("id = ?", org.ID).Update("ref", org.ID).Error; err != nil {
			return err
		}
	}

	return nil
}
