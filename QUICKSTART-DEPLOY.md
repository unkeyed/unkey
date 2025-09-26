# Quickstart Guide

This guide will help you get the Unkey deployment platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- Go 1.24 or later
- A terminal/command line

## Step 1: Start the Platform

1. Change to the deployment directory:

```bash
cd deployment
```

2. Create an `.env` file in `apps/dashboard` (use .env.example as a template):

```bash
cp apps/dashboard/.env.example apps/dashboard/.env
```

Follow the instructions in .env.example to fill the values.


3. Start services using Docker Compose:

```bash
docker compose up -d metald-aio dashboard ctrl
```


If the dashboard build fails in Docker, run it locally:

```bash
docker compose up planetscale agent 
cd apps/dashboard 
pnpm i
pnpm dev
```

4. Wait for all services to be healthy

The platform now uses a Docker backend that creates containers instead of VMs, making it much faster and easier to run locally.

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

2. Create a version using the CLI with your copied IDs:

```bash
go run . version create \
  --context=./demo_api \
  --workspace-id=YOUR_WORKSPACE_ID \
  --project-id=YOUR_PROJECT_ID
```

Keep the context as shown, there's a demo api in that folder.
Replace `YOUR_WORKSPACE_ID` and `YOUR_PROJECT_ID` with the actual values you copied from the dashboard.

3. The CLI will:
   - Always build a fresh Docker image from your code
   - Set the PORT environment variable to 8080 in the container
   - Use the Docker backend to create a container instead of a VM
   - Automatically allocate a random host port (e.g., 35432) to avoid conflicts
   - Show real-time progress as your deployment goes through the stages

## Step 4: View Your Deployment

1. Once the deployment completes, the CLI will show you the available domains:

```
Deployment Complete
  Version ID: v_xxxxxxxxxxxxxxxxxx
  Status: Ready
  Environment: Production

Domains
  https://main-commit-workspace.unkey.app
  http://localhost:35432
```

2. If you're using the `demo_api` you can curl the `/v1/liveness` endpoint
3. Return to the dashboard and navigate to:

```
http://localhost:3000/versions
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
