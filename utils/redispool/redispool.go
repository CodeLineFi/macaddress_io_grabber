package redispool

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sync"
	"time"
)

type Interface interface {
	Get() redis.Conn
	UpdateTags(production, candidate int) error
}

type Pool struct {
	*redis.Pool
	databases   []int
	dbsInfoLock sync.Mutex
	dbCurrent   []int
	projectName string
}

func New(url string, tag string, name string, maxIdleCons, maxActiveCons int, databases []int) *Pool {

	rp := &Pool{
		projectName: name,
		databases:   databases,
	}

	if maxIdleCons <= 0 {
		maxIdleCons = 10
	}

	if maxActiveCons < 0 {
		maxActiveCons = 0
	}

	rp.Pool = &redis.Pool{
		MaxIdle:     maxIdleCons,
		MaxActive:   maxActiveCons,
		Wait:        false,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.DialURL(url)
			if err != nil {
				return conn, err
			}

			for i := 0; ; i++ {
				if i == 4 {
					return conn, err
				}

				var dbsInfo *Databases

				dbsInfo, err = rp.getDatabasesInfo(conn)
				if err != nil {
					return nil, err
				}

				err = selectByTag(conn, tag, dbsInfo)
				if err == nil {
					break
				}
			}

			return conn, nil
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) (err error) {
			var dbsInfo *Databases

			for i := 0; ; i++ {
				if i == 4 {
					return err
				}

				var curTag string

				curTag, err = redis.String(conn.Do("GET", fieldDatabaseTag))
				if err != nil {
					return err
				}

				if tag == curTag {
					break
				}

				dbsInfo, err = rp.getDatabasesInfo(conn)
				if err != nil {
					return err
				}

				err = selectByTag(conn, tag, dbsInfo)
				if err == nil {
					break
				}
			}

			return nil
		},
	}

	return rp
}

func selectByTag(conn redis.Conn, tag string, databases *Databases) error {

	if databases == nil {
		return errors.New("databases is nil")
	}

	num := -1

	switch tag {
	case TagProduction:
		if databases.Production < 0 {
			return errors.New("there's no production database")
		}
		num = databases.List[databases.Production].Number
	case TagCandidate:
		if databases.Candidate < 0 {
			return errors.New("there's no candidate database")
		}
		num = databases.List[databases.Candidate].Number
	case TagTarget:
		if databases.Target >= 0 {
			num = databases.List[databases.Target].Number
		}
	}

	if err := selectByNum(conn, num); err != nil {
		return err
	}

	return nil
}

func selectByNum(conn redis.Conn, num int) error {
	if _, err := conn.Do("SELECT", num); err != nil {
		return errors.New("cannot change database: " + err.Error())
	}
	return nil
}

func (p *Pool) GetDatabasesInfo() (*Databases, error) {
	conn := p.Get()
	defer conn.Close()
	return p.getDatabasesInfo(conn)
}

func (p *Pool) getDatabasesInfo(conn redis.Conn) (*Databases, error) {

	p.dbsInfoLock.Lock()
	defer p.dbsInfoLock.Unlock()

	var list = make([]DatabaseInfo, len(p.databases))

	DBs := &Databases{
		Production: -1,
		Candidate:  -1,
		Target:     -1,
	}

	for i, num := range p.databases {

		list[i].Number = num

		if err := selectByNum(conn, num); err != nil {
			return nil, errors.New("cannot change database: " + err.Error())
		}

		res, err := redis.Values(conn.Do("MGET", fieldDatabaseName, fieldDatabaseTag))
		if err != nil {
			return nil, err
		}

		list[i].Name, _ = redis.String(res[0], nil)
		list[i].Tag, _ = redis.String(res[1], nil)

		switch list[i].Name {
		case p.projectName:
			if list[i].Tag == TagProduction {
				if DBs.Production == -1 {
					DBs.Production = i
				}
				break
			}
			fallthrough
		case "":
			switch list[i].Tag {
			case TagCandidate:
				if DBs.Candidate == -1 {
					DBs.Candidate = i
					DBs.Target = i
				}
			default:
				if DBs.Target == -1 {
					DBs.Target = i
				}
			}
		default:
			return nil, fmt.Errorf("database [%d] name is not '%s'", list[i].Number, p.projectName)
		}
	}

	DBs.List = list

	return DBs, nil
}

func (p *Pool) UpdateTags(production, candidate int) error {

	conn := p.Get()
	defer conn.Close()

	if _, err := conn.Do("MULTI"); err != nil {
		return err
	}

	for i := range p.databases {

		if err := selectByNum(conn, p.databases[i]); err != nil {
			_, _ = conn.Do("DISCARD")
			return err
		}

		var tag = "-"

		switch i {
		case production:
			tag = TagProduction
		case candidate:
			tag = TagCandidate
		}

		if _, err := conn.Do("MSET", fieldDatabaseTag, tag, fieldDatabaseName, p.projectName); err != nil {
			_, _ = conn.Do("DISCARD")
			return err
		}
	}

	if _, err := conn.Do("EXEC"); err != nil {
		return err
	}

	return nil
}
