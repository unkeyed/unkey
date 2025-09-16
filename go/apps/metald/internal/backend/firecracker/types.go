package firecracker

import (
	"context"
	"log/slog"

	sdk "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/unkeyed/unkey/go/apps/metald/internal/assetmanager"
	"github.com/unkeyed/unkey/go/apps/metald/internal/config"
	"github.com/unkeyed/unkey/go/apps/metald/internal/jailer"
	"github.com/unkeyed/unkey/go/apps/metald/internal/network"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Client implements the Backend interface using firecracker-go-sdk
// with integrated jailer functionality for secure VM isolation.
type Client struct {
	logger          *slog.Logger
	assetClient     assetmanager.Client
	vmRegistry      map[string]*VM
	vmAssetLeases   map[string][]string // VM ID -> asset lease IDs
	jailer          *jailer.Jailer
	jailerConfig    *config.JailerConfig
	baseDir         string
	tracer          trace.Tracer
	meter           metric.Meter
	vmCreateCounter metric.Int64Counter
	vmDeleteCounter metric.Int64Counter
	vmBootCounter   metric.Int64Counter
	vmErrorCounter  metric.Int64Counter
}

// VM represents a VM managed by the Firecracker backend
type VM struct {
	ID           string
	Config       *metaldv1.VmConfig
	State        metaldv1.VmState
	Machine      *sdk.Machine
	NetworkInfo  *network.VMNetwork
	CancelFunc   context.CancelFunc
	AssetMapping *assetMapping     // Asset mapping for lease acquisition
	AssetPaths   map[string]string // Prepared asset paths
}

// assetRequirement represents a required asset for VM creation
type assetRequirement struct {
	Type     assetv1.AssetType
	Labels   map[string]string
	Required bool
}

// assetMapping tracks the mapping between requirements and actual assets
type assetMapping struct {
	requirements []assetRequirement
	assets       map[string]*assetv1.Asset // requirement index -> asset
	assetIDs     []string
	leaseIDs     []string
}

func (am *assetMapping) AssetIDs() []string {
	return am.assetIDs
}

func (am *assetMapping) LeaseIDs() []string {
	return am.leaseIDs
}

// queryKey is used for grouping asset requirements by type and labels
type queryKey struct {
	assetType assetv1.AssetType
	labels    string // Serialized labels for grouping
}
