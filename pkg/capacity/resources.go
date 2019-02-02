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

type clusterResource struct {
	cpuAllocatable resource.Quantity
	cpuRequest     resource.Quantity
	cpuLimit       resource.Quantity
	memAllocatable resource.Quantity
	memRequest     resource.Quantity
	memLimit       resource.Quantity
	capacityByNode map[string]*nodeResource
}

type nodeResource struct {
	cpuAllocatable resource.Quantity
	cpuRequest     resource.Quantity
	cpuLimit       resource.Quantity
	memAllocatable resource.Quantity
	memRequest     resource.Quantity
	memLimit       resource.Quantity
	podResources   []podResource
}

type podResource struct {
	name       string
	namespace  string
	cpuRequest resource.Quantity
	cpuLimit   resource.Quantity
	memRequest resource.Quantity
	memLimit   resource.Quantity
}

func (nr *nodeResource) addPodResources(pod *corev1.Pod) {
	req, limit := resourcehelper.PodRequestsAndLimits(pod)

	nr.podResources = append(nr.podResources, podResource{
		name:       pod.Name,
		namespace:  pod.Namespace,
		cpuRequest: req["cpu"],
		cpuLimit:   limit["cpu"],
		memRequest: req["memory"],
		memLimit:   limit["memory"],
	})

	nr.cpuRequest.Add(req["cpu"])
	nr.cpuLimit.Add(limit["cpu"])
	nr.memRequest.Add(req["memory"])
	nr.memLimit.Add(limit["memory"])
}

func (cr *clusterResource) addNodeCapacity(nr *nodeResource) {
	cr.cpuAllocatable.Add(nr.cpuAllocatable)
	cr.cpuRequest.Add(nr.cpuRequest)
	cr.cpuLimit.Add(nr.cpuLimit)
	cr.memAllocatable.Add(nr.memAllocatable)
	cr.memRequest.Add(nr.memRequest)
	cr.memLimit.Add(nr.memLimit)
}

func (cr *clusterResource) cpuRequestString() string {
	return resourceString(cr.cpuRequest, cr.cpuAllocatable)
}

func (cr *clusterResource) cpuLimitString() string {
	return resourceString(cr.cpuLimit, cr.cpuAllocatable)
}

func (cr *clusterResource) memRequestString() string {
	return resourceString(cr.memRequest, cr.memAllocatable)
}

func (cr *clusterResource) memLimitString() string {
	return resourceString(cr.memLimit, cr.memAllocatable)
}

func (nr *nodeResource) cpuRequestString() string {
	return resourceString(nr.cpuRequest, nr.cpuAllocatable)
}

func (nr *nodeResource) cpuLimitString() string {
	return resourceString(nr.cpuLimit, nr.cpuAllocatable)
}

func (nr *nodeResource) memRequestString() string {
	return resourceString(nr.memRequest, nr.memAllocatable)
}

func (nr *nodeResource) memLimitString() string {
	return resourceString(nr.memLimit, nr.memAllocatable)
}

func (pr *podResource) cpuRequestString(nr *nodeResource) string {
	return resourceString(pr.cpuRequest, nr.cpuAllocatable)
}

func (pr *podResource) cpuLimitString(nr *nodeResource) string {
	return resourceString(pr.cpuLimit, nr.cpuAllocatable)
}

func (pr *podResource) memRequestString(nr *nodeResource) string {
	return resourceString(pr.memRequest, nr.memAllocatable)
}

func (pr *podResource) memLimitString(nr *nodeResource) string {
	return resourceString(pr.memLimit, nr.memAllocatable)
}

func resourceString(actual, allocatable resource.Quantity) string {
	utilPercent := float64(0)
	if allocatable.MilliValue() > 0 {
		utilPercent = float64(actual.MilliValue()) / float64(allocatable.MilliValue()) * 100
	}
	return fmt.Sprintf("%s (%d%%)", actual.String(), int64(utilPercent))
}
