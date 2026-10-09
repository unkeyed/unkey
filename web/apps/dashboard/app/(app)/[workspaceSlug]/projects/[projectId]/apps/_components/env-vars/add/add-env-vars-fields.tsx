import { addVariables, envVarErrorToast } from "@/lib/collections/deploy/env-vars";
import { plural } from "@/lib/fmt";
import { useMutation } from "@tanstack/react-query";
import { FormField, toast } from "@unkey/ui";
import { type FormEvent, useState } from "react";
import { useAppId, useProjectData } from "../../../[appId]/(overview)/data-provider";
import { ALL_ENVIRONMENTS, EnvironmentSelect } from "../shared/environment-select";
import { useVariableRows } from "../variables-table/use-variable-rows";
import { conflictErrors, isBlankRow, parseVariableRows } from "../variables-table/variable-rows";
import { VariablesTable } from "../variables-table/variables-table";

export function useAddEnvVarsForm({ onAdded }: { onAdded?: () => void } = {}) {
  const appId = useAppId();
  const { environments } = useProjectData();
  const draft = useVariableRows();
  const [environmentId, setEnvironmentId] = useState(ALL_ENVIRONMENTS);
  const targetEnvironmentIds =
    environmentId === ALL_ENVIRONMENTS ? environments.map((e) => e.id) : [environmentId];

  const isDirty = draft.rows.some((row) => !isBlankRow(row));
  const reset = () => {
    draft.reset();
    setEnvironmentId(ALL_ENVIRONMENTS);
  };

  const add = useMutation({
    mutationFn: addVariables,
    onSuccess: (result) => {
      if (result.status === "taken") {
        draft.showErrors(conflictErrors(draft.rows, new Set(result.keys)));
        return;
      }
      toast.success(`Added ${plural(result.count, "variable")}`);
      reset();
      onAdded?.();
    },
    onError: (err) => {
      const { message, description } = envVarErrorToast(
        err,
        "Failed to create environment variables",
      );
      toast.error(message, { description });
    },
  });

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const parsed = parseVariableRows(draft.rows);
    if (!parsed.ok) {
      draft.showErrors(parsed.errors);
      return;
    }
    if (parsed.variables.length === 0 || targetEnvironmentIds.length === 0 || add.isLoading) {
      return;
    }
    add.mutate({ appId, environmentIds: targetEnvironmentIds, variables: parsed.variables });
  };

  return {
    draft,
    environmentId,
    setEnvironmentId,
    targetEnvironmentIds,
    isDirty,
    isPending: add.isLoading,
    reset,
    onSubmit,
  };
}

type AddEnvVarsForm = ReturnType<typeof useAddEnvVarsForm>;

export function AddEnvVarsFields({ form }: { form: AddEnvVarsForm }) {
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
