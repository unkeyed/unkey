import { eq, schema } from "@unkey/db";

import { Client } from "@planetscale/database";
import { drizzle } from "drizzle-orm/planetscale-serverless";
import type { Subscriptions } from "../../internal/billing/src";

async function main() {
  const db = drizzle(
    new Client({
      host: process.env.DATABASE_HOST,
      username: process.env.DATABASE_USERNAME,
      password: process.env.DATABASE_PASSWORD,
    }),
    {
      schema,
    },
  );

  const workspaces = await db.query.workspaces.findMany({
    where: (table, { and, isNull, isNotNull }) =>
      and(isNull(table.deletedAt), isNotNull(table.subscriptions)),
    columns: {
      id: true,
      name: true,
      stripeCustomerId: true,
      subscriptions: true,
    },
  });

  for (const ws of workspaces) {
    console.log(ws.name);

    const subscriptions = {
      ...ws.subscriptions,
      ratelimits: {
        productId: "prod_PpiPuVkph7t9fI",
        tiers: [
          {
            firstUnit: 1,
            lastUnit: 2_500_000,
            centsPerUnit: null,
          },
          {
            firstUnit: 2_500_001,
            lastUnit: null,
            centsPerUnit: "0.001", // $0.00001 per ratelimit or  $1 per 100k verifications
          },
        ],
      },
    } satisfies Subscriptions;

    console.log("OLD", JSON.stringify(ws.subscriptions));
    console.log("NEW", JSON.stringify(subscriptions));

    await db
      .update(schema.workspaces)
      .set({
        subscriptions,
      })
      .where(eq(schema.workspaces.id, ws.id));
  }
}

main();
