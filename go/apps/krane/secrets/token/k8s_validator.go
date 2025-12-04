package token

import (
	"context"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sValidatorConfig struct {
	Clientset kubernetes.Interface
}

type K8sValidator struct {
	clientset kubernetes.Interface
}

func NewK8sValidator(cfg K8sValidatorConfig) *K8sValidator {
	return &K8sValidator{clientset: cfg.Clientset}
}

func (v *K8sValidator) Validate(ctx context.Context, token string, deploymentID string) (*ValidationResult, error) {
	//nolint:exhaustruct // k8s API types have many optional fields
	tokenReview := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{Token: token},
	}

	result, err := v.clientset.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to validate token via TokenReview: %w", err)
	}

	if !result.Status.Authenticated {
		return nil, fmt.Errorf("token not authenticated: %s", result.Status.Error)
	}

	username := result.Status.User.Username
	podName := ""

	if podNames, ok := result.Status.User.Extra["authentication.kubernetes.io/pod-name"]; ok && len(podNames) > 0 {
		podName = podNames[0]
	}

	if podName == "" {
		return nil, fmt.Errorf("could not determine pod name from token (username: %s)", username)
	}

	parts := strings.Split(username, ":")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid service account username format (expected 4 parts): %s", username)
	}
	if parts[0] != "system" || parts[1] != "serviceaccount" {
		return nil, fmt.Errorf("username is not a service account (expected system:serviceaccount:...): %s", username)
	}
	tokenNamespace := parts[2]

	pod, err := v.clientset.CoreV1().Pods(tokenNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	podDeploymentID := pod.Annotations["unkey.com/deployment-id"]
	if podDeploymentID == "" {
		return nil, fmt.Errorf("pod %s missing unkey.com/deployment-id annotation", podName)
	}

	if podDeploymentID != deploymentID {
		return nil, fmt.Errorf("pod %s belongs to deployment %s, not %s", podName, podDeploymentID, deploymentID)
	}

	return &ValidationResult{DeploymentID: deploymentID}, nil
}
