import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import type { PolicySwitches } from "../../hooks/policy-switches";
import type { Env, PolicyEnvs } from "./merge";

export type PolicyEnvBadge = { env: Env; slug: string; enabled: boolean };

export type PolicyRowState = {
  dimmed: boolean;
  environments: PolicyEnvBadge[];
};

export function policyRowState(switches: PolicySwitches, envs: PolicyEnvs): PolicyRowState {
  const environments = ENVIRONMENT_KINDS.map(
    (env): PolicyEnvBadge => ({ env, slug: envs[env].slug, enabled: switches[env] }),
  );

  return {
    dimmed: environments.every((badge) => !badge.enabled),
    environments,
  };
}
