FROM golang:buster
COPY . /go/src
WORKDIR /go/src
RUN go mod download
