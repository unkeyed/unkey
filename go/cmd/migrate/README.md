# Credits Migration Guide

This guide explains how to migrate key credits from the `keys` table to the separate `credits` table.

## Overview

The migration moves credit-related data from:
- `keys.remaining_requests`
- `keys.refill_day`
- `keys.refill_amount`
- `keys.last_refill_at`

To the `credits` table with proper relationships.

## Zero-Downtime Migration Process

### Phase 1: Deploy Dual-Write Code (v1)

Deploy application code that:
1. **Reads** from both tables (keys first, credits as fallback)
2. **Writes** to both tables simultaneously

Example pattern:
```go
// Read credits - check both places
func GetKeyCredits(keyID string) (*Credits, error) {
    // Try keys table first (backward compatibility)
    key, err := db.FindKeyByID(keyID)
    if err == nil && key.RemainingRequests != nil {
        return &Credits{
            Remaining: *key.RemainingRequests,
            RefillDay: key.RefillDay,
            // ... convert other fields
        }, nil
    }
    
    // Fall back to credits table
    return db.FindCreditsByKeyID(keyID)
}

// Write credits - update both places
func UpdateKeyCredits(keyID string, remaining int) error {
    tx := db.Begin()
    
    // Update keys table (legacy)
    err := tx.UpdateKeyRemainingRequests(keyID, remaining)
    if err != nil {
        return tx.Rollback()
    }
    
    // Update credits table (new)
    err = tx.UpsertCredits(keyID, remaining)
    if err != nil {
        return tx.Rollback()
    }
    
    return tx.Commit()
}
```

### Phase 2: Run Migration

Once v1 is deployed in all regions, run the migration:

```bash
# Dry run to preview changes
unkey migrate credits \
  --primary-dsn="mysql://user:pass@host:3306/db?parseTime=true" \
  --dry-run

# Run the actual migration
unkey migrate credits \
  --primary-dsn="mysql://user:pass@host:3306/db?parseTime=true" \
  --batch-size=1000

# Migrate specific workspace
unkey migrate credits \
  --primary-dsn="mysql://user:pass@host:3306/db?parseTime=true" \
  --workspace="ws_abc123" \
  --batch-size=500
```

### Phase 3: Deploy Read-Switch Code (v2)

Deploy code that:
1. **Reads** from credits table only
2. **Writes** to credits table only
3. Stops updating keys.remaining_requests fields

### Phase 4: Cleanup

Once v2 is stable:
1. Remove dual-write code paths
2. Optionally drop columns from keys table (keep for rollback safety initially)

## Migration Command Options

- `--dry-run`: Preview the migration without making changes
- `--batch-size`: Number of keys to process per batch (default: 500)
- `--workspace`: Migrate only a specific workspace
- `--primary-dsn`: Primary database connection string (required)
- `--readonly-dsn`: Read-only replica connection string (optional)

## Monitoring

During migration, monitor:

1. **Migration Progress**:
   ```sql
   -- Count keys with credits vs total
   SELECT 
     (SELECT COUNT(*) FROM keys WHERE remaining_requests IS NOT NULL) as keys_with_credits,
     (SELECT COUNT(*) FROM credits WHERE key_id IS NOT NULL) as migrated_credits;
   ```

2. **Data Consistency**:
   ```sql
   -- Find mismatches between tables
   SELECT k.id, k.remaining_requests, c.remaining
   FROM keys k
   LEFT JOIN credits c ON c.key_id = k.id
   WHERE k.deleted_at_m IS NULL
     AND k.remaining_requests != c.remaining;
   ```

3. **Recent Credit Updates**:
   ```sql
   -- Check recently updated credits
   SELECT * FROM credits 
   WHERE updated_at > UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL 1 HOUR)) * 1000
   ORDER BY updated_at DESC
   LIMIT 10;
   ```

## Rollback Plan

If issues arise:

1. **During Migration**: The migration is idempotent and can be safely stopped and restarted
2. **After v2 Deploy**: Since keys table still has data, redeploy v1 code that reads from keys table
3. **Emergency SQL**: If needed, sync credits back to keys:
   ```sql
   UPDATE keys k
   INNER JOIN credits c ON c.key_id = k.id
   SET k.remaining_requests = c.remaining
   WHERE k.deleted_at_m IS NULL;
   ```

## Future Enhancement: Identity Credits

The schema already supports identity-based credits:
```sql
CREATE TABLE `credits` (
    `key_id` varchar(256),      -- For key-based credits
    `identity_id` varchar(256),  -- For identity-based credits
    -- ... other fields
);
```

This migration focuses on key credits. Identity credits will be handled separately.

## Testing

The migration has been tested with:
- Idempotent execution (safe to run multiple times)
- Large batch processing
- Null handling for optional fields
- Transaction safety

Run tests:
```bash
go test -v ./cmd/migrate/actions -run TestCreditsMigration
```