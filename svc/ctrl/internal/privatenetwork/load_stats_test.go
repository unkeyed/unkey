package privatenetwork

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestConnectionLoadCompare compares two TestConnectionLoad reports and
// prints, for every query and dataset, the change in median latency with a
// 95% bootstrap interval and a two-sided Mann-Whitney U p-value.
//
//	UNKEY_CONNECTION_LOAD_BEFORE=/tmp/before.json UNKEY_CONNECTION_LOAD_AFTER=/tmp/after.json \
//	mise exec -- go test -run TestConnectionLoadCompare -v ./svc/ctrl/internal/privatenetwork/
func TestConnectionLoadCompare(t *testing.T) {
	beforePath, afterPath := os.Getenv("UNKEY_CONNECTION_LOAD_BEFORE"), os.Getenv("UNKEY_CONNECTION_LOAD_AFTER")
	if beforePath == "" || afterPath == "" {
		t.Skip("set UNKEY_CONNECTION_LOAD_BEFORE and UNKEY_CONNECTION_LOAD_AFTER to compare two load reports")
	}
	before, after := readLoadReport(t, beforePath), readLoadReport(t, afterPath)
	afterByKey := make(map[string]loadResult, len(after.Results))
	for _, result := range after.Results {
		afterByKey[result.key()] = result
	}

	var table strings.Builder
	fmt.Fprintf(&table, "%-44s %-13s %10s %10s %8s %17s %9s %12s\n", "query", "dataset", "before", "after", "ratio", "ratio 95% CI", "p", "rows read")
	for _, b := range before.Results {
		a, ok := afterByKey[b.key()]
		if !ok || len(b.SamplesNs) == 0 || len(a.SamplesNs) == 0 {
			continue
		}
		ratio, low, high := loadRatioInterval(b.SamplesNs, a.SamplesNs)
		fmt.Fprintf(&table, "%-44s %-13s %10s %10s %8.3f [%6.3f, %6.3f] %9.2g %5d→%-6d\n",
			b.Op, fmt.Sprintf("%s-x%d", b.Axis, b.Scale),
			time.Duration(loadMedian(b.SamplesNs)).Round(time.Microsecond), time.Duration(loadMedian(a.SamplesNs)).Round(time.Microsecond),
			ratio, low, high, loadMannWhitney(b.SamplesNs, a.SamplesNs), b.RowsRead, a.RowsRead)
	}
	t.Log("\n" + table.String())
}

func readLoadReport(t *testing.T, path string) loadReport {
	t.Helper()
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	var report loadReport
	require.NoError(t, json.Unmarshal(encoded, &report))
	return report
}

func (r loadResult) key() string {
	return fmt.Sprintf("%s/%s/%d", r.Op, r.Axis, r.Scale)
}

func loadMedian(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := slices.Sorted(slices.Values(values))
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

func loadRatioInterval(before, after []int64) (ratio, low, high float64) {
	const resamples = 2000
	random := rand.New(rand.NewPCG(5, 6))
	resample := func(values []int64) int64 {
		drawn := make([]int64, len(values))
		for i := range drawn {
			drawn[i] = values[random.IntN(len(values))]
		}
		return loadMedian(drawn)
	}
	ratios := make([]float64, resamples)
	for i := range ratios {
		ratios[i] = float64(resample(after)) / float64(resample(before))
	}
	slices.Sort(ratios)
	return float64(loadMedian(after)) / float64(loadMedian(before)), ratios[resamples*25/1000], ratios[resamples*975/1000]
}

func loadMannWhitney(a, b []int64) float64 {
	type ranked struct {
		value int64
		first bool
	}
	all := make([]ranked, 0, len(a)+len(b))
	for _, v := range a {
		all = append(all, ranked{v, true})
	}
	for _, v := range b {
		all = append(all, ranked{v, false})
	}
	slices.SortFunc(all, func(x, y ranked) int { return cmp.Compare(x.value, y.value) })

	n1, n2, n := float64(len(a)), float64(len(b)), float64(len(all))
	var rankSum, ties float64
	for i := 0; i < len(all); {
		j := i
		for j < len(all) && all[j].value == all[i].value {
			j++
		}
		rank := float64(i+j+1) / 2
		for k := i; k < j; k++ {
			if all[k].first {
				rankSum += rank
			}
		}
		count := float64(j - i)
		ties += count*count*count - count
		i = j
	}
	u := rankSum - n1*(n1+1)/2
	sigma := math.Sqrt(n1 * n2 / 12 * ((n + 1) - ties/(n*(n-1))))
	if sigma == 0 {
		return 1
	}
	z := (u - n1*n2/2) / sigma
	return math.Erfc(math.Abs(z) / math.Sqrt2)
}
