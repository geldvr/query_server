package db

import (
	"gopkg.in/mgo.v2"
	"itv/shared/config"
	"log"
	"sync"
	"time"
)

var (
	mgoOnce sync.Once
	session *mgo.Session
	dbName  string
)

func connectMongoDb() *mgo.Session {
	dsn := config.GetString("mongodb.dsn")
	dialInfo, err := mgo.ParseURL(dsn)

	dbName = dialInfo.Database
	dialInfo.FailFast = false
	dialInfo.Timeout = 10 * time.Second

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		log.Fatalf("DB connection fatal error: %s", err.Error())
	}

	return session
}

func GetMongoInstance() (*mgo.Session, *mgo.Database) {
	mgoOnce.Do(func() {
		session = connectMongoDb()
	})

	s := session.Copy()
	return s, s.DB(dbName)
}
