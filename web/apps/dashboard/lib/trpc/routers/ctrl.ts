import { ProjectService } from "@/gen/proto/ctrl/v1/project_pb";
import { env } from "@/lib/env";
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TRPCError } from "@trpc/server";

function getTransport() {
  const { CTRL_URL, CTRL_API_KEY } = env();
  if (!CTRL_URL || !CTRL_API_KEY) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "ctrl service is not configured",
    });
  }

  return createConnectTransport({
    baseUrl: CTRL_URL,
    interceptors: [
      (next) => (req) => {
        req.header.set("Authorization", `Bearer ${CTRL_API_KEY}`);
        return next(req);
      },
    ],
  });
}

export function getCtrlClients() {
  const transport = getTransport();
  return {
    project: createClient(ProjectService, transport),
    // more typed clients can be added here
  };
}
