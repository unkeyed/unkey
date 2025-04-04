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

// AwsEcs implements service discovery for applications running in Amazon ECS.
// It discovers other tasks within the same ECS service by:
//   - Querying the ECS API for running tasks in the same service
//   - Extracting network addresses from task metadata
//   - Filtering out the current task to prevent self-discovery
//
// This implementation requires:
//   - The ECS Task Metadata endpoint to be enabled
//   - Appropriate IAM permissions to call ECS APIs
//   - Tasks to be running in the same ECS cluster and service
type AwsEcs struct {
	// client is the AWS SDK ECS client used to query task information
	client *ecs.Client
	
	// logger receives operational logs for monitoring and debugging
	logger logging.Logger

	// clusterArn identifies the ECS cluster containing the tasks
	clusterArn string
	
	// family is the name of the task definition family
	family string

	// taskArn uniquely identifies this task in ECS
	// Used to filter out self-discovery when querying for peers
	taskArn string
}

// AwsEcsConfig configures the AWS ECS service discovery implementation.
type AwsEcsConfig struct {
	// Logger receives operational logs for monitoring and debugging
	Logger logging.Logger

	// Region specifies the AWS region where the ECS cluster is running
	// Example: "us-east-1", "eu-west-1"
	Region string
}

// NewAwsEcs creates a new ECS-based service discoverer.
// It initializes an AWS SDK client and retrieves task metadata from the
// ECS Task Metadata endpoint. The metadata is used to identify the current
// task and its cluster for subsequent discovery operations.
//
// Returns an error if:
//   - Required environment variables are missing
//   - Task metadata endpoint is unreachable
//   - Task metadata is invalid or incomplete
//   - AWS credentials are invalid or missing
func NewAwsEcs(config AwsEcsConfig) (*AwsEcs, error) {
	config.Logger.Debug("initializing ecs service discovery", "config", config)
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

// Discover implements the Discoverer interface for ECS tasks.
// It queries the ECS API to find other running tasks in the same service,
// extracting their network addresses for peer-to-peer communication.
//
// The method:
//   - Lists all running tasks in the same family
//   - Filters out this task's own ARN
//   - Retrieves detailed task information including network interfaces
//   - Extracts private IPv4 addresses from network interfaces
//
// Returns an error if:
//   - ECS API calls fail
//   - Tasks have no network interfaces
//   - Required task information is missing
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

// getAddrs retrieves network addresses for the given task ARNs.
// For each task, it:
//   - Calls DescribeTasks to get detailed task information
//   - Extracts the private IPv4 address from the first network interface
//   - Logs any failures for individual tasks
//
// Returns an error if:
//   - The input slice is empty or too large (AWS limits)
//   - The DescribeTasks call fails
//   - A task has no network interfaces
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

// getTaskArns retrieves ARNs for all running tasks in the same family.
// It handles pagination automatically and filters out this task's ARN.
//
// The method uses ListTasks to find tasks that are:
//   - In the same cluster
//   - From the same task definition family
//   - In RUNNING desired status
//
// Returns an error if the ListTasks API call fails.
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

// getMeta retrieves and validates task metadata from the ECS Task Metadata endpoint.
// This information is used to identify:
//   - The current task (via TaskARN)
//   - The ECS cluster this task belongs to
//   - The task definition family
//
// The method requires the ECS_CONTAINER_METADATA_URI_V4 environment variable
// to be set, which is automatically provided by the ECS agent.
//
// Returns an error if:
//   - The metadata endpoint environment variable is missing
//   - The HTTP request fails
//   - The response is invalid or missing required fields
//   - Required metadata fields are empty
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
