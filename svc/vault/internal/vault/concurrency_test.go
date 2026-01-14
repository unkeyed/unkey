package vault

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

// TestConcurrency_ParallelEncrypt verifies that parallel encryption
// operations don't interfere with each other.
func TestConcurrency_ParallelEncrypt(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-parallel-enc"

	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    fmt.Sprintf("data-%d", idx),
			})
			encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Encrypt(ctx, encReq)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", idx, err)
				return
			}
			results <- res.Msg.GetEncrypted()
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	// Check for errors
	for err := range errors {
		t.Errorf("encryption error: %v", err)
	}

	// Verify all results are unique (different nonces)
	seen := make(map[string]bool)
	for enc := range results {
		require.False(t, seen[enc], "duplicate ciphertext found")
		seen[enc] = true
	}
}

// TestConcurrency_ParallelDecrypt verifies that parallel decryption
// operations return correct results.
func TestConcurrency_ParallelDecrypt(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-parallel-dec"

	// First, encrypt test data
	data := "shared-secret-data"
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	encrypted := encRes.Msg.GetEncrypted()

	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encrypted,
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Decrypt(ctx, decReq)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: %w", idx, err)
				return
			}
			if res.Msg.GetPlaintext() != data {
				errors <- fmt.Errorf("goroutine %d: wrong plaintext: got %q, want %q",
					idx, res.Msg.GetPlaintext(), data)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("decryption error: %v", err)
	}
}

// TestConcurrency_ParallelEncryptDecrypt verifies that mixed parallel
// encrypt/decrypt operations work correctly.
func TestConcurrency_ParallelEncryptDecrypt(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-parallel-mix"

	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	// Encrypt some data first to have something to decrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    "initial-data",
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	initialEncrypted := encRes.Msg.GetEncrypted()

	// Run encryptions and decryptions in parallel
	for i := 0; i < numGoroutines; i++ {
		// Encrypt goroutine
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    fmt.Sprintf("parallel-data-%d", idx),
			})
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			_, err := service.Encrypt(ctx, req)
			if err != nil {
				errors <- fmt.Errorf("encrypt goroutine %d: %w", idx, err)
			}
		}(i)

		// Decrypt goroutine
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: initialEncrypted,
			})
			req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			res, err := service.Decrypt(ctx, req)
			if err != nil {
				errors <- fmt.Errorf("decrypt goroutine %d: %w", idx, err)
				return
			}
			if res.Msg.GetPlaintext() != "initial-data" {
				errors <- fmt.Errorf("decrypt goroutine %d: wrong data", idx)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("parallel operation error: %v", err)
	}
}

// TestConcurrency_ParallelMultipleKeyrings verifies that parallel operations
// on different keyrings don't interfere.
func TestConcurrency_ParallelMultipleKeyrings(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	const numKeyrings = 10
	const opsPerKeyring = 10

	var wg sync.WaitGroup
	errors := make(chan error, numKeyrings*opsPerKeyring)

	for kr := 0; kr < numKeyrings; kr++ {
		keyring := fmt.Sprintf("keyring-%d", kr)
		expectedData := fmt.Sprintf("data-for-keyring-%d", kr)

		// First encrypt for this keyring
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    expectedData,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)
		encrypted := encRes.Msg.GetEncrypted()

		// Parallel decryptions for this keyring
		for op := 0; op < opsPerKeyring; op++ {
			wg.Add(1)
			go func(keyring, encrypted, expectedData string, opIdx int) {
				defer wg.Done()

				decReq := connect.NewRequest(&vaultv1.DecryptRequest{
					Keyring:   keyring,
					Encrypted: encrypted,
				})
				decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

				res, err := service.Decrypt(ctx, decReq)
				if err != nil {
					errors <- fmt.Errorf("%s op %d: %w", keyring, opIdx, err)
					return
				}
				if res.Msg.GetPlaintext() != expectedData {
					errors <- fmt.Errorf("%s op %d: got %q, want %q",
						keyring, opIdx, res.Msg.GetPlaintext(), expectedData)
				}
			}(keyring, encrypted, expectedData, op)
		}
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("multi-keyring error: %v", err)
	}
}

// TestConcurrency_SequentialReEncrypt verifies that re-encryption
// operations work correctly when run sequentially.
//
// Note: Parallel re-encryption is not tested here because ReEncrypt
// calls cache.Clear() which has known concurrency limitations with
// the otter cache library. This is a known limitation documented here.
func TestConcurrency_SequentialReEncrypt(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-seq-reenc"
	data := "data-to-reencrypt-sequentially"

	// Encrypt initial data
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	encrypted := encRes.Msg.GetEncrypted()

	// Run re-encryptions sequentially
	for i := 0; i < 10; i++ {
		req := connect.NewRequest(&vaultv1.ReEncryptRequest{
			Keyring:   keyring,
			Encrypted: encrypted,
		})
		req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.ReEncrypt(ctx, req)
		require.NoError(t, err, "re-encryption %d failed", i)

		// Verify the re-encrypted data
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: res.Msg.GetEncrypted(),
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err)
		require.Equal(t, data, decRes.Msg.GetPlaintext())

		// Use the new encrypted value for next iteration
		encrypted = res.Msg.GetEncrypted()
	}
}

// TestConcurrency_RaceConditionDetection is designed to be run with -race flag.
// It performs operations that would expose race conditions if they exist.
func TestConcurrency_RaceConditionDetection(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Shared resources that might have race conditions
	keyrings := []string{"race-kr-1", "race-kr-2", "race-kr-3"}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			keyring := keyrings[idx%len(keyrings)]

			// Encrypt
			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    fmt.Sprintf("race-data-%d", idx),
			})
			encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			encRes, err := service.Encrypt(ctx, encReq)
			if err != nil {
				return
			}

			// Immediately decrypt
			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: encRes.Msg.GetEncrypted(),
			})
			decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

			_, _ = service.Decrypt(ctx, decReq)
		}(i)
	}

	wg.Wait()
	// If we reach here without race detector complaints, test passes
}
