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

import { instrument } from "@microlabs/otel-cf-workers";
// Legacy Routes
import { registerLegacyKeysCreate } from "./routes/legacy_keys_createKey";
import { registerLegacyKeysVerifyKey } from "./routes/legacy_keys_verifyKey";

// Export Durable Objects for cloudflare
export { DurableObjectRatelimiter } from "@/pkg/ratelimit/durable_object";
export { DurableObjectUsagelimiter } from "@/pkg/usagelimit/durable_object";
import { cors, init, metrics, otel } from "@/pkg/middleware";
import { traceConfig } from "./pkg/tracing/config";

const app = newApp();

app.use("*", (c, next) => {
  for (let i = 0; i < 10; i++) {
    console.log(JSON.stringify({ metric: "fake", url: c.req.url, i }));
  }
  return next();
});
app.use("*", init());
app.use("*", cors());
app.use(otel());
app.use("*", metrics());

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

// legacy REST style routes
registerLegacyKeysCreate(app);
registerLegacyKeysVerifyKey(app);
registerLegacyApisListKeys(app);

app.get("/routes", (c) => {
  return c.json(
    app.routes.map((r) => ({
      method: r.method,
      path: r.path,
    })),
  );
});

const handler = {
  fetch: (req: Request, env: Env, executionCtx: ExecutionContext) => {
    const parsedEnv = zEnv.safeParse(env);
    if (!parsedEnv.success) {
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

export default instrument(
  handler,
  traceConfig((env) => ({
    name: `api.${env.ENVIRONMENT}`,
    version: env.VERSION,
  })),
);
