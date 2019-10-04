FROM golang:latest

RUN go get github.com/VantageSports/reflex # https://github.com/cespare/reflex/issues/32

WORKDIR /go/src/github.com/VantageSports/lolelo
ADD . /go/src/github.com/VantageSports/lolelo

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

RUN go install ./cmd/convert_runner

CMD ["/go/bin/convert_runner"]
