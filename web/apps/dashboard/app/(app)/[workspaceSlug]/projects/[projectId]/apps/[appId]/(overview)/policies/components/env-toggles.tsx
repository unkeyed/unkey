"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { Switch } from "@/components/ui/switch";
import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import { InfoTooltip } from "@unkey/ui";
import type { Env, MergedPolicy, PolicyEnvs } from "../model/merge";

export function PolicyEnvironmentToggles({
  policy,
  envs,
  onToggle,
  dirty,
  locked,
}: {
  policy: MergedPolicy;
  dirty: boolean;
  locked: boolean;
  envs: PolicyEnvs;
  onToggle: (env: Env, enabled: boolean) => void;
}) {
  return (
    <div className="divide-y divide-grayA-4 rounded-lg border">
      {ENVIRONMENT_KINDS.map((env) => {
        const { slug } = envs[env];
        const environment = { slug, kind: env };
        return (
          <div key={env} className="flex h-12 items-center justify-between gap-3 px-4">
            <EnvironmentLabel environment={environment} className="text-sm text-gray-12" />
            <InfoTooltip
              asChild
              disabled={!dirty}
              position={{ side: "top", align: "end" }}
              content="Save or discard your changes first."
            >
              <span className="flex">
                <Switch
                  size="sm"
                  checked={policy[env]?.enabled ?? false}
                  disabled={dirty || locked}
                  aria-label={`Run ${policy.name} in ${slug}`}
                  onCheckedChange={(checked) => onToggle(env, checked)}
                />
              </span>
            </InfoTooltip>
          </div>
        );
      })}
    </div>
  );
}
