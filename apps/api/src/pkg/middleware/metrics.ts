import { sha256 } from "@unkey/hash";
import type { Metric } from "@unkey/metrics";
import type { Context, MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";

type DiscriminateMetric<T, M = Metric> = M extends { metric: T } ? M : never;

export function metrics(): MiddlewareHandler<HonoEnv> {
  return async (c, next) => {
    const { metrics, analytics } = c.get("services");

    let requestBody = await c.req.raw.clone().text();
    requestBody = requestBody.replaceAll(/"key":\s*"[a-zA-Z0-9_]+"/g, '"key": "<REDACTED>"');
    requestBody = requestBody.replaceAll(
      /"plaintext":\s*"[a-zA-Z0-9_]+"/g,
      '"plaintext": "<REDACTED>"',
    );
    const start = performance.now();
    const m = {
      isolateId: c.get("isolateId"),
      isolateLifetime: Date.now() - c.get("isolateCreatedAt"),
      metric: "metric.http.request",
      path: c.req.path,
      host: new URL(c.req.url).host,
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
      fromAgent: c.req.header("Unkey-Redirect"),
      context: {},
    } as DiscriminateMetric<"metric.http.request">;

    try {
      // const telemetry = {
      //   runtime: c.req.header("Unkey-Telemetry-Runtime"),
      //   platform: c.req.header("Unkey-Telemetry-Platform"),
      //   versions: c.req.header("Unkey-Telemetry-SDK")?.split(","),
      // };
      // if (
      //   telemetry.runtime &&
      //   telemetry.platform &&
      //   telemetry.versions &&
      //   telemetry.versions.length > 0
      // ) {
      //   const event = {
      //     runtime: telemetry.runtime || "unknown",
      //     platform: telemetry.platform || "unknown",
      //     versions: telemetry.versions || [],
      //     requestId: c.get("requestId"),
      //     time: Date.now(),
      //   };

      //   c.executionCtx.waitUntil(
      //     analytics
      //       .insertSdkTelemetry({
      //         ...event,
      //         request_id: event.requestId,
      //       })
      //       .catch((err) => {
      //         logger.error("Error inserting SDK telemetry", {
      //           method: c.req.method,
      //           path: c.req.path,
      //           error: err.message,
      //           telemetry,
      //           event,
      //         });
      //       }),
      //   );
      // }

      await next();
    } catch (e) {
      m.error = (e as Error).message;
      c.get("services").logger.error("request", {
        method: c.req.method,
        path: c.req.path,
        error: e,
      });
      throw e;
    } finally {
      m.status = c.res.status;
      m.context = c.get("metricsContext") ?? {};
      m.serviceLatency = performance.now() - start;
      c.res.headers.append("Unkey-Latency", `service=${m.serviceLatency}ms`);
      c.res.headers.append("Unkey-Version", c.env.VERSION);
      metrics.emit(m);
      c.executionCtx.waitUntil(metrics.flush());

      const responseHeaders: Array<string> = [];
      c.res.headers.forEach((v, k) => {
        responseHeaders.push(`${k}: ${v}`);
      });

      let responseBody = await c.res.clone().text();
      responseBody = responseBody.replaceAll(/"key":\s*"[a-zA-Z0-9_]+"/g, '"key": "<REDACTED>"');
      responseBody = responseBody.replaceAll(
        /"plaintext":\s*"[a-zA-Z0-9_]+"/g,
        '"plaintext": "<REDACTED>"',
      );

      c.executionCtx.waitUntil(
        analytics.insertApiRequest({
          request_id: c.get("requestId"),
          time: c.get("requestStartedAt"),
          workspace_id: await getWorkspaceId(c),
          host: new URL(c.req.url).host,
          method: c.req.method,
          path: c.req.path,
          request_headers: Object.entries(c.req.header()).map(([k, v]) => {
            if (k.toLowerCase() === "authorization") {
              return `${k}: <REDACTED>`;
            }
            return `${k}: ${v}`;
          }),
          request_body: requestBody,
          response_status: c.res.status,
          response_headers: responseHeaders,
          response_body: responseBody,
          error: m.error ?? "",
          service_latency: Date.now() - c.get("requestStartedAt"),
          ip_address: c.req.header("True-Client-IP") ?? c.req.header("CF-Connecting-IP") ?? "",
          user_agent: c.req.header("User-Agent") ?? "",
          // @ts-ignore - this is a bug in the types
          continent: c.req.raw?.cf?.continent,
          // @ts-ignore - this is a bug in the types
          country: c.req.raw?.cf?.country,
          // @ts-ignore - this is a bug in the types
          colo: c.req.raw?.cf?.colo,
          // @ts-ignore - this is a bug in the types
          city: c.req.raw?.cf?.city,
        }),
      );
    }
  };
}

async function getWorkspaceId(c: Context<HonoEnv>): Promise<string> {
  const workspaceId = c.get("workspaceId");
  if (workspaceId) {
    return workspaceId;
  }
  const rootKey = c.req.header("authorization")?.replace("Bearer ", "");
  if (!rootKey) {
    return "";
  }
  const hash = await sha256(rootKey);
  const { cache, db, logger } = c.get("services");
  const { val, err } = await cache.workspaceIdByRootKeyHash.swr(hash, async () => {
    const key = await db.readonly.query.keys.findFirst({
      where: (table, { eq, and, isNull }) => and(isNull(table.deletedAt), eq(table.hash, hash)),
    });
    if (!key) {
      return null;
    }
    return key.forWorkspaceId!;
  });

  if (err) {
    logger.error("unable to get root key from hash", {
      error: err.message,
    });
    return "";
  }
  return val ?? "";
}
