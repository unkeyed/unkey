import { createConnectTransport } from "@connectrpc/connect-web";
import { RatelimitService } from "./gen/proto/ratelimit/v1/service_connect";

import { type PromiseClient, createPromiseClient } from "@connectrpc/connect";
export { protoInt64 } from "@bufbuild/protobuf";
export * from "./gen/proto/ratelimit/v1/service_pb";
export type Ratelimit = PromiseClient<typeof RatelimitService>;

export type Config = {
  baseUrl: string;
  token: string;
};

export function createRatelimitClient(config: Config): Ratelimit {
  const transport = createConnectTransport({
    baseUrl: config.baseUrl,
    interceptors: [
      (fn) => (req) => {
        console.log(config);
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
  return createPromiseClient(RatelimitService, transport);
}
