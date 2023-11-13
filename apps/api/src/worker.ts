import { Env } from "@/pkg/env";
import { handleError } from "@/pkg/errors/http";
import { init, logger, metrics } from "@/pkg/global";
import { app } from "@/pkg/hono/app";
import { prettyJSON } from "hono/pretty-json";
import { newId } from "./pkg/id";
import { Metric } from "./pkg/metrics";
import { registerV1KeysDeleteKey } from "./routes/v1_keys_deleteKey";
import { registerV1KeysGetKey } from "./routes/v1_keys_getKey";
import { registerV1KeysVerifyKey } from "./routes/v1_keys_verifyKey";
import { registerV1Liveness } from "./routes/v1_liveness";

export { DurableObjectRatelimiter } from "@/pkg/ratelimit/durable_object";
export { DurableObjectUsagelimiter } from "@/pkg/usagelimit/durable_object";

app.onError(handleError);
app.use(prettyJSON());

app.doc("/openapi.json", {
  openapi: "3.0.0",
  info: {
    title: "Unkey Api",
    version: "1.0.0",
  },
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

registerV1Liveness(app);
registerV1KeysGetKey(app);
registerV1KeysDeleteKey(app);
registerV1KeysVerifyKey(app);

export default {
  fetch: (req: Request, env: Env["Bindings"], executionCtx: ExecutionContext) => {
    init({ env });

    return app.fetch(req, env, executionCtx);
  },
};
