// This example has relative imports to do type checks, you need to import from "@unkey/cache"
import { DefaultStatefulContext, Namespace, createCache } from ".."; // @unkey/cache
import { MemoryStore } from "../stores"; // @unkey/cache/stores

/**
 * In serverless you'd get this from the request handler
 * See https://unkey.com/docs/libraries/ts/cache/overview#context
 */
const ctx = new DefaultStatefulContext();

/**
 * Define the type of your data, or perhaps generate the types from your database
 */
type User = {
  id: string;
  email: string;
};

const memory = new MemoryStore({ persistentMap: new Map() });

const userNamespace = new Namespace<User>(ctx, {
  stores: [memory],
  fresh: 60_000, // Data is fresh for 60 seconds
  stale: 300_000, // Data is stale for 300 seconds
});

const cache = createCache({ user: userNamespace });

async function main() {
  await cache.user.set("userId", { id: "userId", email: "user@email.com" });

  const user = await cache.user.get("userId");

  console.info(user);

  const users = await cache.user.getMany(["userId", "userId2"]);

  if (users.val) {
    const { userId, userId2 } = users.val;

    console.info({ userId, userId2 });
  }

  await cache.user.setMany({
    userId1: { id: "userId1", email: "user@email.com" },
    userId2: { id: "userId2", email: "user@email.com" },
  });

  if (users.val) {
    const { userId, userId2 } = users.val;

    console.info({ userId, userId2 });
  }

  const usersSwr = await cache.user.swrMany(["userId", "userId2"], async (_) => {
    return {
      userId: {
        email: "user@email.com",
        id: "userId",
      },
    };
  });

  if (usersSwr.val) {
    const { userId, userId2 } = usersSwr.val;

    console.info({ userId, userId2 });
  }
}

main();
