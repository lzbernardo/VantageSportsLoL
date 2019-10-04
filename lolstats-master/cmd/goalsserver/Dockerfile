FROM golang:latest

RUN go get github.com/cespare/reflex

WORKDIR /go/src/github.com/VantageSports/lolstats
ADD . /go/src/github.com/VantageSports/lolstats

RUN go install ./cmd/goalsserver

EXPOSE 8080
CMD ["/go/bin/goalsserver"]

