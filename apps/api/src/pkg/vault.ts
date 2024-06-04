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

  return {
    liveness: async (...args: Parameters<Vault["liveness"]>) => {
      const start = performance.now();
      const res = await vault.liveness(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "liveness",
        latency: performance.now() - start,
      });
      return res;
    },
    createDEK: async (...args: Parameters<Vault["createDEK"]>) => {
      const start = performance.now();
      const res = await vault.createDEK(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "createDEK",
        latency: performance.now() - start,
      });
      return res;
    },
    decrypt: async (...args: Parameters<Vault["decrypt"]>) => {
      const start = performance.now();
      const res = await vault.decrypt(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "decrypt",
        latency: performance.now() - start,
      });
      return res;
    },
    encrypt: async (...args: Parameters<Vault["encrypt"]>) => {
      const start = performance.now();
      const res = await vault.encrypt(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "encrypt",
        latency: performance.now() - start,
      });
      return res;
    },
    encryptBulk: async (...args: Parameters<Vault["encryptBulk"]>) => {
      const start = performance.now();
      const res = await vault.encryptBulk(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "encryptBulk",
        latency: performance.now() - start,
      });
      return res;
    },
    reEncrypt: async (...args: Parameters<Vault["reEncrypt"]>) => {
      const start = performance.now();
      const res = await vault.reEncrypt(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "reEncrypt",
        latency: performance.now() - start,
      });
      return res;
    },
    reEncryptDEKs: async (...args: Parameters<Vault["reEncryptDEKs"]>) => {
      const start = performance.now();
      const res = await vault.reEncryptDEKs(...args);
      metrics.emit({
        metric: "metric.vault.latency",
        op: "reEncryptDEKs",
        latency: performance.now() - start,
      });
      return res;
    },
  };
}
