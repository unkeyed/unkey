package deploy

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/moby/buildkit/client"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
)

const (
	// buildkitPort is the TCP port buildkitd listens on inside the build Job.
	buildkitPort = 1234

	// buildkitPodTimeout bounds the wait for the build pod to pass its
	// readiness probe. Sized for scheduling plus the one-time buildkit image
	// pull on a fresh cluster.
	buildkitPodTimeout = 5 * time.Minute

	// buildJobDeadlineSeconds kills a build Job that outlives any plausible
	// build. buildkitd never exits on its own, so without this a Job leaked
	// by a worker crash would run forever.
	buildJobDeadlineSeconds = 3600

	// buildJobTTLSeconds removes finished Jobs (deadline-killed or failed).
	// The happy path deletes the Job explicitly; this reaps crash leftovers
	// once the deadline has turned them into finished Jobs.
	buildJobTTLSeconds = 300
)

// k8sNameInvalidChars matches everything RFC 1123 forbids in a resource
// name. Deployment IDs contain underscores, which must become hyphens.
var k8sNameInvalidChars = regexp.MustCompile(`[^a-z0-9-]+`)

// sanitizeK8sName converts an arbitrary identifier into a valid RFC 1123
// name fragment.
func sanitizeK8sName(s string) string {
	return strings.Trim(k8sNameInvalidChars.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// withKubernetesBuildkit runs fn against a BuildKit daemon started as a
// one-off Job in the cluster the worker runs in. Local development only:
// the pod runs privileged (buildkitd needs that for its overlayfs
// snapshotter) and provides no isolation beyond the pod boundary, which is
// acceptable only for builds you already trust.
//
// The Job is deleted when fn returns; the deadline and TTL on the Job spec
// reap orphans if the worker dies mid-build. Returns the deployment ID as
// the build ID since there is no external build system to reference.
func (w *Workflow) withKubernetesBuildkit(
	runCtx context.Context,
	params gitBuildParams,
	fn func(buildClient *client.Client) error,
) (string, error) {
	jobs := w.k8s.BatchV1().Jobs(w.buildConfig.Kubernetes.Namespace)

	//nolint: exhaustruct
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			// GenerateName gives every attempt a fresh Job, so a Restate
			// retry never collides with a half-deleted Job from a crashed
			// prior attempt.
			GenerateName: fmt.Sprintf("buildkit-%s-", sanitizeK8sName(params.DeploymentID)),
			Labels: map[string]string{
				"app":                     "buildkit",
				"unkey.com/deployment-id": params.DeploymentID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            ptr.P(int32(0)),
			ActiveDeadlineSeconds:   ptr.P(int64(buildJobDeadlineSeconds)),
			TTLSecondsAfterFinished: ptr.P(int32(buildJobTTLSeconds)),
			Template: corev1.PodTemplateSpec{
				//nolint: exhaustruct
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                     "buildkit",
						"unkey.com/deployment-id": params.DeploymentID,
					},
				},
				//nolint: exhaustruct
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						//nolint: exhaustruct
						{
							Name:  "buildkitd",
							Image: w.buildConfig.Kubernetes.Image,
							Args: []string{
								"--addr", "unix:///run/buildkit/buildkitd.sock",
								"--addr", fmt.Sprintf("tcp://0.0.0.0:%d", buildkitPort),
							},
							Ports: []corev1.ContainerPort{
								//nolint: exhaustruct
								{ContainerPort: buildkitPort},
							},
							//nolint: exhaustruct
							SecurityContext: &corev1.SecurityContext{
								Privileged: ptr.P(true),
							},
							// buildctl talks over the unix socket, so ready
							// means buildkitd accepts RPCs, not merely that
							// the process started.
							//nolint: exhaustruct
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									//nolint: exhaustruct
									Exec: &corev1.ExecAction{
										Command: []string{"buildctl", "debug", "workers"},
									},
								},
								InitialDelaySeconds: 2,
								PeriodSeconds:       2,
							},
						},
					},
				},
			},
		},
	}

	created, err := jobs.Create(runCtx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create build job: %w", err)
	}

	defer func() {
		// WithoutCancel: the deferred delete must run even when runCtx is
		// already canceled — that cancellation is a common reason we're here.
		delCtx, cancel := context.WithTimeout(context.WithoutCancel(runCtx), 30*time.Second)
		defer cancel()
		if delErr := jobs.Delete(delCtx, created.Name, metav1.DeleteOptions{
			PropagationPolicy: ptr.P(metav1.DeletePropagationBackground),
		}); delErr != nil {
			logger.Error("unable to delete build job",
				"job", created.Name,
				"namespace", w.buildConfig.Kubernetes.Namespace,
				"error", delErr)
		}
	}()

	logger.Info("Build job created",
		"job", created.Name,
		"namespace", w.buildConfig.Kubernetes.Namespace,
		"deployment_id", params.DeploymentID)

	podIP, err := w.waitForBuildkitPod(runCtx, created.Name)
	if err != nil {
		return "", err
	}

	buildClient, err := client.New(runCtx, fmt.Sprintf("tcp://%s:%d", podIP, buildkitPort))
	if err != nil {
		return "", fmt.Errorf("unable to create build client: %w", err)
	}
	defer func() {
		if closeErr := buildClient.Close(); closeErr != nil {
			logger.Error("unable to close build client", "error", closeErr)
		}
	}()

	return params.DeploymentID, fn(buildClient)
}

