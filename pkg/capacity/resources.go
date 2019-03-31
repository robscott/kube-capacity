// Copyright 2019 Kube Capacity Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package capacity

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	resourcehelper "k8s.io/kubernetes/pkg/kubectl/util/resource"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type resourceMetric struct {
	resourceType string
	allocatable  resource.Quantity
	utilization  resource.Quantity
	request      resource.Quantity
	limit        resource.Quantity
}

type clusterMetric struct {
	cpu         *resourceMetric
	memory      *resourceMetric
	nodeMetrics map[string]*nodeMetric
	podMetrics  map[string]*podMetric
}

type nodeMetric struct {
	cpu        *resourceMetric
	memory     *resourceMetric
	podMetrics map[string]*podMetric
}

type podMetric struct {
	name      string
	namespace string
	cpu       *resourceMetric
	memory    *resourceMetric
}

func (rm *resourceMetric) addMetric(m *resourceMetric) {
	rm.allocatable.Add(m.allocatable)
	rm.utilization.Add(m.utilization)
	rm.request.Add(m.request)
	rm.limit.Add(m.limit)
}

func (cm *clusterMetric) addPodMetric(pod *corev1.Pod, podMetrics v1beta1.PodMetrics) {
	req, limit := resourcehelper.PodRequestsAndLimits(pod)
	key := fmt.Sprintf("%s-%s", pod.Namespace, pod.Name)

	pm := &podMetric{
		name:      pod.Name,
		namespace: pod.Namespace,
		cpu: &resourceMetric{
			resourceType: "cpu",
			request:      req["cpu"],
			limit:        limit["cpu"],
		},
		memory: &resourceMetric{
			resourceType: "memory",
			request:      req["memory"],
			limit:        limit["memory"],
		},
	}
	cm.podMetrics[key] = pm

	nm := cm.nodeMetrics[pod.Spec.NodeName]
	if nm != nil {
		cm.cpu.request.Add(req["cpu"])
		cm.cpu.limit.Add(limit["cpu"])
		cm.memory.request.Add(req["memory"])
		cm.memory.limit.Add(limit["memory"])

		cm.podMetrics[key].cpu.allocatable = nm.cpu.allocatable
		cm.podMetrics[key].memory.allocatable = nm.memory.allocatable
		nm.podMetrics[key] = cm.podMetrics[key]
		nm.cpu.request.Add(req["cpu"])
		nm.cpu.limit.Add(limit["cpu"])
		nm.memory.request.Add(req["memory"])
		nm.memory.limit.Add(limit["memory"])
	}

	for _, container := range podMetrics.Containers {
		pm.cpu.utilization.Add(container.Usage["cpu"])
		pm.memory.utilization.Add(container.Usage["memory"])

		if nm == nil {
			continue
		}

		nm.cpu.utilization.Add(container.Usage["cpu"])
		nm.memory.utilization.Add(container.Usage["memory"])

		cm.cpu.utilization.Add(container.Usage["cpu"])
		cm.memory.utilization.Add(container.Usage["memory"])
	}
}

func (cm *clusterMetric) addNodeMetric(nm *nodeMetric) {
	cm.cpu.addMetric(nm.cpu)
	cm.memory.addMetric(nm.memory)
}

func (rm *resourceMetric) requestString() string {
	return resourceString(rm.request, rm.allocatable, rm.resourceType)
}

func (rm *resourceMetric) limitString() string {
	return resourceString(rm.limit, rm.allocatable, rm.resourceType)
}

func (rm *resourceMetric) utilString() string {
	return resourceString(rm.utilization, rm.allocatable, rm.resourceType)
}

func resourceString(actual, allocatable resource.Quantity, resourceType string) string {
	utilPercent := float64(0)
	if allocatable.MilliValue() > 0 {
		utilPercent = float64(actual.MilliValue()) / float64(allocatable.MilliValue()) * 100
	}

	if resourceType == "cpu" {
		return fmt.Sprintf("%dm (%d%%)", actual.MilliValue(), int64(utilPercent))
	}
	return fmt.Sprintf("%dMi (%d%%)", actual.Value()/1048576, int64(utilPercent))
}

// NOTE: This might not be a great place for closures due to the cyclical nature of how resourceType works. Perhaps better implemented another way.
func (rm resourceMetric) valueFunction() (f func(r resource.Quantity) string) {
	switch rm.resourceType {
	case "cpu":
		f = func(r resource.Quantity) string {
			return fmt.Sprintf("%dm", r.MilliValue())
		}
	case "memory":
		f = func(r resource.Quantity) string {
			return fmt.Sprintf("%dMi", r.Value()/1048576)
		}
	}
	return f
}

// NOTE: This might not be a great place for closures due to the cyclical nature of how resourceType works. Perhaps better implemented another way.
func (rm resourceMetric) percentFunction() (f func(r resource.Quantity) string) {
	f = func(r resource.Quantity) string {
		return fmt.Sprintf("%v%%", int64(float64(r.MilliValue())/float64(rm.allocatable.MilliValue())*100))
	}
	return f
}
