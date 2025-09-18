package client

import (
	"fmt"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// AIDEV-NOTE: VM configuration builder for customizable VM creation
// This provides a fluent interface for building VM configurations with sensible defaults

// VMConfigBuilder provides a fluent interface for building VM configurations
type VMConfigBuilder struct {
	config *metaldv1.VmConfig
}

// GetConfig returns the current configuration (for accessing intermediate state)
func (b *VMConfigBuilder) GetConfig() *metaldv1.VmConfig {
	return b.config
}

// NewVMConfigBuilder creates a new VM configuration builder with defaults
func NewVMConfigBuilder() *VMConfigBuilder {
	return &VMConfigBuilder{
		config: &metaldv1.VmConfig{
			VcpuCount:     1,
			MemorySizeMib: 512,
			Boot:          "",
			Storage:       &metaldv1.StorageDevice{},
			NetworkConfig: "",
			Console: &metaldv1.ConsoleConfig{
				Enabled:     true,
				Output:      "/tmp/vm-console.log",
				ConsoleType: "serial",
			},
			Metadata: make(map[string]string),
		},
	}
}

// WithCPU configures CPU settings
func (b *VMConfigBuilder) WithCPU(vcpuCount uint32) *VMConfigBuilder {
	b.config.VcpuCount = vcpuCount
	return b
}

// WithMemory configures memory settings
func (b *VMConfigBuilder) WithMemory(sizeBytes uint64) *VMConfigBuilder {
	b.config.MemorySizeMib = sizeBytes
	return b
}

// WithBoot configures boot settings
func (b *VMConfigBuilder) WithBoot(kernelArgs string) *VMConfigBuilder {
	b.config.Boot = kernelArgs
	return b
}

// WithDefaultBoot configures standard boot settings with kernel args
func (b *VMConfigBuilder) WithDefaultBoot(kernelArgs string) *VMConfigBuilder {
	if kernelArgs == "" {
		kernelArgs = "console=ttyS0 reboot=k panic=1 pci=off"
	}
	return b.WithBoot(kernelArgs)
}

// WithConsole configures console settings
func (b *VMConfigBuilder) WithConsole(enabled bool, output, consoleType string) *VMConfigBuilder {
	b.config.Console = &metaldv1.ConsoleConfig{
		Enabled:     enabled,
		Output:      output,
		ConsoleType: consoleType,
	}
	return b
}

// WithDefaultConsole configures standard console settings
func (b *VMConfigBuilder) WithDefaultConsole(output string) *VMConfigBuilder {
	if output == "" {
		output = "/tmp/vm-console.log"
	}
	return b.WithConsole(true, output, "serial")
}

// DisableConsole disables console output
func (b *VMConfigBuilder) DisableConsole() *VMConfigBuilder {
	return b.WithConsole(false, "", "")
}

// AddMetadata adds metadata key-value pairs
func (b *VMConfigBuilder) AddMetadata(key, value string) *VMConfigBuilder {
	if b.config.Metadata == nil {
		b.config.Metadata = make(map[string]string)
	}
	b.config.Metadata[key] = value
	return b
}

// WithMetadata sets all metadata at once
func (b *VMConfigBuilder) WithMetadata(metadata map[string]string) *VMConfigBuilder {
	b.config.Metadata = metadata
	return b
}

// Build returns the configured VM configuration
func (b *VMConfigBuilder) Build() *metaldv1.VmConfig {
	return b.config
}

// VMTemplate represents common VM configuration templates
type VMTemplate string

const (
	// TemplateMinimal creates a minimal VM with basic resources
	TemplateMinimal VMTemplate = "minimal"
	// TemplateStandard creates a standard VM with balanced resources
	TemplateStandard VMTemplate = "standard"
	// TemplateHighCPU creates a VM optimized for CPU-intensive workloads
	TemplateHighCPU VMTemplate = "high-cpu"
	// TemplateHighMemory creates a VM optimized for memory-intensive workloads
	TemplateHighMemory VMTemplate = "high-memory"
	// TemplateDevelopment creates a VM suitable for development work
	TemplateDevelopment VMTemplate = "development"
)

// sanitizeImageName converts a Docker image name to a safe filename
func sanitizeImageName(imageName string) string {
	// Replace special characters with underscores
	safe := imageName
	replacements := map[string]string{
		"/": "_",
		":": "_",
		"@": "_",
		"+": "_",
		" ": "_",
	}

	for old, new := range replacements {
		safe = fmt.Sprintf("%s", safe)
		// Simple replacement without complex string manipulation
		result := ""
		for _, char := range safe {
			if string(char) == old {
				result += new
			} else {
				result += string(char)
			}
		}
		safe = result
	}
	return safe
}

// ForceBuild configures the VM to force rebuild assets even if cached versions exist
func (b *VMConfigBuilder) ForceBuild(force bool) *VMConfigBuilder {
	// Add force build metadata that will be picked up by the asset management system
	if force {
		b.AddMetadata("force_rebuild", "true")
	} else {
		// Remove force rebuild metadata if it exists
		if b.config.Metadata != nil {
			delete(b.config.Metadata, "force_rebuild")
		}
	}
	return b
}
