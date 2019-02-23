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
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"k8s.io/client-go/kubernetes/fake"
)

func TestBuildClusterMetricEmpty(t *testing.T) {
	cm := buildClusterMetric(
		&corev1.PodList{}, &v1beta1.PodMetricsList{}, &corev1.NodeList{},
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
		podMetrics:  map[string]*podMetric{},
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
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							"cpu":    resource.MustParse("1000m"),
							"memory": resource.MustParse("4000Mi"),
						},
					},
				},
			},
		},
	)

	cpuExpected := &resourceMetric{
		allocatable: resource.MustParse("1000m"),
		request:     resource.MustParse("350m"),
		limit:       resource.MustParse("400m"),
		utilization: resource.MustParse("23m"),
	}

	memoryExpected := &resourceMetric{
		allocatable: resource.MustParse("4000Mi"),
		request:     resource.MustParse("400Mi"),
		limit:       resource.MustParse("700Mi"),
		utilization: resource.MustParse("299Mi"),
	}

	assert.Len(t, cm.podMetrics, 1)

	assert.NotNil(t, cm.cpu)
	ensureEqualResourceMetric(t, cm.cpu, cpuExpected)
	assert.NotNil(t, cm.memory)
	ensureEqualResourceMetric(t, cm.memory, memoryExpected)

	assert.NotNil(t, cm.nodeMetrics["example-node-1"])
	assert.NotNil(t, cm.nodeMetrics["example-node-1"].cpu)
	ensureEqualResourceMetric(t, cm.nodeMetrics["example-node-1"].cpu, cpuExpected)
	assert.NotNil(t, cm.nodeMetrics["example-node-1"].memory)
	ensureEqualResourceMetric(t, cm.nodeMetrics["example-node-1"].memory, memoryExpected)

	// Change to pod specific util numbers
	cpuExpected.utilization = resource.MustParse("23m")
	memoryExpected.utilization = resource.MustParse("299Mi")

	assert.NotNil(t, cm.podMetrics["default-example-pod"])
	assert.NotNil(t, cm.podMetrics["default-example-pod"].cpu)
	ensureEqualResourceMetric(t, cm.podMetrics["default-example-pod"].cpu, cpuExpected)
	assert.NotNil(t, cm.podMetrics["default-example-pod"].memory)
	ensureEqualResourceMetric(t, cm.podMetrics["default-example-pod"].memory, memoryExpected)
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

func node(name string, labels map[string]string) *corev1.Node {
	return &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind: "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: labels,
		},
	}
}

func namespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind: "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: labels,
		},
	}
}

func pod(node, namespace, name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Namespace: namespace,
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			NodeName: node,
		},
	}
}

func TestGetPodsAndNodes(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		node("mynode", map[string]string{"hello": "world"}),
		node("mynode2", map[string]string{"hello": "world", "moon": "lol"}),
		namespace("default", map[string]string{"app": "true"}),
		namespace("kube-system", map[string]string{"system": "true"}),
		namespace("other", map[string]string{"app": "true", "system": "true"}),
		namespace("another", map[string]string{"hello": "world"}),
		pod("mynode", "default", "mypod", map[string]string{"a": "test"}),
		pod("mynode2", "kube-system", "mypod1", map[string]string{"b": "test"}),
		pod("mynode", "other", "mypod2", map[string]string{"c": "test"}),
		pod("mynode2", "other", "mypod3", map[string]string{"d": "test"}),
		pod("mynode2", "default", "mypod4", map[string]string{"e": "test"}),
		pod("mynode", "another", "mypod5", map[string]string{"f": "test"}),
		pod("mynode", "default", "mypod6", map[string]string{"g": "test"}),
	)

	podList, nodeList := getPodsAndNodes(clientset, "", "", "")
	assert.Equal(t, []string{"mynode", "mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"default/mypod", "kube-system/mypod1", "other/mypod2", "other/mypod3", "default/mypod4",
		"another/mypod5", "default/mypod6",
	}, listPods(podList))

	podList, nodeList = getPodsAndNodes(clientset, "", "hello=world", "")
	assert.Equal(t, []string{"mynode", "mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"default/mypod", "kube-system/mypod1", "other/mypod2", "other/mypod3", "default/mypod4",
		"another/mypod5", "default/mypod6",
	}, listPods(podList))

	podList, nodeList = getPodsAndNodes(clientset, "", "moon=lol", "")
	assert.Equal(t, []string{"mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"kube-system/mypod1", "other/mypod3", "default/mypod4",
	}, listPods(podList))

	podList, nodeList = getPodsAndNodes(clientset, "a=test", "", "")
	assert.Equal(t, []string{"mynode", "mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"default/mypod",
	}, listPods(podList))


	podList, nodeList = getPodsAndNodes(clientset, "a=test,b!=test", "", "app=true")
	assert.Equal(t, []string{"mynode", "mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"default/mypod",
	}, listPods(podList))
}
