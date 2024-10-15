import { createRoute, OpenAPIHono } from "@hono/zod-openapi";
import type {
  ExecutionContext,
  Env as HonoEnv,
  Context as GenericContext,
  Schema,
  MiddlewareHandler,
} from "hono";
import { createFactory, createMiddleware } from "hono/factory";
import { handleZodError } from "./errors";
import { Ratelimit } from "@unkey/ratelimit";
import { verifyKey } from "@unkey/api";

export type UnkeyBindings = {
  ratelimit: {
    limit: (identifier: string) => Promise<{ success: boolean }>;
  };
  auth: {
    // returns early if unauthorized
    verifyKey: () => Promise<{ identity?: { id: string; externalId: string } }>;
  };

  // we should not load this via env but rather inject a `secrets.ts` file or something during build
  UNKEY_ROOT_KEY: string;
};

export type Env = {
  Variables: HonoEnv["Variables"];

  Bindings: HonoEnv["Bindings"] & UnkeyBindings;
};
export interface Deployable<TEnv extends Env> {
  fetch: (req: Request, env: TEnv["Bindings"], ctx: ExecutionContext) => Promise<Response>;
}

export type Context = GenericContext<Env>;

export class Router<TEnv extends Env = Env, S extends Schema = {}, BasePath extends string = "/">
  implements Deployable<TEnv> {
  private readonly hono: OpenAPIHono<TEnv, S, BasePath>;
  private bindingsReady = false;

  constructor(opts?: { openapi?: { name: string; url: string } }) {
    this.hono = new OpenAPIHono<TEnv, S, BasePath>({
      defaultHook: handleZodError,
    });

    this.hono.doc("/openapi.json", {
      openapi: "3.0.0",
      info: {
        title: opts?.openapi?.name ?? "Unnamed",
        version: "1.0.0",
      },

      servers: opts?.openapi
        ? [
          {
            url: opts.openapi.url,
            description: "Production",
          },
        ]
        : [],
    });
  }

  async fetch(request: Request, env: TEnv["Bindings"], ctx: ExecutionContext): Promise<Response> {
    if (!this.bindingsReady) {
      env.ratelimit = new Ratelimit({
        namespace: "deploy-demo",
        rootKey: env.UNKEY_ROOT_KEY,
        limit: 10,
        duration: "60s",
      });
      this.bindingsReady = true;
    }
    return await this.hono.fetch(request, env, ctx);
  }

  get createRoute() {
    return createRoute;
  }

  get register() {
    return this.hono.openapi;
  }
}

export const withAuth = createMiddleware<{
  Variables: {
    identity: string;
  };
}>(async (c, next) => {
  // pretend we're doing soemthing
  c.set("identity", "id_123");
  return next();
});
