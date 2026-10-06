import { agentSignupJwks } from "@/lib/agent-signup/credential";
import { NextResponse } from "next/server";

export async function GET() {
  const jwks = await agentSignupJwks();
  if (!jwks) {
    return NextResponse.json({ error: "not_found", message: "Not found." }, { status: 404 });
  }
  return NextResponse.json(jwks, {
    headers: { "Cache-Control": "public, max-age=60" },
  });
}
