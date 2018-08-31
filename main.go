package main

import (
	"bufio"
	"log"
	"os"
	"sync"

	"github.com/tusupov/gologtail/db"
	"github.com/tusupov/gologtail/tail"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	dbHost = kingpin.Flag(`dbh`, `MongoDB host`).Default(`localhost:27017`).String()
	dbName = kingpin.Flag(`dbn`, `MongoDB database name`).Default(`logs`).String()
	dbTable = kingpin.Flag(`dbt`, `MongoDB table name`).Default(`logs`).String()
	workers = kingpin.Flag(`dbw`, `Workers for add item to MongoDB`).Default("10").Int()
	timeFormat = kingpin.Flag(`format`, "Time format for log files\n Values:\n\t1 = \"" + tail.TIME_FORMAT_UTC + "\"\n\t2 = \"" + tail.TIME_FORMAT_TZ + "\"").Short('f').Default("1").Int()
)

func init() {
	kingpin.Parse()
}

func main()  {

	log.Println("Start ...")
	defer log.Println("End.")

	// Connection to db
	log.Printf("Start connection to db [%s] ...\n", *dbHost)
	store, err := db.New(*dbHost, *dbName)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected.")

	// Get collection
	collection := store.C(*dbTable)

	// Item workers for add db
	itemWg := &sync.WaitGroup{}
	itemCh := make(chan db.Item, *workers)
	for i := 0; i < *workers; i++ {
		itemWg.Add(1)
		go db.ItemWorker(itemWg, itemCh, collection)
	}

	wg := &sync.WaitGroup{}

	// Read filename from input
	sc := bufio.NewScanner(os.Stdin)
	log.Println("Start read log file list ...")
	for sc.Scan() {

		filename := sc.Text()

		log.Printf("File: [%s]\n", filename)
		t, err := tail.New(string(filename), *timeFormat)
		if err != nil {
			log.Printf("Error: %s", err)
			continue
		}
		t.Debug(true)

		wg.Add(1)
		go t.Run(wg, itemCh)

	}
	log.Println("End list.")

	// Waiting log file events
	wg.Wait()

	// Close item workers and waiting for add all items
	close(itemCh)
	itemWg.Wait()

}
