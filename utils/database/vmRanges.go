package database

import (
	"macaddress_io_grabber/models"
	"time"
)

type VMRange struct {
	ID                 uint64 `gorm:"primary_key"`
	LeftBorder         string `gorm:"column:l_border;type:varchar(100);not null;index:l_border_idx"`
	RightBorder        string `gorm:"column:r_border;type:varchar(100);not null;index:r_border_idx"`
	VirtualMachineName string `gorm:"column:vm_name;type:varchar(100);not null;index:vm_idx"`
	Reference          string `gorm:"column:ref;type:text"`
	CreatedAt          *time.Time
	UpdatedAt          *time.Time
}

func (vmRange VMRange) TableName() string {
	return "vm_ranges"
}

func (vmRange *VMRange) ToJSONModel() models.VMRangeJSON {
	return models.VMRangeJSON{
		RangeJSON: models.RangeJSON{
			LeftBorder:  vmRange.LeftBorder,
			RightBorder: vmRange.RightBorder,
		},
		VirtualMachineName: vmRange.VirtualMachineName,
		Reference:          vmRange.Reference,
	}
}
