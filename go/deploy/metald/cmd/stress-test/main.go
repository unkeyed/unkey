package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"

	"connectrpc.com/connect"
	"golang.org/x/sync/semaphore"
)

type VMState int

const (
	StateCreated VMState = iota
	StateRunning
	StateStopped
	StateDeleted
)

func (s VMState) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	case StateDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

type VM struct {
	ID            string
	State         VMState
	Created       time.Time
	Booted        *time.Time
	Stopped       *time.Time
	Deleted       *time.Time
	CustomerToken string // Track which customer token owns this VM
}

type TestStats struct {
	TotalCreated    int
	TotalBooted     int
	TotalStopped    int
	TotalDeleted    int
	TotalErrors     int
	CreateDurations []time.Duration
	BootDurations   []time.Duration
	StopDurations   []time.Duration
	DeleteDurations []time.Duration
}

type StressTest struct {
	client         vmprovisionerv1connect.VmServiceClient
	vms            map[string]*VM
	mutex          sync.RWMutex
	stats          TestStats
	logger         *slog.Logger
	concSem        *semaphore.Weighted // Limits concurrent VM operations
	customerTokens []string            // Pool of customer tokens for multi-tenant testing
	authEnabled    bool                // Whether to use authentication headers
}

func NewStressTest(serverURL string, heavyLoad bool, authEnabled bool) *StressTest {
	client := vmprovisionerv1connect.NewVmServiceClient(
		http.DefaultClient,
		serverURL,
	)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Limit concurrent operations based on load level
	maxConcurrentOps := 50
	if heavyLoad {
		maxConcurrentOps = 200 // Push Firecracker's concurrency limits
	}
	concSem := semaphore.NewWeighted(int64(maxConcurrentOps))

	// Initialize customer tokens for multi-tenant testing
	customerTokens := []string{
		"dev_customer_stress-test-1",
		"dev_customer_stress-test-2", 
		"dev_customer_stress-test-3",
		"dev_customer_stress-test-4",
		"dev_customer_stress-test-5",
	}

	return &StressTest{
		client:         client,
		vms:            make(map[string]*VM),
		logger:         logger,
		concSem:        concSem,
		customerTokens: customerTokens,
		authEnabled:    authEnabled,
	}
}

// getRandomCustomerToken returns a random customer token for multi-tenant testing
func (s *StressTest) getRandomCustomerToken() string {
	return s.customerTokens[rand.Intn(len(s.customerTokens))]
}

// addAuthHeader adds authentication header to a ConnectRPC request
func (s *StressTest) addAuthHeader(req connect.AnyRequest, token string) {
	if s.authEnabled && token != "" {
		req.Header().Set("Authorization", "Bearer "+token)
	}
}

// addRandomAuthHeader adds a random customer authentication header to a ConnectRPC request
func (s *StressTest) addRandomAuthHeader(req connect.AnyRequest) string {
	if !s.authEnabled {
		return "" // Return empty token when auth is disabled
	}
	token := s.getRandomCustomerToken()
	req.Header().Set("Authorization", "Bearer "+token)
	return token
}

func (s *StressTest) CreateVM(ctx context.Context) (*VM, error) {
	start := time.Now()

	req := &metaldv1.CreateVmRequest{
		Config: &metaldv1.VmConfig{
			Cpu: &metaldv1.CpuConfig{
				VcpuCount: 1,
			},
			Memory: &metaldv1.MemoryConfig{
				SizeBytes: 134217728, // 128MB
			},
			Boot: &metaldv1.BootConfig{
				KernelPath: "/opt/vm-assets/vmlinux",
				KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
			},
			Storage: []*metaldv1.StorageDevice{
				{
					Path:     "/opt/vm-assets/rootfs.ext4",
					ReadOnly: false,
				},
			},
		},
	}

	connectReq := connect.NewRequest(req)
	customerToken := s.addRandomAuthHeader(connectReq)
	resp, err := s.client.CreateVm(ctx, connectReq)
	if err != nil {
		s.mutex.Lock()
		s.stats.TotalErrors++
		s.mutex.Unlock()
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	duration := time.Since(start)
	vm := &VM{
		ID:            resp.Msg.VmId,
		State:         StateCreated,
		Created:       time.Now(),
		CustomerToken: customerToken,
	}

	s.mutex.Lock()
	s.vms[vm.ID] = vm
	s.stats.TotalCreated++
	s.stats.CreateDurations = append(s.stats.CreateDurations, duration)
	s.mutex.Unlock()

	if s.authEnabled && customerToken != "" {
		s.logger.Info("VM created", "vm_id", vm.ID, "customer_token", customerToken, "duration", duration)
	} else {
		s.logger.Info("VM created", "vm_id", vm.ID, "duration", duration)
	}
	return vm, nil
}

func (s *StressTest) BootVM(ctx context.Context, vmID string) error {
	start := time.Now()

	// Get VM to use its customer token
	s.mutex.RLock()
	vm, exists := s.vms[vmID]
	s.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("VM %s not found in registry", vmID)
	}

	req := &metaldv1.BootVmRequest{
		VmId: vmID,
	}

	connectReq := connect.NewRequest(req)
	s.addAuthHeader(connectReq, vm.CustomerToken)
	_, err := s.client.BootVm(ctx, connectReq)
	if err != nil {
		s.mutex.Lock()
		s.stats.TotalErrors++
		// Mark VM as error state so we don't try to use it again
		if vm, exists := s.vms[vmID]; exists {
			vm.State = StateDeleted // Treat as deleted to prevent further operations
		}
		s.mutex.Unlock()
		return fmt.Errorf("failed to boot VM %s: %w", vmID, err)
	}

	duration := time.Since(start)
	now := time.Now()

	s.mutex.Lock()
	if vm, exists := s.vms[vmID]; exists {
		vm.State = StateRunning
		vm.Booted = &now
	}
	s.stats.TotalBooted++
	s.stats.BootDurations = append(s.stats.BootDurations, duration)
	s.mutex.Unlock()

	s.logger.Info("VM booted", "vm_id", vmID, "duration", duration)
	return nil
}

