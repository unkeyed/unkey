import {
  MAX_VARIABLES,
  PartialAddError,
  envVarErrorToast,
  saveVariables,
} from "@/lib/collections/deploy/env-vars";
import { useAppScope, useEnvironments } from "@/lib/deploy-data";
import { plural } from "@/lib/fmt";
import { useMutation } from "@tanstack/react-query";
import { FormField, toast } from "@unkey/ui";
import { type FormEvent, useState } from "react";
import { environmentName } from "../../environment-label";
import { ALL_ENVIRONMENTS, targetEnvironmentIds } from "../../environment-scope";
import { EnvironmentSelect } from "../../environment-select";
import { useAppEnvVars } from "../use-app-env-vars";
import { useVariableRows } from "../variables-table/use-variable-rows";
import {
  type SavedRow,
  conflictErrors,
  isChangedRow,
  savedVariableChanges,
  savedVariableRows,
  splitVariableRows,
} from "../variables-table/variable-rows";
import { VariablesTable } from "../variables-table/variables-table";

const NO_SAVED: readonly SavedRow[] = [];

export function useAddEnvVarsForm({
  onSaved,
  showSaved = false,
}: { onSaved?: () => void; showSaved?: boolean } = {}) {
  const { projectId, appId } = useAppScope();
  const { data: environments } = useEnvironments();
  const [environmentId, setEnvironmentId] = useState(ALL_ENVIRONMENTS);
  const targetIds = targetEnvironmentIds(
    environmentId,
    environments.map((e) => e.id),
  );
  const envVars = useAppEnvVars(projectId, appId);
  const saved =
    showSaved && envVars.status === "ready" ? savedVariableRows(envVars.data, targetIds) : NO_SAVED;
  const draft = useVariableRows(saved);

  const canSubmit = targetIds.length > 0 && (!showSaved || envVars.status === "ready");
  const isDirty = draft.rows.some(isChangedRow) || draft.removed.length > 0;
  const reset = () => {
    draft.reset();
    setEnvironmentId(ALL_ENVIRONMENTS);
  };

  const add = useMutation({
    mutationFn: saveVariables,
    onSuccess: (result) => {
      if (result.status === "taken") {
        draft.showErrors(conflictErrors(draft.rows, new Set(result.keys)));
        return;
      }
      toast.success(
        result.count > 0 ? `Added ${plural(result.count, "variable")}` : "Variables saved",
      );
      reset();
      onSaved?.();
    },
    onError: (err) => {
      if (err instanceof PartialAddError) {
        const saved = environments.filter((e) => err.savedEnvironmentIds.includes(e.id));
        const unsaved = targetIds.filter((id) => !err.savedEnvironmentIds.includes(id));
        if (unsaved.length === 1) {
          setEnvironmentId(unsaved[0]);
        }
        toast.error("Some variables were saved", {
          description:
            saved.length > 0
              ? `Saved to ${saved.map((e) => environmentName(e.slug)).join(", ")}. ${err.message}`
              : err.message,
        });
        return;
      }
      const { message, description } = envVarErrorToast(
        err,
        "Failed to create environment variables",
      );
      toast.error(message, { description });
    },
  });

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const parsed = splitVariableRows(draft.rows);
    if (parsed.errors.size > 0) {
      draft.showErrors(parsed.errors);
      return;
    }
    const changes = savedVariableChanges(draft.rows, draft.removed);
    const unchanged =
      parsed.variables.length === 0 && changes.edited.length === 0 && changes.removed.length === 0;
    if (unchanged || !canSubmit || add.isLoading) {
      return;
    }
    if (parsed.variables.length > MAX_VARIABLES) {
      toast.error(`Add at most ${plural(MAX_VARIABLES, "variable")} at once`, {
        description: `Remove ${plural(parsed.variables.length - MAX_VARIABLES, "row")}, then save again.`,
      });
      return;
    }
    add.mutate({
      projectId,
      appId,
      environmentIds: [...targetIds],
      variables: parsed.variables,
      ...changes,
    });
  };

  return {
    draft,
    environmentId,
    setEnvironmentId: (id: string) => {
      draft.clearEdits();
      setEnvironmentId(id);
    },
    canSubmit,
    isDirty,
    isPending: add.isLoading,
    reset,
    onSubmit,
  };
}

type AddEnvVarsForm = ReturnType<typeof useAddEnvVarsForm>;

export function AddEnvVarsFields({
  form,
}: {
  form: Pick<AddEnvVarsForm, "draft" | "environmentId" | "setEnvironmentId">;
}) {
  return (
    <>
      <VariablesTable draft={form.draft} />
      <FormField label="Environment">
        {({ id }) => (
          <EnvironmentSelect
            id={id}
            value={form.environmentId}
            onValueChange={form.setEnvironmentId}
            wrapperClassName="w-60"
          />
        )}
      </FormField>
    </>
  );
}
