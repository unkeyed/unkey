import crypto from "node:crypto";
import { githubAppEnv } from "@/lib/env";
import { getBaseUrl } from "@/lib/utils";
import { TRPCError } from "@trpc/server";
import { cookies } from "next/headers";
import { z } from "zod";

export const relayHandle = z.string().regex(/^[A-Za-z0-9_-]{43}$/);

const httpsOrigin = z.string().refine((raw) => {
  try {
    const url = new URL(raw);
    return url.protocol === "https:" && url.origin === raw && !url.hostname.includes("*");
  } catch {
    return false;
  }
});

const relaySettings = z.union([
  z.object({ url: httpsOrigin, token: relayHandle, adminToken: z.undefined() }),
  z.object({
    url: httpsOrigin,
    token: z.undefined(),
    adminToken: z.string().min(32),
  }),
]);

export function githubRelayConfig() {
  const url = process.env.GITHUB_INSTALL_RELAY_URL;
  const token = process.env.GITHUB_INSTALL_RELAY_TOKEN || undefined;
  const adminToken = process.env.GITHUB_INSTALL_RELAY_ADMIN_TOKEN || undefined;
  if (!url && !token && !adminToken) {
    return null;
  }
  const result = relaySettings.safeParse({ url, token, adminToken });
  if (!result.success) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "GitHub installation relay is not configured correctly",
    });
  }
  return result.data;
}

export function githubInstallAvailable(): boolean {
  try {
    githubRelayConfig();
    return Boolean(
      githubAppEnv() &&
        /^[a-zA-Z0-9][a-zA-Z0-9-]{0,99}$/.test(process.env.NEXT_PUBLIC_GITHUB_APP_NAME ?? ""),
    );
  } catch {
    return false;
  }
}

async function postRelay(url: string, token: string, body: unknown): Promise<unknown> {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    redirect: "error",
    cache: "no-store",
    signal: AbortSignal.timeout(10_000),
  });
  if (!response.ok) {
    throw new Error("Relay rejected request");
  }
  return await response.json();
}

async function relayRequest(path: string, body: unknown): Promise<unknown> {
  const config = githubRelayConfig();
  if (!config) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "GitHub installation relay is disabled",
    });
  }
  try {
    let token: string;
    if (config.adminToken !== undefined) {
      const origin = httpsOrigin.parse(getBaseUrl());
      const enrollment = z
        .object({
          id: relayHandle,
          token: relayHandle,
          origin: httpsOrigin,
          expiresAt: z.number(),
        })
        .parse(
          await postRelay(`${config.url}/v1/environments`, config.adminToken, {
            origin,
            expiresAt: Date.now() + 86400_000,
            reuse: true,
          }),
        );
      if (
        enrollment.origin !== origin ||
        enrollment.expiresAt <= Date.now() ||
        enrollment.expiresAt > Date.now() + 30 * 86400_000
      ) {
        throw new Error("Invalid GitHub relay enrollment");
      }
      token = enrollment.token;
    } else {
      token = config.token;
    }
    return await postRelay(`${config.url}${path}`, token, body);
  } catch {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "GitHub installation could not be verified. Restart from this dashboard.",
    });
  }
}

const bindingCookie = (id: string) => `__Host-github-install-${relayHandle.parse(id)}`;
const cookieOptions = {
  secure: true,
  httpOnly: true,
  sameSite: "lax",
  path: "/",
} as const;

export async function prepareGithubInstall(state: string, userId: string, workspaceId: string) {
  if (!githubInstallAvailable()) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "GitHub installation is unavailable",
    });
  }
  const relay = githubRelayConfig();
  if (!relay) {
    return `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
  }
  const origin = httpsOrigin.parse(getBaseUrl());
  const binding = crypto.randomBytes(32).toString("base64url");
  const result = z
    .object({
      id: relayHandle,
      url: z.string(),
      expiresAt: z.number().int().positive(),
    })
    .parse(
      await relayRequest("/v1/transactions", {
        origin,
        state,
        userId,
        workspaceId,
        binding: crypto.createHash("sha256").update(binding).digest("hex"),
      }),
    );
  if (
    result.url !== `${relay.url}/start?transaction=${result.id}` ||
    result.expiresAt <= Date.now() ||
    result.expiresAt > Date.now() + 15 * 60 * 1000
  ) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "Invalid GitHub installation transaction",
    });
  }
  (await cookies()).set(bindingCookie(result.id), binding, {
    ...cookieOptions,
    expires: new Date(result.expiresAt),
  });
  return result.url;
}

export async function redeemGithubInstall(
  transaction: string,
  handoff: string,
  userId: string,
  workspaceId: string,
) {
  const binding = (await cookies()).get(bindingCookie(transaction))?.value;
  if (!binding || !relayHandle.safeParse(binding).success) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "GitHub installation session expired",
    });
  }
  const result = z
    .object({
      state: z.string().max(8192),
      installationId: z.number().int().positive(),
      origin: httpsOrigin,
    })
    .parse(
      await relayRequest(`/v1/transactions/${relayHandle.parse(transaction)}/redeem`, {
        handoff: relayHandle.parse(handoff),
        binding,
        userId,
        workspaceId,
      }),
    );
  if (result.origin !== getBaseUrl()) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: "GitHub installation environment changed",
    });
  }
  return result;
}

export async function finishGithubInstall(transaction: string) {
  (await cookies()).set(bindingCookie(transaction), "", {
    ...cookieOptions,
    maxAge: 0,
  });
}
