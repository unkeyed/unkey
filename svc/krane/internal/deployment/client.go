package deployment

import (
	"errors"

	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	autoscalingv2client "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1client "k8s.io/client-go/kubernetes/typed/policy/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

// ClientSet provides the typed Kubernetes API clients used by Controller.
type ClientSet interface {
	CoreV1() corev1client.CoreV1Interface
	AppsV1() appsv1client.AppsV1Interface
	AutoscalingV2() autoscalingv2client.AutoscalingV2Interface
	PolicyV1() policyv1client.PolicyV1Interface
}

type clientSet struct {
	coreV1        corev1client.CoreV1Interface
	appsV1        appsv1client.AppsV1Interface
	autoscalingV2 autoscalingv2client.AutoscalingV2Interface
	policyV1      policyv1client.PolicyV1Interface
}

// NewClientSet constructs the typed Kubernetes API clients used by Controller.
func NewClientSet(config *rest.Config) (ClientSet, error) {
	configCopy := *config
	if configCopy.UserAgent == "" {
		configCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	if configCopy.RateLimiter == nil && configCopy.QPS > 0 {
		if configCopy.Burst <= 0 {
			return nil, errors.New("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configCopy.QPS, configCopy.Burst)
	}

	httpClient, err := rest.HTTPClientFor(&configCopy)
	if err != nil {
		return nil, err
	}

	coreV1, err := corev1client.NewForConfigAndClient(&configCopy, httpClient)
	if err != nil {
		return nil, err
	}

	appsV1, err := appsv1client.NewForConfigAndClient(&configCopy, httpClient)
	if err != nil {
		return nil, err
	}

	autoscalingV2, err := autoscalingv2client.NewForConfigAndClient(&configCopy, httpClient)
	if err != nil {
		return nil, err
	}

	policyV1, err := policyv1client.NewForConfigAndClient(&configCopy, httpClient)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		coreV1:        coreV1,
		appsV1:        appsV1,
		autoscalingV2: autoscalingV2,
		policyV1:      policyV1,
	}, nil
}

func (c *clientSet) CoreV1() corev1client.CoreV1Interface {
	return c.coreV1
}

func (c *clientSet) AppsV1() appsv1client.AppsV1Interface {
	return c.appsV1
}

func (c *clientSet) AutoscalingV2() autoscalingv2client.AutoscalingV2Interface {
	return c.autoscalingV2
}

func (c *clientSet) PolicyV1() policyv1client.PolicyV1Interface {
	return c.policyV1
}
