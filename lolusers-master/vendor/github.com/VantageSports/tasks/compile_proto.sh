#!/bin/bash

# Install proto3 from source
#  apt-get install autoconf automake libtool curl make g++ unzip
#    OR
#  brew install autoconf automake libtool
#
#  git clone https://github.com/google/protobuf
#  ./autogen.sh ; ./configure ; make ; sudo make install
#
# Update protoc Go bindings via
#  go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
#
# See also
#  https://github.com/grpc/grpc-go/tree/master/examples

protoc --go_out=Mvendor/github.com/VantageSports/common/queue/messages/nba.proto=github.com/VantageSports/common/queue/messages,Mvendor/github.com/VantageSports/common/queue/messages/common.proto=github.com/VantageSports/common/queue/messages,plugins=grpc:. tasks.proto
