package vercel

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
)

func TestReferenceFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	s, err := parseDatafile(body, time.Now())
	require.NoError(t, err)
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(s)

	var tests []struct {
		Flag     string                       `json:"flag"`
		Entities openfeature.FlattenedContext `json:"entities"`
		Value    bool                         `json:"value"`
		Reason   string                       `json:"reason"`
	}
	expected, err := os.ReadFile("testdata/expected.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(expected, &tests))
	require.NotEmpty(t, tests)
	reasons := map[string]openfeature.Reason{
		"target_match": openfeature.TargetingMatchReason,
		"fallthrough":  openfeature.DefaultReason,
		"paused":       openfeature.StaticReason,
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%s/%d", test.Flag, i), func(t *testing.T) {
			detail := p.BooleanEvaluation(t.Context(), test.Flag, !test.Value, test.Entities)
			require.Equal(t, test.Value, detail.Value)
			require.NoError(t, detail.Error())
			require.Contains(t, reasons, test.Reason)
			require.Equal(t, reasons[test.Reason], detail.Reason)
		})
	}
}

func TestErrorsReturnAsymmetricDefaults(t *testing.T) {
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(&snapshot{fetchedAt: time.Now(), flags: map[string]compiledFlag{
		"unsupported": {err: errors.New("rules are unsupported")},
	}})

	for _, defaultValue := range []bool{false, true} {
		missing := p.BooleanEvaluation(t.Context(), "missing", defaultValue, nil)
		require.Equal(t, defaultValue, missing.Value)
		require.Equal(t, openfeature.FlagNotFoundCode, missing.ResolutionDetail().ErrorCode)
		unsupported := p.BooleanEvaluation(t.Context(), "unsupported", defaultValue, nil)
		require.Equal(t, defaultValue, unsupported.Value)
		require.Equal(t, openfeature.ParseErrorCode, unsupported.ResolutionDetail().ErrorCode)
	}
}

func TestStalenessAndInvalidContext(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	s, err := parseDatafile(body, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(s)
	stale := p.BooleanEvaluation(t.Context(), "direct", true, nil).ResolutionDetail()
	require.Equal(t, openfeature.ProviderNotReadyCode, stale.ErrorCode)
	require.False(t, p.Diagnostics().Ready)
	s.fetchedAt = time.Now()
	invalid := p.BooleanEvaluation(t.Context(), "direct", true, openfeature.FlattenedContext{"team": "bad"}).ResolutionDetail()
	require.Equal(t, openfeature.InvalidContextCode, invalid.ErrorCode)
}

func TestConcurrentEvaluation(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	s, err := parseDatafile(body, time.Now())
	require.NoError(t, err)
	p := &Provider{config: Config{MaxStaleness: time.Minute}}
	p.snapshot.Store(s)
	var wait sync.WaitGroup
	results := make(chan openfeature.BoolResolutionDetail, 100)
	for range 100 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- p.BooleanEvaluation(t.Context(), "direct", false, openfeature.FlattenedContext{"team": map[string]any{"id": "matching"}})
		}()
	}
	wait.Wait()
	close(results)
	for detail := range results {
		require.True(t, detail.Value, detail.ResolutionDetail().ErrorMessage)
	}
}

func TestInvalidContextCannotBeHiddenByAnotherMatchingTarget(t *testing.T) {
	body, err := os.ReadFile("testdata/direct-targets.json")
	require.NoError(t, err)
	p := testProvider(t, body)
	for range 100 {
		detail := p.BooleanEvaluation(t.Context(), "direct", false, openfeature.FlattenedContext{
			"team":    map[string]any{"id": "matching"},
			"account": "invalid",
		})
		require.Equal(t, openfeature.InvalidContextCode, detail.ResolutionDetail().ErrorCode)
	}
}
