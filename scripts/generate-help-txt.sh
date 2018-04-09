#!/usr/bin/env bash

echo "$ kube-pod-exporter -h" > _output/help.txt
_output/linux/amd64/kube-pod-exporter -h 2>> _output/help.txt
exit 0
