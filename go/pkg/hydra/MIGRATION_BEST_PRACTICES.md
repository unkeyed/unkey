# Database Layer Migration Best Practices

This document captures the key learnings from migrating Hydra from GORM to SQLC, providing guidance for future database layer migrations.

## Migration Strategy

### 1. Incremental Migration with Panic Protection

**Problem**: Ensuring complete migration without missing any operations.

**Solution**: Use panic-driven migration validation:

```go
// GORM implementation - add panic to force migration
func (s *gormStore) CreateWorkflow(ctx context.Context, workflow *WorkflowExecution) error {
    panic("CreateWorkflow has been migrated to SQLC - use engine.GetSQLCStore().CreateWorkflow() instead")
}
```

**Benefits**:
- Forces immediate identification of unmigrated code paths
- Prevents partial migrations
- Provides clear migration instructions

### 2. Dual Store Architecture During Migration

**Problem**: Maintaining system availability during migration.

**Solution**: Run both stores temporarily:

```go
type Engine struct {
    store store.Store // GORM store  
    sqlc  store.Store // SQLC store for migration
}
```

**Benefits**:
- Allows gradual migration
- Enables A/B testing of implementations
- Provides rollback capability

### 3. Operation-by-Operation Migration

**Problem**: Managing complexity of large-scale migrations.

**Solution**: Migrate operations in logical groups:

1. **Core Operations**: CreateWorkflow, GetWorkflow, etc.
2. **Step Operations**: CreateStep, UpdateStepStatus, etc.
3. **Worker Coordination**: AcquireWorkflowLease, HeartbeatLease, etc.
4. **Advanced Features**: Cron jobs, cleanup operations, etc.
5. **Testing Helpers**: GetAllWorkflows, GetAllSteps, etc.

## Code Quality Patterns

### 1. Type Conversion Helpers

Create reusable conversion functions:

```go
// Helper functions for converting from nullable types
func nullInt64ToPtr(n sql.NullInt64) *int64 {
    if !n.Valid {
        return nil
    }
    return &n.Int64
}

func nullStringToPtr(n sql.NullString) *string {
    if !n.Valid {
        return nil
    }
    return &n.String
}
```

### 2. Consistent Error Handling

```go
func (s *sqlcStore) GetWorkflow(ctx context.Context, namespace, id string) (*WorkflowExecution, error) {
    workflow, err := s.queries.GetWorkflow(ctx, sqlcstore.GetWorkflowParams{
        ID:        id,
        Namespace: namespace,
    })
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, errors.New("workflow not found")
        }
        return nil, err
    }
    // ... conversion logic
}
```

### 3. Transaction Support

Leverage SQLC's built-in transaction support:

```go
func (s *sqlcStore) WithTx(ctx context.Context, fn func(Store) error) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    txQueries := s.queries.WithTx(tx)
    txStore := &sqlcStore{
        db:      s.db,
        queries: txQueries,
        clock:   s.clock,
    }

    if err := fn(txStore); err != nil {
        return err
    }
    return tx.Commit()
}
```

## Testing Strategy

### 1. Migration Validation Tests

- **Architectural constraints**: Prevent GORM dependencies from returning
- **Interface completeness**: Ensure all methods are implemented
- **Type conversion validation**: Verify model mappings are correct

### 2. Operation Coverage Tests

- **End-to-end testing**: Test complete workflows through the new store
- **Error path testing**: Verify error handling and edge cases
- **Performance testing**: Ensure new implementation meets performance requirements

### 3. Integration Tests

- **Cross-operation testing**: Verify operations work together correctly
- **Concurrency testing**: Test multiple workers and operations
- **Failure scenario testing**: Test behavior under database failures

## Performance Considerations

### 1. Query Optimization

- Use SQLC's type-safe queries for optimal performance
- Leverage database indexes for frequently queried fields
- Use prepared statements through SQLC for repeated operations

### 2. Connection Management

- Share database connections between SQLC and other components
- Configure appropriate connection pool settings
- Use context for query timeouts and cancellation

### 3. Type Conversions

- Minimize allocations in conversion functions
- Use nullable types appropriately to avoid unnecessary conversions
- Cache conversion results where appropriate

## Common Pitfalls

### 1. Incomplete Migration

**Problem**: Missing some usage sites during migration.

**Prevention**: 
- Use panic-driven validation
- Comprehensive grep/search for old patterns
- Architectural tests to prevent regression

### 2. Type Conversion Errors

**Problem**: Incorrect mapping between SQLC and domain models.

**Prevention**:
- Create comprehensive type conversion tests
- Use consistent conversion patterns
- Validate all nullable field handling

### 3. Transaction Boundary Issues

**Problem**: Incorrect transaction usage with SQLC.

**Prevention**:
- Understand SQLC's transaction patterns
- Test transaction rollback scenarios
- Use appropriate isolation levels

## Migration Checklist

### Pre-Migration
- [ ] Analyze current database usage patterns
- [ ] Design new schema and SQLC queries
- [ ] Create comprehensive test suite
- [ ] Plan migration phases

### During Migration
- [ ] Implement panic protection in old code
- [ ] Migrate operations in logical groups
- [ ] Test each phase thoroughly
- [ ] Monitor system performance

### Post-Migration
- [ ] Remove old implementation code
- [ ] Clean up unused dependencies
- [ ] Update documentation
- [ ] Add architectural constraint tests

### Final Verification
- [ ] Run full test suite
- [ ] Performance testing
- [ ] Security review
- [ ] Documentation updates

## Lessons Learned

1. **Panic-driven migration** is highly effective for ensuring completeness
2. **Incremental migration** reduces risk and allows for validation at each step
3. **Type conversion helpers** reduce code duplication and errors
4. **Comprehensive testing** is crucial for migration confidence
5. **Architectural tests** prevent regression over time
6. **Documentation updates** are essential for team knowledge transfer

This migration strategy successfully migrated a complex workflow orchestration engine with zero downtime and improved performance characteristics.