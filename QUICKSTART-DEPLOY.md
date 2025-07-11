# Quickstart Guide

This guide will help you get the Unkey deployment platform up and running locally for development and testing.

## Prerequisites

- Docker and Docker Compose
- A terminal/command line

## Step 1: Start the Platform

1. Start all services using Docker Compose:

```bash
docker-compose up -d
```

This will start:

- MySQL database (port 3306)
- Dashboard (port 3000)
- Control plane services
- Supporting infrastructure

2. Wait for all services to be healthy (this may take 1-2 minutes):

```bash
docker-compose ps
```

## Step 2: Set Up Your Workspace

1. Open your browser and navigate to the dashboard:

```
http://localhost:3000
```

2. Sign in or create an account through the authentication flow

3. Once logged in, you'll automatically have a workspace created. Navigate to:

```
http://localhost:3000/projects
```

4. Create a new project by filling out the form:

   - **Name**: Choose any name (e.g., "My Test App")
   - **Slug**: This will auto-generate based on the name
   - **Git URL**: Optional, leave blank for testing

5. After creating the project, **copy the Project ID** from the project details. It will look like:

```
proj_xxxxxxxxxxxxxxxxxx
```

6. Also note your **Workspace ID** (you can find this settings). It will look like:

```
ws_xxxxxxxxxxxxxxxxxx
```

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

3. The CLI will show real-time progress as your deployment goes through these stages:
   - Downloading Docker image
   - Building rootfs
   - Uploading rootfs
   - Creating VM
   - Booting VM
   - Assigning domains
   - Completed

## Step 4: View Your Deployment

1. Return to the dashboard and navigate to:

```
http://localhost:3000/versions
http://localhost:3000/deployments
```

