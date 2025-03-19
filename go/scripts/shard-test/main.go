// The shard-test utility deterministically distributes Go test packages
// across multiple shards, allowing for parallel test execution in CI environments.
// It outputs a space-separated list of package paths that belong to the specified
// shard, which can be piped directly into the `go test` command.
//
// Basic usage:
//
//	# Run tests for the first shard out of 8
//	go test $(go run scripts/shard-test/main.go 1/8)
//
//	# Run tests with additional flags
//	go test -v -race $(go run scripts/shard-test/main.go 2/4)
//
// This utility is designed to be used in CI workflows to parallelize test runs.
// It ensures that each package is tested exactly once, and that the distribution
// of packages is deterministic for reproducible builds.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// main is the entry point for the shard-test utility.
//
// It parses command-line arguments to determine the shard specification,
// lists all Go packages in the current module and its subdirectories,
// calculates which packages belong to the requested shard, and
// prints those packages as a space-separated list to stdout.
//
// Command-line arguments:
//   - Exactly one argument in the format "N/M" where:
//   - N is the current shard number (1-indexed)
//   - M is the total number of shards
//
// Output:
//   - To stdout: Space-separated list of package paths for the specified shard
//   - To stderr: Error messages and informational output
//
// Exit codes:
//   - 0: Success or no packages to test in the shard
//   - 1: Invalid arguments or other errors
//
// Example usage in a CI workflow:
//
//	# GitHub Actions workflow snippet
//	jobs:
//	  test:
//	    strategy:
//	      matrix:
//	        shard: ["1/4", "2/4", "3/4", "4/4"]
//	    steps:
//	      - name: Run tests for shard
//	        run: go test $(go run scripts/shard-test/main.go ${{ matrix.shard }})
//
// Notes:
//   - The distribution of packages is deterministic as long as the package list
//     doesn't change between runs
//   - Empty shards (with no packages to test) will produce no output to stdout
//   - The utility requires the `go` command to be available in the PATH
func main() {
	// Parse command line arguments
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run shard-packages.go SHARD/TOTAL (e.g., 1/10)\n")
		os.Exit(1)
	}

	parts := strings.Split(os.Args[1], "/")
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "Error: Argument must be in format N/M (e.g., 1/10)\n")
		os.Exit(1)
	}

	shard, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Shard must be a number\n")
		os.Exit(1)
	}

	totalShards, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Total shards must be a number\n")
		os.Exit(1)
	}

	// Validate inputs
	if shard < 1 || shard > totalShards {
		fmt.Fprintf(os.Stderr, "Error: Shard number must be between 1 and total shards\n")
		os.Exit(1)
	}

	// Adjust shard number to be 0-indexed for calculations
	shard--

	// Get all packages
	cmd := exec.Command("go", "list", "./...")
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing packages: %v\n", err)
		os.Exit(1)
	}

	packages := strings.Split(strings.TrimSpace(string(output)), "\n")
	sort.Strings(packages)

	packageCount := len(packages)
	if packageCount == 0 {
		fmt.Fprintf(os.Stderr, "No packages found\n")
		os.Exit(0)
	}

	// Calculate which packages belong to this shard
	startIndex := packageCount * shard / totalShards
	endIndex := packageCount * (shard + 1) / totalShards

	if startIndex >= packageCount || startIndex == endIndex {
		fmt.Fprintf(os.Stderr, "No packages to test in this shard\n")
		os.Exit(0)
	}

	shardPackages := packages[startIndex:endIndex]

	// Print all packages on one line, space-separated
	fmt.Print(strings.Join(shardPackages, " "))
}
