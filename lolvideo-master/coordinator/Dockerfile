FROM ubuntu:14.04

RUN apt-get update

RUN apt-get -y install git-all

# Install vncViewer
RUN apt-get -y install xvnc4viewer

# Install xorg dummy
RUN apt-get -y install xserver-xorg-video-dummy

# Install pip
RUN apt-get -y install python-pip

# Install vncdotool
RUN apt-get -y install libjpeg-dev libpng-dev python-dev
RUN pip install vncdotool

# Install VNC Password decrypter
RUN git clone https://github.com/jeroennijhof/vncpwd.git
RUN gcc -o /usr/local/bin/vncpwd vncpwd/vncpwd.c vncpwd/d3des.c

# Install Go
RUN apt-get -y install curl
RUN curl -O https://storage.googleapis.com/golang/go1.7.3.linux-amd64.tar.gz
RUN tar -xvf go1.7.3.linux-amd64.tar.gz
RUN mv go /usr/local
ENV GOPATH /go
ENV PATH $PATH:/usr/local/go/bin:/go/bin
RUN go get github.com/tools/godep

WORKDIR /go/src/github.com/VantageSports/lolvideo
ADD . /go/src/github.com/VantageSports/lolvideo

RUN godep go install ./coordinator/cmd/coordinator_server

ENV GOOS windows
RUN godep go install ./worker/cmd/videogenWorker
RUN godep go install ./worker/cmd/datagenWorker
ADD worker/replay.bat /go/bin/windows_amd64/
ENV GOOS linux

EXPOSE 8080
CMD ["/go/bin/coordinator_server"]

