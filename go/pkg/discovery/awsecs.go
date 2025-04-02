package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
)

type AwsEcs struct {
	client *ecs.Client
	logger logging.Logger

	clusterArn string
	family     string

	// Task ARN of the container running right now.
	// We need this to prevent joining a cluster with just ourselves.
	// [Discover()] must not return addresses of ourself
	taskArn string
}

type AwsEcsConfig struct {
	Logger logging.Logger
	Region string
}

func NewAwsEcs(config AwsEcsConfig) (*AwsEcs, error) {

	config.Logger.Warn("ecs sd config", "config", config)
	a := &AwsEcs{
		client: ecs.NewFromConfig(aws.Config{
			Region: config.Region,
		}),
		logger:     config.Logger,
		clusterArn: "",
		family:     "",
		taskArn:    "",
	}

	err := a.getMeta()
	if err != nil {
		return nil, err

	}
	return a, nil
}

func (a AwsEcs) Discover() ([]string, error) {
	ctx := context.Background()

	taskArns, err := a.getTaskArns(ctx)
	if err != nil {
		return nil, err
	}

	addrs, err := a.getAddrs(ctx, taskArns)
	if err != nil {
		return nil, err
	}

	return addrs, nil
}

func (a *AwsEcs) getAddrs(ctx context.Context, taskArns []string) ([]string, error) {

	err := assert.All(
		assert.GreaterOrEqual(len(taskArns), 1, "at least one task arn must be provided"),
		assert.LessOrEqual(len(taskArns), 100, "AWS can only describe 100 tasks at most"),
	)
	if err != nil {
		return nil, err
	}

	res, err := a.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: &a.clusterArn,
		Tasks:   taskArns,
	})
	if err != nil {
		return nil, err
	}
	for _, f := range res.Failures {
		a.logger.Error("failed to describe task",
			"arn", ptr.SafeDeref(f.Arn),
			"detail", ptr.SafeDeref(f.Detail),
			"reason", ptr.SafeDeref(f.Reason),
		)
	}

	addrs := []string{}
	for _, task := range res.Tasks {

		for _, container := range task.Containers {
			if len(container.NetworkInterfaces) == 0 {
				return nil, fault.New("container has no network interfaces")
			}
			addrs = append(addrs, *container.NetworkInterfaces[0].PrivateIpv4Address)

		}
	}
	return addrs, nil

}

func (a *AwsEcs) getTaskArns(ctx context.Context) ([]string, error) {
	taskArns := []string{}

	var nextToken *string = nil

	for {

		res, err := a.client.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:       &a.clusterArn,
			Family:        &a.family,
			DesiredStatus: types.DesiredStatusRunning,
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, arn := range res.TaskArns {
			if arn == a.taskArn {
				continue
			}
			taskArns = append(taskArns, arn)
		}
		nextToken = res.NextToken

		if nextToken == nil {
			break
		}

	}

	return taskArns, nil
}

func (a *AwsEcs) getMeta() error {

	// Retrieve the metadata URI from environment
	url := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	err := assert.NotEmpty(url)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("ECS_CONTAINER_METADATA_URI_V4 is empty", ""))
	}

	// Query the task metadata endpoint
	metaRes, err := http.Get(fmt.Sprintf("%s/task", url))
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to get task metadata", ""))
	}
	defer metaRes.Body.Close()

	// The API returns much more, we just don't care about it.
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-examples.html
	type ecsTaskMetadataV2Response struct {
		Cluster string `json:"Cluster"`
		TaskARN string `json:"TaskARN"`
		Family  string `json:"Family"`
	}

	b, err := io.ReadAll(metaRes.Body)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to read task metadata body", ""))
	}

	body := ecsTaskMetadataV2Response{} // nolint:exhaustruct

	err = json.Unmarshal(b, &body)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to parse task metadata", ""))
	}

	err = assert.All(
		assert.NotEmpty(body.Cluster, "Cluster ARN must not be empty"),
		assert.NotEmpty(body.Family, "Family must not be empty"),
		assert.NotNil(body.TaskARN, "Task ARN must not be empty"),
	)

	if err != nil {
		return err
	}

	a.clusterArn = body.Cluster
	a.family = body.Family
	a.taskArn = body.TaskARN
	return nil

}
