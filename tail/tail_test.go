package tail

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tusupov/gologtail/db"
)

var (
	testLogFileDir = "test/"
	testMsg        = strings.Repeat("a", 10)
	testMsgCnt     = int(1e2)
	testFilesCnt   = int(10)
)

func TestTail_Run(t *testing.T) {

	defer deleteFile(t)

	writeFile(t)

	itemCh := make(chan db.Item, testFilesCnt)
	wgCh := &sync.WaitGroup{}

	itemCnt := 0
	wgCh.Add(1)
	go func() {
		defer func() {
			wgCh.Done()
		}()
		for range itemCh {
			itemCnt++
		}
	}()

	wg := &sync.WaitGroup{}
	tailList := make([]*tail, testFilesCnt)
	for i := 0; i < testFilesCnt; i++ {
		timeFormat := 1
		if i&2 == 1 {
			timeFormat = 2
		}
		var err error
		tailList[i], err = New(fmt.Sprintf("%s/test%d.log", testLogFileDir, i), timeFormat)
		if err != nil {
			t.Fatalf("%d, %s", i, err)
		}
		tailList[i].Debug(false)
		wg.Add(1)
		go tailList[i].Run(wg, itemCh)
	}

	writeFile(t)

	for i := 0; i < testFilesCnt; i++ {
		tailList[i].Stop()
	}

	wg.Wait()

	close(itemCh)
	wgCh.Wait()

}

func writeFile(t *testing.T) {

	err := os.MkdirAll(testLogFileDir, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	f := make([]*os.File, testFilesCnt)
	for i := 0; i < testFilesCnt; i++ {
		f[i], err = os.OpenFile(fmt.Sprintf("%s/test%d.log", testLogFileDir, i), os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}

	date := time.Now().Add(- (time.Hour * 365 * 24))
	for i := 0; i < testMsgCnt; i++ {
		for j := 0; j < testFilesCnt; j++ {
			timeFormat := TIME_FORMAT_UTC
			if j&2 == 1 {
				timeFormat = TIME_FORMAT_TZ
			}
			f[j].WriteString(date.Format(timeFormat+" | "+testMsg) + "\n")
			date = date.Add(time.Second * time.Duration(rand.Intn(100)))
		}
	}

	for i := 0; i < testFilesCnt; i++ {
		err := f[i].Close()
		if err != nil {
			t.Fatal(err)
		}
	}

}

func deleteFile(t *testing.T) {
	err := os.RemoveAll(testLogFileDir)
	if err != nil {
		t.Fatal(err)
	}
}
