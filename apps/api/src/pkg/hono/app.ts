import type { Env } from "@/pkg/env";
import { OpenAPIHono } from "@hono/zod-openapi";
import { handleZodError } from "../errors";

export type Variables = {
  requestId: string;
};
export function newHonoApp() {
  return new OpenAPIHono<{ Bindings: Env["Bindings"]; Variables: Variables }>({
    defaultHook: handleZodError,
  });
}
export type App = ReturnType<typeof newHonoApp>;
