FROM alpine:3.7

COPY _output/linux/amd64/kube-pod-exporter /

ENTRYPOINT ["/kube-pod-exporter"]
