import { z } from "zod";

const apiErrorSchema = z.object({
  error: z.union([
    z.string(),
    z.object({
      detail: z.string().optional(),
      title: z.string().optional(),
    }),
  ]),
});

const deviceLoginSchema = z.object({
  data: z.object({
    userCode: z.string(),
    deviceName: z.string().optional(),
    status: z.string(),
    expiresAt: z.number(),
    workosVerificationUri: z.string(),
  }),
});

export type DeviceLogin = z.infer<typeof deviceLoginSchema>["data"];

async function post(path: string, body: unknown): Promise<unknown> {
  const response = await fetch(`/proxy/v2/cli.${path}`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });

  const payload: unknown = await response.json();
  if (!response.ok) {
    const error = apiErrorSchema.safeParse(payload);
    const message = error.success
      ? typeof error.data.error === "string"
        ? error.data.error
        : (error.data.error.detail ?? error.data.error.title)
      : undefined;
    throw new Error(message ?? "CLI login request failed");
  }

  return payload;
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
