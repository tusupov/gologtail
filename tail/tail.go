package tail

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tusupov/gologtail/db"
)

const (
	TIME_FORMAT_TZ  = "2006-01-02T15:04:05Z"
	TIME_FORMAT_UTC = "Jan 2, 2006 at 3:04:05pm (UTC)"

	WATCHER_STOP = 0
)

var (
	errFileRemove = errors.New(`File removed`)
)

type FileInfo struct {
	Path       string
	Name       string
	Offset     int64
	TimeFormat string
}

type LineCnt struct {
	Success uint64
	Error   uint64
}

type tail struct {
	file    FileInfo
	watcher *fsnotify.Watcher

	lineCnt LineCnt
	line    chan string
	err     chan error
	done    chan struct{}

	closed  chan struct{}

	debug bool
	log   *log.Logger
}

func New(path string, format int) (t *tail, err error) {

	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return
	}
	offset := info.Size()

	timeFormat := TIME_FORMAT_UTC
	if format == 2 {
		timeFormat = TIME_FORMAT_TZ
	}

	_, fileName := filepath.Split(path)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}

	t = &tail{
		// File info
		file: FileInfo{
			Path:       path,
			Name:       fileName,
			Offset:     offset,
			TimeFormat: timeFormat,
		},

		line: make(chan string),
		err:  make(chan error),
		done: make(chan struct{}),

		closed: make(chan struct{}),

		watcher: watcher,

		debug: false,
		log:   log.New(os.Stdout, fmt.Sprintf("[%s]: ", fileName), log.LstdFlags),
	}

	return
}

func (t *tail) Run(wg *sync.WaitGroup, itemCh chan<- db.Item) {

	t.Println("Start listening ...")

	defer func() {

		// Recover
		if r := recover(); r != nil {
			t.Printf("recover: %#v\n", r)
		}

		// Stop process
		t.close()
		wg.Done()

		t.Printf("End [Success = %d, Error = %d].\n", t.lineCnt.Success, t.lineCnt.Error)

	}()

	// Start listen file event
	go t.fileSync()

	for {

		select {

		// Get line from file
		case l := <-t.line:

			// Format line
			lineArr := strings.SplitN(l, "|", 2)
			if len(lineArr) != 2 {
				t.Println("Line error: couldn't split by `|`:", l)
				t.lineCnt.Error++
				continue
			}
			date, errTime := time.Parse(t.file.TimeFormat, strings.TrimSpace(lineArr[0]))
			if errTime != nil {
				t.Println("Line error: time format:", lineArr[0])
				t.lineCnt.Error++
				continue
			}

			t.lineCnt.Success++

			// Send Item
			itemCh <- db.Item{
				Date:       date,
				Msg:        strings.TrimSpace(lineArr[1]),
				FilePath:   t.file.Path,
				TimeFormat: t.file.TimeFormat,
			}

		// Has error
		case err := <-t.err:
			t.Println(err)
			return

		// End
		case <-t.done:
			return

		}

	}

}

func (t *tail) fileSync() {

	// Recover
	defer func() {
		if r := recover(); r != nil {
			t.Printf("recover: %#v\n", r)
		}
	}()

	err := t.watcher.Add(t.file.Path)
	if err != nil {
		t.err <- err
		return
	}

	for {

		select {

		case event := <-t.watcher.Events:

			switch event.Op {

			// File write
			case fsnotify.Write:
				err := t.readLines()
				if err != nil {
					t.err <- errFileRemove
					return
				}

			// File remove
			case fsnotify.Remove:
				t.err <- errFileRemove
				return

			// Stop event
			case WATCHER_STOP:
				t.done <- struct{}{}
				return

			default:
				t.err <- errors.New(event.String())
				return

			}

		case err := <-t.watcher.Errors:
			t.err <- err
			return

		}
	}

}

func (t *tail) readLines() (err error) {

	var readLineCnt int
	defer func() {
		t.Printf("Read line: %d\n", readLineCnt)
	}()

	// Open file for reading
	f, err := os.Open(t.file.Path)
	if err != nil {
		return
	}
	defer f.Close()

	// Set to offset
	n, err := f.Seek(t.file.Offset, os.SEEK_CUR)
	if err != nil {
		return
	}

	// file size small than current read offset
	if n < t.file.Offset {
		t.file.Offset = n
		return
	}

	var offset int
	var lineStr string

	reader := bufio.NewReader(f)

LOOP:
	for {

		line, errRead := reader.ReadSlice('\n')
		offset += len(line)
		if errRead != nil {
			switch errRead {
			case bufio.ErrBufferFull:
				lineStr += string(line)
				continue
			case io.EOF:
				break LOOP
			default:
				err = errRead
				return
			}
		}

		if line[len(line)-1] == '\n' {
			drop := 1
			if len(line) > 1 && line[len(line)-2] == '\r' {
				drop = 2
			}
			line = line[:len(line)-drop]
		}
		lineStr += string(line)

		l := string(lineStr)
		t.line <- l
		t.file.Offset += int64(offset)

		lineStr = ""
		offset = 0
		readLineCnt++

	}

	return
}

func (t *tail) Debug(debug bool) {
	t.debug = debug
}

func (t *tail) SetOutput(w io.Writer) {
	t.log.SetOutput(w)
}

func (t tail) Printf(format string, v ...interface{}) {
	if t.debug {
		t.log.Printf(format, v...)
	}
}

func (t tail) Println(v ...interface{}) {
	if t.debug {
		t.log.Println(v...)
	}
}

// Stop send event for stopping watcher
func (t *tail) Stop() {
	if t.isClosed() {
		return
	}
	t.watcher.Events <- fsnotify.Event{
		Name: "Close",
		Op:   WATCHER_STOP,
	}
}

func (t *tail) isClosed() bool {
	select {
	case <-t.closed:
		return true
	default:
		return false
	}
}

func (t *tail) close() {
	close(t.line)
	close(t.err)
	close(t.done)
	t.watcher.Close()
}
