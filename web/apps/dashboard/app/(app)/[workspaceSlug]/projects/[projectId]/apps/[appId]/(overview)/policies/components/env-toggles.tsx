"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/environment-label";
import { Switch } from "@/components/ui/switch";
import { ENVIRONMENT_KINDS } from "@/lib/collections/deploy/environments";
import { IconPlusOutline12 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { Env, MergedPolicy, PolicyEnvs } from "./list/merge";

export function PolicyEnvironmentToggles({
  policy,
  envs,
  onToggle,
  onAdd,
  locked,
}: {
  policy: MergedPolicy;
  locked: boolean;
  envs: PolicyEnvs;
  onToggle: (key: string, env: Env) => void;
  onAdd: (key: string, env: Env) => void;
}) {
  return (
    <div className="divide-y divide-grayA-4 rounded-lg border">
      {ENVIRONMENT_KINDS.map((env) => {
        const row = policy[env];
        const { slug } = envs[env];
        const environment = { slug, kind: env };
        return (
          <div key={env} className="flex h-12 items-center justify-between gap-3 px-4">
            <EnvironmentLabel environment={environment} className="text-gray-12" />
            <InfoTooltip
              asChild
              disabled={!locked}
              position={{ side: "top", align: "end" }}
              content="Save or discard your changes first."
            >
              <span className="flex">
                {row ? (
                  <Switch
                    size="sm"
                    checked={row.enabled}
                    disabled={locked}
                    aria-label={`Run ${policy.name || "this policy"} in ${slug}`}
                    onCheckedChange={() => onToggle(policy.key, env)}
                  />
                ) : (
                  <button
                    type="button"
                    disabled={locked}
                    onClick={() => onAdd(policy.key, env)}
                    className="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-gray-11 transition-colors enabled:hover:bg-grayA-3 enabled:hover:text-gray-12 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <IconPlusOutline12 className="size-3" />
                    Add to {slug}
                  </button>
                )}
              </span>
            </InfoTooltip>
          </div>
        );
      })}
    </div>
  );
}
