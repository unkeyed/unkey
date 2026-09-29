import { z } from "zod";
import { bindingNameSchema } from "./validation";

export const target = z.discriminatedUnion("targetType", [
  z.object({ targetType: z.literal("automatic") }),
  z.object({
    targetType: z.literal("environment"),
    targetEnvironmentId: z.string().min(1),
  }),
  z.object({
    targetType: z.literal("deployment"),
    targetDeploymentId: z.string().min(1),
  }),
]);
type Target = z.infer<typeof target>;
export const projectInput = z.object({ projectId: z.string().min(1) });
export const optionalName = z.object({ name: bindingNameSchema.optional() });

export function targetColumns(input: Target) {
  return {
    selectionMode: input.targetType,
    targetEnvironmentId: input.targetType === "environment" ? input.targetEnvironmentId : null,
    targetDeploymentId: input.targetType === "deployment" ? input.targetDeploymentId : null,
  };
}
