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

package kube

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	// Required for GKE, OIDC, and more
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// NewClientSet returns a new Kubernetes clientset
func NewClientSet() (*kubernetes.Clientset, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// NewMetricsClientSet returns a new clientset for Kubernetes metrics
func NewMetricsClientSet() (*metrics.Clientset, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	return metrics.NewForConfig(config)
}

func getKubeConfig() (*rest.Config, error) {
	var kubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		fmt.Println("Parsing kubeconfig failed, please set KUBECONFIG env var")
		os.Exit(1)
	}

	if _, err := os.Stat(kubeconfig); err != nil {
		// kubeconfig doesn't exist
		fmt.Printf("%s does not exist - please make sure you have a kubeconfig configured.\n", kubeconfig)
		panic(err.Error())
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
