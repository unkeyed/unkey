import { proxyPost } from "@/lib/proxy-post";
import { z } from "zod";

const deviceLoginSchema = z.object({
  data: z.object({
    userCode: z.string(),
    deviceName: z.string().optional(),
    requesterIp: z.string().optional(),
    requesterUserAgent: z.string().optional(),
    workspaceId: z.string().optional(),
    status: z.string(),
    createdAt: z.number(),
    expiresAt: z.number(),
    grantablePermissions: z.array(z.string()).optional(),
  }),
});

const sessionSchema = z.object({
  id: z.string(),
  userId: z.string(),
  deviceName: z.string().optional(),
  permissions: z.array(z.string()),
  createdAt: z.number(),
  lastUsedAt: z.number(),
  expiresAt: z.number(),
  revokedAt: z.number().optional(),
});

const sessionsSchema = z.object({
  data: z.object({
    sessions: z.array(sessionSchema),
  }),
});

export type DeviceLogin = z.infer<typeof deviceLoginSchema>["data"];
export type CliSession = z.infer<typeof sessionSchema>;

function post(path: string, body: unknown): Promise<unknown> {
  return proxyPost(`cli.${path}`, body, "CLI login request failed");
}

export async function getDeviceLogin(userCode: string): Promise<DeviceLogin> {
  return deviceLoginSchema.parse(await post("getDeviceLogin", { userCode })).data;
}

export async function approveDeviceLogin(request: {
  userCode: string;
  permissions: string[];
}): Promise<void> {
  await post("decideDeviceLogin", { ...request, decision: "approve" });
}

export async function denyDeviceLogin(userCode: string): Promise<void> {
  await post("decideDeviceLogin", { userCode, decision: "deny" });
}

export async function listCliSessions(): Promise<CliSession[]> {
  return sessionsSchema.parse(await post("listSessions", undefined)).data.sessions;
}

export async function revokeCliSession(sessionId: string): Promise<void> {
  await post("revokeSession", { sessionId });
}
