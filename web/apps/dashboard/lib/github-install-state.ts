import { z } from "zod";

const stateBaseFields = {
  workspaceId: z.string().min(1),
  nonce: z.string().min(1),
  exp: z.number().int().positive(),
  installationId: z.number().int().positive().optional(),
};

const apiInstallState = z.object({ ...stateBaseFields, flow: z.literal("api") }).strict();

const workspaceInstallState = z
  .object({
    ...stateBaseFields,
    flow: z.literal("workspace"),
    userId: z.string().min(1),
  })
  .strict();

const appInstallState = z
  .object({
    ...stateBaseFields,
    flow: z.literal("app"),
    userId: z.string().min(1),
    projectId: z.string().min(1),
    appId: z.string().min(1),
    returnTo: z.enum(["settings"]).optional(),
  })
  .strict();

const signedStatePayload = z.discriminatedUnion("flow", [
  apiInstallState,
  workspaceInstallState,
  appInstallState,
]);

export const signedState = z.discriminatedUnion("flow", [
  apiInstallState.extend({ sig: z.string().min(1) }),
  workspaceInstallState.extend({ sig: z.string().min(1) }),
  appInstallState.extend({ sig: z.string().min(1) }),
]);

export type SignedStatePayload = z.infer<typeof signedStatePayload>;
