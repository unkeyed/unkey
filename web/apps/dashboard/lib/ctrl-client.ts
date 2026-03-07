import { env } from "@/lib/env";
import type { DescService } from "@bufbuild/protobuf";
import { type Client, createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TRPCError } from "@trpc/server";

export function createCtrlClient<T extends DescService>(service: T): Client<T> {
  const { CTRL_URL, CTRL_API_KEY } = env();
  if (!CTRL_URL || !CTRL_API_KEY) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "ctrl service is not configured",
    });
  }

  return createClient(
    service,
    createConnectTransport({
      baseUrl: CTRL_URL,
      interceptors: [
        (next) => (req) => {
          req.header.set("Authorization", `Bearer ${CTRL_API_KEY}`);
          return next(req);
        },
      ],
    }),
  );
}
