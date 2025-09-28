# Quickstart Guide

This guide will help you get the Unkey deployment platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- Go 1.24 or later
- A terminal/command line
- dnsmasq (for wildcard DNS setup)

## Step 1: Start the Platform

1. Set up the API key for ctrl service authentication by adding it to `go/apps/ctrl/.env`:

```bash
UNKEY_API_KEY="your-local-dev-key"
```

2. Set up dashboard environment variables for ctrl authentication in `apps/dashboard/.env.local`:

```bash
CTRL_URL="http://127.0.0.1:7091"
CTRL_API_KEY="your-local-dev-key"
```

Note: Use the same API key value in both files for authentication to work properly.

3. Start all necessary services using Docker Compose:

```bash
docker compose -f ./deployment/docker-compose.yaml up mysql planetscale clickhouse redis s3 dashboard gw metald ctrl -d --build
```

This will start:
- **mysql**: Database for storing workspace, project, and deployment data
- **planetscale**: PlanetScale HTTP simulator for database access
- **clickhouse**: Analytics database for metrics and logs
- **redis**: Caching layer for session and temporary data
- **s3**: MinIO S3-compatible storage for assets and vault data
- **dashboard**: Web UI for managing deployments (port 3000)
- **gw**: Gateway service for routing traffic (ports 80/443)
- **metald**: VM/container management service (port 8090)
- **ctrl**: Control plane service for managing deployments (port 7091)

4. Set up wildcard DNS for `unkey.local`:

```bash
./deployment/setup-wildcard-dns.sh
```

5. **OPTIONAL**: Install self-signed certificate for HTTPS (to avoid SSL errors):

```bash
# For macOS
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./deployment/certs/unkey.local.crt
```

Note: Certificates should be mounted to `deployment/certs`. You can skip this if you're fine with SSL errors in your browser.

## Step 2: Set Up Your Workspace

1. Open your browser and navigate to the dashboard:

```
http://localhost:3000
```

2. Create a workspace and copy its id

3. Create a new project by filling out the form:

Go to http://localhost:3000/projects

- **Name**: Choose any name (e.g., "My Test App")
- **Slug**: This will auto-generate based on the name
- **Git URL**: Optional, leave blank for testing

4. After creating the project, **copy the Project ID** from the project details. It will look like:

## Step 3: Deploy a Version

1. Navigate to the go directory:

```bash
cd go
```

2. Set up API key authentication for the CLI (choose one option):

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
  --keyspace-id="REPLACE_ME" # This is optional if you want key verifications
```

Replace the placeholder values:
- `REPLACE_ME` with your actual workspace ID, project ID, and keyspace ID
- `your-local-dev-key` with the same API key value you set in steps 1 and 2
- Keep `--context=./demo_api` as shown (there's a demo API in that folder)

**Note**: If using Option A (environment variable), you can omit the `--api-key` flag from the command.

3. The CLI will:
   - Build a Docker image from the demo_api code
   - Create a deployment on the Unkey platform
   - Show real-time progress through deployment stages
   - Deploy using metald's VM/container backend

## Step 4: Test Your Deployment

1. Once deployment completes, test the API in the unkey root directory:

```bash
curl --cacert ./deployment/certs/unkey.local.crt https://REPLACE_ME/v1/liveness
```

Replace:
- `REPLACE_ME` (URL) with your deployment domain

**Note:** The liveness endpoint is public and doesn't require authentication. For protected endpoints, include an Authorization header:

```bash
curl --cacert ./deployment/certs/unkey.local.crt -H "Authorization: Bearer YOUR_API_KEY" https://YOUR_DOMAIN/protected/endpoint
```

2. Return to the dashboard to monitor your deployment:

```
http://localhost:3000/deployments
```

### Important: Your Application Must Listen on the PORT Environment Variable

**Your deployed application MUST read the `PORT` environment variable and listen on that port.** The platform sets `PORT=8080` in the container, and your code needs to use this value.

**Example for different languages:**

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

- If you see "port is already allocated" errors, the system will automatically retry with a new random port
- Check container logs: `docker logs <container-name>`
- Verify the demo_api is listening on the PORT environment variable (should be 8080)
- Make sure your Dockerfile exposes the correct port (8080 in the demo_api example)
