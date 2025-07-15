package deploy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/git"
)

func generateImageTag(opts *DeployOptions, gitInfo git.Info) string {
	if gitInfo.ShortSHA != "" {
		return fmt.Sprintf("%s-%s", opts.Branch, gitInfo.ShortSHA)
	}
	return fmt.Sprintf("%s-%d", opts.Branch, time.Now().Unix())
}

func buildImage(ctx context.Context, opts *DeployOptions, dockerImage string) error {
	buildArgs := []string{"build"}
	if opts.Dockerfile != "Dockerfile" {
		buildArgs = append(buildArgs, "-f", opts.Dockerfile)
	}
	buildArgs = append(buildArgs,
		"-t", dockerImage,
		"--build-arg", fmt.Sprintf("VERSION=%s", opts.Commit),
		opts.Context,
	)

	cmd := exec.CommandContext(ctx, "docker", buildArgs...)

	// Stream output directly instead of complex pipe handling
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Docker build failed:\n%s\n", string(output))
		return ErrDockerBuildFailed
	}

	return nil
}

func pushImage(ctx context.Context, dockerImage, registry string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", dockerImage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		detailedMsg := classifyPushError(string(output), registry)
		fmt.Printf("Docker push failed: %s\n", detailedMsg)
		return fmt.Errorf("%s", detailedMsg)
	}
	fmt.Printf("%s\n", string(output))
	return nil
}

func classifyPushError(output, registry string) string {
	output = strings.TrimSpace(output)
	registryHost := getRegistryHost(registry)

	switch {
	case strings.Contains(output, "denied"):
		return fmt.Sprintf("registry access denied. Try: docker login %s", registryHost)

	case strings.Contains(output, "not found") || strings.Contains(output, "404"):
		return "registry not found. Create repository or use --registry=your-registry/your-app"

	case strings.Contains(output, "unauthorized"):
		return fmt.Sprintf("authentication required. Run: docker login %s", registryHost)

	default:
		return output
	}
}

// ## HELPERS
func getRegistryHost(registry string) string {
	parts := strings.Split(registry, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return "docker.io"
}

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "--version")
	return cmd.Run() == nil
}
