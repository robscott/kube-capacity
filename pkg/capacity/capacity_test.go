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
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/fake"
)

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
		"another/mypod5",
		"default/mypod",
		"default/mypod4",
		"default/mypod6",
		"kube-system/mypod1",
		"other/mypod2",
		"other/mypod3",
	}, listPods(podList))

	podList, nodeList = getPodsAndNodes(clientset, "", "hello=world", "")
	assert.Equal(t, []string{"mynode", "mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"another/mypod5",
		"default/mypod",
		"default/mypod4",
		"default/mypod6",
		"kube-system/mypod1",
		"other/mypod2",
		"other/mypod3",
	}, listPods(podList))

	podList, nodeList = getPodsAndNodes(clientset, "", "moon=lol", "")
	assert.Equal(t, []string{"mynode2"}, listNodes(nodeList))
	assert.Equal(t, []string{
		"default/mypod4",
		"kube-system/mypod1",
		"other/mypod3",
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

func node(name string, labels map[string]string) *corev1.Node {
	return &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func namespace(name string, labels map[string]string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func pod(node, namespace, name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			NodeName: node,
		},
	}
}
