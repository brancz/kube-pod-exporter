/*
Copyright 2018 Frederic Branczyk All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"github.com/brancz/kube-pod-exporter/collector"
	cri "github.com/brancz/kube-pod-exporter/runtime"
)

type config struct {
	containerRuntimeEndpoint string
	listenAddress            string
}

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

func dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(unixProtocol, addr, timeout)
}

func GetAddressAndDialer(endpoint string) (string, func(addr string, timeout time.Duration) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != unixProtocol {
		return "", nil, fmt.Errorf("only support unix socket endpoint")
	}

	return addr, dial, nil
}

func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = parseEndpoint(fallbackEndpoint)
		if err == nil {
			glog.Warningf("Using %q as endpoint is deprecated, please consider using full url format %q.", endpoint, fallbackEndpoint)
		}
	}
	return
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	if u.Scheme == "tcp" {
		return "tcp", u.Host, nil
	} else if u.Scheme == "unix" {
		return "unix", u.Path, nil
	} else if u.Scheme == "" {
		return "", "", fmt.Errorf("Using %q as endpoint is deprecated, please consider using full url format", endpoint)
	} else {
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
	}
}

func Main() int {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	cfg := config{}
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// kube-pod-exporter flags
	flagset.StringVar(&cfg.listenAddress, "listen-address", "", "The address the kube-pod-exporter HTTP server should listen on.")
	flagset.StringVar(&cfg.containerRuntimeEndpoint, "container-runtime-endpoint", "unix///var/run/dockershim.sock", "The endpoint to connect to the CRI via.")
	flagset.Parse(os.Args[1:])

	r := prometheus.NewRegistry()

	logger.Log("msg", fmt.Sprintf("dialing gRPC endpoint %s", cfg.containerRuntimeEndpoint))
	addr, dailer, err := GetAddressAndDialer(cfg.containerRuntimeEndpoint)
	if err != nil {
		logger.Log("msg", "failed to get address and dialer", "err", err)
		return 1
	}

	conn, err := grpc.Dial(
		addr,
		grpc.WithInsecure(),
		grpc.WithDialer(dailer),
	)
	if err != nil {
		logger.Log("msg", "failed to dial container runtime endpoint", "err", err)
		return 1
	}
	defer conn.Close()

	cri := cri.NewRuntimeServiceClient(conn)
	criCollector := collector.New(log.With(logger, "component", "collector"), cri)
	err = r.Register(criCollector)
	if err != nil {
		logger.Log("msg", "failed to register cri-collector", "err", err)
		return 1
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))

	srv := &http.Server{Handler: mux}

	l, err := net.Listen("tcp", cfg.listenAddress)
	if err != nil {
		logger.Log("msg", "failed listen on address", "err", err)
		return 1
	}
	go srv.Serve(l)
	logger.Log("msg", fmt.Sprintf("listening on %s", cfg.listenAddress))

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		logger.Log("msg", "Received SIGTERM, exiting gracefully...")
		return 0
	}
}

func main() {
	os.Exit(Main())
}
