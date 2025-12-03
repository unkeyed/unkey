package mutator

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (m *Mutator) buildInitContainer() corev1.Container {
	return corev1.Container{
		Name:            "copy-unkey-env",
		Image:           m.cfg.UnkeyEnvImage,
		ImagePullPolicy: corev1.PullNever,
		Command:         []string{"cp", "/unkey-env", unkeyEnvBinary},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      unkeyEnvVolumeName,
				MountPath: unkeyEnvMountPath,
			},
		},
	}
}

func (m *Mutator) buildVolume() corev1.Volume {
	return corev1.Volume{
		Name: unkeyEnvVolumeName,
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
) ([]map[string]interface{}, error) {
	var patches []map[string]interface{}
	basePath := fmt.Sprintf("/spec/containers/%d", containerIndex)

	volumeMount := corev1.VolumeMount{
		Name:      unkeyEnvVolumeName,
		MountPath: unkeyEnvMountPath,
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
	for _, env := range envVars {
		if len(container.Env) == 0 {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  fmt.Sprintf("%s/env", basePath),
				"value": []corev1.EnvVar{env},
			})
		} else {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  fmt.Sprintf("%s/env/-", basePath),
				"value": env,
			})
		}
	}

	var args []string
	if len(container.Command) == 0 {
		m.logger.Info("container has no command, fetching from registry",
			"container", container.Name,
			"image", container.Image,
		)

		imageConfig, err := m.registry.GetImageConfig(ctx, namespace, container, podSpec)
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
		"value": []string{unkeyEnvBinary},
	})

	if len(args) > 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/args", basePath),
			"value": args,
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
