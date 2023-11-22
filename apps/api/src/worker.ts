import { Env, zEnv } from "@/pkg/env";
import { init, logger, metrics } from "@/pkg/global";
import { newApp } from "@/pkg/hono/app";
import { newId } from "./pkg/id";
import { Metric } from "./pkg/metrics";
import { registerV1ApisCreateApi } from "./routes/v1_apis_createApi";
import { registerV1ApisDeleteApi } from "./routes/v1_apis_deleteApi";
import { registerV1ApisGetApi } from "./routes/v1_apis_getApi";
import { registerV1ApisListKeys } from "./routes/v1_apis_listKeys";
import { registerV1KeysCreateKey } from "./routes/v1_keys_createKey";
import { registerV1KeysDeleteKey } from "./routes/v1_keys_deleteKey";
import { registerV1KeysGetKey } from "./routes/v1_keys_getKey";
import { registerV1KeysUpdateRemaining } from "./routes/v1_keys_updateRemaining";
import { registerV1KeysVerifyKey } from "./routes/v1_keys_verifyKey";
import { registerV1Liveness } from "./routes/v1_liveness";

// Legacy Routes
import { registerGetKey } from "./routes/keys_get";
import { registerCreateKey } from "./routes/keys_post";
import { registerVerifyKey } from "./routes/keys_verify";

export { DurableObjectRatelimiter } from "@/pkg/ratelimit/durable_object";
export { DurableObjectUsagelimiter } from "@/pkg/usagelimit/durable_object";

const app = newApp();

app.get("/routes", (c) => {
  return c.jsonT(
    app.routes.map((r) => ({
      method: r.method,
      path: r.path,
    })),
  );
});

app.use("*", async (c, next) => {
  logger.info("request", {
    method: c.req.method,
    path: c.req.path,
  });
  const start = performance.now();
  const m = {
    path: c.req.path,
    method: c.req.method,
    // @ts-ignore - this is a bug in the types
    continent: c.req.raw?.cf?.continent,
    // @ts-ignore - this is a bug in the types
    country: c.req.raw?.cf?.country,
    // @ts-ignore - this is a bug in the types
    colo: c.req.raw?.cf?.colo,
    // @ts-ignore - this is a bug in the types
    city: c.req.raw?.cf?.city,
    userAgent: c.req.header("user-agent"),
  } as Metric["metric.http.request"];
  try {
    const requestId = newId("request", 16);
    m.requestId = requestId;
    c.set("requestId", requestId);
    await next();
    // headers should be set after calling `next()`, otherwise they will be lowercased by the framework
    c.res.headers.append("Unkey-Request-Id", requestId);
  } catch (e) {
    m.error = (e as Error).message;
    throw e;
  } finally {
    m.status = c.res.status;
    m.serviceLatency = performance.now() - start;
    c.res.headers.append("Unkey-Latency", `service=${m.serviceLatency}ms`);
    metrics.emit("metric.http.request", m);
    c.executionCtx.waitUntil(Promise.all([metrics.flush(), logger.flush()]));
  }
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
registerV1KeysUpdateRemaining(app);

// apis
registerV1ApisGetApi(app);
registerV1ApisCreateApi(app);
registerV1ApisListKeys(app);
registerV1ApisDeleteApi(app);

// Legacy Routes
registerGetKey(app);
registerCreateKey(app);
registerVerifyKey(app);

export default {
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
    init({ env: parsedEnv.data });

    return app.fetch(req, parsedEnv.data, executionCtx);
  },
};
