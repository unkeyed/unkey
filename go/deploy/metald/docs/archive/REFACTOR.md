# Refactoring Guide: Remaining nestif Issues

This document outlines refactoring strategies for the remaining complex nested blocks (nestif) in the metald codebase. These issues represent genuinely complex business logic that would benefit from thoughtful refactoring given sufficient time.

## Overview

As of the last linting run, there are 4 remaining nestif issues:
- `internal/network/implementation.go:371` - Network namespace operations (complexity: 18)
- `internal/process/manager.go:506` - Jailer chroot setup (complexity: 12)
- `internal/process/manager.go:800` - Jailer network namespace configuration (complexity: 18)
- `internal/process/manager.go:875` - Asset client preparation (complexity: 12)

## Refactoring Strategies

### 1. Network Namespace Operations (`implementation.go:371`)

**Current Issue**: Complex conditional logic for nsenter vs native operations with 18 levels of nesting.

**Proposed Refactoring**:
```go
// Create interface for namespace operations
type NamespaceOperator interface {
    AddLinkToNamespace(ctx context.Context, link netlink.Link, nsPath string) error
    ConfigureInterfaceInNamespace(ctx context.Context, ifName, nsPath string, config InterfaceConfig) error
}

// Implement two strategies
type NsenterOperator struct {
    logger *slog.Logger
}

type NativeOperator struct {
    logger *slog.Logger
}

// Factory method to choose strategy
func NewNamespaceOperator(useNsenter bool, logger *slog.Logger) NamespaceOperator {
    if useNsenter {
        return &NsenterOperator{logger: logger}
    }
    return &NativeOperator{logger: logger}
}
```

**Benefits**:
- Separates nsenter and native implementations
- Each implementation can be tested independently
- Reduces cognitive complexity
- Makes it easier to add new namespace operation strategies

### 2. Jailer Chroot Setup (`manager.go:506`)

**Current Issue**: Deep nesting for directory creation, permission setting, and device node management.

**Proposed Refactoring**:
```go
// Create a JailerChrootBuilder
type JailerChrootBuilder struct {
    logger      *slog.Logger
    chrootPath  string
    deviceNodes []DeviceNode
}

type DeviceNode struct {
    Path  string
    Mode  uint32
    Major uint32
    Minor uint32
}

func (b *JailerChrootBuilder) Build(ctx context.Context) error {
    steps := []func(context.Context) error{
        b.createDirectoryStructure,
        b.createDeviceNodes,
        b.setPermissions,
        b.validateStructure,
    }
    
    for _, step := range steps {
        if err := step(ctx); err != nil {
            return fmt.Errorf("chroot build failed: %w", err)
        }
    }
    return nil
}

func (b *JailerChrootBuilder) createDeviceNodes(ctx context.Context) error {
    for _, node := range b.deviceNodes {
        if err := b.createDeviceNode(node); err != nil {
            b.logger.ErrorContext(ctx, "failed to create device node",
                slog.String("path", node.Path),
                slog.String("error", err.Error()),
            )
            // Continue with other nodes or fail fast based on requirements
        }
    }
    return nil
}
```

**Benefits**:
- Builder pattern provides clear separation of concerns
- Each step is independently testable
- Error handling is centralized
- Easy to add new steps or modify existing ones

### 3. Jailer Network Namespace Configuration (`manager.go:800`)

**Current Issue**: Complex network namespace setup with TAP device management and permission handling.