func (s *StressTest) StopVM(ctx context.Context, vmID string) error {
	start := time.Now()

	// Get VM to use its customer token
	s.mutex.RLock()
	vm, exists := s.vms[vmID]
	s.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("VM %s not found in registry", vmID)
	}

	req := &metaldv1.ShutdownVmRequest{
		VmId:           vmID,
		Force:          true,
		TimeoutSeconds: 10,
	}

	connectReq := connect.NewRequest(req)
	s.addAuthHeader(connectReq, vm.CustomerToken)
	_, err := s.client.ShutdownVm(ctx, connectReq)
	if err != nil {
		s.mutex.Lock()
		s.stats.TotalErrors++
		// Mark VM as error state so we don't try to use it again
		if vm, exists := s.vms[vmID]; exists {
			vm.State = StateDeleted // Treat as deleted to prevent further operations
		}
		s.mutex.Unlock()
		return fmt.Errorf("failed to stop VM %s: %w", vmID, err)
	}

	duration := time.Since(start)
	now := time.Now()

	s.mutex.Lock()
	if vm, exists := s.vms[vmID]; exists {
		vm.State = StateStopped
		vm.Stopped = &now
	}
	s.stats.TotalStopped++
	s.stats.StopDurations = append(s.stats.StopDurations, duration)
	s.mutex.Unlock()

	s.logger.Info("VM stopped", "vm_id", vmID, "duration", duration)
	return nil
}

func (s *StressTest) DeleteVM(ctx context.Context, vmID string) error {
	start := time.Now()

	// Get VM to use its customer token
	s.mutex.RLock()
	vm, exists := s.vms[vmID]
	s.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("VM %s not found in registry", vmID)
	}

	req := &metaldv1.DeleteVmRequest{
		VmId: vmID,
	}

	connectReq := connect.NewRequest(req)
	s.addAuthHeader(connectReq, vm.CustomerToken)
	_, err := s.client.DeleteVm(ctx, connectReq)
	if err != nil {
		s.mutex.Lock()
		s.stats.TotalErrors++
		// Remove VM from tracking even if delete failed to prevent retry loops
		if vm, exists := s.vms[vmID]; exists {
			vm.State = StateDeleted
			delete(s.vms, vmID)
		}
		s.mutex.Unlock()
		return fmt.Errorf("failed to delete VM %s: %w", vmID, err)
	}

	duration := time.Since(start)
	now := time.Now()

	s.mutex.Lock()
	if vm, exists := s.vms[vmID]; exists {
		vm.State = StateDeleted
		vm.Deleted = &now
		delete(s.vms, vmID) // Remove from tracking map after deletion
	}
	s.stats.TotalDeleted++
	s.stats.DeleteDurations = append(s.stats.DeleteDurations, duration)
	s.mutex.Unlock()

	s.logger.Info("VM deleted", "vm_id", vmID, "duration", duration)
	return nil
}

func (s *StressTest) GetVMsByState(state VMState) []*VM {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*VM
	for _, vm := range s.vms {
		if vm.State == state {
			// Create a copy to avoid race conditions
			vmCopy := &VM{
				ID:            vm.ID,
				State:         vm.State,
				Created:       vm.Created,
				Booted:        vm.Booted,
				Stopped:       vm.Stopped,
				Deleted:       vm.Deleted,
				CustomerToken: vm.CustomerToken,
			}
			result = append(result, vmCopy)
		}
	}
	return result
}

