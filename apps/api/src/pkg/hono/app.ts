import { OpenAPIHono } from "@hono/zod-openapi";
import { prettyJSON } from "hono/pretty-json";
import { handleError, handleZodError } from "../errors";
import type { HonoEnv } from "./env";

export function newApp() {
  const app = new OpenAPIHono<HonoEnv>({
    defaultHook: handleZodError,
  });

  app.use(prettyJSON());
  app.onError(handleError);

  app.use("*", (c, next) => {
    c.set(
      "location",
      c.req.header("True-Client-IP") ??
        c.req.header("CF-Connecting-IP") ??
        // @ts-expect-error - the cf object will be there on cloudflare
        c.req.raw?.cf?.colo ??
        "",
    );
    c.set("userAgent", c.req.header("User-Agent"));

    return next();
  });

  app.doc("/openapi.json", {
    openapi: "3.0.0",
    info: {
      title: "Unkey Api",
      version: "1.0.0",
    },

    servers: [
      {
        url: "http://localhost:8787",
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
