package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvidence_RequiresObservedWitness(t *testing.T) {
	cases := []struct {
		name     string
		outcome  string
		witness  string
		expected string
		pkg      string
		wantErr  bool
	}{
		{
			name:     "green",
			outcome:  "pass",
			witness:  "TestWitness",
			expected: "pass",
			pkg:      "./testdata/witness",
		},
		{
			name:     "red",
			outcome:  "fail",
			witness:  "TestWitness",
			expected: "fail",
			pkg:      "./testdata/witness",
		},
		{
			name:     "failure stays failed",
			outcome:  "fail",
			witness:  "TestWitness",
			expected: "pass",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "later pass cannot hide failure",
			outcome:  "fail-once",
			witness:  "TestWitness",
			expected: "pass",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "TestMain cannot hide failure",
			outcome:  "swallowed-failure",
			expected: "pass",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "passing test is not red",
			outcome:  "pass",
			witness:  "TestWitness",
			expected: "fail",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "skipped witness",
			outcome:  "skip",
			witness:  "TestWitness",
			expected: "pass",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "missing witness",
			outcome:  "pass",
			witness:  "TestMissing",
			expected: "pass",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "unrelated failure is not red",
			outcome:  "fail",
			witness:  "TestWitness/missing",
			expected: "fail",
			pkg:      "./testdata/witness",
			wantErr:  true,
		},
		{
			name:     "no tests",
			expected: "pass",
			pkg:      "./testdata/empty",
			wantErr:  true,
		},
		{
			name:     "build failure is not red",
			witness:  "TestWitness",
			expected: "fail",
			pkg:      "./testdata/broken",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("UNKEY_EVIDENCE_TEST_OUTCOME", tc.outcome)
			args := []string{"--seeds=7,19", "--repeat=2", "--expect=" + tc.expected}
			if tc.witness != "" {
				args = append(args, "--witness="+tc.witness)
			}
			args = append(args, tc.pkg)

			var output, logs bytes.Buffer
			err := run(args, &output, &logs)
			if tc.wantErr {
				require.Error(t, err, logs.String())
			} else {
				require.NoError(t, err, logs.String())
			}

			var evidence receipt
			require.NoError(t, json.Unmarshal(output.Bytes(), &evidence))
			require.Equal(t, tc.witness, evidence.Witness)
			if tc.wantErr {
				require.NotEmpty(t, evidence.Error)
				require.Len(t, evidence.Runs, 1)
			} else {
				require.Empty(t, evidence.Error)
				require.Len(t, evidence.Runs, 2)
				require.Contains(t, evidence.Runs[0].Command, "-shuffle=7")
				require.Contains(t, evidence.Runs[1].Command, "-shuffle=19")
				require.Contains(t, evidence.Runs[0].Command, "-count=2")
			}
		})
	}
}

func TestEvidence_KeepsPackageDiagnosticsOutOfSelection(t *testing.T) {
	t.Setenv("UNKEY_EVIDENCE_TEST_OUTCOME", "pass")
	args := []string{
		"--seeds=7",
		"--witness=TestWitness",
		"github.com/unkeyed/unkey/tools/testevidence/testdata/no-packages/...",
		"./testdata/witness",
	}

	var output, logs bytes.Buffer
	err := run(args, &output, &logs)
	require.NoError(t, err, logs.String())
	require.Contains(t, logs.String(), "matched no packages")

	var evidence receipt
	require.NoError(t, json.Unmarshal(output.Bytes(), &evidence))
	require.Empty(t, evidence.Error)
	require.Len(t, evidence.Runs, 1)
	require.Equal(t, 0, evidence.Runs[0].ExitCode)
	require.Equal(t, map[string]string{
		"github.com/unkeyed/unkey/tools/testevidence/testdata/witness": "pass",
	}, evidence.Runs[0].Packages)
	require.Equal(t, map[string][]string{
		"github.com/unkeyed/unkey/tools/testevidence/testdata/witness/TestWitness": {"pass"},
	}, evidence.Runs[0].Tests)
}
