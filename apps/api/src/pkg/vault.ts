import { type Vault, createVaultClient } from "@unkey/vault";
import type { Env } from "./env";
import type { Metrics } from "./metrics";

export function connectVault(
  env: Pick<Env, "VAULT_URL" | "VAULT_TOKEN">,
  metrics?: Metrics,
): Vault {
  const vault = createVaultClient({
    baseUrl: env.VAULT_URL,
    token: env.VAULT_TOKEN,
  });
  if (!metrics) {
    return vault;
  }

  return new Proxy(vault, {
    get: async (target, p) => {
      // @ts-expect-error
      const fn = target[p];

      return async (...args: any[]) => {
        const start = performance.now();
        const res = fn(...args);
        metrics.emit({
          metric: "metric.vault.latency",
          op: "liveness",
          latency: performance.now() - start,
        });
        return res;
      };
    },
  });
}
