// Package k8sdiag is the shared Kubernetes diagnostics layer for
// preflight probe Diagnose() implementations. Given a deployment ID,
// it captures every pod's manifest, the pod's recent events, and the
// last few minutes of container logs as failure artifacts.
//
// It loads cluster credentials in this order:
//
//  1. In-cluster config (the production case: a probe pod with a
//     ServiceAccount mounted at /var/run/secrets/kubernetes.io).
//  2. The default kubeconfig loading rules ($KUBECONFIG, then
//     ~/.kube/config), so dev runs against minikube work without
//     extra wiring.
//
// All public methods are best-effort: any error is captured as a
// note artifact in the bundle rather than returned to the caller,
// because Diagnose() is itself best-effort and a kube outage should
// not mask the probe failure that triggered diagnosis.
package k8sdiag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// logTail caps the wall-clock window we pull from each container.
// Five minutes is enough to cover a full deploy + crash window
// without ballooning the artifact bundle when an app is chatty.
const logTail = 5 * time.Minute

// logBytesPerContainer caps the per-container log size. Without this
// a runaway log emitter could pin the runner's heap. 256 KiB is
// generous: an app emitting 1KB/s for the full logTail window only
// produces ~300 KB.
const logBytesPerContainer = 256 << 10

// Diagnoser captures kubelet- and kube-API-side artifacts for the
// pods backing a deployment. Construct once and share, or once per
// Diagnose() call: there is no per-call state.
type Diagnoser struct {
	kube kubernetes.Interface
}

// New connects to the cluster using in-cluster config first, falling
// back to the standard kubeconfig loading rules. Returns nil + an
// error when neither path produces a usable client; callers usually
// wrap that error into a note artifact rather than propagating.
func New() (*Diagnoser, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("k8sdiag: load kube config: %w", err)
	}
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8sdiag: build clientset: %w", err)
	}

	return &Diagnoser{kube: kc}, nil
}

func loadConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	} else if !errors.Is(err, rest.ErrNotInCluster) {
		return nil, err
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	//nolint:exhaustruct // overrides intentionally empty; we want the user's defaults
	overrides := &clientcmd.ConfigOverrides{}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
}

// CaptureDeployment captures every pod with label
// unkey.com/deployment.id=<deploymentID> across all namespaces, plus
// recent events and container logs for each. Empty deploymentID
// returns nil.
func (d *Diagnoser) CaptureDeployment(ctx context.Context, deploymentID string) []core.Artifact {
	if d == nil || deploymentID == "" {
		return nil
	}

	selector := labels.LabelKeyDeploymentID + "=" + deploymentID
	//nolint:exhaustruct // ListOptions defaults are correct for a one-shot list
	pods, err := d.kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return []core.Artifact{noteArtifact("pods_list.txt",
			fmt.Sprintf("List pods (selector=%s): %v\n", selector, err))}
	}
	if len(pods.Items) == 0 {
		return []core.Artifact{noteArtifact("pods_list.txt",
			fmt.Sprintf("no pods match %s\n", selector))}
	}

	artifacts := make([]core.Artifact, 0, 4*len(pods.Items))
	for i := range pods.Items {
		pod := &pods.Items[i]
		artifacts = append(artifacts, capturePod(pod)...)
		artifacts = append(artifacts, d.captureEvents(ctx, pod)...)
		artifacts = append(artifacts, d.captureLogs(ctx, pod)...)
	}

	return artifacts
}

func capturePod(pod *corev1.Pod) []core.Artifact {
	body, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return []core.Artifact{noteArtifact(pod.Name+".pod.txt",
			fmt.Sprintf("marshal pod: %v\n", err))}
	}

	return []core.Artifact{{
		Name:        pod.Name + ".pod.json",
		ContentType: "application/json",
		Body:        body,
	}}
}

func (d *Diagnoser) captureEvents(ctx context.Context, pod *corev1.Pod) []core.Artifact {
	fieldSel := fmt.Sprintf("involvedObject.name=%s,involvedObject.namespace=%s", pod.Name, pod.Namespace)
	//nolint:exhaustruct // ListOptions defaults are correct for a field-selected list
	events, err := d.kube.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSel})
	if err != nil {
		return []core.Artifact{noteArtifact(pod.Name+".events.txt",
			fmt.Sprintf("List events (%s): %v\n", fieldSel, err))}
	}

	if len(events.Items) == 0 {
		return nil
	}

	// Newest first so the bundle reader sees the most recent failure
	// reason at the top.
	sort.Slice(events.Items, func(i, j int) bool {
		return events.Items[i].LastTimestamp.After(events.Items[j].LastTimestamp.Time)
	})

	var buf bytes.Buffer
	for _, ev := range events.Items {
		fmt.Fprintf(&buf, "%s  %-7s  %-20s  %s\n",
			ev.LastTimestamp.Format(time.RFC3339),
			ev.Type,
			ev.Reason,
			ev.Message,
		)
	}

	return []core.Artifact{{
		Name:        pod.Name + ".events.txt",
		ContentType: "text/plain",
		Body:        buf.Bytes(),
	}}
}

func (d *Diagnoser) captureLogs(ctx context.Context, pod *corev1.Pod) []core.Artifact {
	containers := allContainerNames(pod)
	artifacts := make([]core.Artifact, 0, len(containers))
	since := int64(logTail.Seconds())

	for _, container := range containers {
		opts := &corev1.PodLogOptions{
			Container:    container,
			SinceSeconds: &since,
			Timestamps:   true,
			//nolint:exhaustruct // remaining fields are nil-by-design (no follow, no preview)
		}

		req := d.kube.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, opts)
		stream, err := req.Stream(ctx)
		if err != nil {
			artifacts = append(artifacts, noteArtifact(
				fmt.Sprintf("%s.logs.%s.txt", pod.Name, container),
				fmt.Sprintf("stream logs: %v\n", err)))
			continue
		}

		body, err := io.ReadAll(io.LimitReader(stream, logBytesPerContainer))
		_ = stream.Close()
		if err != nil {
			artifacts = append(artifacts, noteArtifact(
				fmt.Sprintf("%s.logs.%s.txt", pod.Name, container),
				fmt.Sprintf("read logs: %v\n", err)))
			continue
		}

		artifacts = append(artifacts, core.Artifact{
			Name:        fmt.Sprintf("%s.logs.%s.txt", pod.Name, container),
			ContentType: "text/plain",
			Body:        body,
		})
	}
	return artifacts
}

func allContainerNames(pod *corev1.Pod) []string {
	out := make([]string, 0, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))
	for _, c := range pod.Spec.InitContainers {
		out = append(out, c.Name)
	}

	for _, c := range pod.Spec.Containers {
		out = append(out, c.Name)
	}

	return out
}

func noteArtifact(name, body string) core.Artifact {
	return core.Artifact{
		Name:        name,
		ContentType: "text/plain",
		Body:        []byte(body),
	}
}
