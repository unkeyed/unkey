// Package repofetch provides a Kubernetes Job client for spawning and monitoring
// repository fetch jobs that download tarballs from GitHub and upload to S3.
//
// These jobs run with gVisor isolation in the "builds" namespace to securely
// handle untrusted repository content without exposing the control plane.
package repofetch

import (
	"context"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// Namespace where fetch jobs run.
	namespace = "builds"

	// fieldManager is the SSA field manager name.
	fieldManager = "ctrl-worker"

	// runtimeClassGvisor is the gVisor runtime class for sandboxing.
	runtimeClassGvisor = "gvisor"
)

// Client manages Kubernetes Jobs for repository fetch operations.
type Client struct {
	clientSet kubernetes.Interface
	logger    logging.Logger
	image     string
}

// Config holds configuration for creating a fetch client.
type Config struct {
	// ClientSet is the Kubernetes client.
	ClientSet kubernetes.Interface

	// Logger for structured logging.
	Logger logging.Logger

	// Image is the container image for fetch jobs (e.g., "repofetch:latest").
	Image string
}

// NewClient creates a new fetch job client.
func NewClient(cfg Config) *Client {
	return &Client{
		clientSet: cfg.ClientSet,
		logger:    cfg.Logger.With("component", "repofetch"),
		image:     cfg.Image,
	}
}

// FetchParams contains parameters for spawning a fetch job.
type FetchParams struct {
	// DeploymentID is used as the idempotency key for the job.
	DeploymentID string

	// ProjectID for logging and labeling.
	ProjectID string

	// Repo is the GitHub repository full name (e.g., "owner/repo").
	Repo string

	// SHA is the commit SHA to fetch.
	SHA string

	// GitHubToken is the short-lived installation access token.
	GitHubToken string

	// UploadURL is the presigned S3 PUT URL.
	UploadURL string
}

// SpawnFetchJob creates a fetch job and returns its name.
func (c *Client) SpawnFetchJob(ctx context.Context, params FetchParams) (string, error) {
	jobName := uid.DNS1035(16)

	c.logger.Info("spawning fetch job",
		"job_name", jobName,
		"deployment_id", params.DeploymentID,
		"repo", params.Repo,
		"sha", params.SHA,
	)

	job := c.buildJobSpec(jobName, params)

	_, err := c.clientSet.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to create job"))
	}

	return jobName, nil
}

// GetJobStatus returns the current status of a fetch job.
func (c *Client) GetJobStatus(ctx context.Context, jobName string) (JobStatus, error) {
	job, err := c.clientSet.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return JobStatusUnknown, fault.Wrap(err, fault.Internal("failed to get job"))
	}

	return statusFromJob(job), nil
}

func (c *Client) buildJobSpec(jobName string, params FetchParams) *batchv1.Job {
	labels := map[string]string{
		"app.kubernetes.io/component":  "repo-fetch",
		"app.kubernetes.io/managed-by": "ctrl-worker",
	}

	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            ptr.P(int32(0)),
			ActiveDeadlineSeconds:   ptr.P(int64(600)), // 10 minutes
			TTLSecondsAfterFinished: ptr.P(int32(300)), // 5 minutes
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RuntimeClassName:             ptr.P(runtimeClassGvisor),
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: ptr.P(false),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.P(true),
						RunAsUser:    ptr.P(int64(65534)), // nobody
						RunAsGroup:   ptr.P(int64(65534)),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Volumes: []corev1.Volume{{
						Name: "tmp",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "fetch",
						Image:           c.image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{Name: "UNKEY_GITHUB_TOKEN", Value: params.GitHubToken},
							{Name: "UNKEY_REPO", Value: params.Repo},
							{Name: "UNKEY_SHA", Value: params.SHA},
							{Name: "UNKEY_UPLOAD_URL", Value: params.UploadURL},
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "tmp",
							MountPath: "/tmp",
						}},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.P(false),
							ReadOnlyRootFilesystem:   ptr.P(true),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
						},
						Resources: corev1.ResourceRequirements{},
					}},
				},
			},
		},
	}
}
