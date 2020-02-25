package main

import (
	"flag"
	"fmt"
	"log"
	cnf "macaddress_io_grabber/config"
	"macaddress_io_grabber/modules/auto"
	"macaddress_io_grabber/modules/download"
	"macaddress_io_grabber/modules/export"
	"macaddress_io_grabber/modules/normalize"
	"macaddress_io_grabber/modules/parse"
	"macaddress_io_grabber/modules/stats"
	db "macaddress_io_grabber/utils/database"
	"os"
	"path/filepath"
	"time"
)

var (
	sourceDir  *string
	configFile *string
	outputFile *string
	curDate    *string
	forceRedo  *bool
)

const (
	cmdAuto      = "auto"
	cmdCheck     = "check"
	cmdStats     = "stats"
	cmdDownload  = "download"
	cmdMigrate   = "migrate"
	cmdParse     = "parse"
	cmdNormalize = "normalize"
	cmdExport    = "export"

	cmdExportRedis = "redis"
	cmdExportJSON  = "json"
	cmdExportXML   = "xml"
	cmdExportCisco = "cisco"
	cmdExportCSV   = "csv"
)

var (
	command  = ""
	exportTo = ""
	config   = &cnf.Config{}
)

func init() {

	log.SetFlags(log.Flags() | log.Lshortfile)

	command = parseCommand()

	var usageHeader string

	flag.Usage = func() {
		if _, err := fmt.Fprintf(flag.CommandLine.Output(), usageHeader); err != nil {
			panic(err)
		}
		flag.PrintDefaults()
	}

	usageHeader = fmt.Sprintf("Usage of %s %s:\n", os.Args[0], command)

	switch command {
	case cmdAuto:
		usageHeader += "Automated update\n"
		initConfig()
		initDate()
	case cmdCheck:
		usageHeader += "Check automated update and push changes to production\n"
		initConfig()
		initDate()
	case cmdStats:
		usageHeader += "Calculate statistics\n"
		initConfig()
	case cmdMigrate:
		usageHeader += "Creates tables\n"
		initConfig()
	case cmdDownload:
		usageHeader += "Download csv from IEEE\n"
		initDir()
	case cmdParse:
		usageHeader += "Parse csv and save them into database\n"
		initConfig()
		initDir()
		initDate()
	case cmdNormalize:
		usageHeader += "Normalize existing records in database\n"
		initConfig()
		initForce()
	case cmdExport:
		exportTo = parseCommand()
		switch exportTo {
		case "":
			exportTo = cmdExportRedis
			fallthrough
		case cmdExportRedis:
		case cmdExportJSON, cmdExportXML, cmdExportCisco, cmdExportCSV:
			initOutput()
		}
		usageHeader = fmt.Sprintf("Usage of %s %s (%s|%s|%s|%s|%s):\n",
			os.Args[0], command, cmdExportRedis, cmdExportJSON, cmdExportXML, cmdExportCisco, cmdExportCSV)
		usageHeader += "Export database to redis\n"
		initConfig()
	default:
		usageHeader = fmt.Sprintf("Usage of %s %s:\n",
			os.Args[0], "(migrate|download|parse|normalize|export|auto|check)")
		usageHeader += "Use " + os.Args[0] +
			" (migrate|download|parse|normalize|export|auto|check) (-help|-h) to show a command usage\n"

		flag.Usage()
		os.Exit(2)
	}

	flag.Parse()

	if configFile != nil && *configFile == "" {
		flag.Usage()
		os.Exit(2)
	}
}

func parseCommand() string {
	if len(os.Args) >= 2 {
		c := os.Args[1]
		if c != "" && c[0] != '-' {
			//Remove command from os.Args
			//It's needed to parse flags after command
			os.Args = append(os.Args[0:1], os.Args[2:]...)
			return c
		}
		return ""
	}
	return ""
}

