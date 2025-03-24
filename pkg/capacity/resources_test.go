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
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestBuildClusterMetricEmpty(t *testing.T) {
	cm := buildClusterMetric(
		&corev1.PodList{}, &v1beta1.PodMetricsList{}, &corev1.NodeList{}, &v1beta1.NodeMetricsList{},
	)

	expected := clusterMetric{
		cpu: &resourceMetric{
			resourceType: "cpu",
			allocatable:  resource.Quantity{},
			request:      resource.Quantity{},
			limit:        resource.Quantity{},
			utilization:  resource.Quantity{},
		},
		memory: &resourceMetric{
			resourceType: "memory",
			allocatable:  resource.Quantity{},
			request:      resource.Quantity{},
			limit:        resource.Quantity{},
			utilization:  resource.Quantity{},
		},
		nodeMetrics: map[string]*nodeMetric{},
		podCount:    &podCount{},
	}

	assert.EqualValues(t, cm, expected)
}

func TestBuildClusterMetricFull(t *testing.T) {
	cm := buildClusterMetric(
		&corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-pod",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						NodeName: "example-node-1",
						Containers: []corev1.Container{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"cpu":    resource.MustParse("250m"),
										"memory": resource.MustParse("250Mi"),
									},
									Limits: corev1.ResourceList{
										"cpu":    resource.MustParse("250m"),
										"memory": resource.MustParse("500Mi"),
									},
								},
							},
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"cpu":    resource.MustParse("100m"),
										"memory": resource.MustParse("150Mi"),
									},
									Limits: corev1.ResourceList{
										"cpu":    resource.MustParse("150m"),
										"memory": resource.MustParse("200Mi"),
									},
								},
							},
						},
					},
				},
			},
		}, &v1beta1.PodMetricsList{
			Items: []v1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-pod",
						Namespace: "default",
					},
					Containers: []v1beta1.ContainerMetrics{
						{
							Usage: corev1.ResourceList{
								"cpu":    resource.MustParse("10m"),
								"memory": resource.MustParse("188Mi"),
							},
						},
						{
							Usage: corev1.ResourceList{
								"cpu":    resource.MustParse("13m"),
								"memory": resource.MustParse("111Mi"),
							},
						},
					},
				},
			},
		}, &corev1.NodeList{
			Items: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "example-node-1",
						Labels: map[string]string{
							"example.io/os": "example-os-1",
							"zone":          "example-zone-1",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							"cpu":    resource.MustParse("1000m"),
							"memory": resource.MustParse("4000Mi"),
						},
					},
				},
			},
		}, &v1beta1.NodeMetricsList{
			Items: []v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "example-node-1",
					},
					Usage: corev1.ResourceList{
						"cpu":    resource.MustParse("43m"),
						"memory": resource.MustParse("349Mi"),
					},
				},
			},
		},
	)

	cpuExpected := &resourceMetric{
		allocatable: resource.MustParse("1000m"),
		request:     resource.MustParse("350m"),
		limit:       resource.MustParse("400m"),
		utilization: resource.MustParse("43m"),
	}

	memoryExpected := &resourceMetric{
		allocatable: resource.MustParse("4000Mi"),
		request:     resource.MustParse("400Mi"),
		limit:       resource.MustParse("700Mi"),
		utilization: resource.MustParse("349Mi"),
	}

	assert.NotNil(t, cm.cpu)
	ensureEqualResourceMetric(t, cm.cpu, cpuExpected)
	assert.NotNil(t, cm.memory)
	ensureEqualResourceMetric(t, cm.memory, memoryExpected)

	assert.NotNil(t, cm.nodeMetrics["example-node-1"])
	assert.NotNil(t, cm.nodeMetrics["example-node-1"].cpu)
	ensureEqualResourceMetric(t, cm.nodeMetrics["example-node-1"].cpu, cpuExpected)
	assert.NotNil(t, cm.nodeMetrics["example-node-1"].memory)
	ensureEqualResourceMetric(t, cm.nodeMetrics["example-node-1"].memory, memoryExpected)

	assert.Len(t, cm.nodeMetrics["example-node-1"].podMetrics, 1)

	pm := cm.nodeMetrics["example-node-1"].podMetrics
	// Change to pod specific util numbers
	cpuExpected.utilization = resource.MustParse("23m")
	memoryExpected.utilization = resource.MustParse("299Mi")

	assert.NotNil(t, pm["default-example-pod"])
	assert.NotNil(t, pm["default-example-pod"].cpu)
	ensureEqualResourceMetric(t, pm["default-example-pod"].cpu, cpuExpected)
	assert.NotNil(t, pm["default-example-pod"].memory)
	ensureEqualResourceMetric(t, pm["default-example-pod"].memory, memoryExpected)

	node1LabelsExpected := map[string]string{
		"example.io/os": "example-os-1",
		"zone":          "example-zone-1",
	}
	assert.Equal(t, cm.nodeMetrics["example-node-1"].labels, node1LabelsExpected)
}

func TestSortByPodCount(t *testing.T) {
	nodeList := &corev1.NodeList{
		Items: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1",
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						"pods": resource.MustParse("110"),
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-2",
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						"pods": resource.MustParse("110"),
					},
				},
			},
		},
	}

	podList := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-1",
				},
				Spec: corev1.PodSpec{
					NodeName: "node-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-2",
				},
				Spec: corev1.PodSpec{
					NodeName: "node-1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-3",
				},
				Spec: corev1.PodSpec{
					NodeName: "node-2",
				},
			},
		},
	}

	cm := buildClusterMetric(podList, nil, nodeList, nil)
	sortedNodes := cm.getSortedNodeMetrics("pod.count")

	// Node 1 should come first as it has 2 pods vs 1 pod on node 2
	assert.Equal(t, "node-1", sortedNodes[0].name)
	assert.Equal(t, "node-2", sortedNodes[1].name)
	assert.Equal(t, int64(2), sortedNodes[0].podCount.current)
	assert.Equal(t, int64(1), sortedNodes[1].podCount.current)
}

func ensureEqualResourceMetric(t *testing.T, actual *resourceMetric, expected *resourceMetric) {
	assert.Equal(t, actual.allocatable.MilliValue(), expected.allocatable.MilliValue())
	assert.Equal(t, actual.utilization.MilliValue(), expected.utilization.MilliValue())
	assert.Equal(t, actual.request.MilliValue(), expected.request.MilliValue())
	assert.Equal(t, actual.limit.MilliValue(), expected.limit.MilliValue())
}

func listNodes(n *corev1.NodeList) []string {
	nodes := []string{}

	for _, node := range n.Items {
		nodes = append(nodes, node.GetName())
	}

	return nodes
}

func listPods(p *corev1.PodList) []string {
	pods := []string{}

	for _, pod := range p.Items {
		pods = append(pods, fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName()))
	}

	return pods
}
