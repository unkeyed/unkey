package keyring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/uid"
)

const deletedEnvironmentPrefix = "deleted-environment-keyrings/"

var ErrDeleted = errors.New("environment keyring is permanently deleted")

func ValidateEnvironmentID(id string) error {
	suffix, ok := strings.CutPrefix(id, string(uid.EnvironmentPrefix)+"_")
	if !ok || suffix == "" {
		return errors.New("environment keyring must be an environment ID")
	}
	for _, c := range suffix {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return errors.New("environment ID must have an alphanumeric suffix")
		}
	}
	return nil
}

func (k *Keyring) CheckActive(ctx context.Context, ringID string) error {
	if !strings.HasPrefix(ringID, string(uid.EnvironmentPrefix)+"_") {
		return nil
	}
	if err := ValidateEnvironmentID(ringID); err != nil {
		return err
	}
	_, deleted, err := k.store.GetObject(ctx, deletedEnvironmentPrefix+ringID)
	if err != nil {
		return fmt.Errorf("check environment keyring deletion: %w", err)
	}
	if deleted {
		return ErrDeleted
	}
	return nil
}

func (k *Keyring) DeleteEnvironment(ctx context.Context, environmentID string) error {
	if err := ValidateEnvironmentID(environmentID); err != nil {
		return err
	}
	if err := k.store.PutObject(ctx, deletedEnvironmentPrefix+environmentID, nil); err != nil {
		return fmt.Errorf("persist environment keyring deletion: %w", err)
	}
	return k.deleteKeys(ctx, environmentID)
}

// SweepDeletedEnvironments revisits permanent tombstones because a writer can
// persist a key after deletion and crash before its post-write check.
func (k *Keyring) SweepDeletedEnvironments(ctx context.Context) error {
	tombstones, err := k.store.ListObjectKeys(ctx, deletedEnvironmentPrefix)
	if err != nil {
		return fmt.Errorf("list deleted environment keyrings: %w", err)
	}
	deleted := make(map[string]bool, len(tombstones))
	var errs []error
	for _, tombstone := range tombstones {
		environmentID, ok := strings.CutPrefix(tombstone, deletedEnvironmentPrefix)
		if !ok {
			continue
		}
		if err := ValidateEnvironmentID(environmentID); err != nil {
			errs = append(errs, err)
			continue
		}
		deleted[environmentID] = true
	}
	if len(deleted) == 0 {
		return errors.Join(errs...)
	}
	keys, err := k.store.ListObjectKeys(ctx, "keyring/"+string(uid.EnvironmentPrefix)+"_")
	if err != nil {
		return errors.Join(append(errs, fmt.Errorf("list environment keys: %w", err))...)
	}
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(errs, err)...)
		}
		path, ok := strings.CutPrefix(key, "keyring/")
		environmentID, _, hasKey := strings.Cut(path, "/")
		if !ok || !hasKey || !deleted[environmentID] {
			continue
		}
		if err := k.store.DeleteObject(ctx, key); err != nil {
			errs = append(errs, fmt.Errorf("clean up environment keyring %s: %w", environmentID, err))
		}
	}
	return errors.Join(errs...)
}

func (k *Keyring) deleteKeys(ctx context.Context, ringID string) error {
	prefix := k.buildLookupKey(ringID, "")
	keys, err := k.store.ListObjectKeys(ctx, prefix)
	if err != nil {
		return fmt.Errorf("list environment keys: %w", err)
	}
	var errs []error
	for _, key := range keys {
		if err := k.store.DeleteObject(ctx, key); err != nil {
			errs = append(errs, fmt.Errorf("delete environment key: %w", err))
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
