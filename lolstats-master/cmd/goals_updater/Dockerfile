FROM golang:latest

RUN go get github.com/cespare/reflex

ADD . /go/src/github.com/VantageSports/lolstats
WORKDIR /go/src/github.com/VantageSports/lolstats

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version
RUN go install ./cmd/goals_updater

CMD ["/go/bin/goals_updater"]
