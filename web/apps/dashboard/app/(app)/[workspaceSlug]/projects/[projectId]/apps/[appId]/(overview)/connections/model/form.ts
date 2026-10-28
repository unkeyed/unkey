import { ENVIRONMENT_KINDS, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import { match } from "@unkey/match";
import { z } from "zod";
import { connectionNameSchema } from "../connection-rules";
import { type Environment, type TargetRule, columnsOf, ruleOf } from "../connection-rules";
import type { ConnectionEnvs, MergedConnection } from "./merge";

const targetDraftSchema = z.discriminatedUnion("mode", [
  z.object({ mode: z.literal("automatic") }),
  z.object({ mode: z.literal("environment"), environmentId: z.string() }),
  z.object({ mode: z.literal("deployment"), deploymentId: z.string() }),
]);
const envDraftSchema = z.object({ enabled: z.boolean(), target: targetDraftSchema });
const formShape = z.object({
  targetAppId: z.string().min(1, "Choose an app"),
  name: z.string(),
  envs: z.object({ production: envDraftSchema, preview: envDraftSchema }),
});

export type TargetDraft = z.infer<typeof targetDraftSchema>;
export type EnvDraft = z.infer<typeof envDraftSchema>;
export type ConnectionFormValues = z.infer<typeof formShape>;

export type ConnectionOp =
  | { type: "create"; env: EnvironmentKind; environmentId: string; rule: TargetRule; name?: string }
  | { type: "update"; env: EnvironmentKind; id: string; rule: TargetRule; name?: string }
  | { type: "delete"; env: EnvironmentKind; id: string };

export const IMAGE_PREVIEW_ERROR =
  "Image deployments have no Git branch to match. Choose an environment or deployment.";

export function targetError(target: TargetDraft): string | null {
  return match(target)
    .with({ mode: "automatic" }, () => null)
    .with({ mode: "environment" }, ({ environmentId }) =>
      environmentId ? null : "Choose an environment",
    )
    .with({ mode: "deployment" }, ({ deploymentId }) =>
      deploymentId ? null : "Choose a deployment",
    )
    .exhaustive();
}

export function connectionFormSchema({
  requiresExplicitPreviewTarget,
  nameRequired,
}: {
  requiresExplicitPreviewTarget: boolean;
  nameRequired: boolean;
}) {
  return formShape.superRefine((values, ctx) => {
    const name = values.name.trim();
    if (name !== "" || nameRequired) {
      const parsed = connectionNameSchema.safeParse(name);
      if (!parsed.success) {
        ctx.addIssue({
          code: "custom",
          path: ["name"],
          message: parsed.error.issues[0]?.message ?? "Enter a valid name",
        });
      }
    }
    if (ENVIRONMENT_KINDS.every((kind) => !values.envs[kind].enabled)) {
      ctx.addIssue({
        code: "custom",
        path: ["envs"],
        message: "Turn on at least one environment",
      });
    }
    for (const kind of ENVIRONMENT_KINDS) {
      const { enabled, target } = values.envs[kind];
      if (!enabled) {
        continue;
      }
      const message =
        kind === "preview" && requiresExplicitPreviewTarget && target.mode === "automatic"
          ? IMAGE_PREVIEW_ERROR
          : targetError(target);
      if (message) {
        ctx.addIssue({ code: "custom", path: ["envs", kind, "target"], message });
      }
    }
  });
}

export function defaultEnvs(
  previewEnvironmentId: string | null,
  envs?: ConnectionEnvs,
): ConnectionFormValues["envs"] {
  return {
    production: { enabled: envs ? envs.production !== null : true, target: { mode: "automatic" } },
    preview: {
      enabled: envs ? envs.preview !== null : true,
      target:
        previewEnvironmentId === null
          ? { mode: "automatic" }
          : { mode: "environment", environmentId: previewEnvironmentId },
    },
  };
}

export function previewTargetFor({
  requiresExplicitPreviewTarget,
  environments,
  targetAppId,
}: {
  requiresExplicitPreviewTarget: boolean;
  environments: Environment[];
  targetAppId: string;
}): string | null {
  if (!requiresExplicitPreviewTarget) {
    return null;
  }
  return environments.find((env) => env.appId === targetAppId && env.kind === "preview")?.id ?? "";
}

export function draftOf(rule: TargetRule): TargetDraft {
  return match(rule)
    .with({ targetType: "automatic" }, (): TargetDraft => ({ mode: "automatic" }))
    .with(
      { targetType: "environment" },
      ({ targetEnvironmentId }): TargetDraft => ({
        mode: "environment",
        environmentId: targetEnvironmentId,
      }),
    )
    .with(
      { targetType: "deployment" },
      ({ targetDeploymentId }): TargetDraft => ({
        mode: "deployment",
        deploymentId: targetDeploymentId,
      }),
    )
    .exhaustive();
}

export function ruleOfDraft(draft: TargetDraft): TargetRule {
  return match(draft)
    .with({ mode: "automatic" }, (): TargetRule => ({ targetType: "automatic" }))
    .with(
      { mode: "environment" },
      ({ environmentId }): TargetRule => ({
        targetType: "environment",
        targetEnvironmentId: environmentId,
      }),
    )
    .with(
      { mode: "deployment" },
      ({ deploymentId }): TargetRule => ({
        targetType: "deployment",
        targetDeploymentId: deploymentId,
      }),
    )
    .exhaustive();
}

export function formValuesOf(
  connection: MergedConnection,
  fallback: ConnectionFormValues["envs"],
): ConnectionFormValues {
  const envOf = (kind: EnvironmentKind): EnvDraft => {
    const rule = connection.byEnv[kind] && ruleOf(connection.byEnv[kind]);
    return rule ? { enabled: true, target: draftOf(rule) } : { ...fallback[kind], enabled: false };
  };
  return {
    targetAppId: connection.targetAppId,
    name: connection.name,
    envs: { production: envOf("production"), preview: envOf("preview") },
  };
}

function sameRule(a: TargetRule, b: TargetRule | undefined): boolean {
  if (!b) {
    return false;
  }
  const left = columnsOf(a);
  const right = columnsOf(b);
  return (
    left.targetType === right.targetType &&
    left.targetEnvironmentId === right.targetEnvironmentId &&
    left.targetDeploymentId === right.targetDeploymentId
  );
}

export function planSave(
  before: MergedConnection | null,
  values: ConnectionFormValues,
  envs: ConnectionEnvs,
): ConnectionOp[] {
  const name = values.name.trim();
  const renamed = before !== null && name !== before.name;
  return ENVIRONMENT_KINDS.flatMap((env): ConnectionOp[] => {
    const draft = values.envs[env];
    const existing = before?.byEnv[env];
    const environment = envs[env];
    if (!draft.enabled) {
      return existing ? [{ type: "delete", env, id: existing.id }] : [];
    }
    const rule = ruleOfDraft(draft.target);
    if (!existing) {
      return environment
        ? [
            {
              type: "create",
              env,
              environmentId: environment.id,
              rule,
              ...(name ? { name } : {}),
            },
          ]
        : [];
    }
    const nameChanged = renamed && name !== existing.name;
    if (!nameChanged && sameRule(rule, ruleOf(existing))) {
      return [];
    }
    return [{ type: "update", env, id: existing.id, rule, ...(nameChanged ? { name } : {}) }];
  });
}

export function planRemove(connection: MergedConnection): ConnectionOp[] {
  return ENVIRONMENT_KINDS.flatMap((env): ConnectionOp[] => {
    const row = connection.byEnv[env];
    return row ? [{ type: "delete", env, id: row.id }] : [];
  });
}
