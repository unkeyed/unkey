import {
  agentError,
  beginAgentRequest,
  readWorkspaceBody,
  requestAudit,
} from "@/lib/agent-signup/http";
import { createAgentWorkspace } from "@/lib/agent-signup/service";
import type { NextRequest } from "next/server";

export async function POST(req: NextRequest) {
  try {
    const { agent } = await beginAgentRequest(req);
    const body = await readWorkspaceBody(req);
    const created = await createAgentWorkspace({
      agent,
      name: body.name,
      slug: body.slug,
      audit: requestAudit(req),
    });
    return Response.json(
      {
        workspaceId: created.workspaceId,
        orgId: created.orgId,
        slug: created.slug,
        name: body.name,
      },
      { status: 200, headers: { "Cache-Control": "no-store" } },
    );
  } catch (error) {
    return agentError(error);
  }
}
