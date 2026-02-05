package seed

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

var sentinelCmd = &cli.Command{
	Name:        "sentinel",
	Usage:       "Seed sentinel request events for last 12 hours",
	Description: "Generates realistic sentinel request data for testing RPS and latency charts",
	Flags: []cli.Flag{
		cli.String("deployment-id", "Deployment ID to seed data for (if not provided, uses first available deployment)", cli.Default("")),
		cli.Int("num-requests", "Number of requests to generate", cli.Default(5_000_000)),
		cli.String("clickhouse-url", "ClickHouse URL", cli.Default("clickhouse://default:password@127.0.0.1:9000")),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
	},
	Action: seedSentinel,
}

func seedSentinel(ctx context.Context, cmd *cli.Command) error {
	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Connect to ClickHouse
	ch, err := clickhouse.New(clickhouse.Config{
		URL: cmd.String("clickhouse-url"),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Get or find deployment ID
	deploymentID := cmd.String("deployment-id")
	if deploymentID == "" {
		deploymentID, err = findFirstDeployment(ctx, database)
		if err != nil {
			return fmt.Errorf("no deployment ID provided and couldn't find one: %w", err)
		}
		log.Printf("Using deployment: %s", deploymentID)
	}

	// Create seeder and run
	seeder := &SentinelSeeder{
		deploymentID: deploymentID,
		numRequests:  cmd.RequireInt("num-requests"),
		db:           database,
		clickhouse:   ch,
	}

	return seeder.Seed(ctx)
}

type SentinelSeeder struct {
	deploymentID string
	numRequests  int
	db           db.Database
	clickhouse   clickhouse.ClickHouse
}

func (s *SentinelSeeder) Seed(ctx context.Context) error {
	log.Printf("Starting sentinel seed for deployment: %s", s.deploymentID)

	// 1. Get deployment details
	deployment, domain, err := s.getDeploymentDetails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// 2. Generate placeholder instance/sentinel IDs
	instanceIDs, sentinelIDs := s.generatePlaceholderIDs()

	// 3. Generate and buffer requests (like verifications.go does)
	if err := s.generateRequests(ctx, deployment, domain, instanceIDs, sentinelIDs); err != nil {
		return fmt.Errorf("failed to generate requests: %w", err)
	}

	log.Printf("Successfully seeded %d sentinel requests", s.numRequests)
	return nil
}

func findFirstDeployment(ctx context.Context, database db.Database) (string, error) {
	log.Printf("No deployment ID provided, finding first available deployment...")
	row := database.RO().QueryRowContext(ctx, "SELECT id FROM deployments ORDER BY created_at DESC LIMIT 1")
	var deploymentID string
	err := row.Scan(&deploymentID)
	if err != nil {
		return "", fmt.Errorf("no deployments found - please create a deployment first or specify --deployment-id: %w", err)
	}
	return deploymentID, nil
}

func (s *SentinelSeeder) getDeploymentDetails(ctx context.Context) (db.Deployment, string, error) {
	log.Printf("Fetching deployment details...")

	// Fetch deployment (like verifications.go line 183)
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), s.deploymentID)
	if err != nil {
		return db.Deployment{}, "", fmt.Errorf("deployment not found: %w", err)
	}

	log.Printf("  Using workspace %s, project %s, environment %s",
		deployment.WorkspaceID, deployment.ProjectID, deployment.EnvironmentID)

	// Get domain from frontline routes
	routes, err := db.Query.FindFrontlineRoutesByDeploymentID(ctx, s.db.RO(), s.deploymentID)
	if err != nil {
		return db.Deployment{}, "", fmt.Errorf("failed to get routes: %w", err)
	}

	domain := "example.com" // fallback
	if len(routes) > 0 {
		domain = routes[0].FullyQualifiedDomainName
	}
	log.Printf("  Using domain: %s", domain)

	return deployment, domain, nil
}

func (s *SentinelSeeder) generatePlaceholderIDs() ([]string, []string) {
	instanceIDs := []string{uid.New("inst"), uid.New("inst"), uid.New("inst"), uid.New("inst")}
	sentinelIDs := []string{uid.New("sent"), uid.New("sent"), uid.New("sent")}

	log.Printf("  Using %d instances and %d sentinels", len(instanceIDs), len(sentinelIDs))
	return instanceIDs, sentinelIDs
}

