package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewClient() (*kubernetes.Clientset, error) {

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	return kubernetes.NewForConfig(inClusterConfig)

}
