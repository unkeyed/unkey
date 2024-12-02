package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go/aws"
)

func ComposeGreeting(ctx context.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %s", name), nil
}

type EcrRepository struct {
	ARN string
}

type awsapi struct {
	cfg config.Config
	ecr *ecr.Client
}

func NewAws() (*awsapi, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %w", err)
	}

	return &awsapi{
		cfg: cfg,
		ecr: ecr.NewFromConfig(cfg),
	}, nil
}

func (api *awsapi) CreateECR(ctx context.Context, repositoryName string) (EcrRepository, error) {

	createRepoResponse, err := api.ecr.CreateRepository(ctx, &ecr.CreateRepositoryInput{
		RepositoryName: aws.String(repositoryName),
	})
	if err != nil && !strings.Contains(err.Error(), "RepositoryAlreadyExistsException") {

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

}
