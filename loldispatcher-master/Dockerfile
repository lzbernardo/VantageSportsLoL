FROM golang:latest

RUN go get github.com/VantageSports/reflex

WORKDIR /go/src/github.com/VantageSports/loldispatch
ADD . /go/src/github.com/VantageSports/loldispatch

RUN echo "Built `date` @ `git rev-parse HEAD`" > ./version

# Since the main is currently in the base repo, we force the binary name.
RUN go install


CMD ["/go/bin/loldispatch"]
