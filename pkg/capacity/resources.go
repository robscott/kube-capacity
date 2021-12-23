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
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	resourcehelper "k8s.io/kubectl/pkg/util/resource"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// SupportedSortAttributes lists the valid sorting options
var SupportedSortAttributes = [...]string{
	"cpu.util",
	"cpu.request",
	"cpu.limit",
	"mem.util",
	"mem.request",
	"mem.limit",
	"name",
}

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
}

type nodeMetric struct {
	name       string
	cpu        *resourceMetric
	memory     *resourceMetric
	podMetrics map[string]*podMetric
	podCount   string
}

type podMetric struct {
	name             string
	namespace        string
	cpu              *resourceMetric
	memory           *resourceMetric
	containerMetrics map[string]*containerMetric
}

type containerMetric struct {
	name   string
	cpu    *resourceMetric
	memory *resourceMetric
}

func buildClusterMetric(podList *corev1.PodList, pmList *v1beta1.PodMetricsList,
	nodeList *corev1.NodeList, nmList *v1beta1.NodeMetricsList) clusterMetric {
	cm := clusterMetric{
		cpu:         &resourceMetric{resourceType: "cpu"},
		memory:      &resourceMetric{resourceType: "memory"},
		nodeMetrics: map[string]*nodeMetric{},
	}

	for _, node := range nodeList.Items {
		var tmpPodCount int
		for _, pod := range podList.Items {
			if pod.Spec.NodeName == node.Name {
				tmpPodCount++
			}
		}
		podCount := fmt.Sprintf("%d/%d", tmpPodCount, node.Status.Allocatable.Pods().Value())

		cm.nodeMetrics[node.Name] = &nodeMetric{
			name: node.Name,
			cpu: &resourceMetric{
				resourceType: "cpu",
				allocatable:  node.Status.Allocatable["cpu"],
			},
			memory: &resourceMetric{
				resourceType: "memory",
				allocatable:  node.Status.Allocatable["memory"],
			},
			podMetrics: map[string]*podMetric{},
			podCount: podCount,
		}
	}

	for _, nm := range nmList.Items {
		cm.nodeMetrics[nm.Name].cpu.utilization = nm.Usage["cpu"]
		cm.nodeMetrics[nm.Name].memory.utilization = nm.Usage["memory"]
	}

	podMetrics := map[string]v1beta1.PodMetrics{}
	for _, pm := range pmList.Items {
		podMetrics[fmt.Sprintf("%s-%s", pm.GetNamespace(), pm.GetName())] = pm
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			cm.addPodMetric(&pod, podMetrics[fmt.Sprintf("%s-%s", pod.GetNamespace(), pod.GetName())])
		}
	}

	for _, node := range nodeList.Items {
		nm := cm.nodeMetrics[node.Name]
		if nm != nil {
			cm.addNodeMetric(cm.nodeMetrics[node.Name])
		}
	}

	return cm
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
	nm := cm.nodeMetrics[pod.Spec.NodeName]

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
		containerMetrics: map[string]*containerMetric{},
	}

	for _, container := range pod.Spec.Containers {
		pm.containerMetrics[container.Name] = &containerMetric{
			name: container.Name,
			cpu: &resourceMetric{
				resourceType: "cpu",
				request:      container.Resources.Requests["cpu"],
				limit:        container.Resources.Limits["cpu"],
				allocatable:  nm.cpu.allocatable,
			},
			memory: &resourceMetric{
				resourceType: "memory",
				request:      container.Resources.Requests["memory"],
				limit:        container.Resources.Limits["memory"],
				allocatable:  nm.memory.allocatable,
			},
		}
	}

	if nm != nil {
		nm.podMetrics[key] = pm
		nm.podMetrics[key].cpu.allocatable = nm.cpu.allocatable
		nm.podMetrics[key].memory.allocatable = nm.memory.allocatable

		nm.cpu.request.Add(req["cpu"])
		nm.cpu.limit.Add(limit["cpu"])
		nm.memory.request.Add(req["memory"])
		nm.memory.limit.Add(limit["memory"])
	}

	for _, container := range podMetrics.Containers {
		cm := pm.containerMetrics[container.Name]
		if cm != nil {
			pm.containerMetrics[container.Name].cpu.utilization = container.Usage["cpu"]
			pm.cpu.utilization.Add(container.Usage["cpu"])
			pm.containerMetrics[container.Name].memory.utilization = container.Usage["memory"]
			pm.memory.utilization.Add(container.Usage["memory"])
		}
	}
}

func (cm *clusterMetric) addNodeMetric(nm *nodeMetric) {
	cm.cpu.addMetric(nm.cpu)
	cm.memory.addMetric(nm.memory)
}

