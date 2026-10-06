import { agentSignupEnv } from "@/lib/agent-signup/config";
import { getBaseUrl } from "@/lib/utils";
import { NextResponse } from "next/server";

export async function GET() {
  const config = agentSignupEnv();
  if (!config) {
    return NextResponse.json({ error: "not_found", message: "Not found." }, { status: 404 });
  }

  try {
    const response = await fetch(new URL("/agent/auth.md", config.issuer), {
      signal: AbortSignal.timeout(5000),
    });
    if (!response.ok) {
      return NextResponse.json(
        { error: "upstream", message: "AuthKit auth.md is unavailable." },
        { status: 502 },
      );
    }
    const base = getBaseUrl();
    const preface = `# Unkey agent signup

The WorkOS registration skill follows. After you exchange the identity assertion for an access token:

- If the person already admins a workspace, \`POST ${base}/api/agent/root-key\` with that \`workspaceId\` or \`slug\`.
- If they need a workspace, \`POST ${base}/api/agent/workspace\` with a new name and slug, then create a root key for it.

Save \`identity.refresh_token.value\` from claim complete. Before each Unkey call, and on a 401 \`expired\`, exchange the saved assertion for a new access token and retry that call once. When the assertion expires, refresh it, then exchange again. The calls are in https://www.unkey.com/docs/agent.md.

---

`;
    return new NextResponse(preface + (await response.text()), {
      status: 200,
      headers: {
        "Content-Type": "text/markdown; charset=utf-8",
        "Cache-Control": "public, max-age=60",
      },
    });
  } catch {
    return NextResponse.json(
      { error: "upstream", message: "AuthKit auth.md is unavailable." },
      { status: 502 },
    );
  }
}
