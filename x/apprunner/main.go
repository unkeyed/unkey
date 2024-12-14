package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
)

const BACKEND_URL = "file://./state"

func main() {

	// to destroy our program, we can run `go run main.go destroy`
	destroy := false
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "destroy" {
			destroy = true
		}
	}
	ctx := context.Background()

	// we use a simple stack name here, but recommend using auto.FullyQualifiedStackName for maximum specificity.
	stackName := "stackname"

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	projectName := "abc"

	// I don't know what this does
	pul, err := auto.NewPulumiCommand(&auto.PulumiCommandOptions{
		SkipVersionCheck: true,
	})
	if err != nil {
		fmt.Printf("Failed to create pulumi command: %v\n", err)
		os.Exit(1)
	}

	proj := workspace.Project{
		Name:    tokens.PackageName(projectName),
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		Main:    cwd,
		Backend: &workspace.ProjectBackend{
			URL: BACKEND_URL,
		},
	}
	ws, err := auto.NewLocalWorkspace(ctx,
		auto.Pulumi(pul),
		auto.Project(proj),
		auto.Program(deploy),
		auto.WorkDir(cwd),
		auto.EnvVars(map[string]string{
			"PULUMI_CONFIG_PASSPHRASE": "passphrase",
		}),
	)
	if err != nil {
		fmt.Printf("Failed to create workspace: %v\n", err)
		os.Exit(1)
	}

	s, err := auto.UpsertStack(ctx, stackName, ws)

	if err != nil {
		fmt.Printf("Failed to upsert stack: %v\n", err)
		os.Exit(1)
	}

	err = ws.InstallPlugin(ctx, "aws", "v4.0.0")
	if err != nil {
		fmt.Printf("Failed to install aws plugin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created/Selected stack %q\n", stackName)

	// set stack configuration specifying the AWS region to deploy
	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: "eu-central-1"})
	if err != nil {
		fmt.Printf("Failed to set region: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully set config")
	fmt.Println("Starting refresh")

	_, err = s.Refresh(ctx)
	if err != nil {
		fmt.Printf("Failed to refresh state: %v\n", err)
		os.Exit(1)
	}

	if destroy {
		fmt.Println("Starting stack destroy")

		// wire up our destroy to stream progress to stdout
		stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)

		// destroy our stack and exit early
		_, err := s.Destroy(ctx, stdoutStreamer)
		if err != nil {
			fmt.Printf("Failed to destroy stack: %v", err)
			os.Exit(1)
		}
		fmt.Println("Stack successfully destroyed")
		os.Exit(0)
	}

	fmt.Println("Starting update")

	// wire up our update to stream progress to stdout
	stdoutStreamer := optup.ProgressStreams(os.Stdout)

	// run the update to deploy our fargate web service
	res, err := s.Up(ctx, stdoutStreamer)
	if err != nil {
		fmt.Printf("Failed to update stack: %v\n\n", err)
		os.Exit(1)
	}

	fmt.Println("Update succeeded!")

	// get the URL from the stack outputs
	url, ok := res.Outputs["url"].Value.(string)
	if !ok {
		fmt.Println("Failed to unmarshall output URL")
		os.Exit(1)
	}
	imageUri, ok := res.Outputs["image"].Value.(string)
	if !ok {
		fmt.Println("Failed to unmarshall output imageUri")
		os.Exit(1)
	}

	fmt.Printf("URL: %s\n", url)
	fmt.Printf("image: %s\n", imageUri)
}
