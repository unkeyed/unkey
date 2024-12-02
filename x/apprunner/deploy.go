package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/apprunner"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func deploy(ctx *pulumi.Context) error {

	repo, err := ecr.NewRepository(ctx, "repo", &ecr.RepositoryArgs{
		Name:        pulumi.String("repo"),
		ForceDelete: pulumi.Bool(true),
		LifecyclePolicy: &ecr.LifecyclePolicyArgs{
			Rules: ecr.LifecyclePolicyRuleArray{
				&ecr.LifecyclePolicyRuleArgs{
					TagStatus:       ecr.LifecycleTagStatusAny,
					MaximumAgeLimit: pulumi.Float64(30),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	image, err := ecr.NewImage(ctx, "image", &ecr.ImageArgs{
		ImageName:     pulumi.String("image"),
		RepositoryUrl: repo.Url,
		Context:       pulumi.String("./app"),
		Dockerfile:    pulumi.String("./app/Dockerfile"),
		Platform:      pulumi.String("linux/amd64"),
	})
	if err != nil {
		return nil
	}
	ctx.Export("image", image.ImageUri)

	role, err := iam.NewRole(ctx, "ecrAccessRole", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(`
			{
				"Version": "2012-10-17",
    		"Statement": [{
    			"Action": "sts:AssumeRole",
      		"Effect": "Allow",
       		"Sid": "",
         	"Principal": {
       	  	"Service": ["build.apprunner.amazonaws.com"]
           }
        }]
      }
    `),
	})
	if err != nil {
		return err
	}

	_, err = iam.NewRolePolicy(ctx, "ecrACcessRolePolicy", &iam.RolePolicyArgs{
		Role: role.Name,
		Policy: pulumi.String(`
			{
    		"Version": "2012-10-17",
      	"Statement": [
       		{
         		"Effect": "Allow",
           	"Action": [
            	"ecr:GetAuthorizationToken",
             	"ecr:BatchCheckLayerAvailability",
              "ecr:GetDownloadUrlForLayer",
              "ecr:BatchGetImage",
              "ecr:DescribeImages"
            ],
            "Resource": "*"
          }
        ]
      }
  `),
	})

	if err != nil {
		return err
	}

	asg, err := apprunner.NewAutoScalingConfigurationVersion(ctx, "asg", &apprunner.AutoScalingConfigurationVersionArgs{
		AutoScalingConfigurationName: pulumi.String("autoscaling"),
		MinSize:                      pulumi.Int(1),
		MaxSize:                      pulumi.Int(3),
		MaxConcurrency:               pulumi.Int(10),
	})
	if err != nil {
		return err
	}

	svc, err := apprunner.NewService(ctx, "svc", &apprunner.ServiceArgs{
		ServiceName:                 pulumi.String("service"),
		AutoScalingConfigurationArn: asg.Arn,
		HealthCheckConfiguration: &apprunner.ServiceHealthCheckConfigurationArgs{
			Protocol: pulumi.String("HTTP"),
			Path:     pulumi.String("/"),
		},
		InstanceConfiguration: apprunner.ServiceInstanceConfigurationArgs{
			Memory: pulumi.String("1 GB"),
			Cpu:    pulumi.String("0.5 vCPU"),
		},
		SourceConfiguration: apprunner.ServiceSourceConfigurationArgs{
			AuthenticationConfiguration: apprunner.ServiceSourceConfigurationAuthenticationConfigurationArgs{
				AccessRoleArn: role.Arn,
			},
			ImageRepository: apprunner.ServiceSourceConfigurationImageRepositoryArgs{
				ImageRepositoryType: pulumi.String("ECR"),
				ImageIdentifier:     image.ImageUri,
				ImageConfiguration: apprunner.ServiceSourceConfigurationImageRepositoryImageConfigurationArgs{
					Port: pulumi.String("80"),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	ctx.Export("url", svc.ServiceUrl)

	return nil
}
