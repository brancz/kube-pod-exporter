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

package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	cri "github.com/brancz/kube-pod-exporter/runtime"
)

var (
	descContainerCpuUsageSeconds = prometheus.NewDesc(
		"container_cpu_usage_seconds_total",
		"Total CPU usage of the container.",
		[]string{"namespace", "pod", "container"}, nil,
	)
	descContainerMemoryUsageWorkingSetBytes = prometheus.NewDesc(
		"container_memory_usage_working_set_bytes",
		"Current memory usage of the container.",
		[]string{"namespace", "pod", "container"}, nil,
	)
)

type criSubset interface {
	ListPodSandbox(context.Context, *cri.ListPodSandboxRequest, ...grpc.CallOption) (*cri.ListPodSandboxResponse, error)
	ListContainers(context.Context, *cri.ListContainersRequest, ...grpc.CallOption) (*cri.ListContainersResponse, error)
	ListContainerStats(context.Context, *cri.ListContainerStatsRequest, ...grpc.CallOption) (*cri.ListContainerStatsResponse, error)
}

type collector struct {
	cri    criSubset
	logger log.Logger
}

func New(logger log.Logger, cri criSubset) *collector {
	return &collector{
		logger: logger,
		cri:    cri,
	}
}

// Describe implements the prometheus.Collector interface.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descContainerCpuUsageSeconds
	ch <- descContainerMemoryUsageWorkingSetBytes
}

// Collect implements the prometheus.Collector interface.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	listPodsResponse, err := c.cri.ListPodSandbox(ctx, &cri.ListPodSandboxRequest{})
	if err != nil {
		c.logger.Log("err", err)
	}
	podIndex := indexPods(listPodsResponse.Items)

	listContainersResponse, err := c.cri.ListContainers(ctx, &cri.ListContainersRequest{})
	if err != nil {
		c.logger.Log("err", err)
	}
	containerIndex := indexContainers(listContainersResponse.Containers)

	listContainerStatsResponse, err := c.cri.ListContainerStats(ctx, &cri.ListContainerStatsRequest{})
	if err != nil {
		c.logger.Log("err", err)
	}

	c.collectContainerMetrics(ch, listContainerStatsResponse.Stats, containerIndex, podIndex)
}

func (c *collector) collectContainerMetrics(ch chan<- prometheus.Metric, stats []*cri.ContainerStats, containerIndex map[string]*cri.Container, podIndex map[string]*cri.PodSandbox) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addCounter := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.CounterValue, v, lv...)
	}

	for _, s := range stats {
		container, found := containerIndex[s.Attributes.Id]
		if !found {
			c.logger.Log(fmt.Sprintf("container stat found, but container not found in index (%v)", s.Attributes.Id))
			continue
		}

		pod, found := podIndex[container.PodSandboxId]
		if !found {
			c.logger.Log(fmt.Sprintf("container stat and container found, but pod not found in index (%v)", container.PodSandboxId))
			continue
		}

		usageCoreSeconds := float64(s.Cpu.UsageCoreNanoSeconds.Value) / float64(time.Second)
		addCounter(descContainerCpuUsageSeconds, float64(usageCoreSeconds), pod.Metadata.Namespace, pod.Metadata.Name, container.Metadata.Name)
		addGauge(descContainerMemoryUsageWorkingSetBytes, float64(s.Memory.WorkingSetBytes.Value), pod.Metadata.Namespace, pod.Metadata.Name, container.Metadata.Name)
	}
}

func indexPods(pods []*cri.PodSandbox) map[string]*cri.PodSandbox {
	index := make(map[string]*cri.PodSandbox, len(pods))
	for _, p := range pods {
		index[p.Id] = p
	}

	return index
}

func indexContainers(containers []*cri.Container) map[string]*cri.Container {
	index := make(map[string]*cri.Container, len(containers))
	for _, c := range containers {
		index[c.Id] = c
	}

	return index
}
