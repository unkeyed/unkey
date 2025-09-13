// generate_seed.go
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <network-cidr> <subnet-size>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s 10.0.0.0/24 /28    # Split a /24 into /28 subnets\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 192.168.0.0/16 /24 # Split a /16 into /24 subnets\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 10.0.0.0/22 /27    # Split a /22 into /27 subnets\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Output can be piped directly to sqlite3:\n")
		fmt.Fprintf(os.Stderr, "  %s 10.0.0.0/24 /28 | sqlite3 network.db\n", os.Args[0])
	}

	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	rootCIDR := flag.Arg(0)
	subnetSize := flag.Arg(1)

	// Parse the root network
	_, rootNet, err := net.ParseCIDR(rootCIDR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid CIDR notation '%s': %v\n", rootCIDR, err)
		os.Exit(1)
	}

	// Parse desired subnet size (e.g., "/28" -> 28)
	subnetSize = strings.TrimPrefix(subnetSize, "/")
	newPrefix, err := strconv.Atoi(subnetSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid subnet size '%s'. Use format like '/28' or '28'\n", flag.Arg(1))
		os.Exit(1)
	}

	// Validate subnet size
	ones, bits := rootNet.Mask.Size()
	if newPrefix <= ones {
		fmt.Fprintf(os.Stderr, "Error: Subnet /%d must be smaller than root network /%d\n", newPrefix, ones)
		os.Exit(1)
	}
	if newPrefix > 32 {
		fmt.Fprintf(os.Stderr, "Error: Invalid subnet size /%d (maximum is /32)\n", newPrefix)
		os.Exit(1)
	}

	// Calculate how many subnets we'll create
	subnetCount := 1 << (newPrefix - ones)

	// Print info as SQL comments
	fmt.Printf("-- Generated network seed\n")
	fmt.Printf("-- Splitting %s into %d x /%d subnets\n", rootCIDR, subnetCount, newPrefix)
	fmt.Printf("-- Each /%d subnet has %d total IPs (%d usable)\n",
		newPrefix,
		1<<(32-newPrefix),
		(1<<(32-newPrefix))-2)
	fmt.Println()

	// Generate the INSERT statement
	fmt.Println("INSERT INTO networks (base_network) VALUES")

	subnets := []string{}
	for ip := rootNet.IP.Mask(rootNet.Mask); rootNet.Contains(ip); {
		subnet := &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(newPrefix, bits),
		}
		subnets = append(subnets, fmt.Sprintf("  ('%s')", subnet.String()))

		// Move to next subnet
		inc := 1 << (bits - newPrefix)
		ipInt := ipToInt(ip)
		ipInt += uint32(inc)
		ip = intToIP(ipInt)
	}

	// Output with proper SQL formatting
	for i, subnet := range subnets {
		if i < len(subnets)-1 {
			fmt.Println(subnet + ",")
		} else {
			fmt.Println(subnet + ";")
		}
	}
}

// ipToInt converts an IP address to a 32-bit integer
func ipToInt(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

// intToIP converts a 32-bit integer to an IP address
func intToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
