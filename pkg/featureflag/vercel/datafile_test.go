package vercel

import (
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
)

func TestParseRejectsMalformedAndUnsupportedShapes(t *testing.T) {
	tests := []string{
		`{"variants":[null,true],"environments":{"production":1}}`,
		`{"variants":[true,false],"environments":{"production":null}}`,
		`{"variants":[true,false],"environments":{"production":{"fallthrough":null}}}`,
		`{"variants":[true,false],"environments":{"production":{"targets":[{"team":{"id":[null]}}}],"fallthrough":0}}}`,
		`{"variants":[false,true],"prerequisite":"other","environments":{"production":1}}`,
		`{"variants":[false,true],"environments":{"production":{"fallthrough":2}}}`,
		`{"variants":[false,true],"environments":{"production":{"fallthrough":{"type":"split"}}}}`,
		`{"variants":[false,true],"environments":{"production":{"fallthrough":0,"unknown":true}}}`,
		`{"variants":[false,true],"experiment":{},"environments":{"production":0}}`,
		`{"variants":[false,true],"environments":{"production":{"reuse":"preview"}}}`,
	}
	for _, input := range tests {
		_, err := parseFlag([]byte(input), "production")
		require.Error(t, err)
	}
}

func TestUnsupportedDefinitionsAreIsolatedAndReplacePriorValue(t *testing.T) {
	valid := []byte(`{"environment":"production","definitions":{"good":{"variants":[false,true],"environments":{"production":1}},"changed":{"variants":[false,true],"environments":{"production":1}}}}`)
	updated := []byte(`{"environment":"production","definitions":{"good":{"variants":[false,true],"environments":{"production":1}},"changed":{"variants":[false,true],"environments":{"production":{"rules":[{"conditions":[],"outcome":1}],"fallthrough":0}}}}}`)
	p := testProvider(t, valid, updated)
	require.NoError(t, p.refresh(t.Context()))
	require.True(t, p.BooleanEvaluation(t.Context(), "good", false, nil).Value)
	detail := p.BooleanEvaluation(t.Context(), "changed", false, nil)
	require.False(t, detail.Value)
	require.Equal(t, openfeature.ParseErrorCode, detail.ResolutionDetail().ErrorCode)
}
