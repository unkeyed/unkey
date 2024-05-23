import { type Env, zEnv } from "@/pkg/env";
import { newApp } from "@/pkg/hono/app";

import { registerLegacyApisListKeys } from "./routes/legacy_apis_listKeys";
import { registerV1ApisCreateApi } from "./routes/v1_apis_createApi";
import { registerV1ApisDeleteApi } from "./routes/v1_apis_deleteApi";
import { registerV1ApisGetApi } from "./routes/v1_apis_getApi";
import { registerV1ApisListKeys } from "./routes/v1_apis_listKeys";
import { registerV1KeysCreateKey } from "./routes/v1_keys_createKey";
import { registerV1KeysDeleteKey } from "./routes/v1_keys_deleteKey";
import { registerV1KeysGetKey } from "./routes/v1_keys_getKey";
import { registerV1KeysGetVerifications } from "./routes/v1_keys_getVerifications";
import { registerV1KeysUpdate } from "./routes/v1_keys_updateKey";
import { registerV1KeysUpdateRemaining } from "./routes/v1_keys_updateRemaining";
import { registerV1KeysVerifyKey } from "./routes/v1_keys_verifyKey";
import { registerV1Liveness } from "./routes/v1_liveness";
import { registerV1RatelimitLimit } from "./routes/v1_ratelimit_limit";

// import { instrument } from "@microlabs/otel-cf-workers";
// Legacy Routes
import { registerLegacyKeysCreate } from "./routes/legacy_keys_createKey";
import { registerLegacyKeysVerifyKey } from "./routes/legacy_keys_verifyKey";

// Export Durable Objects for cloudflare
export { DurableObjectRatelimiter } from "@/pkg/ratelimit/durable_object";
export { DurableObjectUsagelimiter } from "@/pkg/usagelimit/durable_object";
import { cors, init, metrics } from "@/pkg/middleware";
import { ConsoleLogger } from "./pkg/logging";
// import { traceConfig } from "./pkg/tracing/config";
import { registerV1MigrationsCreateKeys } from "./routes/v1_migrations_createKey";

const app = newApp();

app.use("*", init());
app.use("*", cors());
app.use("*", metrics());

app.use("*", async (c, next) => {
  try {
    if (c.env.TINYBIRD_PROXY_URL) {
      const start = performance.now();
      const p = fetch(new URL("/v0/incr", c.env.TINYBIRD_PROXY_URL), {
        method: "POST",
      }).then(() => {
        const { metrics } = c.get("services");

        metrics.emit({
          metric: "metric.koyeb.latency",
          // @ts-expect-error
          continent: c.req.raw?.cf?.continent,
          // @ts-expect-error
          colo: c.req.raw?.cf?.colo,
          latency: performance.now() - start,
        });
      });

      c.executionCtx.waitUntil(p);
    }
  } catch (e) {
    console.error(e);
  }

  return next();
});

/**
 * Registering all route handlers
 */

// misc
registerV1Liveness(app);

// keys
registerV1KeysGetKey(app);
registerV1KeysDeleteKey(app);
registerV1KeysCreateKey(app);
registerV1KeysVerifyKey(app);
registerV1KeysUpdate(app);
registerV1KeysUpdateRemaining(app);
registerV1KeysGetVerifications(app);

// apis
registerV1ApisGetApi(app);
registerV1ApisCreateApi(app);
registerV1ApisListKeys(app);
registerV1ApisDeleteApi(app);

// ratelimit
registerV1RatelimitLimit(app);

// migrations
registerV1MigrationsCreateKeys(app);

// legacy REST style routes
registerLegacyKeysCreate(app);
registerLegacyKeysVerifyKey(app);
registerLegacyApisListKeys(app);

const handler = {
  fetch: (req: Request, env: Env, executionCtx: ExecutionContext) => {
    const parsedEnv = zEnv.safeParse(env);
    if (!parsedEnv.success) {
      new ConsoleLogger({ requestId: "" }).fatal(`BAD_ENVIRONMENT: ${parsedEnv.error.message}`);
      return Response.json(
        {
          code: "BAD_ENVIRONMENT",
          message: "Some environment variables are missing or are invalid",
          errors: parsedEnv.error,
        },
        { status: 500 },
      );
    }

    return app.fetch(req, parsedEnv.data, executionCtx);
  },
} satisfies ExportedHandler<Env>;

export default handler;
