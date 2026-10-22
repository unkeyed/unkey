package keyring

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

const deletedKeyringPrefix = "deleted-keyrings/"

var ErrDeleted = errors.New("keyring is permanently deleted")

func (k *Keyring) CheckActive(ctx context.Context, ringID string) error {
	_, deleted, err := k.store.GetObject(ctx, deletionMarkerKey(ringID))
	if err != nil {
		return fmt.Errorf("check keyring deletion: %w", err)
	}
	if deleted {
		return ErrDeleted
	}
	return nil
}

func (k *Keyring) Delete(ctx context.Context, ringID string) error {
	if err := k.store.PutObject(ctx, deletionMarkerKey(ringID), nil); err != nil {
		return fmt.Errorf("persist keyring deletion: %w", err)
	}
	return k.deleteKeys(ctx, ringID)
}

// SweepDeleted revisits permanent tombstones because a writer can
// persist a key after deletion and crash before its post-write check.
func (k *Keyring) SweepDeleted(ctx context.Context) error {
	tombstones, err := k.store.ListObjectKeys(ctx, deletedKeyringPrefix)
	if err != nil {
		return fmt.Errorf("list deleted keyrings: %w", err)
	}
	deleted := make(map[string]bool, len(tombstones))
	for _, tombstone := range tombstones {
		deleted[tombstone] = true
	}
	if len(deleted) == 0 {
		return nil
	}

	keys, err := k.store.ListObjectKeys(ctx, "keyring/")
	if err != nil {
		return fmt.Errorf("list keys: %w", err)
	}
	var errs []error
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		path, ok := strings.CutPrefix(key, "keyring/")
		separator := strings.LastIndex(path, "/")
		if !ok || separator < 0 || !deleted[deletionMarkerKey(path[:separator])] {
			continue
		}
		if err := k.store.DeleteObject(ctx, key); err != nil {
			errs = append(errs, fmt.Errorf("clean up key %s: %w", key, err))
		}
	}
	return errors.Join(errs...)
}

func deletionMarkerKey(ringID string) string {
	return fmt.Sprintf("%s%x", deletedKeyringPrefix, sha256.Sum256([]byte(ringID)))
}

func (k *Keyring) deleteKeys(ctx context.Context, ringID string) error {
	prefix := k.buildLookupKey(ringID, "")
	keys, err := k.store.ListObjectKeys(ctx, prefix)
	if err != nil {
		return fmt.Errorf("list keys: %w", err)
	}
	var errs []error
	for _, key := range keys {
		keyID, ok := strings.CutPrefix(key, prefix)
		if !ok || keyID == "" || strings.Contains(keyID, "/") {
			continue
		}
		if err := k.store.DeleteObject(ctx, key); err != nil {
			errs = append(errs, fmt.Errorf("delete key %s: %w", key, err))
		}
	}
	return errors.Join(errs...)
}

func (k *Keyring) storeKey(ctx context.Context, ringID, keyID string, data []byte) error {
	if err := k.CheckActive(ctx, ringID); err != nil {
		return err
	}

	key := k.buildLookupKey(ringID, keyID)
	if err := k.store.PutObject(ctx, key, data); err != nil {
		return err
	}
	if err := k.CheckActive(ctx, ringID); err != nil {
		if errors.Is(err, ErrDeleted) {
			return errors.Join(err, k.store.DeleteObject(ctx, key))
		}
		return err
	}
	return nil
}
