import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Eye, EyeSlash, PenWriting3, Trash } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { EnvVarForm } from "./components/env-var-form";
import type { EnvVar } from "./types";

type EnvVarRowProps = {
  envVar: EnvVar;
  projectId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
};

export function EnvVarRow({ envVar, projectId, getExistingEnvVar }: EnvVarRowProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [isDecrypted, setIsDecrypted] = useState(false);
  // INFO: Won't be necessary once we add tRPC then we can use isSubmitting
  const [isSecretLoading, setIsSecretLoading] = useState(false);
  const [decryptedValue, setDecryptedValue] = useState<string>();

  const trpcUtils = trpc.useUtils();

  // TODO: Add mutations when available
  // const deleteMutation = trpc.deploy.project.envs.delete.useMutation();
  // const decryptMutation = trpc.deploy.project.envs.decrypt.useMutation();

  const handleDelete = async () => {
    try {
      // TODO: Call tRPC delete when available
      // await deleteMutation.mutateAsync({
      //   projectId,
      //   id: envVar.id
      // });

      // Mock successful delete for now
      await new Promise((resolve) => setTimeout(resolve, 300));

      // Invalidate to refresh data
      await trpcUtils.deploy.project.envs.getEnvs.invalidate({ projectId });
    } catch (error) {
      console.error("Failed to delete env var:", error);
    }
  };

  const handleToggleSecret = async () => {
    if (envVar.type !== "secret") {
      return;
    }

    // This stupid nested branching won't be necessary once we have the actual tRPC. So disregard this when reviewing
    if (isDecrypted) {
      setIsDecrypted(false);
    } else {
      if (decryptedValue) {
        setIsDecrypted(true);
      } else {
        setIsSecretLoading(true);
        try {
          // TODO: Call tRPC decrypt when available
          // const result = await decryptMutation.mutateAsync({
          //   projectId,
          //   envVarId: envVar.id
          // });

          // Mock decrypted value for now
          await new Promise((resolve) => setTimeout(resolve, 800));
          const mockDecrypted = `decrypted-${envVar.key}`;

          setDecryptedValue(mockDecrypted);
          setIsDecrypted(true);
        } catch (error) {
          console.error("Failed to decrypt secret:", error);
        } finally {
          setIsSecretLoading(false);
        }
      }
    }
  };

  if (isEditing) {
    return (
      <EnvVarForm
        initialData={{
          key: envVar.key,
          value: envVar.type === "secret" && !isDecrypted ? "" : (decryptedValue ?? envVar.value),
          type: envVar.type,
        }}
        projectId={projectId}
        decrypted={isDecrypted}
        getExistingEnvVar={getExistingEnvVar}
        excludeId={envVar.id}
        onSuccess={() => setIsEditing(false)}
        onCancel={() => setIsEditing(false)}
        className="w-full flex px-4 py-3 bg-gray-2 h-12"
      />
    );
  }

  return (
    <div className="w-full px-4 py-3 flex items-center hover:bg-gray-2 transition-colors border-b border-gray-4 last:border-b-0 h-12">
      <div className="flex items-center flex-1 min-w-0">
        <div className="text-gray-12 font-medium text-xs font-mono w-28 truncate">{envVar.key}</div>
        <span className="text-gray-9 text-xs px-2">=</span>
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <div
            className={cn(
              "text-gray-11 text-xs font-mono truncate flex-1",
              isSecretLoading && "text-gray-7",
            )}
          >
            {envVar.type === "secret" && !isDecrypted
              ? "••••••••••••••••"
              : envVar.type === "secret" && isSecretLoading
                ? "Loading..."
                : envVar.type === "secret" && isDecrypted && decryptedValue
                  ? decryptedValue
                  : envVar.value}
          </div>
          {envVar.type === "secret" && (
            <Button
              size="icon"
              variant="outline"
              onClick={handleToggleSecret}
              disabled={isSecretLoading}
              className="size-7 text-gray-9 hover:text-gray-11 shrink-0"
              loading={isSecretLoading}
            >
              {isDecrypted ? (
                <EyeSlash className="!size-[14px]" size="sm-medium" />
              ) : (
                <Eye className="!size-[14px]" size="sm-medium" />
              )}
            </Button>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2 shrink-0 ml-2">
        <Button
          size="icon"
          variant="outline"
          onClick={() => {
            handleToggleSecret().then(() => setIsEditing(true));
          }}
          className="size-7 text-gray-9"
        >
          <PenWriting3 className="!size-[14px]" size="sm-medium" />
        </Button>
        <Button size="icon" variant="outline" onClick={handleDelete} className="size-7 text-gray-9">
          <Trash className="!size-[14px]" size="sm-medium" />
        </Button>
      </div>
    </div>
  );
}
