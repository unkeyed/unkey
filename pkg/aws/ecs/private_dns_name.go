package ecs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// GetPrivateDnsName queries the AWS ECS task metadata endpoint to retrieve
// the private DNS name of the current container. This is used for service
// discovery within AWS ECS tasks.
//
// The function relies on the ECS_CONTAINER_METADATA_URI_V4 environment variable,
// which is automatically injected by the ECS agent when running in AWS ECS.
//
// This private DNS name can be used by other containers or services to
// communicate with this container within the same VPC.
//
// Returns:
//   - string: The private DNS name of the container
//   - error: Any error encountered during the metadata retrieval
//
// Example:
//
//	dnsName, err := ecs.GetPrivateDnsName()
//	if err != nil {
//	    logger.Error("Failed to get private DNS name", err)
//	    return err
//	}
//	logger.Info("Container private DNS name", slog.String("dns", dnsName))
func GetPrivateDnsName() (string, error) {
	// Retrieve the metadata URI from environment
	url := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	err := assert.NotEmpty(url)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("ECS_CONTAINER_METADATA_URI_V4 is empty"))
	}

	// Query the task metadata endpoint
	metaRes, err := http.Get(fmt.Sprintf("%s/task", url))
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to get task metadata"))
	}
	defer metaRes.Body.Close()

	// The API returns much more, we just don't care about it.
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4-examples.html
	type ecsTaskMetadataV2Response struct {
		Containers []struct {
			Networks []struct {
				PrivateDNSName string `json:"PrivateDNSName"`
			} `json:"Networks"`
		} `json:"Containers"`
	}

	b, err := io.ReadAll(metaRes.Body)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to read task metadata body"))
	}

	body := ecsTaskMetadataV2Response{} // nolint:exhaustruct

	err = json.Unmarshal(b, &body)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("failed to parse task metadata"))
	}

	// Validate the response structure
	err = assert.True(len(body.Containers) > 0)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("no containers found"))
	}

	err = assert.True(len(body.Containers[0].Networks) > 0)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("no networks found"))
	}

	// Extract the private DNS name
	addr := body.Containers[0].Networks[0].PrivateDNSName

	err = assert.NotEmpty(addr)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("PrivateDNSName is empty"))
	}
	return addr, nil
}
