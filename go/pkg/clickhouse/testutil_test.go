package clickhouse_test

import (
	"math"
	"sort"
)

// taken from standard library
// https://cs.opensource.google/go/x/perf/+/2f7363a0:internal/stats/sample.go;l=237
func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0.0
	}

	sort.Float64s(data)

	if p <= 0 {

		return data[0]
	} else if p >= 1 {
		return data[len(data)-1]
	}

	N := float64(len(data))
	n := 1/3.0 + p*(N+1/3.0)
	kf, frac := math.Modf(n)
	k := int(kf)
	if k <= 0 {
		return data[0]
	} else if k >= len(data) {
		return data[len(data)-1]
	}
	return data[k-1] + frac*(data[k]-data[k-1])

}

// calculateAverage calculates the average from a slice of float64 values
func calculateAverage(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}

	var sum float64
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}
