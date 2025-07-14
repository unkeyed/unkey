package deploy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/git"
)

func buildDockerImage(ctx context.Context, opts *DeployOptions, gitInfo git.Info) (string, error) {
	if !isDockerAvailable() {
		return "", ErrDockerNotFound
	}

	imageTag := generateImageTag(opts, gitInfo)
	dockerImage := fmt.Sprintf("%s:%s", opts.Registry, imageTag)

	if err := buildImage(ctx, opts, dockerImage); err != nil {
		return "", err
	}

	if opts.SkipPush {
		fmt.Printf("Skipping Docker push (--skip-push enabled)\n")
		return dockerImage, nil
	}

	// Push failure shouldn't be fatal in development
	if err := pushImage(ctx, dockerImage, opts.Registry); err != nil {
		fmt.Printf("Push failed but continuing: %v\n", err)
	}

	return dockerImage, nil
}

func generateImageTag(opts *DeployOptions, gitInfo git.Info) string {
	if gitInfo.ShortSHA != "" {
		return fmt.Sprintf("%s-%s", opts.Branch, gitInfo.ShortSHA)
	}
	return fmt.Sprintf("%s-%d", opts.Branch, time.Now().Unix())
}

func buildImage(ctx context.Context, opts *DeployOptions, dockerImage string) error {
	fmt.Printf("Building Docker image %s...\n", dockerImage)

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

	fmt.Printf("%s\n", string(output))
	return nil
}

func pushImage(ctx context.Context, dockerImage, registry string) error {
	fmt.Printf("\nPublishing Docker image...\n")

	cmd := exec.CommandContext(ctx, "docker", "push", dockerImage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return classifyPushError(string(output), registry)
	}

	fmt.Printf("%s\n", string(output))
	return nil
}

func classifyPushError(output, registry string) error {
	output = strings.TrimSpace(output)
	registryHost := getRegistryHost(registry)

	switch {
	case strings.Contains(output, "denied"):
		fmt.Printf("Docker push failed: Registry access denied\n")
		fmt.Printf("  Registry: %s\n", registry)
		fmt.Printf("  Solutions:\n")
		fmt.Printf("  • Login: docker login %s\n", registryHost)
		fmt.Printf("  • Use your own registry: --registry=your-registry/your-app\n")
		fmt.Printf("  • Skip push: --skip-push\n")
		return ErrDockerPushFailed

	case strings.Contains(output, "not found") || strings.Contains(output, "404"):
		fmt.Printf("Docker push failed: Registry not found\n")
		fmt.Printf("  Registry: %s\n", registry)
		fmt.Printf("  Solutions:\n")
		fmt.Printf("  • Create repository first\n")
		fmt.Printf("  • Use different registry: --registry=your-registry/your-app\n")
		fmt.Printf("  • Skip push: --skip-push\n")
		return ErrDockerPushFailed

	case strings.Contains(output, "unauthorized"):
		fmt.Printf("Docker push failed: Authentication required\n")
		fmt.Printf("  Run: docker login %s\n", registryHost)
		fmt.Printf("  Or skip: --skip-push\n")
		return ErrDockerPushFailed

	default:
		fmt.Printf("Docker push failed: %s\n", output)
		return ErrDockerPushFailed
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
