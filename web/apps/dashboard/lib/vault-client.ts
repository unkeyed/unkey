import { env } from "@/lib/env";
import type { DescService } from "@bufbuild/protobuf";
import { type Client, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

export function createVaultClient<T extends DescService>(service: T): Client<T> {
  const { VAULT_URL, VAULT_TOKEN } = env();

  return createClient(
    service,
    createConnectTransport({
      baseUrl: VAULT_URL,
      interceptors: [
        (next) => (req) => {
          req.header.set("Authorization", `Bearer ${VAULT_TOKEN}`);
          return next(req);
        },
      ],
    }),
  );
}
