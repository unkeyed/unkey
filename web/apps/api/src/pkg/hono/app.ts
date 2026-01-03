import { OpenAPIHono } from "@hono/zod-openapi";
import type { Context as GenericContext } from "hono";
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
      title: "Unkey API",
      version: "1.0.0",
    },

    servers: [
      {
        url: "https://api.unkey.dev",
        description: "Production",
      },
    ],

    "x-speakeasy-retries": {
      strategy: "backoff",
      backoff: {
        initialInterval: 50, // 50ms
        maxInterval: 1_000, // 1s
        maxElapsedTime: 30_000, // 30s
        exponent: 1.5,
      },
      statusCodes: ["5XX"],
      retryConnectionErrors: true,
    },
  });

  app.openAPIRegistry.registerComponent("securitySchemes", "bearerAuth", {
    bearerFormat: "root key",
    type: "http",
    scheme: "bearer",
    "x-speakeasy-example": "UNKEY_ROOT_KEY",
  });
  return app;
}

export type App = ReturnType<typeof newApp>;
export type Context = GenericContext<HonoEnv>;
