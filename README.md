# kube-pod-exporter

[![Docker Repository on Quay](https://quay.io/repository/brancz/kube-pod-exporter/status "Docker Repository on Quay")](https://quay.io/repository/brancz/kube-pod-exporter)

> NOTE: This project is *alpha* stage. Flags, configuration, behavior and design may change significantly in following releases.

The kube-pod-exporter is a shim between the Kubernetes CRI and the Prometheus metrics format. On metric scrapes from a Prometheus server, the kube-pod-exporter uses the CRI stats endpoints to retrieve CPU and memory metrics from all Pods running on the respective Kubernetes Node.

## Usage

All command line flags:

[embedmd]:# (_output/help.txt)
```txt
$ kube-pod-exporter -h
Usage of _output/linux/amd64/kube-pod-exporter:
  -container-runtime-endpoint string
    	The endpoint to connect to the CRI via. (default "passthrough:///unix///var/run/dockershim.sock")
  -listen-address string
    	The address the kube-pod-exporter HTTP server should listen on.
```
