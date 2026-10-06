import { z } from "zod";

const agentSignupSchema = z.object({
  AUTH_PROVIDER: z.literal("workos"),
  WORKOS_API_KEY: z.string().min(1),
  WORKOS_CLIENT_ID: z.string().min(1),
  WORKOS_AUTHKIT_DOMAIN: z.string().min(1),
  WORKOS_AGENT_AUDIENCE: z.string().min(1),
  WORKOS_API_HOSTNAME: z.string().optional(),
});

export type AgentSignupConfig = {
  apiKey: string;
  clientId: string;
  issuer: string;
  audience: string;
  apiBase: string;
  jwksUrl: string;
};

export function authKitIssuer(raw: string): string | null {
  const trimmed = raw.trim().replace(/\/+$/, "");
  if (trimmed.startsWith("http://")) {
    return null;
  }
  const withScheme = trimmed.startsWith("https://") ? trimmed : `https://${trimmed}`;
  try {
    const url = new URL(withScheme);
    if (url.protocol !== "https:" || url.username || url.password || url.search || url.hash) {
      return null;
    }
    if (url.pathname !== "/" && url.pathname !== "") {
      return null;
    }
    return url.origin;
  } catch {
    return null;
  }
}

function workOsApiBase(hostname: string | undefined): string | null {
  if (!hostname || hostname.trim() === "") {
    return "https://api.workos.com";
  }
  const trimmed = hostname.trim().replace(/\/+$/, "");
  if (trimmed.startsWith("http://")) {
    return null;
  }
  const withScheme = trimmed.startsWith("https://") ? trimmed : `https://${trimmed}`;
  try {
    const url = new URL(withScheme);
    if (url.protocol !== "https:" || url.username || url.password || url.search || url.hash) {
      return null;
    }
    if (url.pathname !== "/" && url.pathname !== "") {
      return null;
    }
    return url.origin;
  } catch {
    return null;
  }
}

export function agentSignupEnv(
  source: Record<string, string | undefined> = process.env,
): AgentSignupConfig | null {
  const parsed = agentSignupSchema.safeParse(source);
  if (!parsed.success) {
    return null;
  }
  const issuer = authKitIssuer(parsed.data.WORKOS_AUTHKIT_DOMAIN);
  const apiBase = workOsApiBase(parsed.data.WORKOS_API_HOSTNAME);
  if (!issuer || !apiBase) {
    return null;
  }
  return {
    apiKey: parsed.data.WORKOS_API_KEY,
    clientId: parsed.data.WORKOS_CLIENT_ID,
    issuer,
    audience: parsed.data.WORKOS_AGENT_AUDIENCE,
    apiBase,
    jwksUrl: new URL("/oauth2/jwks", issuer).toString(),
  };
}
