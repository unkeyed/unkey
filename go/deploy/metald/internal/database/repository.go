package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

// VMRepository handles VM state persistence operations
type VMRepository struct {
	db     *Database
	logger *slog.Logger
}

// NewVMRepository creates a new VM repository
func NewVMRepository(db *Database) *VMRepository {
	return &VMRepository{
		db:     db,
		logger: db.logger.With("component", "vm_repository"),
	}
}

// VM represents the database model for a VM
type VM struct {
	ID           string
	CustomerID   string
	Config       []byte // serialized protobuf
	State        metaldv1.VmState
	ProcessID    *string
	PortMappings string // JSON serialized port mappings
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time

	// Parsed configuration (populated by ListVMsByCustomerWithContext)
	ParsedConfig *metaldv1.VmConfig
}

// CreateVM inserts a new VM record
func (r *VMRepository) CreateVM(vmID, customerID string, config *metaldv1.VmConfig, state metaldv1.VmState) error {
	return r.CreateVMWithContext(context.Background(), vmID, customerID, config, state)
}

// CreateVMWithContext inserts a new VM record with context for tracing
func (r *VMRepository) CreateVMWithContext(ctx context.Context, vmID, customerID string, config *metaldv1.VmConfig, state metaldv1.VmState) error {
	_, span := r.db.tracer.Start(ctx, "vm_repository.create_vm",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.String("vm.customer_id", customerID),
			attribute.String("vm.state", state.String()),
		),
	)
	defer span.End()

	r.logger.DebugContext(ctx, "creating VM record",
		slog.String("vm_id", vmID),
		slog.String("customer_id", customerID),
		slog.String("state", state.String()),
	)
	configBytes, err := proto.Marshal(config)
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to marshal VM config",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to marshal VM config: %w", err)
	}

	query := `
		INSERT INTO vms (id, customer_id, config, state, port_mappings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	_, err = r.db.db.Exec(query, vmID, customerID, configBytes, int32(state), "[]")
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to insert VM record",
			slog.String("vm_id", vmID),
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create VM: %w", err)
	}

	r.logger.InfoContext(ctx, "VM record created successfully",
		slog.String("vm_id", vmID),
		slog.String("customer_id", customerID),
		slog.String("state", state.String()),
	)

	return nil
}

// GetVM retrieves a VM by ID
func (r *VMRepository) GetVM(vmID string) (*VM, error) {
	return r.GetVMWithContext(context.Background(), vmID)
}

// GetVMWithContext retrieves a VM by ID with context for tracing
func (r *VMRepository) GetVMWithContext(ctx context.Context, vmID string) (*VM, error) {
	_, span := r.db.tracer.Start(ctx, "vm_repository.get_vm",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
		),
	)
	defer span.End()

	r.logger.DebugContext(ctx, "retrieving VM record",
		slog.String("vm_id", vmID),
	)
	query := `
		SELECT id, customer_id, config, state, process_id, port_mappings, created_at, updated_at, deleted_at
		FROM vms
		WHERE id = ? AND deleted_at IS NULL
	`

	var vm VM
	var processID sql.NullString
	var portMappings sql.NullString
	var deletedAt sql.NullTime

	err := r.db.db.QueryRow(query, vmID).Scan(
		&vm.ID,
		&vm.CustomerID,
		&vm.Config,
		&vm.State,
		&processID,
		&portMappings,
		&vm.CreatedAt,
		&vm.UpdatedAt,
		&deletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.DebugContext(ctx, "VM not found",
				slog.String("vm_id", vmID),
			)
			return nil, fmt.Errorf("VM not found: %s", vmID)
		}
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to query VM record",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	if processID.Valid {
		vm.ProcessID = &processID.String
	}
	if portMappings.Valid {
		vm.PortMappings = portMappings.String
	} else {
		vm.PortMappings = "[]" // Default empty array
	}
	if deletedAt.Valid {
		vm.DeletedAt = &deletedAt.Time
	}

	r.logger.DebugContext(ctx, "VM record retrieved successfully",
		slog.String("vm_id", vmID),
		slog.String("customer_id", vm.CustomerID),
		slog.String("state", vm.State.String()),
	)

	span.SetAttributes(
		attribute.String("vm.customer_id", vm.CustomerID),
		attribute.String("vm.state", vm.State.String()),
	)

	return &vm, nil
}

// UpdateVMState updates the VM state and optionally the process ID
func (r *VMRepository) UpdateVMState(vmID string, state metaldv1.VmState, processID *string) error {
	return r.UpdateVMStateWithContext(context.Background(), vmID, state, processID)
}

// UpdateVMStateWithContext updates the VM state and optionally the process ID with context for tracing
func (r *VMRepository) UpdateVMStateWithContext(ctx context.Context, vmID string, state metaldv1.VmState, processID *string) error {
	_, span := r.db.tracer.Start(ctx, "vm_repository.update_vm_state",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.String("vm.state", state.String()),
		),
	)
	defer span.End()

	r.logger.DebugContext(ctx, "updating VM state",
		slog.String("vm_id", vmID),
		slog.String("state", state.String()),
		slog.Any("process_id", processID),
	)
	query := `
		UPDATE vms 
		SET state = ?, process_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.db.Exec(query, int32(state), processID, vmID)
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to update VM state",
			slog.String("vm_id", vmID),
			slog.String("state", state.String()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to update VM state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to get rows affected",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.WarnContext(ctx, "VM not found or already deleted during state update",
			slog.String("vm_id", vmID),
			slog.String("state", state.String()),
		)
		return fmt.Errorf("VM not found or already deleted: %s", vmID)
	}

	r.logger.InfoContext(ctx, "VM state updated successfully",
		slog.String("vm_id", vmID),
		slog.String("state", state.String()),
		slog.Int64("rows_affected", rowsAffected),
	)

	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))

	return nil
}

