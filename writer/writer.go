package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tusupov/gologtail/tail"
)

var (
	writeTimeFormat = flag.Int(`format`, 1, `Time format for log files \nValues:\n\t1 - ` + tail.TIME_FORMAT_UTC + `\n\t2 - ` + tail.TIME_FORMAT_TZ + `)`)
	writeMsg = strings.Repeat("a", 10)
)

func init() {
	flag.Parse()
}

func main() {

	log.Println("Start writer ...")

	wg := &sync.WaitGroup{}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		filename := sc.Text()
		log.Println("File:", filename)
		wg.Add(1)
		go writeFile(wg, filename, *writeTimeFormat)
	}
	wg.Wait()
	log.Println("End.")

}

func writeFile(wg *sync.WaitGroup, filename string, format int) {

	var err error
	defer func() {
		if err != nil {
			log.Println("File:", filename, "Error:", err)
		}
		wg.Done()
	}()

	dir, _ := filepath.Split(filename)
	if len(dir) > 0 {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return
		}
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return
	}

	timeFormat := tail.TIME_FORMAT_UTC
	if format == 2 {
		timeFormat = tail.TIME_FORMAT_TZ
	}

	date := time.Now().Add(-time.Duration(rand.Intn(1e6)))

LOOP:
	for {
		msg := date.Format(timeFormat+" | "+writeMsg)
		cnt := rand.Intn(10)
		for i := 0; i <	cnt; i++ {
			_, err = f.WriteString(msg + "\n")
			if err != nil {
				break LOOP
			}
		}
		log.Printf("[%s]: Write %d\n", filename, cnt)
		date = date.Add(time.Second * time.Duration(rand.Intn(60)))
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(3 * 1000)))
	}

}
