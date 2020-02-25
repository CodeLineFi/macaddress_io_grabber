package database

import (
	"macaddress_io_grabber/models"
	"time"
)

type ApplicationRange struct {
	ID          uint64 `gorm:"primary_key"`
	LeftBorder  string `gorm:"column:l_border;type:varchar(100);not null;index:l_border_idx"`
	RightBorder string `gorm:"column:r_border;type:varchar(100);not null;index:r_border_idx"`
	Application string `gorm:"column:application;type:varchar(100);not null;index:app_idx"`
	Notes       string `gorm:"column:notes;type:text"`
	Reference   string `gorm:"column:ref;type:text"`
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

func (applicationRange ApplicationRange) TableName() string {
	return "application_ranges"
}

func (applicationRange *ApplicationRange) ToJSONModel() models.ApplicationRangeJSON {
	return models.ApplicationRangeJSON{
		RangeJSON: models.RangeJSON{
			LeftBorder:  applicationRange.LeftBorder,
			RightBorder: applicationRange.RightBorder,
		},
		Application: applicationRange.Application,
		Notes:       applicationRange.Notes,
		Reference:   applicationRange.Reference,
	}
}
