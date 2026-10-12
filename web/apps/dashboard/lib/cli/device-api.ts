import { proxyPost } from "@/lib/proxy-post";
import { z } from "zod";

const deviceLoginSchema = z.object({
  data: z.object({
    userCode: z.string(),
    deviceName: z.string().optional(),
    requesterIp: z.string().optional(),
    requesterUserAgent: z.string().optional(),
    status: z.string(),
    createdAt: z.number(),
    expiresAt: z.number(),
  }),
});

export type DeviceLogin = z.infer<typeof deviceLoginSchema>["data"];

function post(path: string, body: unknown): Promise<unknown> {
  return proxyPost(`cli.${path}`, body, "CLI login request failed");
}

export async function getDeviceLogin(userCode: string): Promise<DeviceLogin> {
  return deviceLoginSchema.parse(await post("getDeviceLogin", { userCode })).data;
}

export async function approveDeviceLogin(request: {
  userCode: string;
  name: string;
  permissions: string[];
}): Promise<void> {
  await post("approveDeviceLogin", request);
}

export async function denyDeviceLogin(userCode: string): Promise<void> {
  await post("denyDeviceLogin", { userCode });
}
