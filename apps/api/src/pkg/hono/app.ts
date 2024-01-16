import type { Env } from "@/pkg/env";
import { OpenAPIHono } from "@hono/zod-openapi";
import { prettyJSON } from "hono/pretty-json";
import { handleError, handleZodError } from "../errors";

export type HonoEnv = {
  Bindings: Env;
  Variables: {
    requestId: string;
  };
};
export function newApp() {
  const app = new OpenAPIHono<HonoEnv>({
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

    servers: [
      {
        url: "https://api.unkey.dev",
        description: "Production",
      },
    ],
  });

  app.openAPIRegistry.registerComponent("securitySchemes", "bearerAuth", {
    bearerFormat: "root key",
    type: "http",
    scheme: "bearer",
  });
  return app;
}

export type App = ReturnType<typeof newApp>;
