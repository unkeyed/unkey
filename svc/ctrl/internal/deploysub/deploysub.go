// Package deploysub manages the Stripe side of cancelling a workspace's Unkey
// Deploy plan. It resolves the Deploy price ids from their stable lookup_keys
// and classifies a subscription's items so the cancel can stop the renewal
// without ever refunding the prepaid plan fee.
//
// Two subscription topologies are supported. A Deploy-only subscription (every
// item is a Deploy item) cancels at period end: Stripe bills the frozen
// usage at the boundary and then stops. A mixed subscription (Deploy plus an
// API plan on the same subscription) keeps its metered items and only removes
// the plan-fee item, so the metered items bill the frozen usage at the boundary
// and then report zero.
package deploysub

import (
	"context"
	"sync"
	"time"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/unkeyed/unkey/pkg/fault"
)

// resolutionTTL bounds how long resolved lookup_key -> price id mappings are
// cached. Reprices transfer the lookup_key onto the new price, so this is short
// enough to pick one up without a redeploy and long enough to avoid a
// prices.list call on every cancel.
const resolutionTTL = 5 * time.Minute

// Config holds the Stripe price lookup_keys for Unkey Deploy billing. Each maps
// to a stable Stripe lookup_key (not a price id) so a reprice does not require a
// config change.
type Config struct {
	// StarterLookupKey is the plan-fee lookup_key for the Starter tier.
	StarterLookupKey string
	// ProLookupKey is the plan-fee lookup_key for the Pro tier.
	ProLookupKey string
	// BusinessLookupKey is the plan-fee lookup_key for the Business tier.
	BusinessLookupKey string
	// MeterCPULookupKey is the metered lookup_key for CPU usage.
	MeterCPULookupKey string
	// MeterMemoryLookupKey is the metered lookup_key for memory usage.
	MeterMemoryLookupKey string
	// MeterEgressLookupKey is the metered lookup_key for egress usage.
	MeterEgressLookupKey string
	// MeterDiskLookupKey is the metered lookup_key for disk usage.
	MeterDiskLookupKey string
	// MeterActiveKeysLookupKey is the metered lookup_key for active keys usage.
	MeterActiveKeysLookupKey string
}

// Topology classifies a subscription against the Deploy price set so the cancel
// path and logging can distinguish the cases.
type Topology int

const (
	// TopologyNone means the subscription has no Deploy items: already cancelled.
	TopologyNone Topology = iota
	// TopologyDeployOnly means every item on the subscription is a Deploy item.
	TopologyDeployOnly
	// TopologyMixed means the subscription has Deploy items alongside other
	// (e.g. API plan) items.
	TopologyMixed
)

// String returns a stable label for logging.
func (t Topology) String() string {
	switch t {
	case TopologyNone:
		return "none"
	case TopologyDeployOnly:
		return "deploy_only"
	case TopologyMixed:
		return "mixed"
	default:
		return "unknown"
	}
}

// resolution is a cached snapshot of the resolved Deploy price ids.
type resolution struct {
	resolvedAt   time.Time
	planFeeIDs   map[string]struct{}
	allDeployIDs map[string]struct{}
}

// Manager resolves Deploy price ids and cancels Deploy subscriptions.
type Manager struct {
	client *stripe.Client

	// planFeeLookupKeys are the 3 plan-fee lookup_keys (Starter, Pro, Business).
	planFeeLookupKeys []string
	// meteredLookupKeys are the 5 metered lookup_keys (cpu, memory, egress,
	// disk, active keys).
	meteredLookupKeys []string

	mu     sync.Mutex
	cached *resolution
}

// New builds a Manager from a Stripe client and the Deploy lookup_key config.
// The client may be nil when Stripe is not configured; [Manager.Configured]
// reports whether the manager can run.
func New(client *stripe.Client, cfg Config) *Manager {
	return &Manager{
		client: client,
		planFeeLookupKeys: []string{
			cfg.StarterLookupKey,
			cfg.ProLookupKey,
			cfg.BusinessLookupKey,
		},
		meteredLookupKeys: []string{
			cfg.MeterCPULookupKey,
			cfg.MeterMemoryLookupKey,
			cfg.MeterEgressLookupKey,
			cfg.MeterDiskLookupKey,
			cfg.MeterActiveKeysLookupKey,
		},
		mu:     sync.Mutex{},
		cached: nil,
	}
}

// Configured reports whether the Stripe client and all 8 lookup_keys are set.
// CancelDeploy fails closed when this is false.
func (m *Manager) Configured() bool {
	if m.client == nil {
		return false
	}
	for _, key := range m.planFeeLookupKeys {
		if key == "" {
			return false
		}
	}
	for _, key := range m.meteredLookupKeys {
		if key == "" {
			return false
		}
	}
	return true
}

