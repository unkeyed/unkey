import { OpenAPIHono } from "@hono/zod-openapi";
import { prettyJSON } from "hono/pretty-json";
import { handleError, handleZodError } from "../errors";
import { HonoEnv } from "./env";

export function newApp() {
  const app = new OpenAPIHono<HonoEnv>({
    defaultHook: handleZodError,
  });

  app.use(prettyJSON());
  app.onError(handleError);

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
