import { ResolveConfigFn, ServiceConfig } from "@microlabs/otel-cf-workers";
import { Context, Span } from "@opentelemetry/api";
import { Env } from "../env";

export function traceConfig(service: (env: Env) => ServiceConfig): ResolveConfigFn {
  return (env: Env) => {
    if (!env.AXIOM_TOKEN) {
      return {
        service: service(env),
        spanProcessors: {
          forceFlush: () => Promise.resolve(),
          onStart: (_span: Span, _parentContext: Context) => {},
          onEnd: (_span) => {},
          shutdown: () => Promise.resolve(),
        },
      };
    }
    return {
      exporter: {
        url: "https://api.axiom.co/v1/traces",
        headers: { authorization: `Bearer ${env.AXIOM_TOKEN}`, "x-axiom-dataset": "tracing" },
      },
      service: service(env),
    };
  };
}
