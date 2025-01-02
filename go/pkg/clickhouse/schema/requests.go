package schema

type Sample struct {
	// Temporality of the metric. This is used to identify the type of the metric. It can be one of
	// the following values:
	// - Unspecified: This is used for gauge metrics.
	// - Cumulative: This is used for monotonic counters.
	// - Delta: This is used for counters that reset after flushing.
	//
	// LowCardinality(String) DEFAULT 'Unspecified',
	Temporality string `ch:"temporality"`

	// Name of the metric.
	//
	// LowCardinality(String),
	Name string `ch:"name"`

	// Fingerprint of the metric. This is used to identify the metric uniquely. Currently, we are
	// using the hash of the labels to generate the fingerprint.
	//
	// UInt64 CODEC(Delta(8), ZSTD(1)),
	Fingerprint uint64 `ch:"fingerprint"`

	// Timestamp in milliseconds when the metric was observed.
	//
	// Int64 CODEC(DoubleDelta, ZSTD(1)),
	UnixMilli int64 `ch:"unix_milli"`

	// Value of the metric.
	//
	// Float64 CODEC(Gorilla, ZSTD(1))
	Value float64 `ch:"value"`
}

type Metric struct {
	// Name of the metric.
	//
	// LowCardinality(String)
	MetricName string

	// Description of the metric.
	//
	// LowCardinality(String)
	Description string

	// Unit of the metric.
	//
	// For example: "s" for seconds
	//
	// LowCardinality(String)
	Unit string

	// Type: Type of the metric. One of the following values:
	// - Sum: This is used for monotonic counters.
	// - Gauge: This is used for gauge metrics.
	// - Histogram: This is used for histogram metrics.
	// - ExponentialHistogram: This is used for exponential histogram metrics.
	Type string

	// Fingerprint of the metric. This is used to identify the metric uniquely.
	// Currently, we are using the hash of the labels to generate the fingerprint.
	Fingerprint uint64

	// Labels of the metric; Stored as a JSON string.
	// The JSON string is sorted lexicographically to merge identical sets of labels using
	// ClickHouse's ReplacingMergeTree.
	Labels string
}