// resolve maps the configured lookup_keys to current active price ids. Results
// are cached for resolutionTTL. Every lookup_key must resolve to an active
// price; a partial resolution returns an error and is not cached so callers
// fail closed rather than act on an incomplete classification.
func (m *Manager) resolve(ctx context.Context) (planFeeIDs, allDeployIDs map[string]struct{}, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cached != nil && time.Since(m.cached.resolvedAt) < resolutionTTL {
		return m.cached.planFeeIDs, m.cached.allDeployIDs, nil
	}

	allLookupKeys := make([]string, 0, len(m.planFeeLookupKeys)+len(m.meteredLookupKeys))
	allLookupKeys = append(allLookupKeys, m.planFeeLookupKeys...)
	allLookupKeys = append(allLookupKeys, m.meteredLookupKeys...)

	lookupKeyParams := make([]*string, 0, len(allLookupKeys))
	for i := range allLookupKeys {
		lookupKeyParams = append(lookupKeyParams, stripe.String(allLookupKeys[i]))
	}

	//nolint:exhaustruct
	params := &stripe.PriceListParams{
		LookupKeys: lookupKeyParams,
		Active:     stripe.Bool(true),
	}
	params.Limit = stripe.Int64(100)

	idByLookupKey := make(map[string]string, len(allLookupKeys))
	list := m.client.V1Prices.List(ctx, params)
	for price, listErr := range list.All(ctx) {
		if listErr != nil {
			return nil, nil, fault.Wrap(listErr, fault.Internal("listing deploy prices"))
		}
		if price.LookupKey != "" {
			idByLookupKey[price.LookupKey] = price.ID
		}
	}

	planFeeResolved := make(map[string]struct{}, len(m.planFeeLookupKeys))
	allResolved := make(map[string]struct{}, len(allLookupKeys))
	for _, key := range m.planFeeLookupKeys {
		id, ok := idByLookupKey[key]
		if !ok {
			return nil, nil, fault.New("deploy plan-fee lookup_key did not resolve to an active price", fault.Internal("incomplete deploy price set"))
		}
		planFeeResolved[id] = struct{}{}
		allResolved[id] = struct{}{}
	}
	for _, key := range m.meteredLookupKeys {
		id, ok := idByLookupKey[key]
		if !ok {
			return nil, nil, fault.New("deploy metered lookup_key did not resolve to an active price", fault.Internal("incomplete deploy price set"))
		}
		allResolved[id] = struct{}{}
	}

	m.cached = &resolution{
		resolvedAt:   time.Now(),
		planFeeIDs:   planFeeResolved,
		allDeployIDs: allResolved,
	}
	return planFeeResolved, allResolved, nil
}

// Cancel stops the Stripe renewal for the given subscription without refunding.
// It returns the detected topology. A subscription with no Deploy items returns
// TopologyNone (already cancelled; idempotent).
func (m *Manager) Cancel(ctx context.Context, subscriptionID string) (Topology, error) {
	planFeeIDs, allDeployIDs, err := m.resolve(ctx)
	if err != nil {
		return TopologyNone, err
	}

	//nolint:exhaustruct
	sub, err := m.client.V1Subscriptions.Retrieve(ctx, subscriptionID, &stripe.SubscriptionRetrieveParams{})
	if err != nil {
		return TopologyNone, fault.Wrap(err, fault.Internal("retrieving subscription"))
	}

	items := make([]itemPrice, 0, len(sub.Items.Data))
	for _, item := range sub.Items.Data {
		priceID := ""
		if item.Price != nil {
			priceID = item.Price.ID
		}
		items = append(items, itemPrice{itemID: item.ID, priceID: priceID})
	}

	topology, planFeeItemIDs := classify(items, planFeeIDs, allDeployIDs)
	switch topology {
	case TopologyNone:
		// No Deploy items: nothing to cancel. Idempotent success.
		return TopologyNone, nil

	case TopologyDeployOnly:
		// Every item is a Deploy item, so cancel the whole subscription at period
		// end. Stripe bills the frozen usage at the boundary and then stops. No
		// refund.
		//nolint:exhaustruct
		if _, err = m.client.V1Subscriptions.Update(ctx, subscriptionID, &stripe.SubscriptionUpdateParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}); err != nil {
			return TopologyNone, fault.Wrap(err, fault.Internal("cancelling deploy-only subscription"))
		}
		return TopologyDeployOnly, nil

	case TopologyMixed:
		// Keep the metered items so they bill the frozen usage at the boundary and
		// then report zero. Remove only the plan-fee item(s) in one update so the
		// removal is atomic. proration_behavior none means no refund.
		updateItems := make([]*stripe.SubscriptionUpdateItemParams, 0, len(planFeeItemIDs))
		for i := range planFeeItemIDs {
			//nolint:exhaustruct
			updateItems = append(updateItems, &stripe.SubscriptionUpdateItemParams{
				ID:      stripe.String(planFeeItemIDs[i]),
				Deleted: stripe.Bool(true),
			})
		}
		//nolint:exhaustruct
		if _, err = m.client.V1Subscriptions.Update(ctx, subscriptionID, &stripe.SubscriptionUpdateParams{
			Items:             updateItems,
			ProrationBehavior: stripe.String("none"),
		}); err != nil {
			return TopologyNone, fault.Wrap(err, fault.Internal("removing plan-fee item from mixed subscription"))
		}
		return TopologyMixed, nil

	default:
		return TopologyNone, fault.New("unreachable topology", fault.Internal("classify returned an unknown topology"))
	}
}

// itemPrice is the minimal view of a subscription item the classifier needs:
// the item id (to delete) and its price id (to match against the Deploy set).
type itemPrice struct {
	itemID  string
	priceID string
}

// classify decides the cancel topology for a subscription's items against the
// resolved Deploy price ids. For a mixed subscription it also returns the
// plan-fee item ids to remove (metered items stay so they bill the frozen usage
// at the boundary). It is pure so the cancel decision can be tested without
// Stripe.
func classify(items []itemPrice, planFeeIDs, allDeployIDs map[string]struct{}) (Topology, []string) {
	var deployItems int
	var planFeeItemIDs []string
	for _, it := range items {
		if _, ok := allDeployIDs[it.priceID]; ok {
			deployItems++
		}
		if _, ok := planFeeIDs[it.priceID]; ok {
			planFeeItemIDs = append(planFeeItemIDs, it.itemID)
		}
	}

	switch {
	case deployItems == 0:
		return TopologyNone, nil
	case deployItems == len(items):
		return TopologyDeployOnly, nil
	default:
		return TopologyMixed, planFeeItemIDs
	}
}
