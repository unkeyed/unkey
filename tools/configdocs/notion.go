package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func upload(ctx context.Context, client *http.Client, pageID, token, markdown string) (err error) {
	var payload struct {
		Type           string `json:"type"`
		ReplaceContent struct {
			NewString            string `json:"new_str"`
			AllowDeletingContent bool   `json:"allow_deleting_content"`
		} `json:"replace_content"`
	}
	payload.Type = "replace_content"
	payload.ReplaceContent.NewString = markdown
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		"https://api.notion.com/v1/pages/"+pageID+"/markdown", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2026-03-11")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, res.Body.Close()) }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("upload to Notion page %s: HTTP %d", pageID, res.StatusCode)
	}
	var result struct {
		Object string `json:"object"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Notion response: %w", err)
	}
	if result.Object != "page_markdown" {
		return fmt.Errorf("unexpected Notion response object %q", result.Object)
	}
	if err := lockPage(ctx, client, pageID, token); err != nil {
		return fmt.Errorf("content uploaded but page lock failed: %w", err)
	}
	return nil
}

func lockPage(ctx context.Context, client *http.Client, pageID, token string) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		"https://api.notion.com/v1/pages/"+pageID, bytes.NewBufferString(`{"is_locked":true}`))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2026-03-11")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, res.Body.Close()) }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("lock Notion page %s: HTTP %d", pageID, res.StatusCode)
	}
	var result struct {
		Object   string `json:"object"`
		IsLocked bool   `json:"is_locked"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Notion lock response: %w", err)
	}
	if result.Object != "page" || !result.IsLocked {
		return fmt.Errorf("Notion did not confirm the page lock")
	}
	return nil
}
