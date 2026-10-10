"use client";

import { LoadError } from "@/components/load-error";
import { plural } from "@/lib/fmt";
import { SettingsForm, SettingsGroupCollapsible, formSaveState } from "@unkey/ui";
import { useAppId, useProjectData } from "../../[appId]/(overview)/data-provider";
import { AddEnvVarsFields, useAddEnvVarsForm } from "./add/add-env-vars-fields";
import { SavedEnvVarsList } from "./list/saved-env-vars-list";
import { useAppEnvVars } from "./use-app-env-vars";

export function EnvVarsGroup() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const envVars = useAppEnvVars(projectId, appId);
  const form = useAddEnvVarsForm();

  return (
    <SettingsGroupCollapsible
      title="Environment variables"
      description="Secrets and config your app reads at build and run time."
      summary={envVarsSummary({
        failed: envVars.failed,
        isLoading: envVars.isLoading,
        keys: envVars.data.map((v) => v.key),
      })}
    >
      {envVars.failed ? (
        <div className="px-5 py-4">
          <LoadError title="Could not load environment variables" onRetry={envVars.retry} />
        </div>
      ) : (
        <>
          {envVars.data.length > 0 && (
            <div className="px-5 pt-4">
              <SavedEnvVarsList envVars={envVars.data} />
            </div>
          )}
          <SettingsForm
            dirty={form.isDirty}
            saveState={formSaveState({
              isSubmitting: form.isPending,
              isValid: form.canSubmit,
              isDirty: form.isDirty,
            })}
            onSubmit={form.onSubmit}
            className="flex flex-col gap-6 px-5 pt-4 pb-6"
          >
            <AddEnvVarsFields form={form} />
          </SettingsForm>
        </>
      )}
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
