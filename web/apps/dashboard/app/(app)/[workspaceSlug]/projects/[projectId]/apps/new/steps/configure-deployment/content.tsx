"use client";

import {
  useAppId,
  useProjectData,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  AddEnvVarsFields,
  useAddEnvVarsForm,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/env-vars/add/add-env-vars-fields";
import { SavedEnvVarsList } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/env-vars/list/saved-env-vars-list";
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
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { useCollectionLoad } from "@/lib/collections/use-collection-load";
import { plural } from "@/lib/fmt";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import {
  SettingsForm,
  SettingsGroupCollapsible,
  SettingsGroups,
  UnsavedChangesScope,
  formSaveState,
  useReportUnsavedChanges,
  useUnsavedChanges,
} from "@unkey/ui";
import { DeployAction } from "./deploy-action";
import { ConfigureFrame } from "./frame";

export function ConfigureDeploymentContent({
  onDeploymentCreated,
}: { onDeploymentCreated: (deploymentId: string) => void }) {
  const { isDirty, report } = useUnsavedChanges();
  useReportUnsavedChanges(isDirty);
  const { settings } = useEnvironmentSettings();

  return (
    <ConfigureFrame>
      <UnsavedChangesScope report={report}>
        <SettingsGroups pendingNote={NEXT_DEPLOY} className="gap-4 p-0">
          <BuildSettings githubReadOnly />
          <ComputeSettings />
          <EnvVarsGroup />
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
      </UnsavedChangesScope>
      <DeployAction state={isDirty ? "dirty" : "ready"} onDeploymentCreated={onDeploymentCreated} />
    </ConfigureFrame>
  );
}

function EnvVarsGroup() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const form = useAddEnvVarsForm();
  const envVars = useLiveQuery(
    (q) =>
      q
        .from({ v: collection.envVars })
        .where(({ v }) => and(eq(v.projectId, projectId), eq(v.appId, appId))),
    [projectId, appId],
  );
  const envVarsLoad = useCollectionLoad(collection.envVars.utils);

  return (
    <SettingsGroupCollapsible
      title="Environment variables"
      description="Secrets and config your app reads at build and run time."
      summary={envVarsSummary({
        failed: envVarsLoad.failed,
        isLoading: envVars.isLoading,
        keys: envVars.data.map((v) => v.key),
      })}
    >
      <div className="px-5 pt-4 empty:hidden">
        <SavedEnvVarsList envVars={envVars.data} />
      </div>
      <SettingsForm
        dirty={form.isDirty}
        saveState={formSaveState({
          isSubmitting: form.isPending,
          isValid: form.targetEnvironmentIds.length > 0,
          isDirty: form.isDirty,
        })}
        onSubmit={form.onSubmit}
        className="flex flex-col gap-6 px-5 pt-4 pb-6"
      >
        <AddEnvVarsFields form={form} />
      </SettingsForm>
    </SettingsGroupCollapsible>
  );
}

function envVarsSummary({
  failed,
  isLoading,
  keys,
}: { failed: boolean; isLoading: boolean; keys: string[] }): string | undefined {
  if (failed) {
    return "Couldn't load variables";
  }
  if (isLoading) {
    return undefined;
  }
  const count = new Set(keys).size;
  return count === 0 ? "No variables" : plural(count, "variable");
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
