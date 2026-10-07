package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
)

//go:embed config.md.tmpl
var markdownTemplate string

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, &http.Client{Timeout: time.Minute}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer, client *http.Client) error {
	command := &cli.Command{
		Name:  "configdocs",
		Usage: "Generate configuration references for Notion",
		Flags: []cli.Flag{
			cli.String("file", "Go source file declaring the config struct", cli.Required()),
			cli.String("struct", "Root config struct", cli.Default("Config")),
			cli.String("ref", "GitHub source revision", cli.Default("main")),
			cli.String("notion-page-id", "Existing Notion page to replace; omit to print Markdown"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			file := cmd.String("file")
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("--file is required")
			}
			uploadRequested := cmd.FlagIsSet("notion-page-id")
			id := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(cmd.String("notion-page-id")), "-", ""))
			token := strings.TrimSpace(os.Getenv("NOTION_TOKEN"))
			if uploadRequested {
				if _, err := hex.DecodeString(id); err != nil || len(id) != 32 {
					return fmt.Errorf("notion-page-id must contain 32 hexadecimal characters")
				}
				if token == "" {
					return fmt.Errorf("NOTION_TOKEN is required for upload")
				}
			}
			page, err := extract(file, cmd.String("struct"), cmd.String("ref"))
			if err != nil {
				return err
			}
			tmpl, err := template.New("config").Parse(markdownTemplate)
			if err != nil {
				return err
			}
			var markdown bytes.Buffer
			if err := tmpl.Execute(&markdown, page); err != nil {
				return err
			}
			if !uploadRequested {
				_, err := io.Copy(output, &markdown)
				return err
			}
			return upload(ctx, client, id, token, markdown.String())
		},
	}
	return command.Run(ctx, append([]string{"configdocs"}, args...))
}
