package deploy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/urfave/cli/v2"
)

var Cmd = &cli.Command{
	Name: "deploy",
	Flags: []cli.Flag{

		&cli.StringFlag{
			Name:     "flightcontrol-deploy-hook-url",
			Required: true,
			EnvVars:  []string{"FLIGHTCONTROL_DEPLOY_HOOK_URL"},
		},
		&cli.StringFlag{
			Name:     "flightcontrol-api-key",
			Required: true,
			EnvVars:  []string{"FLIGHTCONTROL_API_KEY"},
		},
		&cli.DurationFlag{
			Name:  "timeout",
			Value: 15 * time.Minute,
		},
	},
	Action: run,
}

func run(c *cli.Context) error {
	deployHookURL := c.String("flightcontrol-deploy-hook-url")
	apiKey := c.String("flightcontrol-api-key")

	if deployHookURL == "" {
		return fmt.Errorf("missing required flag: --flightcontrol-deploy-hook-url")
	}

	if apiKey == "" {
		return fmt.Errorf("missing required flag: --flightcontrol-api-key")
	}

	client := http.DefaultClient

	deployResponse := struct {
		DeploymentId string `json:"deploymentId"`
		Success      bool   `json:"success"`
	}{}

	deployResp, err := http.Get(deployHookURL)
	if err != nil {
		return fmt.Errorf("failed to send deploy hook: %w", err)
	}
	defer deployResp.Body.Close()

	err = json.NewDecoder(deployResp.Body).Decode(&deployResponse)
	if err != nil {
		return fmt.Errorf("failed to decode deploy response: %w", err)
	}

	if !deployResponse.Success {
		return fmt.Errorf("deploy hook failed")
	}

	timeout := c.Duration("timeout")
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}

	for timeoutC := time.After(timeout); ; {
		select {
		case <-timeoutC:
			return fmt.Errorf("deployment timed out")
		default:
			{

				time.Sleep(10 * time.Second)

				req, err := http.NewRequest("GET", fmt.Sprintf("https://api.flightcontrol.dev/v1/deployments/%s", deployResponse.DeploymentId), nil)
				if err != nil {
					return fmt.Errorf("failed to create deployment status request: %w", err)
				}
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

				resp, err := client.Do(req)
				if err != nil {
					return fmt.Errorf("failed to get deployment status: %w", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("failed to get deployment status: %d", resp.StatusCode)
				}

				deploymentStatus := struct {
					Status string `json:"status"`
				}{}
				err = json.NewDecoder(resp.Body).Decode(&deploymentStatus)
				if err != nil {
					return fmt.Errorf("failed to decode deployment status: %w", err)
				}
				switch deploymentStatus.Status {
				case "PENDING", "INPROGRESS", "BUILDING", "DEPLOYING", "PENDING_DEPENDENCY", "PROVISIONING":
					log.Printf("deployment status: %s\n", deploymentStatus.Status)
					continue
				case "SUCCESS", "NO_CHANGE":
					return nil
				case "CANCELLED", "ERROR", "BUILD_ERROR", "DEPLOY_ERROR":
					return fmt.Errorf("deployment failed: %s", deploymentStatus.Status)
				}
			}
		}
	}
}
