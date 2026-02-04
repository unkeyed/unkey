package mutator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (m *Mutator) buildInitContainer() corev1.Container {
	return corev1.Container{
		Name:            "copy-inject",
		Image:           m.injectImage,
		ImagePullPolicy: corev1.PullPolicy(m.injectImagePullPolicy),
		Command:         []string{"cp", "/ko-app/inject", injectBinary},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      injectVolumeName,
				MountPath: injectMountPath,
			},
		},
	}
}

func (m *Mutator) buildVolume() corev1.Volume {
	return corev1.Volume{
		Name: injectVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
}

func (m *Mutator) buildContainerPatches(
	ctx context.Context,
	containerIndex int,
	container *corev1.Container,
	podSpec *corev1.PodSpec,
	namespace string,
	podCfg *podConfig,
	buildID string,
) ([]map[string]interface{}, error) {
	var patches []map[string]interface{}
	basePath := fmt.Sprintf("/spec/containers/%d", containerIndex)

	volumeMount := corev1.VolumeMount{
		Name:      injectVolumeName,
		MountPath: injectMountPath,
		ReadOnly:  true,
	}

	if len(container.VolumeMounts) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/volumeMounts", basePath),
			"value": []corev1.VolumeMount{volumeMount},
		})
	} else {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/volumeMounts/-", basePath),
			"value": volumeMount,
		})
	}

	envVars := m.buildEnvVars(podCfg)
	if len(container.Env) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/env", basePath),
			"value": envVars,
		})
	} else {
		for _, env := range envVars {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  fmt.Sprintf("%s/env/-", basePath),
				"value": env,
			})
		}
	}

	// We replace the container's command with inject, which decrypts secrets and then
	// exec's the original entrypoint. If the pod spec doesn't define a command, we need to
	// fetch the image's ENTRYPOINT/CMD from the registry so we know what to exec into.
	var args []string
	if len(container.Command) == 0 {
		m.logger.Info("container has no command, fetching from registry",
			"container", container.Name,
			"image", container.Image,
		)

		imageConfig, err := m.registry.GetImageConfig(ctx, namespace, container, podSpec, buildID)
		if err != nil {
			return nil, fmt.Errorf("failed to get image config for %s: %w", container.Image, err)
		}

		args = append(args, imageConfig.Entrypoint...)
		if len(container.Args) == 0 {
			args = append(args, imageConfig.Cmd...)
		}
	} else {
		args = append(args, container.Command...)
	}

	args = append(args, container.Args...)
	patches = append(patches, map[string]interface{}{
		"op":    "add",
		"path":  fmt.Sprintf("%s/command", basePath),
		"value": []string{injectBinary},
	})

	// Always prepend "--" to prevent user-controlled args from being parsed
	// as inject flags. This ensures malicious commands like ["--token", "evil", "/bin/sh"]
	// cannot override inject's environment-based configuration.
	if len(args) > 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/args", basePath),
			"value": append([]string{"--"}, args...),
		})
	}

	return patches, nil
}

func (m *Mutator) buildEnvVars(podCfg *podConfig) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "UNKEY_PROVIDER_ENDPOINT", Value: podCfg.ProviderEndpoint},
		{Name: "UNKEY_DEPLOYMENT_ID", Value: podCfg.DeploymentID},
		{Name: "UNKEY_TOKEN_PATH", Value: ServiceAccountTokenPath},
	}
}
