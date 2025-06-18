# SPIRE Configuration Fix

## Issue

The SPIRE server was failing to start with the following errors:
1. `Referenced but unset environment variable evaluates to an empty string: UNKEY_SPIRE_TRUST_BUNDLE`
2. `could not start logger: not a valid logrus Level: "${UNKEY_SPIRE_LOG_LEVEL:-INFO}"`

## Root Cause

SPIRE's configuration parser doesn't support shell-style variable expansion syntax (`${VAR:-default}`). The configuration file was using this syntax for environment variable substitution, which SPIRE interpreted literally.

## Solution

1. **Configuration Preprocessing**: Created a preprocessing script (`scripts/preprocess-config.sh`) that:
   - Takes the template configuration file
   - Replaces shell-style variable expansions with actual values
   - Outputs a static configuration file for SPIRE to use

2. **Systemd Service Update**: Modified the systemd service to:
   - Run the preprocessing script before starting SPIRE
   - Removed the unnecessary `UNKEY_SPIRE_TRUST_BUNDLE` check (made it optional)

3. **Static Configuration**: Created a static configuration file for immediate use with:
   - Trust domain: `dev.unkey.app`
   - Log level: `INFO`
   - Database: SQLite at `/var/lib/spire/server/data/datastore.sqlite3`

## Files Changed

1. `/spire/scripts/preprocess-config.sh` - New preprocessing script
2. `/spire/server/spire-server.conf.template` - Renamed from `spire-server.conf`
3. `/spire/server/spire-server.conf` - New static configuration
4. `/spire/contrib/systemd/spire-server.service` - Updated to use preprocessing
5. `/spire/Makefile` - Updated to install all necessary files

## Usage

### With Static Configuration (Development)
```bash
# Install and start with static config
make install-server
make service-start-server
```

### With Environment Variables (Production)
```bash
# Set environment variables
export UNKEY_SPIRE_TRUST_DOMAIN=prod.unkey.app
export UNKEY_SPIRE_LOG_LEVEL=WARN
export UNKEY_SPIRE_DB_TYPE=postgres
export UNKEY_SPIRE_DB_CONNECTION="postgresql://user:pass@host/spire"

# Install and start (preprocessing will run automatically)
make install-server
make service-start-server
```

## Environment Variables

The following environment variables are supported:
- `UNKEY_SPIRE_TRUST_DOMAIN` - SPIFFE trust domain (default: `dev.unkey.app`)
- `UNKEY_SPIRE_LOG_LEVEL` - Log level: DEBUG, INFO, WARN, ERROR (default: `INFO`)
- `UNKEY_SPIRE_DB_TYPE` - Database type: sqlite3, postgres, mysql (default: `sqlite3`)
- `UNKEY_SPIRE_DB_CONNECTION` - Database connection string
- `UNKEY_SPIRE_TRUST_BUNDLE` - Optional: Path to external trust bundle file

## AIDEV-NOTE

The preprocessing approach allows us to maintain configuration templates with defaults while ensuring SPIRE receives valid configuration files. This pattern can be extended for other services that don't support environment variable expansion natively.