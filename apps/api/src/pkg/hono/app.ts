import type { Env } from "@/pkg/env";
import { OpenAPIHono } from "@hono/zod-openapi";
import { prettyJSON } from "hono/pretty-json";
import { handleError, handleZodError } from "../errors";

export type Variables = {
  requestId: string;
};
export function newApp() {
  const app = new OpenAPIHono<{ Bindings: Env["Bindings"]; Variables: Variables }>({
    defaultHook: handleZodError,
  });

  app.onError(handleError);
  app.use(prettyJSON());

  app.doc("/openapi.json", {
    openapi: "3.0.0",
    info: {
      title: "Unkey Api",
      version: "1.0.0",
    },
  });

  return app;
}

export type App = ReturnType<typeof newApp>;
