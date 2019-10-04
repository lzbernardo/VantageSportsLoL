FROM golang:latest

RUN go get github.com/VantageSports/reflex

WORKDIR /go/src/github.com/VantageSports/lolobserver
ADD . /go/src/github.com/VantageSports/lolobserver

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

RUN go install ./cmd/observer

CMD ["/go/bin/observer"]
