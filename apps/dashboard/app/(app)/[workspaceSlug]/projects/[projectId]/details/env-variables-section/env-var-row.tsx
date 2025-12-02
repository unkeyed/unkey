import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Eye, EyeSlash, PenWriting3, Trash } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useState } from "react";
import { EnvVarForm } from "./components/env-var-form";
import type { EnvVar } from "./types";

type EnvVarRowProps = {
  envVar: EnvVar;
  projectId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
  onDelete?: () => void;
  onUpdate?: () => void;
};

export function EnvVarRow({
  envVar,
  projectId,
  getExistingEnvVar,
  onDelete,
  onUpdate,
}: EnvVarRowProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [isRevealed, setIsRevealed] = useState(false);
  const [decryptedValue, setDecryptedValue] = useState<string>();
  const [isLoadingForEdit, setIsLoadingForEdit] = useState(false);

  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const deleteMutation = trpc.deploy.envVar.delete.useMutation();

  const handleDelete = async () => {
    const mutation = deleteMutation.mutateAsync({ envVarId: envVar.id });

    toast.promise(mutation, {
      loading: `Deleting environment variable ${envVar.key}...`,
      success: `Deleted environment variable ${envVar.key}`,
      error: (err) => ({
        message: "Failed to delete environment variable",
        description: err.message || "Please try again",
      }),
    });

    try {
      await mutation;
      onDelete?.();
    } catch {
    }
  };

  const isDeleting = deleteMutation.isLoading;

  // Only recoverable vars can be revealed/decrypted
  const isRecoverable = envVar.type === "recoverable";

  const handleEdit = async () => {
    // For recoverable vars, decrypt first if not already decrypted
    if (isRecoverable && !decryptedValue) {
      setIsLoadingForEdit(true);
      try {
        const result = await decryptMutation.mutateAsync({
          envVarId: envVar.id,
        });
        setDecryptedValue(result.value);
        setIsRevealed(true);
      } catch (error) {
        console.error("Failed to decrypt:", error);
        setIsLoadingForEdit(false);
        return;
      }
      setIsLoadingForEdit(false);
    }
    setIsEditing(true);
  };

  const handleToggleReveal = async () => {
    if (!isRecoverable) {
      return;
    }

    if (isRevealed) {
      setIsRevealed(false);
    } else {
      if (decryptedValue) {
        setIsRevealed(true);
      } else {
        try {
          const result = await decryptMutation.mutateAsync({
            envVarId: envVar.id,
          });
          setDecryptedValue(result.value);
          setIsRevealed(true);
        } catch (error) {
          console.error("Failed to decrypt:", error);
        }
      }
    }
  };

  const isLoading = decryptMutation.isLoading && !isLoadingForEdit;

  if (isEditing) {
    return (
      <EnvVarForm
        envVarId={envVar.id}
        initialData={{
          key: envVar.key,
          value: decryptedValue ?? "",
          type: envVar.type,
        }}
        projectId={projectId}
        getExistingEnvVar={getExistingEnvVar}
        excludeId={envVar.id}
        onSuccess={() => {
          setIsEditing(false);
          setIsRevealed(false);
          onUpdate?.();
        }}
        onCancel={() => {
          setIsEditing(false);
          setIsRevealed(false);
        }}
        className="w-full flex px-4 py-3 bg-gray-2 h-12"
      />
    );
  }

  // Determine what value to display
  const displayValue = (() => {
    if (isLoading) {
      return "Loading...";
    }
    if (isRevealed && decryptedValue) {
      return decryptedValue;
    }
    return "••••••••••••••••";
  })();

  return (
    <div className="w-full px-4 py-3 flex items-center hover:bg-gray-2 transition-colors border-b border-gray-4 last:border-b-0 h-12">
      <div className="flex items-center flex-1 min-w-0">
        <div
          className="text-gray-12 font-medium text-xs font-mono w-48 truncate"
          title={envVar.key}
        >
          {envVar.key}
        </div>
        <span className="text-gray-9 text-xs px-2">=</span>
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <div
            className={cn(
              "text-gray-11 text-xs font-mono truncate flex-1",
              isLoading && "text-gray-7",
            )}
          >
            {displayValue}
          </div>
          {isRecoverable && (
            <Button
              size="icon"
              variant="outline"
              onClick={handleToggleReveal}
              disabled={isLoading}
              className="size-7 text-gray-9 hover:text-gray-11 shrink-0"
              loading={isLoading}
            >
              {isRevealed ? (
                <EyeSlash className="!size-[14px]" iconSize="sm-medium" />
              ) : (
                <Eye className="!size-[14px]" iconSize="sm-medium" />
              )}
            </Button>
          )}
        </div>
      </div>
      <div className="flex items-center gap-2 shrink-0 ml-2">
        <Button
          size="icon"
          variant="outline"
          onClick={handleEdit}
          disabled={isLoadingForEdit}
          loading={isLoadingForEdit}
          className="size-7 text-gray-9"
        >
          <PenWriting3 className="!size-[14px]" iconSize="sm-medium" />
        </Button>
        <Button
          size="icon"
          variant="outline"
          onClick={handleDelete}
          disabled={isDeleting}
          loading={isDeleting}
          className="size-7 text-gray-9"
        >
          <Trash className="!size-[14px]" iconSize="sm-medium" />
        </Button>
      </div>
    </div>
  );
}
