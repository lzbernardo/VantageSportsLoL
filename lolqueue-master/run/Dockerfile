FROM golang:latest

RUN go get github.com/VantageSports/reflex # https://github.com/cespare/reflex/issues/32

WORKDIR /go/src/github.com/VantageSports/lolqueue
ADD . /go/src/github.com/VantageSports/lolqueue

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

RUN go install ./cmd/server

CMD ["/go/bin/server"]
