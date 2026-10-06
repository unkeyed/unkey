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

The WorkOS registration skill follows. After you exchange the identity assertion for an access token, create the workspace and the first root key on this dashboard:

- \`POST ${base}/api/agent/workspace\`
- \`POST ${base}/api/agent/root-key\` with \`workspaceId\` or \`slug\`

The full procedure is https://www.unkey.com/docs/agent.md

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
