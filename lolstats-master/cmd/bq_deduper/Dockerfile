FROM golang:latest

RUN go get github.com/cespare/reflex
RUN go get github.com/tools/godep

WORKDIR /go/src/github.com/VantageSports/lolstats
ADD . /go/src/github.com/VantageSports/lolstats

RUN godep go install ./cmd/bq_deduper

CMD ["/go/bin/bq_deduper"]

