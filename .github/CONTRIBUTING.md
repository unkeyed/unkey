
### Services

There are a few 3rd party services that are required to run the app:

- [Planetscale](https://planetscale.com?ref=unkey): Database
- [Tinybird](https://www.tinybird.co?ref=unkey): Time series database
- [Upstash Kafka](https://upstash.com/kafka?ref=unkey): Cache invalidation
- [Clerk](https://clerk.com?ref=unkey): Authentication

Set environment variables in `/apps/web/.env` and/or `/apps/api/.env` respectively and populate the values from the services above.:

```sh-session
cp apps/web/.env.example apps/web/.env
cp apps/api/.env.example apps/api/.env
```

### 0. Install

```sh-session
pnpm install
```

### 1. Prepare databases

Push the database schema to Planetscale:

```sh-session
cd packages/db
DRIZZLE_DATABASE_URL='mysql://{user}:{password}{host}/{db}?ssl={"rejectUnauthorized":true}' pnpm drizzle-kit push:mysql
```
### 2. Tinybird

Download the Tinybird CLI from [here](https://www.tinybird.co/docs/cli.html) and run the following command after authenticating:

```sh-session
cd packages/tinybird
tb push datasources/
tb push
```


### 3. Clerk

Create a new app and set it up as described [here](https://clerk.com/docs/nextjs/get-started-with-nextjs).
Alternatively, ask on our [discord](https://unkey.dev/discord) for temporary credentials.

Afterwards, create a new organization in clerk and make a note of the organization ID, you'll need it in the next step.

### 4. Bootstrap Unkey

Unkey uses itself to manage its own API keys. To bootstrap the app, run the following command:

You need to provide the database credentials as well as the organization ID from clerk as `TENANT_ID`

```sh-session

export DATABASE_HOST=
export DATABASE_USERNAME=
export DATABASE_PASSWORD=
export TENANT_ID=

pnpm bootstrap
```

This sets up the workspace and gets everything ready to run the app.


## Build

```sh-session
pnpm build
```

## Run app

```sh-session
pnpm turbo run dev --filter=web
```


## Run api

```sh-session
pnpm turbo run dev --filter=api
```
