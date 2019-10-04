FROM golang:latest

RUN go get github.com/cespare/reflex

WORKDIR /go/src/github.com/VantageSports/lolcrawl
ADD . /go/src/github.com/VantageSports/lolcrawl

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

# Since the main is currently in the base repo, we force the binary name.
RUN go install ./cmd/replay_server

EXPOSE 8080
CMD ["/go/bin/replay_server"]
