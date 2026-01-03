package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
)

var verificationsCmd = &cli.Command{
	Name:  "verifications",
	Usage: "Seed key verification events",
	Flags: []cli.Flag{
		cli.String("api-id", "API ID to use for seeding", cli.Required()),
		cli.Int("num-verifications", "Number of verifications to generate", cli.Required()),
		cli.Float("unique-keys-percent", "Percentage of verifications that are unique keys (0.0-100.0)", cli.Default(1.0)),
		cli.Float("keys-with-identity-percent", "Percentage of keys that have an identity attached (0.0-100.0)", cli.Default(30.0)),
		cli.Float("identity-usage-percent", "Percentage chance to use identity in verification if key has one (0.0-100.0)", cli.Default(90.0)),
		cli.Int("days-back", "Number of days back to generate data", cli.Default(30)),
		cli.Int("days-forward", "Number of days forward to generate data", cli.Default(30)),
		cli.String("clickhouse-url", "ClickHouse URL", cli.Default("clickhouse://default:password@127.0.0.1:9000")),
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY"), cli.Required()),
	},
	Action: seedVerifications,
}

const chunkSize = 50_000

func seedVerifications(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Connect to ClickHouse
	ch, err := clickhouse.New(clickhouse.Config{
		URL:    cmd.String("clickhouse-url"),
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Create key service for proper key generation
	keyService, err := keys.New(keys.Config{
		Logger:       logger,
		DB:           database,
		RateLimiter:  nil,
		RBAC:         nil,
		Clickhouse:   ch,
		Region:       "test",
		UsageLimiter: nil,
		KeyCache:     nil,
	})
	if err != nil {
		return fmt.Errorf("failed to create key service: %w", err)
	}

	// Calculate derived values
	numVerifications := cmd.RequireInt("num-verifications")
	uniqueKeysPercent := cmd.Float("unique-keys-percent")
	keysWithIdentityPercent := cmd.Float("keys-with-identity-percent")
	identityUsagePercent := cmd.Float("identity-usage-percent")

	// Calculate number of unique keys based on percentage
	numKeys := max(1, int(float64(numVerifications)*(uniqueKeysPercent/100.0)))

	// Calculate number of identities based on percentage of keys
	numIdentities := int(float64(numKeys) * (keysWithIdentityPercent / 100.0))

	seeder := &Seeder{
		apiID:                   cmd.RequireString("api-id"),
		numKeys:                 numKeys,
		numIdentities:           numIdentities,
		numVerifications:        numVerifications,
		keysWithIdentityPercent: keysWithIdentityPercent,
		identityUsagePercent:    identityUsagePercent,
		daysBack:                cmd.Int("days-back"),
		daysForward:             cmd.Int("days-forward"),
		db:                      database,
		clickhouse:              ch,
		keyService:              keyService,
	}

	return seeder.Seed(ctx)
}

type Seeder struct {
	apiID                   string
	numKeys                 int
	numIdentities           int
	numVerifications        int
	keysWithIdentityPercent float64
	identityUsagePercent    float64
	daysBack                int
	daysForward             int
	db                      db.Database
	clickhouse              clickhouse.ClickHouse
	keyService              keys.KeyService
}

type Key struct {
	ID         string
	KeyAuthID  string
	Hash       string
	Start      string
	Name       string
	Enabled    bool
	IdentityID string // Empty string if no identity attached
	ExternalID string // Empty string if no external ID attached
}

type Identity struct {
	ID         string
	ExternalID string
}

func (s *Seeder) Seed(ctx context.Context) error {
	log.Printf("Starting seed for API: %s", s.apiID)

	// 1. Get API details including workspace_id
	log.Printf("Fetching API details...")
	workspaceID, keyAuthID, prefix, err := s.getAPIDetails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get API details: %w", err)
	}
	log.Printf("  Using workspace %s, API %s with keyAuth %s (prefix: %s)", workspaceID, s.apiID, keyAuthID, prefix)

	// 2. Create Identities and Keys with batched transactions
	var identities []Identity
	var allKeys []Key

	// Create identities first (if needed)
	if s.numIdentities > 0 {
		log.Printf("Creating %d identities...", s.numIdentities)
		identities, err = s.createIdentitiesBatched(ctx, workspaceID)
		if err != nil {
			return fmt.Errorf("failed to create identities: %w", err)
		}
	} else {
		log.Printf("No identities will be created (0 keys will have identities)")
	}

	// Create keys and attach identities to some of them
	log.Printf("Creating %d keys (%.1f%% will have identities)...", s.numKeys, s.keysWithIdentityPercent)
	allKeys, err = s.createKeysBatched(ctx, workspaceID, keyAuthID, prefix, identities)
	if err != nil {
		return fmt.Errorf("failed to create keys: %w", err)
	}

	// 4. Generate and insert verifications
	log.Printf("Generating %d verifications...", s.numVerifications)
	if err := s.generateVerifications(ctx, workspaceID, allKeys, keyAuthID); err != nil {
		return fmt.Errorf("failed to generate verifications: %w", err)
	}

	log.Println("Seeding completed successfully!")
	return nil
}

func (s *Seeder) getAPIDetails(ctx context.Context) (workspaceID, keyAuthID, prefix string, err error) {
	// Fetch API from database
	api, err := db.Query.FindApiByID(ctx, s.db.RO(), s.apiID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to find API: %w", err)
	}

	workspaceID = api.WorkspaceID

	if !api.KeyAuthID.Valid {
		return "", "", "", fmt.Errorf("API %s does not have key authentication enabled", s.apiID)
	}

	keyAuthID = api.KeyAuthID.String

	// Fetch keyAuth to get the prefix
	keyAuth, err := db.Query.GetKeyAuthByID(ctx, s.db.RO(), keyAuthID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get keyAuth: %w", err)
	}

	if keyAuth.DefaultPrefix.Valid {
		prefix = keyAuth.DefaultPrefix.String
	} else {
		prefix = "key" // fallback prefix
	}

	return workspaceID, keyAuthID, prefix, nil
}

func (s *Seeder) createKeysBatched(ctx context.Context, workspaceID, keyAuthID, prefix string, identities []Identity) ([]Key, error) {
	allKeys := make([]Key, s.numKeys)
	keyParams := make([]db.InsertKeyParams, s.numKeys)

	environments := []string{"development", "staging", "production", "test"}
	keyNames := []string{"Backend Service", "Frontend App", "Mobile Client", "Admin Dashboard", "API Integration", "Test Key"}

	// Calculate how many keys should have identities
	keysWithIdentityCount := int(float64(s.numKeys) * (s.keysWithIdentityPercent / 100.0))

	now := time.Now().UnixMilli()

	for i := range s.numKeys {
		// Use the key service to create a proper key with real hash
		keyResult, err := s.keyService.CreateKey(ctx, keys.CreateKeyRequest{
			Prefix:     prefix,
			ByteLength: 16,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create key: %w", err)
		}

		keyID := uid.New(uid.Prefix(prefix))

		// Some keys are disabled (5%)
		enabled := rand.Float64() > 0.05 // 95% enabled

		name := fmt.Sprintf("%s - %s",
			keyNames[rand.IntN(len(keyNames))],
			environments[rand.IntN(len(environments))],
		)

		// Determine if this key should have an identity attached
		var identityID string
		var externalID string
		if i < keysWithIdentityCount && len(identities) > 0 {
			// Attach an identity to this key
			identity := identities[rand.IntN(len(identities))]
			identityID = identity.ID
			externalID = identity.ExternalID
		}

		key := Key{
			ID:         keyID,
			KeyAuthID:  keyAuthID,
			Hash:       keyResult.Hash,
			Start:      keyResult.Start,
			Name:       name,
			Enabled:    enabled,
			IdentityID: identityID,
			ExternalID: externalID,
		}

		// Prepare key params for bulk insert
		var identityIDParam sql.NullString
		if identityID != "" {
			identityIDParam = sql.NullString{String: identityID, Valid: true}
		}

		keyParams[i] = db.InsertKeyParams{
			ID:                 keyID,
			KeySpaceID:         keyAuthID,
			Hash:               keyResult.Hash,
			Start:              keyResult.Start,
			WorkspaceID:        workspaceID,
			Name:               sql.NullString{String: name, Valid: true},
			IdentityID:         identityIDParam,
			CreatedAtM:         now,
			Enabled:            enabled,
			ForWorkspaceID:     sql.NullString{},
			Meta:               sql.NullString{},
			Expires:            sql.NullTime{},
			RemainingRequests:  sql.NullInt32{},
			RefillDay:          sql.NullInt16{},
			RefillAmount:       sql.NullInt32{},
			PendingMigrationID: sql.NullString{Valid: false, String: ""},
		}

		allKeys[i] = key

		// Log progress periodically
		if (i+1)%1000 == 0 || i == s.numKeys-1 {
			log.Printf("  Prepared %d/%d keys", i+1, s.numKeys)
		}
	}

	chunks := array.Chunk(keyParams, chunkSize)

	// Commit every 5 chunks (250k rows per transaction)
	batchSize := 5
	for i := 0; i < len(chunks); i += batchSize {
		err := db.Tx(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
			end := min(i+batchSize, len(chunks))
			for j := i; j < end; j++ {
				log.Printf("  Inserting %d keys... chunk %d/%d", len(chunks[j]), j+1, len(chunks))
				if err := db.BulkQuery.InsertKeys(ctx, tx, chunks[j]); err != nil {
					return fmt.Errorf("failed to bulk insert keys: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	keysWithIdentity := 0
	for _, k := range allKeys {
		if k.IdentityID != "" {
			keysWithIdentity++
		}
	}

	log.Printf("  Created %d total keys (%d with identities attached)", s.numKeys, keysWithIdentity)

	return allKeys, nil
}

func (s *Seeder) createIdentitiesBatched(ctx context.Context, workspaceID string) ([]Identity, error) {
	identities := make([]Identity, s.numIdentities)
	identityParams := make([]db.InsertIdentityParams, s.numIdentities)

	domains := []string{"example.com", "test.com", "demo.org", "app.io"}
	now := time.Now().UnixMilli()
	for i := range s.numIdentities {
		identityID := uid.New("id")
		// Generate unique external ID using the identity ID to guarantee uniqueness
		externalID := fmt.Sprintf("user-%s@%s",
			uid.New("ex"),
			array.Random(domains),
		)

		identity := Identity{
			ID:         identityID,
			ExternalID: externalID,
		}

		// Prepare identity params for bulk insert
		identityParams[i] = db.InsertIdentityParams{
			ID:          identityID,
			ExternalID:  externalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   now,
			Meta:        []byte(`{}`),
		}

		identities[i] = identity
	}

	chunks := array.Chunk(identityParams, chunkSize)

	// Commit every 5 chunks (250k rows per transaction)
	batchSize := 5
	for i := 0; i < len(chunks); i += batchSize {
		err := db.Tx(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
			end := min(i+batchSize, len(chunks))
			for j := i; j < end; j++ {
				log.Printf("  Inserting %d identities... chunk %d/%d", len(chunks[j]), j+1, len(chunks))
				if err := db.BulkQuery.InsertIdentities(ctx, tx, chunks[j]); err != nil {
					return fmt.Errorf("failed to bulk insert identities: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	log.Printf("  Created %d identities", s.numIdentities)

	return identities, nil
}

func (s *Seeder) generateVerifications(_ context.Context, workspaceID string, keys []Key, keyAuthID string) error {
	startTime := time.Now().AddDate(0, 0, -s.daysBack)
	endTime := time.Now().AddDate(0, 0, s.daysForward)

	outcomes := []struct {
		name   string
		weight float64
	}{
		{"VALID", 0.85},
		{"RATE_LIMITED", 0.05},
		{"EXPIRED", 0.03},
		{"DISABLED", 0.02},
		{"FORBIDDEN", 0.02},
		{"USAGE_EXCEEDED", 0.02},
		{"INSUFFICIENT_PERMISSIONS", 0.01},
	}

	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "sa-east-1"}
	tagOptions := []string{"api", "web", "mobile", "server", "client", "frontend", "backend", "test", "prod"}

	// Use normal distribution to make some keys "hot"
	mean := float64(len(keys)) * 0.2
	stdDev := float64(len(keys)) / 5.0

	for i := range s.numVerifications {
		// Generate timestamp with bias towards recent data
		timeFraction := rand.Float64()
		timeFraction = math.Pow(timeFraction, 0.5)
		timestamp := startTime.Add(time.Duration(timeFraction * float64(endTime.Sub(startTime))))

		// Select key with normal distribution (creates hot keys)
		keyIndex := int(normalDistribution(mean, stdDev, 0, float64(len(keys)-1)))
		key := keys[keyIndex]

		// Select outcome
		outcomeRand := rand.Float64()
		var outcome string
		cumulative := 0.0
		for _, o := range outcomes {
			cumulative += o.weight
			if outcomeRand < cumulative {
				outcome = o.name
				break
			}
		}

		// Bias outcome based on key properties
		if !key.Enabled && rand.Float64() < 0.6 {
			outcome = "DISABLED"
		}

		// Determine identity for this verification
		var identityID string
		var externalID string
		if key.IdentityID != "" {
			// Key has an identity - use it X% of the time (default 90%)
			if rand.Float64()*100 < s.identityUsagePercent {
				identityID = key.IdentityID
				externalID = key.ExternalID
			}
			// Otherwise leave it blank (simulating key used before identity was attached)
		}
		// If key doesn't have identity, identityID stays empty

		// Generate tags (0-2 tags)
		tagCount := rand.IntN(3)
		tags := make([]string, 0, tagCount)
		for range tagCount {
			tag := tagOptions[rand.IntN(len(tagOptions))]
			if !slices.Contains(tags, tag) {
				tags = append(tags, tag)
			}
		}

		// random latency
		latency := rand.ExpFloat64()*50 + 1 // 1-100ms base range
		if rand.Float64() < 0.1 {           // 10% chance of high latency
			latency += rand.Float64() * 400 // Up to 500ms
		}

		// random credit spent between 0 and 50 but 70% should be 0
		credit := rand.Int64N(51)
		if rand.Float64() < 0.7 {
			credit = 0
		}

		// Use BufferKeyVerification to let the clickhouse client batch automatically
		s.clickhouse.BufferKeyVerification(schema.KeyVerification{
			RequestID:    uid.New("req"),
			Time:         timestamp.UnixMilli(),
			WorkspaceID:  workspaceID,
			KeySpaceID:   keyAuthID,
			KeyID:        key.ID,
			Region:       regions[rand.IntN(len(regions))],
			Tags:         tags,
			Outcome:      outcome,
			IdentityID:   identityID,
			ExternalID:   externalID,
			Latency:      latency,
			SpentCredits: credit,
		})

		// Log progress periodically
		if (i+1)%10000 == 0 {
			log.Printf("  Buffered %d/%d verifications", i+1, s.numVerifications)
		}
	}

	log.Printf("  Buffered all %d verifications, waiting for flush...", s.numVerifications)

	err := s.clickhouse.Close()
	if err != nil {
		return fmt.Errorf("failed to close clickhouse: %w", err)
	}

	log.Printf("  All verifications sent to ClickHouse")
	return nil
}

// normalDistribution returns a random value from a normal distribution
func normalDistribution(mean, stdDev, minimum, maximum float64) float64 {
	// If minimum == maximum, just return that value to avoid infinite loop
	if minimum >= maximum {
		return minimum
	}

	for {
		// Box-Muller transform
		u1 := rand.Float64()
		u2 := rand.Float64()
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)

		value := mean + z*stdDev

		if value >= minimum && value <= maximum {
			return value
		}
	}
}
