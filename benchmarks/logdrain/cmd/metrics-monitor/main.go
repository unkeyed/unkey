// Package main monitors real logdrain Prometheus metrics during performance testing.
// Connects to the actual logdrain service in dev k8s and tracks throughput metrics.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	metricsURL = flag.String("metrics-url", "http://localhost:9402/metrics", "Logdrain metrics endpoint")
	duration   = flag.Duration("duration", 5*time.Minute, "How long to monitor")
	interval   = flag.Duration("interval", 5*time.Second, "Metrics scraping interval")
)

// MetricsMonitor tracks logdrain performance metrics
type MetricsMonitor struct {
	client     *http.Client
	url        string
	startTime  time.Time
	samples    []MetricsSample
}

// MetricsSample represents metrics at a point in time
type MetricsSample struct {
	Timestamp              time.Time
	ActiveGroups           float64
	TickDuration           float64
	ClickHouseQueries      float64
	ClickHouseErrors       float64
	RecordsDelivered       float64
	ProviderErrors         float64
	CursorAdvanceLatency   float64
	MemoryUsageBytes       float64
	GroupsSkipped          float64
}

func main() {
	flag.Parse()
	
	log.Printf("📊 Starting logdrain metrics monitoring...")
	log.Printf("   • Metrics URL: %s", *metricsURL)
	log.Printf("   • Duration: %v", *duration)
	log.Printf("   • Interval: %v", *interval)

	monitor := NewMetricsMonitor(*metricsURL)
	
	log.Printf("🔍 Testing connection to metrics endpoint...")
	if err := monitor.testConnection(); err != nil {
		log.Fatalf("❌ Failed to connect to metrics endpoint: %v", err)
		log.Printf("💡 Make sure logdrain is running with: kubectl port-forward svc/logdrain 9402:9402 -n unkey")
	}
	log.Printf("✅ Connected to logdrain metrics!")

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	log.Printf("\n🚀 Starting metrics collection for %v...", *duration)
	monitor.start(ctx)
	
	log.Printf("\n📈 Analyzing results...")
	monitor.generateReport()
	
	log.Printf("\n✅ Metrics monitoring completed!")
}

func NewMetricsMonitor(url string) *MetricsMonitor {
	return &MetricsMonitor{
		client:    &http.Client{Timeout: 10 * time.Second},
		url:       url,
		startTime: time.Now(),
		samples:   make([]MetricsSample, 0),
	}
}

func (m *MetricsMonitor) testConnection() error {
	resp, err := m.client.Get(m.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}
	
	return nil
}

func (m *MetricsMonitor) start(ctx context.Context) {
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Take initial sample
	if err := m.takeSample(); err != nil {
		log.Printf("⚠️  Failed to take initial sample: %v", err)
	} else {
		log.Printf("📊 Baseline metrics captured")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.takeSample(); err != nil {
				log.Printf("⚠️  Failed to take sample: %v", err)
				continue
			}
			
			if len(m.samples) > 1 {
				current := m.samples[len(m.samples)-1]
				previous := m.samples[len(m.samples)-2]
				m.logProgress(current, previous)
			}
		}
	}
}

