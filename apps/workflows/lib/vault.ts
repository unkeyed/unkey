import { env } from "@/lib/env";
import { createVaultClient } from "@unkey/vault";

export function connectVault() {
  const { AGENT_URL, AGENT_TOKEN } = env();
  return createVaultClient({
    baseUrl: AGENT_URL,
    token: AGENT_TOKEN,
  });
}
