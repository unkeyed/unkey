package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apprunner/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"

	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func main() {
	ctx := context.Background()
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	ecrService := ecr.NewFromConfig(cfg)

	repositoryName := "rawdog"
	var repo *ecrTypes.Repository
	createRepoResponse, err := ecrService.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil && !strings.Contains(err.Error(), "RepositoryAlreadyExistsException") {
		panic(err)
	}
	if createRepoResponse != nil {
		repo = createRepoResponse.Repository
	} else {
		describeRepoResponse, err := ecrService.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
			RepositoryNames: []string{repositoryName},
		})
		if err != nil {
			panic(err)
		}
		repo = &describeRepoResponse.Repositories[0]
	}

	log.Printf("Repository: %s", *repo.RepositoryUri)

	_, err = ecrService.PutLifecyclePolicy(ctx, &ecr.PutLifecyclePolicyInput{
		RepositoryName: aws.String(repositoryName),
		LifecyclePolicyText: aws.String(`
			{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Expire images older than 14 days",
            "selection": {
                "tagStatus": "any",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 14
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
			`),
	})

	iamService := iam.NewFromConfig(cfg)
	roleName := "ecrAccess"
	var roleArn string
	role, err := iamService.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName: aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(`{
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
	if err == nil && !strings.Contains(err.Error(), "EntityAlreadyExists") {
		panic(err)
	}
	if role != nil {
		roleArn = *role.Role.Arn
	} else {
		getRole, err := iamService.GetRole(ctx, &iam.GetRoleInput{
			RoleName: aws.String(roleName),
		})
		if err != nil {
			panic(err)
		}
		roleArn = *getRole.Role.Arn
	}

	policy, err := iamService.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyName: aws.String("ecrPullPolicy"),
		PolicyDocument: aws.String(`{
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
      }`),
	})
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists") {

		panic(err)

	}
	if policy != nil {
		_, err = iamService.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
			RoleName:  aws.String(roleName),
			PolicyArn: policy.Policy.Arn,
		})
		if err != nil {
			panic(err)
		}
	}

	serviceName := "rawdog2"

	log.Println(roleArn)
	apprunnerService := apprunner.NewFromConfig(cfg)
	var app *types.Service
	createApp, err := apprunnerService.CreateService(ctx, &apprunner.CreateServiceInput{
		ServiceName:                 aws.String(serviceName),
		AutoScalingConfigurationArn: nil,
		HealthCheckConfiguration: &types.HealthCheckConfiguration{
			Protocol: types.HealthCheckProtocolHttp,
			Path:     aws.String("/"),
		},
		InstanceConfiguration: &types.InstanceConfiguration{
			Memory: aws.String("1 GB"),
			Cpu:    aws.String("0.5 vCPU"),
		},
		SourceConfiguration: &types.SourceConfiguration{
			AuthenticationConfiguration: &types.AuthenticationConfiguration{
				AccessRoleArn: aws.String(roleArn),
			},
			ImageRepository: &types.ImageRepository{
				ImageRepositoryType: types.ImageRepositoryTypeEcr,
				ImageIdentifier:     aws.String(fmt.Sprintf("%s@sha256:eaa98a09d2d6b1e82c64b585eb0937bbbb23ccd0ef8277f9ddb0d3f886b9b1d4", *repo.RepositoryUri)),
				ImageConfiguration: &types.ImageConfiguration{
					Port:                        aws.String("80"),
					RuntimeEnvironmentVariables: map[string]string{},
				},
			},
		},
	})

	if err != nil && !strings.Contains(err.Error(), "Service with the provided name already exists") {
		panic(err)
	}
	if createApp != nil {
		app = createApp.Service
	} else {

		services, err := apprunnerService.ListServices(ctx, &apprunner.ListServicesInput{})
		if err != nil {
			panic(err)
		}
		for _, s := range services.ServiceSummaryList {
			if *s.ServiceName == serviceName {
				desc, err := apprunnerService.DescribeService(ctx, &apprunner.DescribeServiceInput{
					ServiceArn: s.ServiceArn,
				})
				if err != nil {
					panic(err)
				}
				app = desc.Service
				break
			}
		}
	}

	deployment, err := apprunnerService.StartDeployment(ctx, &apprunner.StartDeploymentInput{
		ServiceArn: app.ServiceArn,
	})

	if err != nil {
		panic(err)
	}

	ops, err := apprunnerService.ListOperations(ctx, &apprunner.ListOperationsInput{
		ServiceArn: app.ServiceArn,
	})
	if err != nil {
		panic(err)
	}

	for _, op := range ops.OperationSummaryList {
		if *op.Id == *deployment.OperationId {
			log.Printf(" [ op ] %+v\n\n", op)

		}
	}

	log.Printf("deployment: %+v\n", deployment)
	log.Printf("url: %s\n", *app.ServiceUrl)

}