// waitForBuildkitPod polls until the Job's pod is ready and returns its pod
// IP. Terminal pod states (image pull failures, config errors) fail fast
// instead of burning the full timeout.
func (w *Workflow) waitForBuildkitPod(runCtx context.Context, jobName string) (string, error) {
	pods := w.k8s.CoreV1().Pods(w.buildConfig.Kubernetes.Namespace)

	var podIP string
	err := wait.PollUntilContextTimeout(runCtx, 2*time.Second, buildkitPodTimeout, true, func(ctx context.Context) (bool, error) {
		// The job controller stamps job-name on every pod it creates.
		list, listErr := pods.List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
		if listErr != nil {
			logger.Warn("unable to list build pods, retrying", "job", jobName, "error", listErr)
			return false, nil
		}

		for i := range list.Items {
			pod := &list.Items[i]
			if pod.Status.Phase == corev1.PodFailed {
				return false, fmt.Errorf("build pod %s failed: %s %s", pod.Name, pod.Status.Reason, pod.Status.Message)
			}
			if reason := stuckContainerReason(pod); reason != "" {
				return false, fmt.Errorf("build pod %s cannot start: %s", pod.Name, reason)
			}
			if pod.Status.PodIP == "" {
				continue
			}
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					podIP = pod.Status.PodIP
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		// Blame the timeout only when it actually fired; a canceled run
		// context (worker shutdown, workflow cancellation) is not a
		// scheduling problem and must not read like one.
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("build pod for job %s not ready after %s: %w", jobName, buildkitPodTimeout, err)
		}
		return "", fmt.Errorf("build pod for job %s not ready: %w", jobName, err)
	}
	return podIP, nil
}

// stuckContainerReason reports a container waiting reason that will not
// resolve on its own within the pod timeout, or "" if none.
func stuckContainerReason(pod *corev1.Pod) string {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting == nil {
			continue
		}
		switch cs.State.Waiting.Reason {
		// ImagePullBackOff (unlike a single ErrImagePull) means the pull has
		// already failed repeatedly; the other two never self-heal.
		case "ImagePullBackOff", "InvalidImageName", "CreateContainerConfigError":
			return fmt.Sprintf("%s: %s", cs.State.Waiting.Reason, cs.State.Waiting.Message)
		}
	}
	return ""
}
