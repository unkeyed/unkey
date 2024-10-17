import { OpenAPIHono, createRoute } from "@hono/zod-openapi";
import { verifyKey } from "@unkey/api";
import { Ratelimit } from "@unkey/ratelimit";
import type { ExecutionContext, Context as GenericContext, Env as HonoEnv, Schema } from "hono";
import { createMiddleware } from "hono/factory";
import { handleZodError } from "./errors";

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
  Variables: HonoEnv["Variables"] & {
    identity?: { id: string; externalId: string; meta?: Record<string, unknown> };
  };

  Bindings: HonoEnv["Bindings"] & UnkeyBindings;
};
export interface Deployable<TEnv extends Env> {
  fetch: (req: Request, env: TEnv["Bindings"], ctx: ExecutionContext) => Promise<Response>;
}

export type Context = GenericContext<Env>;

export class Router<TEnv extends Env = Env, S extends Schema = {}, BasePath extends string = "/">
  implements Deployable<TEnv>
{
  private readonly hono: OpenAPIHono<TEnv, S, BasePath>;
  private bindingsReady = false;
  private openapiSpec: ReturnType<typeof this.hono.getOpenAPIDocument>;

  constructor(opts?: { openapi?: { name: string; url: string } }) {
    this.hono = new OpenAPIHono<TEnv, S, BasePath>({
      defaultHook: handleZodError,
    });

    const openapiConfig = {
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
    };

    this.hono.doc("/openapi.json", openapiConfig);

    this.hono.use(async (c, next) => {
      this.openapiSpec = this.hono.getOpenAPIDocument(openapiConfig);
      for (const matched of c.req.matchedRoutes) {
        try {
          const operation = this.openapiSpec.paths[matched.path][matched.method.toLowerCase()];

          if (operation.security) {
            console.warn(operation.security);
            for (const security of operation.security) {
              if (security.type.includes("apiKey")) {
                const header = c.req.header("Authorization");
                if (!header) {
                  return c.json(
                    {
                      message: "Authorization header missing",
                      error: "Unauthenticated",
                    },
                    { status: 401 },
                  );
                }
                const bearer = header.trim().replaceAll("Bearer ", "").trim();
                const res = await verifyKey(bearer);
                console.info("res", res);
                if (res.error) {
                  return c.json(
                    {
                      error: res.error.message,
                    },
                    { status: 403 },
                  );
                }
                if (!res.result.valid) {
                  return c.json({ error: "Unauthorized" }, { status: 403 });
                }
                c.set("identity", res.result.identity);
                return next();
              }
            }
          }
        } catch (err) {
          console.error({
            message: "shit hit the fan",
            error: JSON.stringify(err.message),
          });
        }
      }
      return next();
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
  c.set("identity", "id_123");
  return next();
});