func (m *MetricsMonitor) takeSample() error {
	resp, err := m.client.Get(m.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	metrics, err := m.parseMetrics(resp.Body)
	if err != nil {
		return err
	}
	
	sample := MetricsSample{
		Timestamp:              time.Now(),
		ActiveGroups:           metrics["logdrain_coordinator_active_groups"],
		TickDuration:           metrics["logdrain_coordinator_tick_duration_seconds"],
		ClickHouseQueries:      metrics["logdrain_clickhouse_query_duration_seconds_count"],
		ClickHouseErrors:       metrics["logdrain_clickhouse_query_errors_total"],
		RecordsDelivered:       metrics["logdrain_provider_records_delivered_total"],
		ProviderErrors:         metrics["logdrain_provider_errors_total"],
		CursorAdvanceLatency:   metrics["logdrain_cursor_advance_latency_seconds"],
		MemoryUsageBytes:       metrics["logdrain_memory_usage_bytes"],
		GroupsSkipped:          metrics["logdrain_coordinator_groups_skipped_limit_total"],
	}
	
	m.samples = append(m.samples, sample)
	return nil
}

func (m *MetricsMonitor) parseMetrics(body io.Reader) (map[string]float64, error) {
	scanner := bufio.NewScanner(body)
	metrics := make(map[string]float64)
	
	// Regex to parse Prometheus metric lines
	metricRegex := regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*)(\{[^}]*\})?\s+([0-9.-]+)`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		
		matches := metricRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		
		metricName := matches[1]
		valueStr := matches[3]
		
		// Only capture logdrain metrics
		if !strings.HasPrefix(metricName, "logdrain_") {
			continue
		}
		
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}
		
		// Store the metric (last value wins for metrics with labels)
		metrics[metricName] = value
	}
	
	return metrics, scanner.Err()
}

func (m *MetricsMonitor) logProgress(current, previous MetricsSample) {
	duration := current.Timestamp.Sub(previous.Timestamp).Seconds()
	
	// Calculate rates
	queryRate := (current.ClickHouseQueries - previous.ClickHouseQueries) / duration
	recordRate := (current.RecordsDelivered - previous.RecordsDelivered) / duration
	errorRate := (current.ProviderErrors - previous.ProviderErrors) / duration
	
	elapsed := current.Timestamp.Sub(m.startTime)
	
	log.Printf("⏱️  [%v] Active: %.0f groups | Queries: %.1f/s | Records: %.1f/s | Errors: %.1f/s | Tick: %.0fms",
		elapsed.Round(time.Second),
		current.ActiveGroups,
		queryRate,
		recordRate,
		errorRate,
		current.TickDuration*1000,
	)
}

func (m *MetricsMonitor) generateReport() {
	if len(m.samples) < 2 {
		log.Printf("⚠️  Not enough samples for meaningful analysis")
		return
	}

	first := m.samples[0]
	last := m.samples[len(m.samples)-1]
	totalDuration := last.Timestamp.Sub(first.Timestamp).Seconds()

	// Calculate totals and averages
	totalQueries := last.ClickHouseQueries - first.ClickHouseQueries
	totalRecords := last.RecordsDelivered - first.RecordsDelivered  
	totalErrors := last.ProviderErrors - first.ProviderErrors
	
	avgQueryRate := totalQueries / totalDuration
	avgRecordRate := totalRecords / totalDuration
	avgErrorRate := totalErrors / totalDuration
	
	// Calculate averages for gauge metrics
	var avgActiveGroups, avgTickDuration, avgMemoryUsage float64
	for _, sample := range m.samples {
		avgActiveGroups += sample.ActiveGroups
		avgTickDuration += sample.TickDuration
		avgMemoryUsage += sample.MemoryUsageBytes
	}
	avgActiveGroups /= float64(len(m.samples))
	avgTickDuration /= float64(len(m.samples))
	avgMemoryUsage /= float64(len(m.samples))

	log.Printf("🎯 Performance Report (%.1f minutes):", totalDuration/60)
	log.Printf("╔═══════════════════════════════════════════════════════╗")
	log.Printf("║                     THROUGHPUT                        ║")
	log.Printf("╠═══════════════════════════════════════════════════════╣")
	log.Printf("║ • Records processed:     %10.0f total          ║", totalRecords)
	log.Printf("║ • Records/second:        %10.1f avg            ║", avgRecordRate)
	log.Printf("║ • ClickHouse queries:    %10.0f total          ║", totalQueries)
	log.Printf("║ • Queries/second:        %10.1f avg            ║", avgQueryRate)
	log.Printf("╠═══════════════════════════════════════════════════════╣")
	log.Printf("║                     PERFORMANCE                       ║")
	log.Printf("╠═══════════════════════════════════════════════════════╣")
	log.Printf("║ • Avg active groups:     %10.1f                ║", avgActiveGroups)
	log.Printf("║ • Avg tick duration:     %10.0f ms             ║", avgTickDuration*1000)
	log.Printf("║ • Avg memory usage:      %10.1f MB             ║", avgMemoryUsage/1024/1024)
	log.Printf("║ • Groups skipped (limit): %9.0f total          ║", last.GroupsSkipped)
	log.Printf("╠═══════════════════════════════════════════════════════╣")
	log.Printf("║                      RELIABILITY                      ║")
	log.Printf("╠═══════════════════════════════════════════════════════╣")
	log.Printf("║ • Provider errors:       %10.0f total          ║", totalErrors)
	log.Printf("║ • Error rate:            %10.3f errors/sec     ║", avgErrorRate)
	log.Printf("║ • Success rate:          %10.1f%%               ║", (1.0-totalErrors/totalRecords)*100)
	log.Printf("╚═══════════════════════════════════════════════════════╝")

	// Performance assessment
	log.Printf("\n📊 Performance Assessment:")
	
	if avgRecordRate >= 1000 {
		log.Printf("✅ EXCELLENT throughput: %.0f records/sec (target: 1000+)", avgRecordRate)
	} else if avgRecordRate >= 500 {
		log.Printf("✅ GOOD throughput: %.0f records/sec", avgRecordRate)
	} else {
		log.Printf("⚠️  LOW throughput: %.0f records/sec (consider optimization)", avgRecordRate)
	}
	
	if avgTickDuration*1000 <= 5000 {
		log.Printf("✅ EXCELLENT tick duration: %.0fms (target: <5000ms)", avgTickDuration*1000)
	} else {
		log.Printf("⚠️  HIGH tick duration: %.0fms (may need sharding)", avgTickDuration*1000)
	}
	
	if avgErrorRate <= 0.1 {
		log.Printf("✅ EXCELLENT reliability: %.3f errors/sec", avgErrorRate)
	} else {
		log.Printf("⚠️  HIGH error rate: %.3f errors/sec (investigate providers)", avgErrorRate)
	}
	
	if avgMemoryUsage/1024/1024 <= 500 {
		log.Printf("✅ GOOD memory usage: %.1fMB (target: <500MB)", avgMemoryUsage/1024/1024)
	} else {
		log.Printf("⚠️  HIGH memory usage: %.1fMB (may need limits)", avgMemoryUsage/1024/1024)
	}

	if last.GroupsSkipped > 0 {
		log.Printf("⚠️  Group limits triggered: %.0f groups skipped", last.GroupsSkipped)
		log.Printf("💡 Consider increasing max_groups_per_shard or adding shards")
	}

	// Recommendations
	log.Printf("\n💡 Recommendations:")
	if avgRecordRate < 500 {
		log.Printf("   • Consider increasing batch_size or reducing poll_interval")
		log.Printf("   • Check ClickHouse query performance")
		log.Printf("   • Verify provider endpoints are responsive")
	}
	if avgTickDuration*1000 > 2000 {
		log.Printf("   • Consider enabling sharding (increase shard_count)")
		log.Printf("   • Reduce max_batch_records if memory pressure high")
	}
	if avgErrorRate > 0.1 {
		log.Printf("   • Check provider credentials and endpoints")
		log.Printf("   • Verify network connectivity to providers")
		log.Printf("   • Review provider rate limits")
	}
}
