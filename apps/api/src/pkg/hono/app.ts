import type { Env } from "@/pkg/env";
import { OpenAPIHono } from "@hono/zod-openapi";
import { prettyJSON } from "hono/pretty-json";
import { handleError, handleZodError } from "../errors";
import { SECURITY_SCHEME_NAME } from "./openapi/create-auth-route";

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
  });

  app.openAPIRegistry.registerComponent("securitySchemes", SECURITY_SCHEME_NAME, {
    type: "http",
    scheme: "bearer",
  });

  return app;
}

export type App = ReturnType<typeof newApp>;
