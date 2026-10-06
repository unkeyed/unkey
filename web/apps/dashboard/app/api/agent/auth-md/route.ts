import { agentSignupEnv } from "@/lib/agent-signup/config";
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
    return new NextResponse(await response.text(), {
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
