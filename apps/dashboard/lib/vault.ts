import { env } from "@/lib/env";
import { createVaultClient } from "@unkey/vault";

export function connectVault() {
  const { VAULT_URL, VAULT_TOKEN } = env();
  return createVaultClient({
    baseUrl: VAULT_URL,
    token: VAULT_TOKEN,
  });
}