**Proposed Refactoring**:
```go
// Create a NetworkNamespaceConfigurator
type NetworkNamespaceConfigurator struct {
    logger        *slog.Logger
    jailerConfig  *JailerConfig
    networkMgr    network.Manager
    processID     string
}

type NetworkSetupStep interface {
    Execute(ctx context.Context) error
    Rollback(ctx context.Context) error
    Name() string
}

// Implement chain of responsibility pattern
type NetworkSetupChain struct {
    steps []NetworkSetupStep
}

func (c *NetworkSetupChain) Execute(ctx context.Context) error {
    completed := make([]NetworkSetupStep, 0, len(c.steps))
    
    for _, step := range c.steps {
        if err := step.Execute(ctx); err != nil {
            // Rollback completed steps
            for i := len(completed) - 1; i >= 0; i-- {
                if rbErr := completed[i].Rollback(ctx); rbErr != nil {
                    // Log rollback error but continue
                }
            }
            return fmt.Errorf("network setup failed at step %s: %w", step.Name(), err)
        }
        completed = append(completed, step)
    }
    return nil
}

// Example steps
type CreateNamespaceStep struct{ /* fields */ }
type MoveTAPDeviceStep struct{ /* fields */ }
type ConfigurePermissionsStep struct{ /* fields */ }
```

**Benefits**:
- Each network configuration step is isolated
- Automatic rollback on failure
- Steps can be reused across different configurations
- Easy to add/remove/reorder steps

### 4. Asset Client Preparation (`manager.go:875`)

**Current Issue**: Nested loops and conditionals for preparing different asset types.

**Proposed Refactoring**:
```go
// Create an AssetPreparer
type AssetPreparer struct {
    logger      *slog.Logger
    assetClient assetmanager.Client
    chrootPath  string
}

type AssetType string

const (
    AssetTypeKernel    AssetType = "kernel"
    AssetTypeRootfs    AssetType = "rootfs"
    AssetTypeDiskImage AssetType = "disk"
)

type AssetPreparationStrategy interface {
    PrepareAssets(ctx context.Context, assets []Asset) ([]string, error)
}

type AssetPreparationPipeline struct {
    strategies map[AssetType]AssetPreparationStrategy
    logger     *slog.Logger
}

func (p *AssetPreparationPipeline) PrepareAll(ctx context.Context) ([]string, error) {
    var allAssetIDs []string
    results := make(chan prepareResult)
    
    // Parallel preparation
    for assetType, strategy := range p.strategies {
        go func(at AssetType, s AssetPreparationStrategy) {
            assets, err := p.fetchAssets(ctx, at)
            if err != nil {
                results <- prepareResult{err: err}
                return
            }
            
            ids, err := s.PrepareAssets(ctx, assets)
            results <- prepareResult{ids: ids, err: err}
        }(assetType, strategy)
    }
    
    // Collect results
    for range p.strategies {
        result := <-results
        if result.err != nil {
            p.logger.WarnContext(ctx, "asset preparation failed",
                slog.String("error", result.err.Error()),
            )
            continue
        }
        allAssetIDs = append(allAssetIDs, result.ids...)
    }
    
    return allAssetIDs, nil
}
```

**Benefits**:
- Parallel asset preparation
- Strategy pattern allows different handling per asset type
- Clear separation between fetching and preparing
- Error isolation - one asset type failure doesn't block others

## Implementation Priority

1. **Network Namespace Operations** - Highest complexity (18), significant impact on readability
2. **Jailer Network Namespace** - High complexity (18), security-critical code
3. **Jailer Chroot Setup** - Medium complexity (12), but security-critical
4. **Asset Preparation** - Medium complexity (12), good candidate for parallelization

## Testing Strategy

For each refactoring:
1. Create comprehensive unit tests for current behavior
2. Implement new structure with same tests passing
3. Add additional tests for new error cases and edge conditions
4. Benchmark before/after to ensure no performance regression

## Migration Path

1. Implement new structures alongside existing code
2. Add feature flags to switch between old and new implementations
3. Gradually migrate usage with careful monitoring
4. Remove old code after validation period

## Conclusion

These refactorings would significantly improve code maintainability and testability. The main trade-offs are:
- Initial implementation time
- Potential for introducing bugs during refactoring
- Need for comprehensive testing

However, the long-term benefits of reduced complexity, improved testability, and easier maintenance justify the investment, especially for security-critical components like the jailer configuration.