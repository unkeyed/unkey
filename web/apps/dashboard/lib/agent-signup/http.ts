import { getClientIp } from "@/lib/client-ip";
import { getBaseUrl } from "@/lib/utils";
import { WorkspaceCreateError } from "@/lib/workspace/create-workspace";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { z } from "zod";
import { type AgentSignupConfig, agentSignupEnv } from "./config";
import { AgentSignupError } from "./errors";
import { agentLimiters, enforceAgentLimit } from "./limiters";
import { type VerifiedAgent, authenticateAgent, remoteJwks } from "./verify";

const workspaceBody = z.object({
  name: z.string().trim().min(3).max(50),
  slug: z
    .string()
    .trim()
    .min(3)
    .max(64)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/),
});

const rootKeyBody = z.object({
  name: z.string().trim().min(1).max(256).optional(),
  permissions: z
    .array(
      z.object({
        path: z.string(),
        action: z.string(),
      }),
    )
    .max(32)
    .optional(),
});

export function protectedResourceDocument(config: AgentSignupConfig) {
  return {
    resource: getBaseUrl(),
    authorization_servers: [config.issuer],
  };
}

export function wwwAuthenticate(): string {
  const metadata = new URL("/.well-known/oauth-protected-resource", getBaseUrl()).toString();
  return `Bearer resource_metadata="${metadata}"`;
}

export function agentJson(body: unknown, status = 200): NextResponse {
  return NextResponse.json(body, {
    status,
    headers: { "Cache-Control": "no-store" },
  });
}

export function agentError(error: unknown): NextResponse {
  let normalized = error;
  if (error instanceof WorkspaceCreateError) {
    normalized =
      error.code === "CONFLICT"
        ? new AgentSignupError(409, "slug_taken", error.message)
        : new AgentSignupError(
            500,
            "internal",
            "Agent signup failed. Try again or email support@unkey.com.",
          );
  }
  if (normalized instanceof AgentSignupError) {
    const headers = new Headers({ "Cache-Control": "no-store" });
    if (normalized.authenticate) {
      headers.set("WWW-Authenticate", wwwAuthenticate());
    }
    return NextResponse.json(
      { error: normalized.code, message: normalized.message },
      { status: normalized.status, headers },
    );
  }
  console.error("agent signup failed", normalized instanceof Error ? normalized.name : "unknown");
  return agentJson(
    {
      error: "internal",
      message: "Agent signup failed. Try again or email support@unkey.com.",
    },
    500,
  );
}

export async function beginAgentRequest(req: NextRequest): Promise<{ agent: VerifiedAgent }> {
  const config = agentSignupEnv();
  if (!config) {
    throw new AgentSignupError(404, "not_found", "Not found.");
  }
  const limiters = agentLimiters();
  await enforceAgentLimit(limiters.ip, getClientIp(req.headers) ?? "unknown");
  const agent = await authenticateAgent(req.headers.get("authorization"), config, {
    fetch,
    key: remoteJwks(config.jwksUrl),
  });
  await enforceAgentLimit(limiters.identity, agent.registrationId);
  return { agent };
}

export async function readWorkspaceBody(req: NextRequest) {
  const parsed = workspaceBody.safeParse(await readJson(req));
  if (!parsed.success) {
    throw new AgentSignupError(
      400,
      "invalid_body",
      "name must be 3 to 50 characters. slug must be 3 to 64 lowercase letters, numbers, and single hyphens.",
    );
  }
  return parsed.data;
}

export async function readRootKeyBody(req: NextRequest) {
  const parsed = rootKeyBody.safeParse(await readJson(req));
  if (!parsed.success) {
    throw new AgentSignupError(
      400,
      "invalid_body",
      "name must be 1 to 256 characters. permissions must be a list of path and action.",
    );
  }
  return parsed.data;
}

export function requestAudit(req: NextRequest): { location: string; userAgent?: string } {
  const userAgent = req.headers.get("user-agent");
  return {
    location: getClientIp(req.headers) ?? "unknown",
    ...(userAgent ? { userAgent } : {}),
  };
}

async function readJson(req: NextRequest): Promise<unknown> {
  try {
    return await req.json();
  } catch {
    throw new AgentSignupError(400, "invalid_body", "Request body must be JSON.");
  }
}
