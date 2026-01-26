package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/fault"
)

const defaultMaxSizeBytes = 10 * 1024 * 1024 * 1024 // 10GB

var Cmd = &cli.Command{
	Name:        "repofetch",
	Usage:       "Fetch a GitHub repository tarball and upload to S3",
	Description: "Fetches a tarball from GitHub, strips the top-level directory, and streams to an S3 presigned URL.",
	Aliases:     []string{},
	Version:     "",
	Commands:    []*cli.Command{},
	AcceptsArgs: false,
	Flags: []cli.Flag{
		cli.String("github-token", "GitHub API token", cli.Required()),
		cli.String("repo", "GitHub repository (owner/repo)", cli.Required()),
		cli.String("sha", "Git commit SHA or ref", cli.Required()),
		cli.String("upload-url", "S3 presigned upload URL", cli.Required()),
		cli.Int64("max-size-bytes", "Maximum tarball size in bytes", cli.Default(fmt.Sprintf("%d", defaultMaxSizeBytes))),
	},
	Action: runAction,
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	githubToken := cmd.RequireString("github-token")
	repo := cmd.RequireString("repo")
	sha := cmd.RequireString("sha")
	uploadURL := cmd.RequireString("upload-url")
	maxSizeBytes := cmd.Int64("max-size-bytes")
	if maxSizeBytes == 0 {
		maxSizeBytes = defaultMaxSizeBytes
	}

	tarballURL := fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", repo, sha)
	logger.Info("fetching tarball", "repo", repo, "sha", sha)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to create github request"))
	}
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	ghResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to fetch tarball from github"))
	}
	defer func() { _ = ghResp.Body.Close() }()

	if ghResp.StatusCode != http.StatusOK {
		return fault.New(fmt.Sprintf("github returned status %d", ghResp.StatusCode))
	}

	logger.Info("repackaging tarball to strip top-level directory")

	pr, pw := io.Pipe()

	errCh := make(chan error, 1)
	go func() {
		defer func() { _ = pw.Close() }()
		if repackErr := repackageTarball(ghResp.Body, pw, maxSizeBytes); repackErr != nil {
			errCh <- repackErr
			return
		}
		errCh <- nil
	}()

	logger.Info("streaming repackaged tarball to s3")

	uploadReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, pr)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to create upload request"))
	}
	uploadReq.Header.Set("Content-Type", "application/gzip")

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to upload to s3"))
	}
	defer func() { _ = uploadResp.Body.Close() }()

	if repackErr := <-errCh; repackErr != nil {
		return repackErr
	}

	if uploadResp.StatusCode < 200 || uploadResp.StatusCode >= 300 {
		return fault.New(fmt.Sprintf("s3 upload returned status %d", uploadResp.StatusCode))
	}

	logger.Info("upload complete", "status", uploadResp.StatusCode)
	return nil
}

func repackageTarball(src io.Reader, dst io.Writer, maxBytes int64) error {
	limitedSrc := io.LimitReader(src, maxBytes)

	gzr, err := gzip.NewReader(limitedSrc)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to create gzip reader"))
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)

	gzw := gzip.NewWriter(dst)
	defer func() { _ = gzw.Close() }()

	tw := tar.NewWriter(gzw)
	defer func() { _ = tw.Close() }()

	for {
		header, readErr := tr.Next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fault.Wrap(readErr, fault.Internal("failed to read tar entry"))
		}

		newName := stripTopLevel(header.Name)
		if newName == "" {
			continue
		}

		header.Name = newName

		if writeErr := tw.WriteHeader(header); writeErr != nil {
			return fault.Wrap(writeErr, fault.Internal("failed to write tar header"))
		}

		if header.Typeflag == tar.TypeReg {
			if _, copyErr := io.Copy(tw, tr); copyErr != nil {
				return fault.Wrap(copyErr, fault.Internal("failed to copy tar content"))
			}
		}
	}

	return nil
}

func stripTopLevel(path string) string {
	idx := strings.Index(path, "/")
	if idx == -1 {
		return ""
	}
	return path[idx+1:]
}
