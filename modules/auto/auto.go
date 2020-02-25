package auto

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"macaddress_io_grabber/config"
	"macaddress_io_grabber/modules/download"
	"macaddress_io_grabber/modules/export"
	"macaddress_io_grabber/modules/normalize"
	"macaddress_io_grabber/modules/parse"
	"path/filepath"
	"time"
)

func selectDB(client redis.Conn, db int) error {
	if _, err := client.Do("SELECT", db); err != nil {
		return errors.New("cannot change database: " + err.Error())
	}
	return nil
}

func Update(config *config.Config, d time.Time) error {
	if config.AutoUpdate.DirSource == "" {
		return errors.New("source directory in config is not specified")
	}
	sDir := filepath.Join(
		config.AutoUpdate.DirSource,
		d.Format("2006-01-02"),
	)
	if err := download.Download(sDir); err != nil {
		return err
	}
	if err := parse.Parse(sDir, d); err != nil {
		return err
	}

	if err := parse.ParseWireshark(sDir, d); err != nil {
		return err
	}

	if err := normalize.All(false); err != nil {
		return err
	}
	if config.AutoUpdate.ExportToRedis {
		if err := export.ToRedis(config.Redis.ConnString, config.Redis.Databases); err != nil {
			return err
		}
	}
	if config.AutoUpdate.ResultJSON != "" {
		if err := export.ToJSON(config.AutoUpdate.ResultJSON); err != nil {
			return err
		}
	}
	if config.AutoUpdate.ResultXML != "" {
		if err := export.ToXML(config.AutoUpdate.ResultXML); err != nil {
			return err
		}
	}
	if config.AutoUpdate.ResultCisco != "" {
		if err := export.ToVendorMacXML(config.AutoUpdate.ResultCisco); err != nil {
			return err
		}
	}
	if config.AutoUpdate.ResultCSV != "" {
		if err := export.ToCSV(config.AutoUpdate.ResultCSV); err != nil {
			return err
		}
	}
	return nil
}
