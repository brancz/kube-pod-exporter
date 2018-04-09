FROM golang:1.9.2

ADD . /go/src/github.com/brancz/kube-pod-exporter

RUN cd /go/src/github.com/brancz/kube-pod-exporter && \
    apt-get update && \
    apt-get install unzip && \
    /go/src/github.com/brancz/kube-pod-exporter/scripts/install-protobuf.sh `cat PROTOC_VERSION | tr -d "\n"` && \
    go get github.com/gogo/protobuf/protoc-gen-gofast

WORKDIR /go/src/github.com/brancz/kube-pod-exporter
