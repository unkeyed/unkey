import { trace } from "@opentelemetry/api";
import { flatten } from "flat";

import type { MiddlewareHandler } from "hono";

interface OtelConfiguration {
  captureRequestBody?: boolean;
  captureResponseBody?: boolean;
}

export function otel(config: OtelConfiguration = {}): MiddlewareHandler {
  const tracer = trace.getTracer("@baselime/hono-js", "0.0.1");
  return async (c, next) => {
    return tracer.startActiveSpan("hono", async (span) => {
      if (config.captureRequestBody) {
        if (c.req.header("content-type") === "application/json") {
          const body = await c.req.json();
          span.setAttributes(flatten({ hono: { request: { body } } }));
        }
        if (
          ["application/x-www-form-urlencoded", "multipart/form-data"].includes(
            c.req.header("content-type") || "",
          )
        ) {
          const body = await c.req.parseBody();
          span.setAttributes(flatten({ hono: { request: { body } } }));
        }
        if (c.req.header("content-type") === "text/plain") {
          const body = await c.req.text();
          span.setAttributes(flatten({ hono: { request: { body } } }));
        }
      }

      span.setAttributes(
        flatten({
          hono: {
            request: {
              path: c.req.path,
              method: c.req.method,
              url: c.req.url,
            },
          },
        }),
      );

      await next();

      if (c.error) {
        span.setStatus({ code: 2, message: c.error.message });
        span.recordException(c.error);
      }

      span.setAttributes({
        "hono.response.status_code": c.res.status,
      });

      if (config.captureResponseBody) {
        if (c.res.headers.get("content-type") === "application/json") {
          const body = await c.res.json();
          span.setAttributes(flatten({ hono: { response: { body } } }));
        }
        if (c.res.headers.get("content-type") === "text/plain") {
          const body = await c.res.text();
          span.setAttributes(flatten({ hono: { response: { body } } }));
        }
        if (c.res.headers.get("content-type") === "text/html") {
          const body = await c.res.text();
          span.setAttributes(flatten({ hono: { response: { body } } }));
        }
      }
      span.end();
    });
  };
}
