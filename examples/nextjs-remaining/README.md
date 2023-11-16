This example is a [Next.js](https://nextjs.org/) project bootstrapped with [`create-next-app`](https://github.com/vercel/next.js/tree/canary/packages/create-next-app). The example has a single page at the root to generate api keys that expire in one minute and an api route at `/api/forecast` to retrieve a fake 7 day weather forecast.

## Environment Variables

The examples only requires two environment variables.

- `UNKEY_ROOT_KEY` - One can be created at [here](`https://unkey.dev/app/settings/root-keys`).
- `UNKEY_API_ID` - One can be created at [here](`https://unkey.dev/app/apis`). After selecting the api the id can be found in the top right.

Configure these environments variables in a `.env` file.

## Walkthrough

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Then navigate to [http://localhost:3000](http://localhost:3000) to generate an api key. The key will expire in one minute. The key will appear underneath the `Create Key` button.

Then using your favorite http client make a GET request to http://localhost:3000/api/forecast with the Authorization header set to `Bearer <NEW_API_KEY>`. Continue making requests. After a minute you should see a 401 response due to the key expiring.

## Code

The code for creating apis keys is in `src/server/unkey-client.ts`. There is a function called `createApiKey` that accepts a name and expiration time in milliseconds. It uses the unkey client to create a new key and returns the key.

The verification happens in the route handler at `/app/api/forecast/route.ts`. It simple checks for the Authorization header and checks if the key is valid using `verifyKey`. All the logic of checking the expiration time is handled by unkey.
