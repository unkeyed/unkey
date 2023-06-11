import { Hono } from "hono";
import { handle } from "hono/vercel";

import { db } from "@unkey/db";
import { init } from "@/lib/api/router";

export const config = {
  runtime: "edge",
};

const router = init(new Hono(), { db });

export default handle(router);
