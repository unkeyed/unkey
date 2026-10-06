import { agentError, beginAgentRequest, readRootKeyBody } from "@/lib/agent-signup/http";
import { issueAgentRootKey } from "@/lib/agent-signup/service";
import type { NextRequest } from "next/server";

export async function POST(req: NextRequest) {
  try {
    const { agent } = await beginAgentRequest(req);
    const body = await readRootKeyBody(req);
    const created = await issueAgentRootKey({
      agent,
      ...(body.workspaceId ? { workspaceId: body.workspaceId } : {}),
      ...(body.slug ? { slug: body.slug } : {}),
      ...(body.name ? { name: body.name } : {}),
      ...(body.permissions ? { permissions: body.permissions } : {}),
    });
    return Response.json(created, {
      status: 200,
      headers: { "Cache-Control": "no-store" },
    });
  } catch (error) {
    return agentError(error);
  }
}