func (s *StressTest) CountVMsByState(state VMState) int {
	return len(s.GetVMsByState(state))
}

func (s *StressTest) RunIteration(ctx context.Context, targetVMs, targetRunning int) {
	s.logger.Info("Starting iteration",
		"target_vms", targetVMs,
		"target_running", targetRunning,
		"current_total", len(s.vms),
		"current_running", s.CountVMsByState(StateRunning),
	)

	// Create VMs to reach target
	currentTotal := len(s.vms)
	if currentTotal < targetVMs {
		toCreate := targetVMs - currentTotal
		s.logger.Info("Creating VMs", "count", toCreate)

		var wg sync.WaitGroup
		for i := 0; i < toCreate; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := s.concSem.Acquire(ctx, 1); err != nil {
					s.logger.Error("Failed to acquire semaphore for create", "error", err)
					return
				}
				defer s.concSem.Release(1)
				_, err := s.CreateVM(ctx)
				if err != nil {
					s.logger.Error("Failed to create VM", "error", err)
				}
			}()
		}
		wg.Wait()
	}

	// Boot VMs to reach running target
	currentRunning := s.CountVMsByState(StateRunning)
	if currentRunning < targetRunning {
		createdVMs := s.GetVMsByState(StateCreated)
		toBoot := min(targetRunning-currentRunning, len(createdVMs))

		s.logger.Info("Booting VMs", "count", toBoot)

		var wg sync.WaitGroup
		for i := 0; i < toBoot; i++ {
			wg.Add(1)
			go func(vm *VM) {
				defer wg.Done()
				if err := s.concSem.Acquire(ctx, 1); err != nil {
					s.logger.Error("Failed to acquire semaphore for boot", "vm_id", vm.ID, "error", err)
					return
				}
				defer s.concSem.Release(1)
				err := s.BootVM(ctx, vm.ID)
				if err != nil {
					s.logger.Error("Failed to boot VM", "vm_id", vm.ID, "error", err)
				}
			}(createdVMs[i])
		}
		wg.Wait()
	}

	// Stop some running VMs (20% chance)
	if rand.Float32() < 0.2 {
		runningVMs := s.GetVMsByState(StateRunning)
		if len(runningVMs) > 0 {
			toStop := max(1, len(runningVMs)/4) // Stop 25% of running VMs
			s.logger.Info("Stopping VMs", "count", toStop)

			var wg sync.WaitGroup
			for i := 0; i < toStop && i < len(runningVMs); i++ {
				wg.Add(1)
				go func(vm *VM) {
					defer wg.Done()
					if err := s.concSem.Acquire(ctx, 1); err != nil {
						s.logger.Error("Failed to acquire semaphore for stop", "vm_id", vm.ID, "error", err)
						return
					}
					defer s.concSem.Release(1)
					err := s.StopVM(ctx, vm.ID)
					if err != nil {
						s.logger.Error("Failed to stop VM", "vm_id", vm.ID, "error", err)
					}
				}(runningVMs[i])
			}
			wg.Wait()
		}
	}

	// Delete some stopped VMs (30% chance)
	if rand.Float32() < 0.3 {
		stoppedVMs := s.GetVMsByState(StateStopped)
		if len(stoppedVMs) > 0 {
			toDelete := max(1, len(stoppedVMs)/2) // Delete 50% of stopped VMs
			s.logger.Info("Deleting VMs", "count", toDelete)

			var wg sync.WaitGroup
			for i := 0; i < toDelete && i < len(stoppedVMs); i++ {
				wg.Add(1)
				go func(vm *VM) {
					defer wg.Done()
					if err := s.concSem.Acquire(ctx, 1); err != nil {
						s.logger.Error("Failed to acquire semaphore for delete", "vm_id", vm.ID, "error", err)
						return
					}
					defer s.concSem.Release(1)
					err := s.DeleteVM(ctx, vm.ID)
					if err != nil {
						s.logger.Error("Failed to delete VM", "vm_id", vm.ID, "error", err)
					}
				}(stoppedVMs[i])
			}
			wg.Wait()
		}
	}

	// Final state report
	s.logger.Info("Iteration complete",
		"total_vms", len(s.vms),
		"created", s.CountVMsByState(StateCreated),
		"running", s.CountVMsByState(StateRunning),
		"stopped", s.CountVMsByState(StateStopped),
		"deleted", s.CountVMsByState(StateDeleted),
	)
}

