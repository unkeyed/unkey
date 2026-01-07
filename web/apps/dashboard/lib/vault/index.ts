import {
  type Client,
  type StreamRequest,
  type StreamResponse,
  type UnaryRequest,
  type UnaryResponse,
  createClient,
} from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { VaultService } from "./gen/vault/v1/service_pb";

export function createVault(baseUrl: string, bearer: string): Client<typeof VaultService> {
  const transport = createConnectTransport({
    baseUrl,
    interceptors: [
      (next) => {
        return (req: UnaryRequest | StreamRequest): Promise<UnaryResponse | StreamResponse> => {
          req.header.append("Authorization", `Bearer ${bearer}`);

          return next(req);
        };
      },
    ],
  });
  return createClient(VaultService, transport);
}
