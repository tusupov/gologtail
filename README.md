# GoLogTail
Parse files with different log formats and insert in MongoDB.

### Usage
```
$ ./gologtail --help
usage: gologtail [<flags>]

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
      --dbh="localhost:27017"  MongoDB host
      --dbn="logs"             MongoDB database name
      --dbt="logs"             MongoDB table name
      --dbw=10                 Workers for add item to MongoDB
  -f, --format=1               Time format for log files
                               Values:
                                     1 = "Jan 2, 2006 at 3:04:05pm (UTC)"
                                     2 = "2006-01-02T15:04:05Z"
```

### Run
```
$ ./gologtail 
```

with flags
```
$ echo -e '/logs/test0.log\n/logs/test1.log' ./gologtail --dbh localhost:27017 --dbn logs --dbt logs --dbw 10 -f 1
```

## Testing
```
$ docker-compose up
```

## Testing
```go
package main

import (
	"log"
	"sync"

	"github.com/tusupov/gologtail/db"
	"github.com/tusupov/gologtail/tail"
)

func main() {

	logFilePath := "/logs/test0.log"
	timeFormat := 1
	
	// Create new
	t, err := tail.New(logFilePath, timeFormat)
	if err != nil {
		log.Fatal(err)
	}
	
	wg := &sync.WaitGroup{}
	lineItemCh := make(chan db.Item)
	
	// Enable debug
	t.Debug(true)

	// Listen file write
	go t.Run(wg, lineItemCh)
	
	// Read line
	for lineItem := range lineItemCh {
		log.Println(lineItem)
	}

}

```
