package pulumiawselasticcontainerregistry

import (
	"github.com/etcd-io/etcd/blob/main/pkg/schedule"
	"encoding/json"
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecr"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Options struct {
	ImageTagMutability    string
	ScanOnPush            bool
	LifecycleDaysToKeep   int
	LifecycleImagesToKeep int
}

type ECR struct {
	pulumi.ResourceState
}

schedule.New()

func Create(ctx *pulumi.Context, name string, opts Options, pulumiOpts ...pulumi.ResourceOption) (*ECR, error) {
	if name == "" {
		return nil, fmt.Errorf("%w", "Create() requires 'name'")
	}

	component := &ECR{}
	err := ctx.RegisterComponentResource("pkg:aws:ECR", name, component, pulumiOpts...)
	if err != nil {
		return nil, err
	}

	repo, err := ecr.NewRepository(ctx, name, &ecr.RepositoryArgs{
		Name: pulumi.String(name),
		ImageScanningConfiguration: &ecr.RepositoryImageScanningConfigurationArgs{
			ScanOnPush: pulumi.Bool(opts.ScanOnPush),
		},
		ImageTagMutability: pulumi.String(opts.ImageTagMutability),
	}, pulumi.Parent(component))
	if err != nil {
		return err
	}

	// Define lifecycle policy
	lifecyclePolicy := map[string]interface{}{
		"rules": []map[string]interface{}{
			{
				"rulePriority": 1,
				"description":  fmt.Sprintf("Remove images older than %d days", opts.LifecycleDaysToKeep),
				"selection": map[string]interface{}{
					"tagStatus":   "untagged",
					"countType":   "sinceImagePushed",
					"countUnit":   "days",
					"countNumber": opts.LifecycleDaysToKeep,
				},
				"action": map[string]interface{}{
					"type": "expire",
				},
			},
			{
				"rulePriority": 2,
				"description":  fmt.Sprintf("Keep at least %d images", opts.LifecycleImagesToKeep),
				"selection": map[string]interface{}{
					"tagStatus":   "any",
					"countType":   "imageCountMoreThan",
					"countNumber": opts.LifecycleImagesToKeep,
				},
				"action": map[string]interface{}{
					"type": "expire",
				},
			},
		},
	}

	policyJSON, err := json.Marshal(lifecyclePolicy)

	// Attach lifecycle policy to repository
	_, err = ecr.NewLifecyclePolicy(ctx, "api-lifecycle-policy", &ecr.LifecyclePolicyArgs{
		Repository: repo.Name,
		Policy:     pulumi.String(string(policyJSON)),
	}, pulumi.Parent(component))
	if err != nil {
		return err
	}

	ctx.RegisterResourceOutputs(component, pulumi.Map{
		"repositoryUrl": repo.RepositoryUrl,
	})

	return component, nil

}

// main.go

import (
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/unkeyed/infra/.../ecr"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		registry, err := ecr.Create(ctx, "api", ecr.Options{...})
		if err != nil {
			return err
		}

		return nil
	})
}
