import { agentSignupEnv } from "@/lib/agent-signup/config";
import { protectedResourceDocument } from "@/lib/agent-signup/http";
import { NextResponse } from "next/server";

export async function GET() {
  const config = agentSignupEnv();
  if (!config) {
    return NextResponse.json({ error: "not_found", message: "Not found." }, { status: 404 });
  }
  return NextResponse.json(protectedResourceDocument(config), {
    headers: { "Cache-Control": "public, max-age=60" },
  });
}
