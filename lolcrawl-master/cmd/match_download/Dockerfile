FROM golang:latest

RUN go get github.com/cespare/reflex

ADD . /go/src/github.com/VantageSports/lolcrawl
WORKDIR /go/src/github.com/VantageSports/lolcrawl

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version
RUN go install ./cmd/match_download

CMD ["/go/bin/match_download"]
