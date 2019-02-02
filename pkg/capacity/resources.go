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

func (nr *nodeResource) cpuRequestString() string {
	return fmt.Sprintf("%s / %s", nr.cpuRequest.String(), nr.cpuAllocatable.String())
}

func (nr *nodeResource) cpuLimitString() string {
	return fmt.Sprintf("%s / %s", nr.cpuLimit.String(), nr.cpuAllocatable.String())
}

func (nr *nodeResource) memRequestString() string {
	return fmt.Sprintf("%s / %s", nr.memRequest.String(), nr.memAllocatable.String())
}

func (nr *nodeResource) memLimitString() string {
	return fmt.Sprintf("%s / %s", nr.memLimit.String(), nr.memAllocatable.String())
}
