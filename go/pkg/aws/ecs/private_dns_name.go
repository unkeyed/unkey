package ecs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// GetPrivateDnsName queries the aws task metadata endpoint to retrieve the private dns name of the current container.
//
// It expects the ECS_CONTAINER_METADATA_URI_V4 environment variable to be set.
func GetPrivateDnsName() (string, error) {
	url := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	err := assert.NotEmpty(url)
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("ECS_CONTAINER_METADATA_URI_V4 is empty", ""))
	}

	metaRes, err := http.Get(fmt.Sprintf("%s/task", url))
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("failed to get task metadata", ""))
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
		return "", fault.Wrap(err, fault.WithDesc("failed to read task metadata body", ""))
	}

	body := ecsTaskMetadataV2Response{} // nolint:exhaustruct
	fmt.Println("body", body)

	err = json.Unmarshal(b, &body)
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("failed to parse task metadata", ""))
	}

	err = assert.True(len(body.Containers) > 0)
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("no containers found", ""))
	}

	err = assert.True(len(body.Containers[0].Networks) > 0)
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("no networks found", ""))
	}

	addr := body.Containers[0].Networks[0].PrivateDNSName

	err = assert.NotEmpty(addr)
	if err != nil {
		return "", fault.Wrap(err, fault.WithDesc("PrivateDNSName is empty", ""))
	}
	return addr, nil
}
