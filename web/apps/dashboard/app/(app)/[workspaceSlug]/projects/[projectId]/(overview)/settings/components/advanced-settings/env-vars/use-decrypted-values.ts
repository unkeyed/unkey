import { trpc } from "@/lib/trpc/client";
import { useEffect, useState } from "react";

type EnvVariable = {
  id: string;
  key: string;
  type: "writeonly" | "recoverable";
};

export type EnvData = {
  variables: EnvVariable[];
};

export function useDecryptedValues(envData: EnvData | undefined) {
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const [decryptedValues, setDecryptedValues] = useState<Record<string, string>>({});
  const [isDecrypting, setIsDecrypting] = useState(false);

  useEffect(() => {
    if (!envData) {
      return;
    }

    const recoverableVars = envData.variables.filter((v) => v.type === "recoverable");
    if (recoverableVars.length === 0) {
      return;
    }

    setIsDecrypting(true);
    Promise.all(
      recoverableVars.map((v) =>
        decryptMutation.mutateAsync({ envVarId: v.id }).then((r) => [v.id, r.value] as const)
      )
    )
      .then((entries) => {
        setDecryptedValues(Object.fromEntries(entries));
      })
      .finally(() => {
        setIsDecrypting(false);
      });
  }, [envData]);

  return { decryptedValues, isDecrypting };
}