// ListVMs retrieves VMs with optional filters
func (r *VMRepository) ListVMs(customerID *string, states []metaldv1.VmState, limit, offset int) ([]*VM, error) {
	baseQuery := `
		SELECT id, customer_id, config, state, process_id, port_mappings, created_at, updated_at, deleted_at
		FROM vms
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	// Add customer filter
	if customerID != nil {
		baseQuery += " AND customer_id = ?"
		args = append(args, *customerID)
	}

	// Add state filters
	if len(states) > 0 {
		baseQuery += " AND state IN ("
		for i, state := range states {
			if i > 0 {
				baseQuery += ", "
			}
			baseQuery += "?"
			args = append(args, int32(state))
		}
		baseQuery += ")"
	}

	// Add ordering and pagination
	baseQuery += " ORDER BY created_at DESC"
	if limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		baseQuery += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := r.db.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}
	defer rows.Close()

	var vms []*VM
	for rows.Next() {
		var vm VM
		var processID sql.NullString
		var portMappings sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&vm.ID,
			&vm.CustomerID,
			&vm.Config,
			&vm.State,
			&processID,
			&portMappings,
			&vm.CreatedAt,
			&vm.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan VM row: %w", err)
		}

		if processID.Valid {
			vm.ProcessID = &processID.String
		}
		if portMappings.Valid {
			vm.PortMappings = portMappings.String
		} else {
			vm.PortMappings = "[]" // Default empty array
		}
		if deletedAt.Valid {
			vm.DeletedAt = &deletedAt.Time
		}

		vms = append(vms, &vm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating VM rows: %w", err)
	}

	return vms, nil
}

// ListVMsByCustomerWithContext lists all VMs for a specific customer with context for tracing
func (r *VMRepository) ListVMsByCustomerWithContext(ctx context.Context, customerID string) ([]*VM, error) {
	_, span := r.db.tracer.Start(ctx, "vm_repository.list_vms_by_customer",
		trace.WithAttributes(
			attribute.String("customer.id", customerID),
		),
	)
	defer span.End()

	r.logger.DebugContext(ctx, "listing VMs for customer",
		slog.String("customer_id", customerID),
	)

	// Use existing ListVMs method with customer filter
	vms, err := r.ListVMs(&customerID, nil, 0, 0)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Deserialize configs for service layer
	for _, vm := range vms {
		if len(vm.Config) > 0 {
			var config metaldv1.VmConfig
			if err := proto.Unmarshal(vm.Config, &config); err != nil {
				r.logger.ErrorContext(ctx, "failed to unmarshal VM config",
					slog.String("vm_id", vm.ID),
					slog.String("error", err.Error()),
				)
				continue
			}
			vm.ParsedConfig = &config
		}
	}

	r.logger.DebugContext(ctx, "listed VMs for customer",
		slog.String("customer_id", customerID),
		slog.Int("count", len(vms)),
	)

	return vms, nil
}

// DeleteVM soft deletes a VM by setting deleted_at
func (r *VMRepository) DeleteVM(vmID string) error {
	return r.DeleteVMWithContext(context.Background(), vmID)
}

// DeleteVMWithContext soft deletes a VM by setting deleted_at with context for tracing
func (r *VMRepository) DeleteVMWithContext(ctx context.Context, vmID string) error {
	_, span := r.db.tracer.Start(ctx, "vm_repository.delete_vm",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
		),
	)
	defer span.End()

	r.logger.DebugContext(ctx, "deleting VM record",
		slog.String("vm_id", vmID),
	)
	query := `
		UPDATE vms 
		SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.db.Exec(query, vmID)
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to delete VM record",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to get rows affected",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.WarnContext(ctx, "VM not found or already deleted during deletion",
			slog.String("vm_id", vmID),
		)
		return fmt.Errorf("VM not found or already deleted: %s", vmID)
	}

	r.logger.InfoContext(ctx, "VM record deleted successfully",
		slog.String("vm_id", vmID),
		slog.Int64("rows_affected", rowsAffected),
	)

	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))

	return nil
}