func (cm *clusterMetric) getSortedNodeMetrics(sortBy string) []*nodeMetric {
	sortedNodeMetrics := make([]*nodeMetric, len(cm.nodeMetrics))

	i := 0
	for name := range cm.nodeMetrics {
		sortedNodeMetrics[i] = cm.nodeMetrics[name]
		i++
	}

	sort.Slice(sortedNodeMetrics, func(i, j int) bool {
		m1 := sortedNodeMetrics[i]
		m2 := sortedNodeMetrics[j]

		switch sortBy {
		case "cpu.util":
			return m2.cpu.utilization.MilliValue() < m1.cpu.utilization.MilliValue()
		case "cpu.limit":
			return m2.cpu.limit.MilliValue() < m1.cpu.limit.MilliValue()
		case "cpu.request":
			return m2.cpu.request.MilliValue() < m1.cpu.request.MilliValue()
		case "mem.util":
			return m2.memory.utilization.Value() < m1.memory.utilization.Value()
		case "mem.limit":
			return m2.memory.limit.Value() < m1.memory.limit.Value()
		case "mem.request":
			return m2.memory.request.Value() < m1.memory.request.Value()
		default:
			return m1.name < m2.name
		}
	})

	return sortedNodeMetrics
}

func (nm *nodeMetric) getSortedPodMetrics(sortBy string) []*podMetric {
	sortedPodMetrics := make([]*podMetric, len(nm.podMetrics))

	i := 0
	for name := range nm.podMetrics {
		sortedPodMetrics[i] = nm.podMetrics[name]
		i++
	}

	sort.Slice(sortedPodMetrics, func(i, j int) bool {
		m1 := sortedPodMetrics[i]
		m2 := sortedPodMetrics[j]

		switch sortBy {
		case "cpu.util":
			return m2.cpu.utilization.MilliValue() < m1.cpu.utilization.MilliValue()
		case "cpu.limit":
			return m2.cpu.limit.MilliValue() < m1.cpu.limit.MilliValue()
		case "cpu.request":
			return m2.cpu.request.MilliValue() < m1.cpu.request.MilliValue()
		case "mem.util":
			return m2.memory.utilization.Value() < m1.memory.utilization.Value()
		case "mem.limit":
			return m2.memory.limit.Value() < m1.memory.limit.Value()
		case "mem.request":
			return m2.memory.request.Value() < m1.memory.request.Value()
		default:
			return m1.name < m2.name
		}
	})

	return sortedPodMetrics
}

func (pm *podMetric) getSortedContainerMetrics(sortBy string) []*containerMetric {
	sortedContainerMetrics := make([]*containerMetric, len(pm.containerMetrics))

	i := 0
	for name := range pm.containerMetrics {
		sortedContainerMetrics[i] = pm.containerMetrics[name]
		i++
	}

	sort.Slice(sortedContainerMetrics, func(i, j int) bool {
		m1 := sortedContainerMetrics[i]
		m2 := sortedContainerMetrics[j]

		switch sortBy {
		case "cpu.util":
			return m2.cpu.utilization.MilliValue() < m1.cpu.utilization.MilliValue()
		case "cpu.limit":
			return m2.cpu.limit.MilliValue() < m1.cpu.limit.MilliValue()
		case "cpu.request":
			return m2.cpu.request.MilliValue() < m1.cpu.request.MilliValue()
		case "mem.util":
			return m2.memory.utilization.Value() < m1.memory.utilization.Value()
		case "mem.limit":
			return m2.memory.limit.Value() < m1.memory.limit.Value()
		case "mem.request":
			return m2.memory.request.Value() < m1.memory.request.Value()
		default:
			return m1.name < m2.name
		}
	})

	return sortedContainerMetrics
}

func (rm *resourceMetric) requestString(availableFormat bool) string {
	return resourceString(rm.request, rm.allocatable, availableFormat)
}

func (rm *resourceMetric) limitString(availableFormat bool) string {
	return resourceString(rm.limit, rm.allocatable, availableFormat)
}

func (rm *resourceMetric) utilString(availableFormat bool) string {
	return resourceString(rm.utilization, rm.allocatable, availableFormat)
}

func resourceString(actual, allocatable resource.Quantity, availableFormat bool) string {
	utilPercent := float64(0)
	if allocatable.MilliValue() > 0 {
		utilPercent = float64(actual.MilliValue()) / float64(allocatable.MilliValue()) * 100
	}

	var actualStr, allocatableStr string

	if availableFormat {
		if actual.Format == resource.DecimalSI {
			actualStr = fmt.Sprintf("%dm", allocatable.MilliValue()-actual.MilliValue())
			allocatableStr = fmt.Sprintf("%dm", allocatable.MilliValue())
		} else {
			actualStr = fmt.Sprintf("%dMi", allocatable.ScaledValue(resource.Mega)-actual.ScaledValue(resource.Mega))
			allocatableStr = fmt.Sprintf("%dMi", allocatable.ScaledValue(resource.Mega))
		}

		return fmt.Sprintf("%s/%s", actualStr, allocatableStr)
	}

	if actual.Format == resource.DecimalSI {
		actualStr = fmt.Sprintf("%dm", actual.MilliValue())
	} else {
		actualStr = fmt.Sprintf("%dMi", actual.ScaledValue(resource.Mega))
	}

	return fmt.Sprintf("%s (%d%%%%)", actualStr, int64(utilPercent))

}

// NOTE: This might not be a great place for closures due to the cyclical nature of how resourceType works. Perhaps better implemented another way.
func (rm resourceMetric) valueFunction() (f func(r resource.Quantity) string) {
	switch rm.resourceType {
	case "cpu":
		f = func(r resource.Quantity) string {
			return fmt.Sprintf("%s", r.String())
		}
	case "memory":
		f = func(r resource.Quantity) string {
			return fmt.Sprintf("%s", r.String())
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
