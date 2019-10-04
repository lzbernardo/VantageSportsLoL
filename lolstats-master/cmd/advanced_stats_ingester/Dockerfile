FROM golang:latest

RUN go get github.com/VantageSports/reflex # https://github.com/cespare/reflex/issues/32

WORKDIR /go/src/github.com/VantageSports/lolstats
ADD . /go/src/github.com/VantageSports/lolstats

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

RUN go install ./cmd/advanced_stats_ingester

CMD ["/go/bin/advanced_stats_ingester"]
