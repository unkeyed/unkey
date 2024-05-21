import { createConnectTransport } from "@connectrpc/connect-web";
import { VaultService } from "./gen/proto/vault/v1/service_connect";

import { type PromiseClient, createPromiseClient } from "@connectrpc/connect";

export type Vault = PromiseClient<typeof VaultService>;

export type Config = {
  baseUrl: string;
  token: string;
};

export function createVaultClient(config: Config): Vault {
  const transport = createConnectTransport({
    baseUrl: config.baseUrl,
    interceptors: [
      (fn) => (req) => {
        req.header.set("Authorization", `Bearer ${config.token}`);
        return fn(req);
      },
    ],
    /**
     * Cloudflare can not handle these configs, so we need to delete them
     */
    fetch: (input, init) => {
      const i = init ?? {};
      delete i.mode;
      delete i.credentials;
      delete i.redirect;
      return fetch(input, i);
    },
  });
  return createPromiseClient(VaultService, transport);
}
