import { ResolveConfigFn, ServiceConfig } from "@microlabs/otel-cf-workers";
import { Env } from "../env";

export function traceConfig(service: (env: Env) => ServiceConfig): ResolveConfigFn {
  return (env: Env) => {
    return {
      exporter: {
        url: "https://api.axiom.co/v1/traces",
        headers: { authorization: `Bearer ${env.AXIOM_TOKEN}`, "x-axiom-dataset": "tracing" },
      },
      service: service(env),
    };
  };
}
