package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/frontline"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := command().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func command() *cli.Command {
	return &cli.Command{
		Name: "frontline", Usage: "Run the Unkey Frontline server",
		Flags: []cli.Flag{
			cli.String("config", "Path to a TOML config file, or the raw TOML config itself",
				cli.Default("unkey.toml"), cli.EnvVar("UNKEY_CONFIG")),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx, cmd.String("config"))
		},
	}
}

func run(ctx context.Context, pathOrContent string) error {
	data := []byte(pathOrContent)
	directory := "."
	if !strings.ContainsAny(pathOrContent, "=\n") {
		var err error
		data, err = os.ReadFile(pathOrContent)
		if err != nil {
			return fmt.Errorf("read config: %w", err)
		}
		directory = filepath.Dir(pathOrContent)
	}
	type localDevFile struct {
		LocalDev frontline.LocalDevConfig `toml:"local-dev"`
	}
	var probe map[string]any
	metadata, err := toml.Decode(os.ExpandEnv(string(data)), &probe)
	if err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if metadata.IsDefined("local-dev") {
		cfg, err := config.LoadBytes[localDevFile](data)
		if err != nil {
			return fmt.Errorf("load local-dev config: %w", err)
		}
		cfg.LocalDev.ConfigDirectory = directory
		return frontline.RunLocalDev(ctx, cfg.LocalDev)
	}
	cfg, err := config.LoadBytes[frontline.Config](data)
	if err != nil {
		return fmt.Errorf("load production config: %w", err)
	}
	return frontline.Run(ctx, cfg)
}
