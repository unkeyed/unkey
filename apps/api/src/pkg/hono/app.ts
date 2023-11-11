import type { Env } from "@/pkg/env";
import { OpenAPIHono } from "@hono/zod-openapi";
import { handleZodError } from "../errors";

export type Variables = {
  requestId: string;
};
export const app = new OpenAPIHono<{ Bindings: Env["Bindings"]; Variables: Variables }>({
  defaultHook: handleZodError,
});
export type App = typeof app;
