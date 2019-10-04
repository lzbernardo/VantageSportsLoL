#!/usr/bin/env sh

# Install proto3 from source
#  brew install autoconf automake libtool
#  git clone https://github.com/google/protobuf
#  ./autogen.sh ; ./configure ; make ; make install
#
# Update protoc Go bindings via
#  go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
#
# See also
#  https://github.com/grpc/grpc-go/tree/master/examples

protoc users.proto --go_out=plugins=grpc:.

# Now remove the two String() methods that we've already defined in string_methods.go
grep -vE "\LoginRequest\) String\(\)|\*User\) String\(\)" users.pb.go > users.pb.go.stripped && mv users.pb.go.stripped users.pb.go
