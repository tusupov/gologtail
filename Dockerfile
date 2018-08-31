FROM golang:1.11

# copy project
WORKDIR /go/src/github.com/tusupov/gologtail
COPY . ./

# run test
RUN go test -v ./...
RUN rm -rf *_test.go

# build app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gologtail .

FROM alpine:latest

# copy app
WORKDIR /app/
COPY --from=0 /go/src/github.com/tusupov/gologtail/gologtail .

# run app
CMD echo -e $LOGS_LIST | ./gologtail -dbh $DB_HOST:$DB_PORT -dbn $DB_NAME -dbt $DB_TABLE -dbw $DB_WORKERS -format $LOGS_FORMAT
