package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
)

type Database struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type RedisConfig struct {
	ConnString string `json:"url"`
	Databases  []int  `json:"databases"`
}

type AutoUpdateConfig struct {
	DirSource     string `json:"sourceDir"`
	ResultJSON    string `json:"exportToJson"`
	ResultXML     string `json:"exportToXml"`
	ResultCisco   string `json:"exportToCisco"`
	ResultCSV     string `json:"exportToCsv"`
	ExportToRedis bool   `json:"exportToRedis"`
	DirServer     string `json:"serverDir"`
	StoreDays     int    `json:"storeDays"`
	Email         Email  `json:"email"`
}

type Email struct {
	Host      string   `json:"host"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	EmailFrom string   `json:"emailFrom"`
	EmailTo   []string `json:"emailTo"`
}

type Config struct {
	Db         Database         `json:"database"`
	Redis      RedisConfig      `json:"redis"`
	AutoUpdate AutoUpdateConfig `json:"autoUpdate"`
}

func Load(filename string) (*Config, error) {

	var instance = Config{}

	if filename == "" {
		return nil, errors.New("config file is not specified")
	}

	file, err := ioutil.ReadFile(filename)
	if err == nil {
		if err := json.Unmarshal(file, &instance); err != nil {
			log.Println(err)
			return nil, err
		}
		return &instance, nil
	}
	return nil, err
}
