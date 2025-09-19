package migrations

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(upSeedNetworks, downSeedNetworks)
}

func upSeedNetworks(tx *sql.Tx) error {
	// Get network CIDR from environment variable, default to 10.0.0.0/8
	rootCIDR := os.Getenv("UNKEY_METALD_NETWORK_CIDR")
	if rootCIDR == "" {
		rootCIDR = "10.0.0.0/8"
	}

	// Get subnet prefix from environment variable, default to /26
	subnetPrefix := 26
	if envPrefix := os.Getenv("UNKEY_METALD_SUBNET_PREFIX"); envPrefix != "" {
		var err error
		if _, err = fmt.Sscanf(envPrefix, "%d", &subnetPrefix); err != nil {
			return fmt.Errorf("invalid UNKEY_METALD_SUBNET_PREFIX %s: %w", envPrefix, err)
		}
	}

	fmt.Printf("Seeding networks from %s with /%d subnets\n", rootCIDR, subnetPrefix)

	_, rootNet, err := net.ParseCIDR(rootCIDR)
	if err != nil {
		return fmt.Errorf("failed to parse root CIDR %s: %w", rootCIDR, err)
	}

	ones, bits := rootNet.Mask.Size()
	if subnetPrefix <= ones {
		return fmt.Errorf("subnet /%d must be smaller than root network /%d", subnetPrefix, ones)
	}

	// Prepare batch insert statement
	stmt, err := tx.Prepare("INSERT INTO networks (base_network) VALUES (?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Generate all /26 subnets
	count := 0
	for ip := rootNet.IP.Mask(rootNet.Mask); rootNet.Contains(ip); {
		subnet := &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(subnetPrefix, bits),
		}

		_, err := stmt.Exec(subnet.String())
		if err != nil {
			return fmt.Errorf("failed to insert subnet %s: %w", subnet.String(), err)
		}

		count++
		if count%10000 == 0 {
			// Log progress for large batches
			fmt.Printf("Inserted %d networks...\n", count)
		}

		// Move to next subnet
		inc := 1 << (bits - subnetPrefix)
		ipInt := ipToInt(ip)
		ipInt += uint32(inc)
		ip = intToIP(ipInt)
	}

	fmt.Printf("Successfully seeded %d /26 networks from %s\n", count, rootCIDR)
	return nil
}

func downSeedNetworks(tx *sql.Tx) error {
	// Remove all seeded networks
	_, err := tx.Exec("DELETE FROM networks")
	if err != nil {
		return fmt.Errorf("failed to delete networks: %w", err)
	}
	return nil
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
