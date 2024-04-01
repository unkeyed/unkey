import { trace } from "@opentelemetry/api";

import type { MiddlewareHandler } from "hono";
import type { HonoEnv } from "../hono/env";

export function otel(): MiddlewareHandler<HonoEnv> {
  const tracer = trace.getTracer("hono", "0.0.1");

  return async (c, next) => {
    return tracer.startActiveSpan(`hono: ${c.req.method} ${c.req.path}`, async (span) => {
      const requestId = `req_${span.spanContext().traceId}`;
      c.set("requestId", requestId);
      c.res.headers.append("Unkey-Request-Id", requestId);

      span.setAttributes({
        "hono.request.path": c.req.path,
        "hono.request.method": c.req.method,
        "hono.request.url": c.req.url,
      });

      await next();

      if (c.error) {
        span.setStatus({ code: 2, message: c.error.message });
        span.recordException(c.error);
      }

      span.setAttributes({
        "hono.response.status_code": c.res.status,
      });

      span.end();
    });
  };
}