func (s *StressTest) PrintStats() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	fmt.Printf("\n=== STRESS TEST STATISTICS ===\n")
	fmt.Printf("Total Created: %d\n", s.stats.TotalCreated)
	fmt.Printf("Total Booted: %d\n", s.stats.TotalBooted)
	fmt.Printf("Total Stopped: %d\n", s.stats.TotalStopped)
	fmt.Printf("Total Deleted: %d\n", s.stats.TotalDeleted)
	fmt.Printf("Total Errors: %d\n", s.stats.TotalErrors)

	if len(s.stats.CreateDurations) > 0 {
		avg := avgDuration(s.stats.CreateDurations)
		fmt.Printf("Avg Create Duration: %v\n", avg)
	}

	if len(s.stats.BootDurations) > 0 {
		avg := avgDuration(s.stats.BootDurations)
		fmt.Printf("Avg Boot Duration: %v\n", avg)
	}

	if len(s.stats.StopDurations) > 0 {
		avg := avgDuration(s.stats.StopDurations)
		fmt.Printf("Avg Stop Duration: %v\n", avg)
	}

	if len(s.stats.DeleteDurations) > 0 {
		avg := avgDuration(s.stats.DeleteDurations)
		fmt.Printf("Avg Delete Duration: %v\n", avg)
	}

	// Export detailed stats as JSON
	statsJSON, _ := json.MarshalIndent(s.stats, "", "  ")
	filename := fmt.Sprintf("stress-test-stats-%d.json", time.Now().Unix())
	os.WriteFile(filename, statsJSON, 0644)
	fmt.Printf("Detailed stats exported to: %s\n", filename)
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	var (
		serverURL    = flag.String("server", "http://localhost:8080", "metald server URL")
		intervals    = flag.Int("intervals", 5, "number of test intervals")
		intervalDur  = flag.Duration("interval-duration", 2*time.Minute, "duration of each interval")
		iterationDur = flag.Duration("iteration-duration", 10*time.Second, "duration between iterations")
		maxVMs       = flag.Int("max-vms", 100, "maximum VMs per interval")
		heavyLoad    = flag.Bool("heavy-load", false, "enable heavy load testing (1000+ VMs, 100+ concurrent ops)")
		authEnabled  = flag.Bool("auth", true, "enable multi-tenant authentication headers (default: true)")
	)
	flag.Parse()

	// Adjust maxVMs for heavy load testing
	if *heavyLoad && *maxVMs < 500 {
		*maxVMs = 1000 // Really stress test Firecracker
	}

	stressTest := NewStressTest(*serverURL, *heavyLoad, *authEnabled)
	ctx := context.Background()

	stressTest.logger.Info("Starting stress test",
		"server", *serverURL,
		"intervals", *intervals,
		"interval_duration", *intervalDur,
		"iteration_duration", *iterationDur,
		"max_vms", *maxVMs,
		"heavy_load", *heavyLoad,
		"auth_enabled", *authEnabled,
	)

	for interval := 1; interval <= *intervals; interval++ {
		stressTest.logger.Info("Starting interval", "interval", interval)

		// Random target VMs for this interval (min 2, max maxVMs)
		minVMs := min(2, *maxVMs)
		targetVMs := minVMs
		if *maxVMs > minVMs {
			targetVMs = rand.Intn(*maxVMs-minVMs+1) + minVMs
		}
		targetRunning := int(float64(targetVMs) * 0.6) // 60% running

		intervalStart := time.Now()
		for time.Since(intervalStart) < *intervalDur {
			stressTest.RunIteration(ctx, targetVMs, targetRunning)
			time.Sleep(*iterationDur)
		}

		stressTest.logger.Info("Interval complete", "interval", interval)
	}

	// Final cleanup - delete all remaining VMs with concurrency limit
	stressTest.logger.Info("Cleaning up remaining VMs")
	
	var wg sync.WaitGroup
	for _, vm := range stressTest.vms {
		if vm.State != StateDeleted {
			wg.Add(1)
			go func(vm *VM) {
				defer wg.Done()
				
				// Acquire semaphore before starting cleanup
				if err := stressTest.concSem.Acquire(ctx, 1); err != nil {
					stressTest.logger.Error("Failed to acquire cleanup semaphore", "vm_id", vm.ID, "error", err)
					return
				}
				defer stressTest.concSem.Release(1)
				
				if vm.State == StateRunning {
					stressTest.StopVM(ctx, vm.ID)
				}
				stressTest.DeleteVM(ctx, vm.ID)
			}(vm)
		}
	}
	wg.Wait()

	stressTest.PrintStats()
	stressTest.logger.Info("Stress test complete!")
}

