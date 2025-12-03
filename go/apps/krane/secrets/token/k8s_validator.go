package token

import (
	"context"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sValidatorConfig configures the Kubernetes-based token validator
type K8sValidatorConfig struct {
	// Clientset is the Kubernetes client for TokenReview API calls
	Clientset kubernetes.Interface
	// Namespace is the namespace where customer workloads run
	Namespace string
}

// K8sValidator validates Kubernetes service account tokens.
// It uses the TokenReview API to validate tokens and maps pod identity to deployments.
type K8sValidator struct {
	clientset kubernetes.Interface
	namespace string
}

// NewK8sValidator creates a new Kubernetes-based token validator
func NewK8sValidator(cfg K8sValidatorConfig) *K8sValidator {
	return &K8sValidator{
		clientset: cfg.Clientset,
		namespace: cfg.Namespace,
	}
}

// Validate checks if a Kubernetes service account token is valid.
// The token is validated via the TokenReview API, and the pod's deployment
// is determined from the pod's annotation (unkey.com/deployment-id).
func (v *K8sValidator) Validate(ctx context.Context, token string, deploymentID string) (*ValidationResult, error) {
	// Create TokenReview request
	tokenReview := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	// Call TokenReview API
	result, err := v.clientset.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to validate token via TokenReview: %w", err)
	}

	if !result.Status.Authenticated {
		return nil, fmt.Errorf("token not authenticated: %s", result.Status.Error)
	}

	// Extract pod info from the token
	// Username format: system:serviceaccount:{namespace}:{serviceaccount}
	username := result.Status.User.Username
	podName := ""

	// Get pod name from Extra fields
	if podNames, ok := result.Status.User.Extra["authentication.kubernetes.io/pod-name"]; ok && len(podNames) > 0 {
		podName = podNames[0]
	}

	if podName == "" {
		return nil, fmt.Errorf("could not determine pod name from token (username: %s)", username)
	}

	// Verify the pod is in the expected namespace
	parts := strings.Split(username, ":")
	if len(parts) >= 3 {
		tokenNamespace := parts[2]
		if tokenNamespace != v.namespace {
			return nil, fmt.Errorf("token namespace %s does not match expected namespace %s", tokenNamespace, v.namespace)
		}
	}

	// Get the pod to check its deployment-id annotation
	pod, err := v.clientset.CoreV1().Pods(v.namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	// Check the deployment ID from the pod annotation (set by krane)
	podDeploymentID := pod.Annotations["unkey.com/deployment-id"]
	if podDeploymentID == "" {
		return nil, fmt.Errorf("pod %s missing unkey.com/deployment-id annotation", podName)
	}

	// Verify it matches the requested deployment
	if podDeploymentID != deploymentID {
		return nil, fmt.Errorf("pod %s belongs to deployment %s, not %s", podName, podDeploymentID, deploymentID)
	}

	// Token is valid and belongs to the expected deployment
	return &ValidationResult{
		DeploymentID: deploymentID,
	}, nil
}
