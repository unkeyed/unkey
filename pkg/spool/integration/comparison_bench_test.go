//go:build linux

package integration_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
	"github.com/unkeyed/unkey/pkg/openapi/validation"
	"github.com/unkeyed/unkey/pkg/spool"
	"golang.org/x/sys/unix"
)

var errRejected = errors.New("request rejected")

type comparisonFixture struct {
	schema map[string]any
	prefix string
	block  string
	bytes  int64
	suffix string
	valid  bool
}

func (f comparisonFixture) body() (io.Reader, int64) {
	return io.MultiReader(strings.NewReader(f.prefix), io.LimitReader(&repeatedBlock{block: f.block, offset: 0}, f.bytes), strings.NewReader(f.suffix)), int64(len(f.prefix)+len(f.suffix)) + f.bytes
}

type repeatedBlock struct {
	block  string
	offset int
}

func (r *repeatedBlock) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		copied := copy(p[n:], r.block[r.offset:])
		r.offset = (r.offset + copied) % len(r.block)
		n += copied
	}
	return n, nil
}

func comparisonFixtures() map[string]comparisonFixture {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"image": map[string]any{"type": "string", "minLength": 1},
			"count": map[string]any{"type": "integer", "minimum": 1},
		},
		"required": []string{"image", "count"}, "additionalProperties": false,
	}
	fixtures := make(map[string]comparisonFixture)
	for name, size := range map[string]int64{"1KiB": 1 << 10, "1MiB": 1 << 20, "8MiB": 8 << 20, "16MiB": 16 << 20, "64MiB": 64 << 20} {
		fixtures[name] = comparisonFixture{schema: schema, prefix: `{"image":"`, block: strings.Repeat("a", 32<<10), bytes: size, suffix: `","count":1}`, valid: true}
	}
	unicode := fixtures["16MiB"]
	unicode.block = strings.Repeat(`é😀\n\ud83d\ude00`, 1024)
	unicode.bytes = (16 << 20) / int64(len(unicode.block)) * int64(len(unicode.block))
	fixtures["unicode16MiB"] = unicode
	late := fixtures["16MiB"]
	late.suffix = `","count":0}`
	late.valid = false
	fixtures["late-invalid16MiB"] = late
	early := late
	early.prefix = `{"count":0,"image":"`
	early.suffix = `"}`
	fixtures["early-invalid16MiB"] = early
	item := `{"name":"alice","n":17}`
	block := strings.Repeat(item+",", 1024)
	fixtures["objects16MiB"] = comparisonFixture{
		schema: map[string]any{"type": "array", "items": map[string]any{
			"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}, "n": map[string]any{"type": "integer"}},
			"required": []string{"name", "n"}, "additionalProperties": false,
		}},
		prefix: "[", block: block, bytes: (16 << 20) / int64(len(block)) * int64(len(block)), suffix: item + "]", valid: true,
	}
	return fixtures
}

type replayBody struct {
	io.Reader
	io.Closer
}

func comparisonAdmission(b *testing.B, mode string, schema map[string]any) func(*http.Request) (io.ReadCloser, error) {
	b.Helper()
	if mode == "before" {
		spec, err := json.Marshal(map[string]any{
			"openapi": "3.1.0", "info": map[string]any{"title": "Comparison", "version": "1.0.0"},
			"paths": map[string]any{"/upload": map[string]any{"post": map[string]any{
				"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}},
				"responses":   map[string]any{"200": map[string]any{"description": "Accepted"}},
			}}},
		})
		require.NoError(b, err)
		validator, err := validation.NewFromBytes(spec)
		require.NoError(b, err)
		return func(r *http.Request) (io.ReadCloser, error) {
			if result := validator.Validate(r); result != nil {
				return nil, errors.Join(errRejected, r.Body.Close())
			}
			return r.Body, nil
		}
	}
	document, err := json.Marshal(schema)
	require.NoError(b, err)
	compiled, err := streamschema.Compile(document, streamschema.DefaultLimits())
	require.NoError(b, err)
	pool, err := spool.NewPool(spool.Config{Dir: b.TempDir(), BodyBytesMax: 128 << 20, TotalBytesMax: 1 << 30, ConcurrentMax: 8})
	require.NoError(b, err)
	return func(r *http.Request) (io.ReadCloser, error) {
		body, err := pool.Admit(r.Context(), r.Body, r.ContentLength, compiled.Validate)
		if err != nil {
			var failure *spool.Error
			if errors.As(err, &failure) && failure.Kind == spool.FailureInvalid {
				return nil, errors.Join(errRejected, err)
			}
			return nil, err
		}
		return replayBody{Reader: body.Reader(), Closer: body}, nil
	}
}

func comparisonDigest(b *testing.B, f comparisonFixture) string {
	b.Helper()
	h := sha256.New()
	body, _ := f.body()
	_, err := io.Copy(h, body)
	require.NoError(b, err)
	return hex.EncodeToString(h.Sum(nil))
}

