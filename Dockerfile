FROM golang:1.13-rc-buster
COPY . /go/src
WORKDIR /go/src
RUN go mod download
