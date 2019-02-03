// Copyright 2019 Rob Scott
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
)

type resourceMetric struct {
	allocatable resource.Quantity
	utilization resource.Quantity
	request     resource.Quantity
	limit       resource.Quantity
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

func (cm *clusterMetric) addPodMetric(pod *corev1.Pod) {
	req, limit := resourcehelper.PodRequestsAndLimits(pod)
	key := fmt.Sprintf("%s-%s", pod.Namespace, pod.Name)

	cm.podMetrics[key] = &podMetric{
		name:      pod.Name,
		namespace: pod.Namespace,
		cpu: &resourceMetric{
			request: req["cpu"],
			limit:   limit["cpu"],
		},
		memory: &resourceMetric{
			request: req["memory"],
			limit:   limit["memory"],
		},
	}

	nm := cm.nodeMetrics[pod.Spec.NodeName]
	if nm != nil {
		nm.podMetrics[key] = cm.podMetrics[key]
		nm.cpu.request.Add(req["cpu"])
		nm.cpu.limit.Add(limit["cpu"])
		nm.memory.request.Add(req["memory"])
		nm.memory.limit.Add(limit["memory"])
	}
}

func (cm *clusterMetric) addNodeMetric(nm *nodeMetric) {
	cm.cpu.addMetric(nm.cpu)
	cm.memory.addMetric(nm.memory)
}

func (rm *resourceMetric) requestString() string {
	return resourceString(rm.request, rm.allocatable)
}

func (rm *resourceMetric) limitString() string {
	return resourceString(rm.limit, rm.allocatable)
}

func (rm *resourceMetric) utilStringMilli() string {
	utilPercent := float64(0)
	if rm.allocatable.MilliValue() > 0 {
		utilPercent = float64(rm.utilization.MilliValue()) / float64(rm.allocatable.MilliValue()) * 100
	}
	return fmt.Sprintf("%dm (%d%%)", rm.utilization.MilliValue(), int64(utilPercent))
}

func (rm *resourceMetric) utilStringMebi() string {
	utilPercent := float64(0)
	if rm.allocatable.MilliValue() > 0 {
		utilPercent = float64(rm.utilization.MilliValue()) / float64(rm.allocatable.MilliValue()) * 100
	}
	return fmt.Sprintf("%dMi (%d%%)", rm.utilization.Value()/1048576, int64(utilPercent))
}

func (rm *resourceMetric) requestStringPar(pm *resourceMetric) string {
	return resourceString(rm.request, pm.allocatable)
}

func (rm *resourceMetric) limitStringPar(pm *resourceMetric) string {
	return resourceString(rm.limit, pm.allocatable)
}

func (rm *resourceMetric) utilStringParMilli(pm *resourceMetric) string {
	utilPercent := float64(0)
	if pm.allocatable.MilliValue() > 0 {
		utilPercent = float64(rm.utilization.MilliValue()) / float64(pm.allocatable.MilliValue()) * 100
	}
	return fmt.Sprintf("%dm (%d%%)", rm.utilization.MilliValue(), int64(utilPercent))
}

func (rm *resourceMetric) utilStringParMebi(pm *resourceMetric) string {
	utilPercent := float64(0)
	if pm.allocatable.MilliValue() > 0 {
		utilPercent = float64(rm.utilization.MilliValue()) / float64(pm.allocatable.MilliValue()) * 100
	}
	return fmt.Sprintf("%dMi (%d%%)", rm.utilization.Value()/1048576, int64(utilPercent))
}

func resourceString(actual, allocatable resource.Quantity) string {
	utilPercent := float64(0)
	if allocatable.MilliValue() > 0 {
		utilPercent = float64(actual.MilliValue()) / float64(allocatable.MilliValue()) * 100
	}
	return fmt.Sprintf("%s (%d%%)", actual.String(), int64(utilPercent))
}
