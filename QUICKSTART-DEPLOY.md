# Quickstart Guide

This guide will help you get the Unkey deployment platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- Go 1.24 or later
- A terminal/command line
- dnsmasq (for wildcard DNS setup)

## Step 1: Configure Build Backend

**You must configure the build backend before starting Docker Compose.** This script generates the `.env` file that Docker Compose requires.

### Why This Matters

The platform builds **unknown user code** in isolation for security. When users deploy their applications, the system:

1. **Isolates each build** - User code runs in separate containers/VMs to prevent interference
2. **Caches layers** - Speeds up rebuilds by reusing unchanged Docker layers
3. **Stores artifacts** - Uses S3-compatible storage for built images and artifacts

You have two options:

- **Docker**: Fully local setup, slower builds, good for development
- **Depot**: Fast remote builds with persistent layer caching

Navigate to the deployment directory:

```bash
cd deployment
```

### Option A: Local Docker

Fully local setup using Docker with MinIO for S3 storage:

```bash
./setup-build-backend.sh docker
```

This configures:

- Local Docker builds (no external services)
- MinIO S3 storage running in Docker

### Option B: Depot

Remote builds with persistent caching for faster iteration. Create `deployment/config/depot.json`:

```json
{
  "token": "depot_org_YOUR_TOKEN_HERE",
  "s3_url": "https://your-s3-endpoint.com",
  "s3_access_key_id": "your_access_key",
  "s3_access_key_secret": "your_secret_key"
}
```

Then run:

```bash
./setup-build-backend.sh depot
```

This configures:

- Remote Depot infrastructure for builds
- Persistent layer caching across builds
- Your S3-compatible storage for artifacts

**Note:** The token must start with `depot_org_` or the script will fail validation.

## Step 2: Configure API Keys

1. Set up the API key for ctrl service authentication in `go/apps/ctrl/.env`:

```bash
UNKEY_API_KEY="your-local-dev-key"
```

2. Set up dashboard environment variables for ctrl authentication in `apps/dashboard/.env.local`:

```bash
CTRL_URL="http://127.0.0.1:7091"
CTRL_API_KEY="your-local-dev-key"
```

**Critical:** Use the same API key value in both files for authentication to work.

## Step 3: Start the Platform

From the project root directory, start all services:

```bash
docker compose -f ./deployment/docker-compose.yaml up mysql planetscale clickhouse redis s3 dashboard gw krane ctrl -d --build
```

This starts:

- **mysql**: Database for workspace, project, and deployment data
- **planetscale**: PlanetScale HTTP simulator for database access
- **clickhouse**: Analytics database for metrics and logs
- **redis**: Caching layer for session and temporary data
- **s3**: MinIO S3-compatible storage for assets and vault data (when using docker backend)
- **dashboard**: Web UI for managing deployments (port 3000)
- **gw**: Gateway service for routing traffic (ports 80/443)
- **krane**: VM/container management service (port 8090)
- **ctrl**: Control plane service for managing deployments (port 7091)

## Step 4: Set Up DNS and Certificates

1. Set up wildcard DNS for `unkey.local`:

```bash
./deployment/setup-wildcard-dns.sh
```

2. **OPTIONAL**: Install self-signed certificate for HTTPS (to avoid SSL errors):

```bash
# For macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./deployment/certs/unkey.local.crt
```

Note: Certificates are in `deployment/certs`. You can skip this if you're fine with SSL warnings in your browser.

## Step 5: Set Up Your Workspace

1. Open your browser and navigate to the dashboard:

```
http://localhost:3000
```

2. Create a workspace and copy its ID

3. Create a new project by going to:

```
http://localhost:3000/projects
```

Fill out the form:

- **Name**: Choose any name (e.g., "My Test App")
- **Slug**: This will auto-generate based on the name
- **Git URL**: Optional, leave blank for testing

4. After creating the project, **copy the Project ID** from the project details

## Step 6: Deploy a Version

1. Navigate to the go directory:

```bash
cd go
```

2. Set up API key authentication for the CLI:

**Option A: Environment variable (recommended)**

```bash
export API_KEY="your-local-dev-key"
```

**Option B: CLI flag**

Use `--api-key="your-local-dev-key"` in the command below.

3. Deploy using the CLI with your copied IDs:

```bash
go run . deploy \
  --context=./demo_api \
  --workspace-id="REPLACE_ME" \
  --project-id="REPLACE_ME" \
  --control-plane-url="http://127.0.0.1:7091" \
  --api-key="your-local-dev-key" \
  --keyspace-id="REPLACE_ME"  # Optional, only needed for key verifications
```

Replace the placeholder values:

- `REPLACE_ME` with your actual workspace ID, project ID, and keyspace ID (if needed)
- `your-local-dev-key` with the same API key value you set in Step 2
- Keep `--context=./demo_api` as shown (there's a demo API in that folder)

**Note**: If using Option A (environment variable), you can omit the `--api-key` flag from the command.

4. The CLI will:
   - Build a Docker image from the demo_api code (using your configured build backend)
   - Create a deployment on the Unkey platform
   - Show real-time progress through deployment stages
   - Deploy using Krane's VM/container backend

## Step 7: Test Your Deployment

1. Once deployment completes, test the API from the project root directory:

```bash
curl --cacert ./deployment/certs/unkey.local.crt https://REPLACE_ME/v1/liveness
```

Replace `REPLACE_ME` with your deployment domain.

**Note:** The liveness endpoint is public and doesn't require authentication. For protected endpoints, include an Authorization header:

```bash
curl --cacert ./deployment/certs/unkey.local.crt \
  -H "Authorization: Bearer YOUR_API_KEY" \
  https://YOUR_DOMAIN/protected/endpoint
```

2. Monitor your deployment in the dashboard:

```
http://localhost:3000/deployments
```

## Important: Application Port Configuration

**Your deployed application MUST read the `PORT` environment variable and listen on that port.** The platform sets `PORT=8080` in the container.

**Examples for different languages:**

```javascript
// Node.js
const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(`Server running on port ${port}`);
});
```

```python
# Python
import os
port = int(os.environ.get('PORT', 3000))
app.run(host='0.0.0.0', port=port)
```

```go
// Go
port := os.Getenv("PORT")
if port == "" {
    port = "3000"
}
http.ListenAndServe(":"+port, handler)
```

The demo_api already follows this pattern and listens on the PORT environment variable.

## Troubleshooting

### Build Issues

- **"depot login failed"**: Check your depot token in `deployment/config/depot.json` - it must start with `depot_org_`
- **"S3 connection failed"**: Verify S3 credentials in your depot.json or ensure MinIO is running (for docker backend)
- **Slow builds**: Switch to depot backend for faster builds with layer caching
- **Stuck builds**: If you are stuck with a deployment go to `http://localhost:9070/ui/invocations` and kill the ongoing invocation.

### Deployment Issues

- **"port is already allocated"**: The system will automatically retry with a new random port
- **Application not responding**: Verify your app listens on the PORT environment variable (should be 8080)
- **Dockerfile issues**: Ensure your Dockerfile exposes the correct port (8080 in the demo_api example)

### Service Logs

Check container logs for any service:

```bash
docker logs <container-name>
```

Service names: `mysql`, `planetscale`, `clickhouse`, `redis`, `s3`, `dashboard`, `gw`, `krane`, `ctrl`
