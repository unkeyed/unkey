import type { Env } from "@/pkg/env";
import { OpenAPIHono } from "@hono/zod-openapi";
import { handleZodError } from "../errors";
import { GlobalContext } from "../context/global";

type Variables = {
  requestId: string;
  ctx: GlobalContext
};
export const app = new OpenAPIHono<{ Bindings: Env["Bindings"]; Variables: Variables }>({
  defaultHook: handleZodError,
});
export type App = typeof app;
