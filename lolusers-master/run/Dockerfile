FROM golang:latest

RUN go get github.com/VantageSports/reflex
RUN go get github.com/tools/godep

WORKDIR /go/src/github.com/VantageSports/lolusers
ADD . /go/src/github.com/VantageSports/lolusers

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

# Since the main is currently in the base repo, we force the binary name.
RUN godep go install ./cmd/loluserserver


EXPOSE 8080
CMD ["/go/bin/loluserserver"]
