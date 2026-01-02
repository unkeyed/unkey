# API Benchmark Tools

This directory contains k6 load testing scripts for comparing v1 and v2 Unkey APIs.

## Scripts

### ratelimit.js - Ratelimit API Benchmark
Tests the ratelimiting endpoints with random identifiers.

```bash
UNKEY_ROOT_KEY=your_key REGION=us-east k6 cloud ratelimit.js
```

**Features:**
- 50/50 split between v1 (`api.unkey.dev`) and v2 (`api.unkey.com`) APIs
- Random user identifiers from pool of 5 users
- Tests `/v1/ratelimits.limit` vs `/v2/ratelimit.limit`

### keyverify.js - Key Verification Benchmark
Tests the key verification endpoints.

```bash
UNKEY_ROOT_KEY=your_key KEY=your_test_key k6 cloud keyverify.js
```

**Features:**
- 50/50 split between v1 and v2 APIs
- Tests `/v1/keys.verifyKey` vs `/v2/keys.verifyKey`

## Setup
1. Install k6: `brew install k6`
2. Set up k6 Cloud account and configure project ID in scripts
3. Configure environment variables

## Environment Variables
- `UNKEY_ROOT_KEY` - Required. Your Unkey root API key
- `KEY` - Required for keyverify.js. Test key to verify

## Configuration
- **Duration:** 10 minutes at 10 req/s
- **Load zones:** Currently US East only (others commented out)
- **Thresholds:** P95 < 500ms, 99% success rate
- **Project ID:** 3788521 (update as needed)

## Metrics
- `request_latency` - Custom trend metric with URL and region tags
- Built-in k6 HTTP metrics (duration, success rate, etc.)
- All metrics automatically available in Grafana Cloud