func main() {

	if command == "" {
		flag.Usage()
		return
	}

	// Init database when it's needed
	if configFile != nil {
		if c, err := cnf.Load(*configFile); err != nil {
			log.Fatalln(err)
		} else {
			config = c
		}

		if err := db.InitDatabase(config.Db); err != nil {
			log.Fatalln(err)
		}

		defer func() {
			if err := db.Instance.Close(); err != nil {
				log.Fatalln(err)
			}
		}()
	}

	switch command {
	case cmdAuto:
		t := time.Now()
		if *curDate != "" {
			n, err := time.ParseInLocation("2006-01-02", *curDate, time.UTC)
			if err != nil {
				log.Fatalln(err)
			}
			t = n
		}
		if err := auto.Update(config, t); err != nil {
			log.Fatalln(err)
		}
	case cmdCheck:
		t := time.Now()
		if *curDate != "" {
			n, err := time.ParseInLocation("2006-01-02", *curDate, time.UTC)
			if err != nil {
				log.Fatalln(err)
			}
			t = n
		}
		if err := auto.Check(config, t); err != nil {
			log.Fatalln(err)
		}
	case cmdStats:
		if err := stats.Calculate(); err != nil {
			log.Fatalln(err)
		}
	case cmdDownload:
		err := download.Download(*sourceDir)
		if err != nil {
			log.Fatalln(err)
		}
	case cmdParse:
		t := time.Now()
		if *curDate != "" {
			n, err := time.ParseInLocation("2006-01-02", *curDate, time.UTC)
			if err != nil {
				log.Fatalln(err)
			}
			t = n
		}

		dir := *sourceDir

		if dir == "" {
			dir = filepath.Join(
				config.AutoUpdate.DirSource,
				t.Format("2006-01-02"),
			)
		}

		err := parse.Parse(dir, t)
		if err != nil {
			log.Fatalln(err)
		}

		err = parse.ParseWireshark(dir, t)
		if err != nil {
			log.Fatalln(err)
		}

		fallthrough
	case cmdNormalize:
		err := normalize.All(forceRedo != nil && *forceRedo)
		if err != nil {
			log.Fatalln(err)
		}
	case cmdMigrate:
		if err := migrate(); err != nil {
			log.Fatalln(err)
		}
	case cmdExport:
		switch exportTo {
		case cmdExportRedis:
			if err := export.ToRedis(config.Redis.ConnString, config.Redis.Databases); err != nil {
				log.Fatalln(err)
			}
		case cmdExportJSON:
			if err := export.ToJSON(*outputFile); err != nil {
				log.Fatalln(err)
			}
		case cmdExportXML:
			if err := export.ToXML(*outputFile); err != nil {
				log.Fatalln(err)
			}
		case cmdExportCisco:
			if err := export.ToVendorMacXML(*outputFile); err != nil {
				log.Fatalln(err)
			}
		case cmdExportCSV:
			if err := export.ToCSV(*outputFile); err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func migrate() error {

	for _, model := range []interface{}{
		&db.OUI{},
		&db.Org{},
		&db.VMRange{},
		&db.ApplicationRange{},
		&db.WiresharkNote{},
		&db.CustomNote{},
	} {
		if err := db.Instance.AutoMigrate(model).Error; err != nil {
			return err
		}
	}

	return nil
}

func initDir() {
	if sourceDir == nil {
		sourceDir = flag.String("d", "", "directory for .csv files")
	}
}

func initOutput() {
	if outputFile == nil {
		outputFile = flag.String("o", "", "output file")
	}
}

func initConfig() {
	if configFile == nil {
		configFile = flag.String("c", "", "path to config file")
	}
}

func initForce() {
	if forceRedo == nil {
		forceRedo = flag.Bool("f", false, "force redo already done records")
	}
}

func initDate() {
	if curDate == nil {
		curDate = flag.String("t", "", "date in format yyyy-mm-dd (2006-01-02)")
	}
}
