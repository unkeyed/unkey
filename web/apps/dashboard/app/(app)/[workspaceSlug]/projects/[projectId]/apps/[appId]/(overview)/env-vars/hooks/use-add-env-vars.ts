import { addVariables } from "@/lib/collections/deploy/env-vars";
import { getErrorMessage } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import type { VariableRows } from "../components/variables-table/use-variable-rows";
import { conflictErrors, parseVariableRows } from "../components/variables-table/variable-rows";

type Args = {
  projectId: string;
  appId: string;
  draft: VariableRows;
  onAdded: () => void;
};

export function useAddEnvVars({ projectId, appId, draft, onAdded }: Args) {
  const mutation = useMutation({
    mutationFn: addVariables,
    onSuccess: (result) => {
      if (result.status === "taken") {
        draft.showErrors(conflictErrors(draft.rows, result.keys));
        return;
      }
      toast.success(`Added ${result.count} ${result.count === 1 ? "variable" : "variables"}`);
      onAdded();
    },
    onError: (err) => {
      toast.error("Failed to create environment variables", {
        description: getErrorMessage(err),
      });
    },
  });

  const submit = (environmentIds: string[]) => {
    const parsed = parseVariableRows(draft.rows);
    if (!parsed.ok) {
      draft.showErrors(parsed.errors);
      return;
    }
    if (parsed.variables.length === 0 || environmentIds.length === 0) {
      return;
    }
    mutation.mutate({ projectId, appId, environmentIds, variables: parsed.variables });
  };

  return { submit, isPending: mutation.isLoading };
}
