import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";

export type EnvVariable = {
  id: string;
  key: string;
  type: "writeonly" | "recoverable";
};

export function useDecryptedValues(variables: EnvVariable[]) {
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const [decryptedValues, setDecryptedValues] = useState<Record<string, string>>({});
  const [isDecrypting, setIsDecrypting] = useState(false);

  const variableFingerprint = useMemo(
    () =>
      variables
        .map((v) => v.id)
        .sort()
        .join(","),
    [variables],
  );

  // biome-ignore lint/correctness/useExhaustiveDependencies: its safe to keep
  useEffect(() => {
    if (variables.length === 0) {
      return;
    }

    const recoverableVars = variables.filter((v) => v.type === "recoverable");
    if (recoverableVars.length === 0) {
      return;
    }

    setIsDecrypting(true);
    Promise.all(
      recoverableVars.map((v) =>
        decryptMutation.mutateAsync({ envVarId: v.id }).then((r) => [v.id, r.value] as const),
      ),
    )
      .then((entries) => {
        setDecryptedValues(Object.fromEntries(entries));
      })
      .finally(() => {
        setIsDecrypting(false);
      });
  }, [variableFingerprint]);

  return { decryptedValues, isDecrypting };
}