// GetVMConfig unmarshals and returns the VM configuration
func (vm *VM) GetVMConfig() (*metaldv1.VmConfig, error) {
	var config metaldv1.VmConfig
	if err := proto.Unmarshal(vm.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VM config: %w", err)
	}
	return &config, nil
}

// CountVMs returns the total count of VMs with optional filters
func (r *VMRepository) CountVMs(customerID *string, states []metaldv1.VmState) (int64, error) {
	baseQuery := "SELECT COUNT(*) FROM vms WHERE deleted_at IS NULL"
	args := []interface{}{}

	// Add customer filter
	if customerID != nil {
		baseQuery += " AND customer_id = ?"
		args = append(args, *customerID)
	}

	// Add state filters
	if len(states) > 0 {
		baseQuery += " AND state IN ("
		for i, state := range states {
			if i > 0 {
				baseQuery += ", "
			}
			baseQuery += "?"
			args = append(args, int32(state))
		}
		baseQuery += ")"
	}

	var count int64
	err := r.db.db.QueryRow(baseQuery, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count VMs: %w", err)
	}

	return count, nil
}

// UpdateVMPortMappings updates the port mappings for a VM
func (r *VMRepository) UpdateVMPortMappings(vmID string, portMappingsJSON string) error {
	return r.UpdateVMPortMappingsWithContext(context.Background(), vmID, portMappingsJSON)
}

// UpdateVMPortMappingsWithContext updates the port mappings for a VM with context for tracing
func (r *VMRepository) UpdateVMPortMappingsWithContext(ctx context.Context, vmID string, portMappingsJSON string) error {
	_, span := r.db.tracer.Start(ctx, "vm_repository.update_vm_port_mappings",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
		),
	)
	defer span.End()

	r.logger.DebugContext(ctx, "updating VM port mappings",
		slog.String("vm_id", vmID),
		slog.String("port_mappings", portMappingsJSON),
	)

	query := `
		UPDATE vms 
		SET port_mappings = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := r.db.db.Exec(query, portMappingsJSON, vmID)
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to update VM port mappings",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to update VM port mappings: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to get rows affected",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.WarnContext(ctx, "VM not found or already deleted during port mappings update",
			slog.String("vm_id", vmID),
		)
		return fmt.Errorf("VM not found or already deleted: %s", vmID)
	}

	r.logger.InfoContext(ctx, "VM port mappings updated successfully",
		slog.String("vm_id", vmID),
		slog.Int64("rows_affected", rowsAffected),
	)

	span.SetAttributes(attribute.Int64("db.rows_affected", rowsAffected))

	return nil
}

// ListAllVMsWithContext retrieves all VMs from the database with context for tracing
func (r *VMRepository) ListAllVMsWithContext(ctx context.Context) ([]*VM, error) {
	_, span := r.db.tracer.Start(ctx, "vm_repository.list_all_vms")
	defer span.End()

	r.logger.DebugContext(ctx, "listing all VMs from database")

	query := `
		SELECT id, customer_id, config, state, process_id, port_mappings, created_at, updated_at, deleted_at
		FROM vms
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.db.QueryContext(ctx, query)
	if err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "failed to query all VMs",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to list all VMs: %w", err)
	}
	defer rows.Close()

	var vms []*VM
	for rows.Next() {
		var vm VM
		var processID sql.NullString
		var portMappings sql.NullString
		var deletedAt sql.NullTime

		err := rows.Scan(
			&vm.ID,
			&vm.CustomerID,
			&vm.Config,
			&vm.State,
			&processID,
			&portMappings,
			&vm.CreatedAt,
			&vm.UpdatedAt,
			&deletedAt,
		)
		if err != nil {
			span.RecordError(err)
			r.logger.ErrorContext(ctx, "failed to scan VM row",
				slog.String("error", err.Error()),
			)
			return nil, fmt.Errorf("failed to scan VM row: %w", err)
		}

		if processID.Valid {
			vm.ProcessID = &processID.String
		}
		if portMappings.Valid {
			vm.PortMappings = portMappings.String
		} else {
			vm.PortMappings = "[]" // Default empty array
		}
		if deletedAt.Valid {
			vm.DeletedAt = &deletedAt.Time
		}

		vms = append(vms, &vm)
	}

	if err := rows.Err(); err != nil {
		span.RecordError(err)
		r.logger.ErrorContext(ctx, "error iterating VM rows",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("error iterating VM rows: %w", err)
	}

	r.logger.InfoContext(ctx, "successfully listed all VMs from database",
		slog.Int("count", len(vms)),
	)

	span.SetAttributes(attribute.Int("vm.count", len(vms)))

	return vms, nil
}

// UpdateVMStateWithContextInt updates VM state with an integer state parameter (used by reconciler)
func (r *VMRepository) UpdateVMStateWithContextInt(ctx context.Context, vmID string, state int) error {
	return r.UpdateVMStateWithContext(ctx, vmID, metaldv1.VmState(state), nil)
}
