package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/svc/krane/pkg/labels"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sValidatorConfig holds configuration for Kubernetes token validator.
//
// This configuration provides the Kubernetes clientset needed for TokenReview
// API calls and pod retrieval during validation.
type K8sValidatorConfig struct {
	// Clientset is the Kubernetes client for API calls.
	// Must have permissions for TokenReview creation and pod retrieval.
	Clientset kubernetes.Interface
}

type K8sValidator struct {
	clientset kubernetes.Interface
}

// NewK8sValidator creates a Kubernetes-based token validator.
//
// This function initializes a K8sValidator that uses Kubernetes TokenReview API
// to validate service account tokens and pod information to verify deployment
// membership. The validator ensures tokens belong to pods with expected
// deployment annotations.
//
// Returns a configured K8sValidator ready for use.
func NewK8sValidator(cfg K8sValidatorConfig) *K8sValidator {
	return &K8sValidator{clientset: cfg.Clientset}
}

// Validate validates Kubernetes service account token against expected deployment and environment.
//
// This method implements token validation using Kubernetes TokenReview API. It
// verifies that the token is valid, extracts pod information from token review,
// retrieves the pod, and validates that the pod belongs to the expected
// deployment and environment by checking pod labels.
//
// Returns ValidationResult on successful validation or error if:
// - Token is invalid or expired
// - Token doesn't represent a service account
// - Pod information cannot be extracted
// - Pod doesn't have required deployment or environment labels
// - Pod labels don't match expected deploymentID or environmentID
func (v *K8sValidator) Validate(ctx context.Context, token string, deploymentID string, environmentID string) (*ValidationResult, error) {
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

	// Validate deployment ID from pod labels
	podDeploymentID, ok := labels.GetDeploymentID(pod.Labels)
	if !ok || podDeploymentID == "" {
		return nil, fmt.Errorf("pod %s missing %s label", podName, labels.LabelKeyDeploymentID)
	}

	if podDeploymentID != deploymentID {
		return nil, fmt.Errorf("pod %s belongs to deployment %s, not %s", podName, podDeploymentID, deploymentID)
	}

	// Validate environment ID from pod labels
	podEnvironmentID, ok := labels.GetEnvironmentID(pod.Labels)
	if !ok || podEnvironmentID == "" {
		return nil, fmt.Errorf("pod %s missing %s label", podName, labels.LabelKeyEnvironmentID)
	}

	if podEnvironmentID != environmentID {
		return nil, fmt.Errorf("pod %s belongs to environment %s, not %s", podName, podEnvironmentID, environmentID)
	}

	return &ValidationResult{
		DeploymentID:  deploymentID,
		EnvironmentID: environmentID,
	}, nil
}
