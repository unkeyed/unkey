# Unkey Limited (Remaining) API Key Demo

This example shows off how one might use limited keys in a product. This Demo is a chat app built with [Vercel AI](https://sdk.vercel.ai), [Open AI](platform.openai.com). It shows off both methods of using Unkey. Programtically via the SDK and via HTTP request.

It also uses [Shadcn/ui](https://ui.shadcn.com/) for the toast, [Prisma](https://prisma.io) for user creation and [T3 Env](https://env.t3.gg/).

## Getting Started

Copy the environment variables via:

```bash
cp .env.example .env.local
```

- Go to [unkey.dev](https://unkey.dev/app) and create a new API.
- Copy your workspace token and `apiId` to `.env.local`
- Go to the OpenAI dashboard and create a new token and copy to `.env.local`

## Install Dependencies

Install dependencies via:

```bash
npm install
# or
yarn install
# or
pnpm install
```

## Run migrations

To run migrations run:

```bash
npx prisma db push
```

This will create `prisma/dev.db`, a local SQLITE DB and generate the client.

## Start the example

You can start the example via:

```bash
npm run dev
```

> **NOTE** If you attempt to run the dev server without specifying all necessary variables, it'll through a run-time error.

## Troubleshooting

You might get a `429` error code when you attempt a chat completion, this comes from OpenAI and basically means your free trial is over and need to input billing details for a paid plan.
