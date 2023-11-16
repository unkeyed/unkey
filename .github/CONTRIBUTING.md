### Services

There are a few 3rd party services that are required to run the app:

Required

- [Planetscale](https://planetscale.com?ref=unkey): Database
- [Clerk](https://clerk.com?ref=unkey): Authentication

Optional

- [Tinybird](https://www.tinybird.co?ref=unkey): Time series database
- [Upstash Kafka](https://upstash.com/kafka?ref=unkey): Cache invalidation

### Tools

- [Docker](https://www.docker.com/?ref=unkey)
- [pnpm](https://pnpm.io/installation/?ref=unkey)

### Setup

[Unkey Repo](https://github.com/unkeyed/unkey)

Set environment variables in `/apps/web/.env` and/or `/apps/agent/.env` respectively and populate the values from the services above.:

```sh-session
cp apps/web/.env.example apps/web/.env
cp apps/agent/.env.example apps/agent/.env
```

For Planetscale the following variables will be used after you create your database:

```
DATABASE_HOST=aws.connect.psdb.cloud
DATABASE_USERNAME=...
DATABASE_PASSWORD=pscale_pw_....
```

### 0.2 Install

```sh-session
pnpm install
```

### 1. Prepare databases

Push the database schema to Planetscale:

> Make sure you replace `user`, `password`, `host` and `db` with your own values

```sh-session
cd internal/db
DRIZZLE_DATABASE_URL='mysql://{user}:{password}@{host}/{db}?ssl={"rejectUnauthorized":true}' pnpm drizzle-kit push:mysql
```

### 2. Clerk

Create a new application via their [dashboard](https://dashboard.clerk.com?ref=unkey).

Once you have created the application, you need to create a single user from the UI and then enable and create an organization.

> You need the organization ID for step 3


### 3. Bootstrap Unkey

Unkey uses itself to manage its own API keys. To bootstrap the app, run the following command:

You need to provide the database credentials as well as the organization ID from clerk as `TENANT_ID`

```sh-session

export DATABASE_HOST=
export DATABASE_USERNAME=
export DATABASE_PASSWORD=
export TENANT_ID=org_xxx

pnpm bootstrap
```

This sets up the workspace and gets everything ready to run the app.

## Build

```sh-session
pnpm build
```

## Run API

Add a `.env` file in `/apps/agent/.env` and populate the values from the services above.:

```sh-session
cp apps/agent/.env.example apps/agent/.env
```

Then run the api via the agent:

```sh-session
cd apps/agent
go run . agent --env ./.env
```

## Run app

```sh-session
pnpm turbo run dev --filter=web
```

## Run api

```sh-session
pnpm turbo run dev --filter=api
```

### 4. Tinybird (Optional)

Download the Tinybird CLI from [here](https://www.tinybird.co/docs/cli.html) and run the following command after authenticating:

```sh-session
cd packages/tinybird
tb push ./*.datasource
tb push
```

Add your auth token to the `.env` file in `/apps/agent/.env` and `/apps/web/.env` respectively:

```sh-session
TINYBIRD_TOKEN=
```
