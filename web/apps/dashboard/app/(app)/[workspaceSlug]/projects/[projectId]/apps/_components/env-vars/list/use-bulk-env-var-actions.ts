import { envVarErrorToast } from "@/lib/collections/deploy/env-vars";
import { type EnvVar, useEnvVarMutations } from "@/lib/deploy-data";
import { plural } from "@/lib/fmt";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";

export function useBulkEnvVarActions(selected: EnvVar[], onDone: () => void) {
  const { deleteEnvVars, makeVariablesSensitive } = useEnvVarMutations();
  const makeSensitive = useMutation({
    mutationFn: makeVariablesSensitive,
    onSuccess: (updated) => {
      toast.success(`Marked ${plural(updated, "variable")} as sensitive`);
    },
    onError: (err) => {
      const { message, description } = envVarErrorToast(
        err,
        "Failed to mark variables as sensitive",
      );
      toast.error(message, { description });
    },
    onSettled: onDone,
  });

  const deleteSelected = () => {
    if (selected.length > 0) {
      deleteEnvVars(selected.map((item) => item.id));
    }
    onDone();
  };

  const makeSelectedSensitive = () => {
    if (makeSensitive.isLoading) {
      return;
    }
    const recoverable = selected.filter((item) => item.type === "recoverable");
    if (recoverable.length === 0) {
      toast.info("No recoverable variables are selected");
      onDone();
      return;
    }
    makeSensitive.mutate(recoverable);
  };

  return { deleteSelected, makeSelectedSensitive, isMakingSensitive: makeSensitive.isLoading };
}