func BenchmarkValidationComparison(b *testing.B) {
	for name, fixture := range comparisonFixtures() {
		b.Run(name, func(b *testing.B) {
			for _, concurrency := range []int{1, 8} {
				b.Run(fmt.Sprintf("c%d", concurrency), func(b *testing.B) {
					for _, mode := range []string{"before", "after"} {
						b.Run(mode, func(b *testing.B) {
							admit := comparisonAdmission(b, mode, fixture.schema)
							digest := comparisonDigest(b, fixture)
							benchmarkRequests(b, concurrency, func() (int64, error) {
								input, size := fixture.body()
								req, err := http.NewRequestWithContext(b.Context(), http.MethodPost, "http://comparison/upload", input)
								if err != nil {
									return 0, err
								}
								req.ContentLength = size
								req.Header.Set("Content-Type", "application/json")
								body, err := admit(req)
								if !fixture.valid && errors.Is(err, errRejected) {
									return 0, nil
								}
								if err != nil {
									return 0, err
								}
								h := sha256.New()
								n, readErr := io.Copy(h, body)
								closeErr := body.Close()
								if err := errors.Join(readErr, closeErr); err != nil {
									return 0, err
								}
								if !fixture.valid || n != size || hex.EncodeToString(h.Sum(nil)) != digest {
									return 0, errors.New("unexpected acceptance or replay mismatch")
								}
								return n, nil
							})
						})
					}
				})
			}
		})
	}
}

func BenchmarkHTTPValidationComparison(b *testing.B) {
	fixtures := comparisonFixtures()
	for _, name := range []string{"1KiB", "16MiB"} {
		b.Run(name, func(b *testing.B) {
			fixture := fixtures[name]
			digest := comparisonDigest(b, fixture)
			for _, concurrency := range []int{1, 8} {
				b.Run(fmt.Sprintf("c%d", concurrency), func(b *testing.B) {
					for _, mode := range []string{"before", "after"} {
						b.Run(mode, func(b *testing.B) {
							admit := comparisonAdmission(b, mode, fixture.schema)
							upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								h := sha256.New()
								if _, err := io.Copy(h, r.Body); err != nil {
									http.Error(w, err.Error(), http.StatusInternalServerError)
									return
								}
								if _, err := io.WriteString(w, hex.EncodeToString(h.Sum(nil))); err != nil {
									b.Error(err)
								}
							}))
							b.Cleanup(upstream.Close)
							upstreamURL, err := url.Parse(upstream.URL)
							require.NoError(b, err)
							proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
							completed := make(chan struct{}, concurrency)
							frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								defer func() { completed <- struct{}{} }()
								body, err := admit(r)
								if err != nil {
									http.Error(w, err.Error(), http.StatusBadRequest)
									return
								}
								defer func() {
									if err := body.Close(); err != nil {
										b.Error(err)
									}
								}()
								r.Body = io.NopCloser(body)
								proxy.ServeHTTP(w, r)
							}))
							b.Cleanup(frontend.Close)
							benchmarkRequests(b, concurrency, func() (int64, error) {
								input, size := fixture.body()
								req, err := http.NewRequestWithContext(b.Context(), http.MethodPost, frontend.URL+"/upload", input)
								if err != nil {
									return 0, err
								}
								req.ContentLength = size
								req.Header.Set("Content-Type", "application/json")
								response, err := frontend.Client().Do(req)
								if err != nil {
									return 0, err
								}
								got, readErr := io.ReadAll(response.Body)
								closeErr := response.Body.Close()
								<-completed
								if err := errors.Join(readErr, closeErr); err != nil {
									return 0, err
								}
								if response.StatusCode != http.StatusOK || string(got) != digest {
									return 0, fmt.Errorf("unexpected HTTP status %d or replay digest", response.StatusCode)
								}
								return size, nil
							})
						})
					}
				})
			}
		})
	}
}

func benchmarkRequests(b *testing.B, concurrency int, request func() (int64, error)) {
	b.Helper()
	var before, after unix.Rusage
	var next, replayed atomic.Int64
	errs := make(chan error, concurrency)
	latencies := make([]int64, b.N)
	var workers sync.WaitGroup
	_, err := request()
	require.NoError(b, err)
	runtime.GC()
	require.NoError(b, unix.Getrusage(unix.RUSAGE_SELF, &before))
	b.ReportAllocs()
	b.ResetTimer()
	for range concurrency {
		workers.Go(func() {
			for {
				index := next.Add(1) - 1
				if index >= int64(b.N) {
					return
				}
				start := time.Now()
				n, err := request()
				latencies[index] = time.Since(start).Nanoseconds()
				if err != nil {
					errs <- err
					return
				}
				replayed.Add(n)
			}
		})
	}
	workers.Wait()
	b.StopTimer()
	require.NoError(b, unix.Getrusage(unix.RUSAGE_SELF, &after))
	close(errs)
	for err := range errs {
		require.NoError(b, err)
	}
	user := after.Utime.Nano() - before.Utime.Nano()
	system := after.Stime.Nano() - before.Stime.Nano()
	b.ReportMetric(float64(user+system)/float64(b.N), "cpu-ns/op")
	b.ReportMetric(float64(system)/float64(b.N), "sys-ns/op")
	b.ReportMetric(float64(after.Maxrss), "peak-RSS-KiB")
	b.ReportMetric(float64(replayed.Load())/float64(b.N), "replayed-B/op")
	slices.Sort(latencies)
	b.ReportMetric(float64(latencies[(len(latencies)-1)/2]), "p50-ns")
	b.ReportMetric(float64(latencies[(len(latencies)-1)*95/100]), "p95-ns")
}
