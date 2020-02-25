package database

import (
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // Export mysql dialect
	"macaddress_io_grabber/config"
)

var Instance *gorm.DB

func InitDatabase(cfg config.Database) error {
	conf := mysql.NewConfig()

	conf.Addr = cfg.URL
	conf.User = cfg.Username
	conf.Passwd = cfg.Password
	conf.DBName = cfg.Database
	conf.Net = ""
	conf.ParseTime = true

	db, err := gorm.Open("mysql", conf.FormatDSN())
	if err != nil {
		return err
	}

	Instance = db.Set("gorm:table_options", "charset=binary")
	return Instance.Error
}
