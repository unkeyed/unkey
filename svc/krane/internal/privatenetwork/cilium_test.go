package privatenetwork

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestCiliumRevocationBlocksCachedIPsAndExistingFlows(t *testing.T) {
	contextName := os.Getenv("UNKEY_CILIUM_TEST_CONTEXT")
	if contextName == "" {
		t.Skip("requires an explicitly selected disposable Cilium cluster; see Krane failure guarantees")
	}
	image := os.Getenv("UNKEY_CILIUM_TEST_IMAGE")
	require.NotEmpty(t, image, "set a digest-pinned image with python3 for the disposable probe Pods")
	program, err := os.ReadFile("testdata/flow_probe.py")
	require.NoError(t, err)
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{CurrentContext: contextName}).ClientConfig()
	require.NoError(t, err)
	config.Timeout = 10 * time.Second
	client, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)
	policies, err := dynamic.NewForConfig(config)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	t.Cleanup(cancel)
	namespace, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "unkey-network-test-"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cleanupCancel()
		require.NoError(t, client.CoreV1().Namespaces().Delete(cleanupCtx, namespace.Name, metav1.DeleteOptions{}))
	})
	t.Logf("context=%s namespace=%s", contextName, namespace.Name)

	baseline := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "cilium.io/v2", "kind": "CiliumNetworkPolicy",
		"metadata": map[string]interface{}{"name": "default-deny"},
		"spec": map[string]interface{}{
			"endpointSelector":  map[string]interface{}{},
			"enableDefaultDeny": map[string]interface{}{"ingress": true, "egress": true},
			"ingress":           []interface{}{map[string]interface{}{}},
			"egress":            []interface{}{map[string]interface{}{}},
		},
	}}
	_, err = policies.Resource(policyResource).Namespace(namespace.Name).Create(ctx, baseline, metav1.CreateOptions{})
	require.NoError(t, err)
	for _, name := range []string{"caller", "control", "target"} {
		command := []string{"python3", "-c", "import time; time.sleep(600)"}
		if name == "target" {
			command = []string{"python3", "-u", "-c", string(program), "server"}
		}
		_, err := client.CoreV1().Pods(namespace.Name).Create(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels.New().ManagedByKrane().ComponentDeployment().WorkspaceID(namespace.Name).ProjectID(namespace.Name).AppID("target").DeploymentID(name).Namespace(namespace.Name)},
			Spec: corev1.PodSpec{
				AutomountServiceAccountToken: new(false), RestartPolicy: corev1.RestartPolicyNever,
				SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: new(true), RunAsUser: new(int64(65532)), SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}},
				Containers: []corev1.Container{{
					Name: "probe", Image: image, Command: command,
					SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: new(false), ReadOnlyRootFilesystem: new(true), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
					Resources:       corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("25m"), corev1.ResourceMemory: resource.MustParse("32Mi")}, Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Mi")}},
				}},
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
	}
	var targetIP string
	require.Eventually(t, func() bool {
		pods, err := client.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil || len(pods.Items) != 3 {
			return false
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodRunning || pod.Status.PodIP == "" || len(pod.Status.ContainerStatuses) != 1 || !pod.Status.ContainerStatuses[0].Ready {
				return false
			}
			if pod.Name == "target" {
				targetIP = pod.Status.PodIP
			}
		}
		return true
	}, 2*time.Minute, time.Second)

	caller := startFlowProbe(t, ctx, contextName, namespace.Name, "caller", targetIP, string(program))
	control := startFlowProbe(t, ctx, contextName, namespace.Name, "control", targetIP, string(program))
	allowed := flowResult{TCP: true, UDP: true}
	denied := flowResult{}
	caller.await(t, "fresh", denied)

	r := &Reconciler{dynamic: policies}
	connection := testConnection("target")
	connection.K8SNamespace, connection.WorkspaceId, connection.ProjectId = namespace.Name, namespace.Name, namespace.Name
	connection.TargetAppId, connection.CallerDeploymentId = "target", "caller"
	controlConnection := testConnection("target")
	controlConnection.K8SNamespace, controlConnection.WorkspaceId, controlConnection.ProjectId = namespace.Name, namespace.Name, namespace.Name
	controlConnection.TargetAppId, controlConnection.CallerDeploymentId, controlConnection.ConnectionId = "target", "control", "control"
	for _, spec := range []*ctrlv1.PrivateNetworkConnection{connection, controlConnection} {
		require.NoError(t, r.ensurePolicy(ctx, spec, policyName(spec), nil))
	}
	caller.await(t, "open", allowed)
	control.await(t, "open", allowed)

	connection.TargetDeploymentId = ""
	require.NoError(t, r.ensurePolicy(ctx, connection, policyName(connection), nil))
	started := time.Now()
	for {
		require.Equal(t, allowed, control.exchange(t, "held"), "the server and unrelated grant must remain usable")
		if caller.exchange(t, "fresh") == denied && caller.exchange(t, "held") == denied {
			break
		}
		require.Less(t, time.Since(started), 30*time.Second, "revoked flow still succeeds")
		time.Sleep(100 * time.Millisecond)
	}
	t.Logf("cached-IP TCP/UDP and pre-existing flows denied after %s", time.Since(started))
	for range 3 {
		require.Equal(t, allowed, control.exchange(t, "held"))
		require.Equal(t, denied, caller.exchange(t, "fresh"))
		require.Equal(t, denied, caller.exchange(t, "held"))
	}
	connection.TargetDeploymentId = "target"
	require.NoError(t, r.ensurePolicy(ctx, connection, policyName(connection), nil))
	caller.await(t, "fresh", allowed)
}

type flowResult struct {
	TCP bool `json:"tcp"`
	UDP bool `json:"udp"`
}

type flowProbe struct {
	stdin   io.WriteCloser
	results <-chan flowResult
}

func startFlowProbe(t *testing.T, ctx context.Context, contextName, namespace, pod, targetIP, program string) *flowProbe {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, "kubectl", "--context", contextName, "--namespace", namespace, "exec", "-i", pod, "--", "python3", "-u", "-c", program, "client", targetIP)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	results := make(chan flowResult)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(results)
		decoder := json.NewDecoder(stdout)
		for {
			var result flowResult
			if err := decoder.Decode(&result); err != nil {
				if err != io.EOF && ctx.Err() == nil {
					t.Errorf("decode flow result: %v", err)
				}
				break
			}
			select {
			case results <- result:
			case <-ctx.Done():
			}
		}
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			t.Errorf("flow probe: %v", err)
		}
	}()
	t.Cleanup(func() {
		cancel()
		if err := stdin.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Errorf("close probe input: %v", err)
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("flow probe did not stop")
		}
	})
	return &flowProbe{stdin: stdin, results: results}
}

func (p *flowProbe) exchange(t *testing.T, command string) flowResult {
	t.Helper()
	_, err := fmt.Fprintln(p.stdin, command)
	require.NoError(t, err)
	select {
	case result, ok := <-p.results:
		require.True(t, ok, "flow probe exited without a result")
		return result
	case <-time.After(5 * time.Second):
		t.Fatal("flow probe did not answer")
		return flowResult{}
	}
}

func (p *flowProbe) await(t *testing.T, command string, want flowResult) {
	t.Helper()
	started := time.Now()
	for {
		got := p.exchange(t, command)
		if got == want {
			return
		}
		require.Less(t, time.Since(started), 30*time.Second, "%s: got %+v, want %+v", command, got, want)
		time.Sleep(100 * time.Millisecond)
	}
}
