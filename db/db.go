package db

import (
	"log"
	"sync"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type db struct {
	*mgo.Database
}

func New(host string, dbname string) (store *db, err error) {

	mgoS, err := mgo.Dial(host)
	if err != nil {
		return
	}

	err = mgoS.Ping()
	if err != nil {
		return
	}

	mgoDb := mgoS.DB(dbname)
	store = &db{
		mgoDb,
	}

	return

}

type Item struct {
	Id         bson.ObjectId `bson:"_id"`
	Date       time.Time     `bson:"log_date"`
	Msg        string        `bson:"log_msg"`
	FilePath   string        `bson:"file_name"`
	TimeFormat string        `bson:"log_format"`
}

func ItemWorker(wg *sync.WaitGroup, ch <-chan Item, collection *mgo.Collection) {

	defer wg.Done()

	for item := range ch {
		item.Id = bson.NewObjectId()
		err := collection.Insert(item)
		if err != nil {
			log.Println("Item add error:", err)
		}
	}

}
