"use client";

import { OpenapiSpecPath } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/advanced-settings/openapi-spec-path";
import {
  PROTOCOLS,
  UpstreamProtocol,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/advanced-settings/upstream-protocol";
import { BuildSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/build-settings";
import { ComputeSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/compute/compute-settings";
import { useEnvironmentSettings } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/environment-provider";
import { Command } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/runtime-settings/command";
import { Healthcheck } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/runtime-settings/healthcheck";
import { Port } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/runtime-settings/port-settings";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { Button, SettingsGroupCollapsible, SettingsGroups, useStepWizard } from "@unkey/ui";

export function ConfigureDeploymentContent() {
  const { next } = useStepWizard();
  const { settings } = useEnvironmentSettings();

  return (
    <div className="w-225">
      <SettingsGroups pendingNote={NEXT_DEPLOY} className="gap-4 p-0">
        <BuildSettings githubReadOnly />
        <ComputeSettings />
        <SettingsGroupCollapsible
          title="Runtime"
          description="Port, start command and health check. The defaults suit most apps."
          summary={runtimeSummary(settings)}
        >
          <Port />
          <Command />
          <Healthcheck />
        </SettingsGroupCollapsible>
        <SettingsGroupCollapsible
          title="Advanced"
          description="OpenAPI spec path and upstream protocol."
          summary={advancedSummary(settings)}
        >
          <OpenapiSpecPath />
          <UpstreamProtocol />
        </SettingsGroupCollapsible>
      </SettingsGroups>
      <div className="flex justify-end mt-6 mb-10 flex-col gap-4">
        <Button type="button" variant="primary" size="xlg" className="rounded-lg" onClick={next}>
          Next
        </Button>
        <span className="text-gray-10 text-sm text-center">
          Start configuring your environment variables
        </span>
      </div>
    </div>
  );
}

function runtimeSummary({ port, command, healthcheck }: EnvironmentSettings): string {
  return [
    `Port ${port}`,
    command.length > 0 ? "Custom command" : "Image command",
    healthcheck ? `${healthcheck.method} ${healthcheck.path}` : "No health check",
  ].join(" · ");
}

function advancedSummary({ upstreamProtocol, openapiSpecPath }: EnvironmentSettings): string {
  const protocol = PROTOCOLS.find((option) => option.value === upstreamProtocol);
  const spec =
    openapiSpecPath === null || openapiSpecPath.trim() === "" ? "No OpenAPI spec" : openapiSpecPath;
  return protocol ? `${protocol.label} · ${spec}` : spec;
}
