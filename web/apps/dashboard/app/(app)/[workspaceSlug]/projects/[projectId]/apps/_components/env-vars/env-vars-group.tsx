"use client";

import { LoadError } from "@/components/load-error";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { plural } from "@/lib/fmt";
import { routes } from "@/lib/navigation/routes";
import { match } from "@unkey/match";
import { Button, SettingsGroupCollapsible, Skeleton, useReportUnsavedChanges } from "@unkey/ui";
import Link from "next/link";
import { useAppId, useProjectData } from "../../[appId]/(overview)/data-provider";
import { AddEnvVarsFields } from "./add/add-env-vars-fields";
import type { AutosaveStatus } from "./autosave-plan";
import { useAutosaveEnvVars } from "./use-autosave-env-vars";

function summary(status: AutosaveStatus): string | undefined {
  return match(status)
    .with({ kind: "loading" }, () => undefined)
    .with({ kind: "load-failed" }, () => "Couldn't load variables")
    .with({ kind: "unrepresentable" }, () => "Manage in app settings")
    .with({ kind: "failed" }, () => "Couldn't save")
    .with({ kind: "invalid" }, ({ invalid }) => `Fix ${plural(invalid, "variable")}`)
    .with({ kind: "saving" }, () => "Saving…")
    .with({ kind: "saved" }, ({ count }) =>
      count === 0 ? "No variables" : plural(count, "variable"),
    )
    .exhaustive();
}

export function EnvVarsGroup() {
  const form = useAutosaveEnvVars();
  useReportUnsavedChanges(form.dirty);

  return (
    <SettingsGroupCollapsible
      title="Environment variables"
      description="Secrets and config your app reads at build and run time."
      summary={summary(form.status)}
    >
      <div className="flex flex-col gap-6 px-5 pt-4 pb-6">
        <EnvVarsBody form={form} />
      </div>
    </SettingsGroupCollapsible>
  );
}

function EnvVarsBody({ form }: { form: ReturnType<typeof useAutosaveEnvVars> }) {
  return match(form.status)
    .with({ kind: "loading" }, () => <Skeleton className="h-28 w-full rounded-lg" />)
    .with({ kind: "load-failed" }, () => (
      <LoadError title="Could not load environment variables" onRetry={form.retryLoad} />
    ))
    .with({ kind: "unrepresentable" }, () => <ManageInSettings />)
    .with({ kind: "failed" }, () => (
      <>
        <AddEnvVarsFields form={form} />
        <div role="alert" className="flex items-center gap-3 text-xs text-error-11">
          Couldn't save your variables.
          <Button variant="outline" size="sm" onClick={form.retrySave}>
            Retry
          </Button>
        </div>
      </>
    ))
    .with({ kind: "invalid" }, { kind: "saving" }, () => <AddEnvVarsFields form={form} />)
    .with({ kind: "saved" }, () => <AddEnvVarsFields form={form} />)
    .exhaustive();
}

function ManageInSettings() {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();
  return (
    <p className="text-sm text-gray-11">
      This app already has variables.{" "}
      <Link
        href={routes.projects.apps.envVars({ workspaceSlug: workspace.slug, projectId, appId })}
        className="text-gray-12 underline underline-offset-2"
      >
        Manage them in app settings
      </Link>
      .
    </p>
  );
}
