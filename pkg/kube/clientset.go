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

package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	// Required for GKE, OIDC, and more
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// NewClientSet returns a new Kubernetes clientset
func NewClientSet(kubeContext, kubeConfig string) (*kubernetes.Clientset, error) {
	config, err := getKubeConfig(kubeContext, kubeConfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// NewMetricsClientSet returns a new clientset for Kubernetes metrics
func NewMetricsClientSet(kubeContext, kubeConfig string) (*metrics.Clientset, error) {
	config, err := getKubeConfig(kubeContext, kubeConfig)
	if err != nil {
		return nil, err
	}

	return metrics.NewForConfig(config)
}

func getKubeConfig(kubeContext, kubeConfig string) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeConfig != "" {
		loadingRules.ExplicitPath = kubeConfig
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext},
	).ClientConfig()
}
