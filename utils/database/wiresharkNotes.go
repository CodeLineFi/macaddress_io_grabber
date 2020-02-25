package database

import (
	"github.com/jinzhu/gorm"
	"time"
)

type WiresharkNote struct {
	ID         uint64 `gorm:"primary_key"`
	Assignment string `gorm:"column:assignment;type:varchar(100);not null;index:assignment_idx"`
	Note       string `gorm:"column:note;type:text;not null"`
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

func (wiresharkNote WiresharkNote) TableName() string {
	return "wireshark_notes"
}

func (wiresharkNote *WiresharkNote) Save() (e error) {

	db := Instance

	var old = new(WiresharkNote)

	if err := db.Where("assignment = ?", wiresharkNote.Assignment).Last(old).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	}

	// Check if record with same assignment exists
	if old.ID != 0 {
		if old.Note != wiresharkNote.Note {

			if err :=
				db.Model(WiresharkNote{}).
					Where("assignment = ?", wiresharkNote.Assignment).
					Update("note", wiresharkNote.Note).
					Error; err != nil {
				return err
			}
		}
		return nil
	}

	if err := db.Create(&wiresharkNote).Error; err != nil {
		return err
	}

	return nil
}
