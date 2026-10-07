import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import type { Env, MergedPolicy, PolicyEnvs } from "./merge";

export type PolicyRowName = { type: "named"; text: string } | { type: "untitled" };

export type PolicyEnvBadge =
  | { type: "on"; env: Env; slug: string }
  | { type: "off"; env: Env; slug: string };

export type PolicyRowState = {
  dimmed: boolean;
  name: PolicyRowName;
  environments: PolicyEnvBadge[];
};

export function policyRowState(policy: MergedPolicy, envs: PolicyEnvs): PolicyRowState {
  const environments = ENVIRONMENT_KINDS.flatMap((env): PolicyEnvBadge[] => {
    const row = policy[env];
    if (!row) {
      return [];
    }
    return [{ type: row.enabled ? "on" : "off", env, slug: envs[env].slug }];
  });

  return {
    dimmed: environments.every((badge) => badge.type === "off"),
    name: policy.name ? { type: "named", text: policy.name } : { type: "untitled" },
    environments,
  };
}