func (s *SentinelSeeder) generateRequests(
	_ context.Context,
	deployment db.Deployment,
	domain string,
	instanceIDs, sentinelIDs []string,
) error {
	endTime := time.Now()
	startTime := endTime.Add(-12 * time.Hour)

	// Pre-define distributions (outside loop for efficiency)
	regions := []struct {
		name   string
		weight float64
	}{
		{"us-east-1", 0.50},
		{"eu-west-1", 0.20},
		{"ap-southeast-1", 0.15},
		{"us-west-2", 0.10},
		{"eu-central-1", 0.05},
	}

	methods := []struct {
		name   string
		weight float64
	}{
		{"GET", 0.60},
		{"POST", 0.25},
		{"PUT", 0.10},
		{"DELETE", 0.05},
	}

	statuses := []struct {
		code   int32
		weight float64
	}{
		{200, 0.90},
		{404, 0.05},
		{500, 0.03},
		{429, 0.02},
	}

	paths := []string{
		"/api/v1/users",
		"/api/v1/products",
		"/api/v1/orders",
		"/api/v1/auth",
		"/api/v1/search",
		"/health",
		"/metrics",
	}

	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		"curl/7.68.0",
		"PostmanRuntime/7.26.8",
	}

	log.Printf("Generating and buffering %d requests...", s.numRequests)

	// Allocate 40% of requests to last 15 minutes for good RPS visibility
	last15MinThreshold := endTime.Add(-15 * time.Minute)
	numRequestsLast15Min := int(float64(s.numRequests) * 0.40)
	numRequestsOlder := s.numRequests - numRequestsLast15Min

	log.Printf("  Allocating %d requests to last 15 min, %d to earlier time", numRequestsLast15Min, numRequestsOlder)

	// CRITICAL: Generate and buffer in SAME loop (like verifications.go line 410-486)
	for i := range s.numRequests {
		// Generate ONE request with better time distribution
		var timestamp time.Time
		if i < numRequestsLast15Min {
			// Last 15 minutes: distribute evenly for consistent RPS
			timeFraction := rand.Float64()
			timestamp = last15MinThreshold.Add(time.Duration(timeFraction * float64(15*time.Minute)))
		} else {
			// Older data: use weighted distribution (favor more recent within this range)
			timestamp = generateWeightedTime(startTime, last15MinThreshold)
		}

		instanceLatency := generateLatency()
		sentinelLatency := rand.Float64()*3 + 1
		totalLatency := instanceLatency + sentinelLatency

		// Buffer it IMMEDIATELY
		s.clickhouse.BufferSentinelRequest(schema.SentinelRequest{
			RequestID:       uid.New("req"),
			Time:            timestamp.UnixMilli(),
			WorkspaceID:     deployment.WorkspaceID,
			ProjectID:       deployment.ProjectID,
			DeploymentID:    deployment.ID,
			EnvironmentID:   deployment.EnvironmentID,
			SentinelID:      sentinelIDs[rand.IntN(len(sentinelIDs))],
			InstanceID:      instanceIDs[rand.IntN(len(instanceIDs))],
			InstanceAddress: generateIP(),
			Region:          weightedSelectString(regions),
			Method:          weightedSelectString(methods),
			Host:            domain,
			Path:            paths[rand.IntN(len(paths))],
			ResponseStatus:  weightedSelectInt32(statuses),
			UserAgent:       userAgents[rand.IntN(len(userAgents))],
			IPAddress:       generateIP(),
			TotalLatency:    int64(totalLatency),
			InstanceLatency: int64(instanceLatency),
			SentinelLatency: int64(sentinelLatency),
			QueryString:     "",
			QueryParams:     make(map[string][]string),
			RequestHeaders:  []string{},
			RequestBody:     "",
			ResponseHeaders: []string{},
			ResponseBody:    "",
		})

		// Progress logging (every 10k like verifications.go line 489)
		if (i+1)%10000 == 0 {
			log.Printf("  Buffered %d/%d requests", i+1, s.numRequests)
		}
	}

	log.Printf("  Buffered all %d requests, waiting for flush...", s.numRequests)

	// Flush by closing (like verifications.go line 496)
	if err := s.clickhouse.Close(); err != nil {
		return fmt.Errorf("failed to close clickhouse: %w", err)
	}

	log.Printf("  All requests sent to ClickHouse")
	return nil
}

// generateWeightedTime creates timestamps weighted towards recent times
// Uses power distribution: t = end - duration * (1 - u^2)
func generateWeightedTime(start, end time.Time) time.Time {
	duration := end.Sub(start)
	u := rand.Float64()
	// Square the random value to weight towards recent (end) time
	offsetFraction := 1 - math.Pow(u, 2)
	offset := time.Duration(float64(duration) * offsetFraction)
	return end.Add(-offset)
}

// generateLatency creates realistic latency with proper percentile distribution
// Target: P50=15ms, P75=25ms, P90=40ms, P95=60ms, P99=150ms
func generateLatency() float64 {
	r := rand.Float64()

	// Use percentile-based distribution for realistic API latencies
	switch {
	case r < 0.50: // P0-P50: 5-15ms (fast responses)
		return 5 + rand.Float64()*10
	case r < 0.75: // P50-P75: 15-25ms (normal responses)
		return 15 + rand.Float64()*10
	case r < 0.90: // P75-P90: 25-40ms (slower responses)
		return 25 + rand.Float64()*15
	case r < 0.95: // P90-P95: 40-60ms (slow responses)
		return 40 + rand.Float64()*20
	case r < 0.99: // P95-P99: 60-150ms (very slow)
		return 60 + rand.Float64()*90
	default: // P99-P100: 150-500ms (outliers)
		return 150 + rand.Float64()*350
	}
}

// weightedSelectString selects a string item based on weights
func weightedSelectString(items []struct {
	name   string
	weight float64
},
) string {
	r := rand.Float64()
	cumulative := 0.0

	for _, item := range items {
		cumulative += item.weight
		if r < cumulative {
			return item.name
		}
	}

	// Fallback to last item
	return items[len(items)-1].name
}

// weightedSelectInt32 selects an int32 item based on weights
func weightedSelectInt32(items []struct {
	code   int32
	weight float64
},
) int32 {
	r := rand.Float64()
	cumulative := 0.0

	for _, item := range items {
		cumulative += item.weight
		if r < cumulative {
			return item.code
		}
	}

	// Fallback to last item
	return items[len(items)-1].code
}

// generateIP creates a random IPv4 address
func generateIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.IntN(256),
		rand.IntN(256),
		rand.IntN(256),
		rand.IntN(256),
	)
}
