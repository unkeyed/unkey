import { drizzle } from "drizzle-orm/planetscale-serverless";

import { Client } from "@planetscale/database";
import * as schema from "./schemas";

export const db = drizzle(
  new Client({
    host: process.env.MARKETING_DATABASE_HOST!,
    username: process.env.MARKETING_DATABASE_USERNAME!,
    password: process.env.MARKETING_DATABASE_PASSWORD!,

    fetch: (url: string, init: any) => {
      (init as any).cache = undefined; // Remove cache header
      return fetch(url, init);
    },
  }),
  {
    schema,
  },
);
