"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, FormTextarea, toast } from "@unkey/ui";
import { useCallback, useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const editEnvVarSchema = z.object({
  key: z.string().min(1, "Variable name is required"),
  value: z.string(),
  description: z.string().optional(),
});

type EditEnvVarFormValues = z.infer<typeof editEnvVarSchema>;

type EnvVarEditRowProps = {
  envVarId: string;
  variableKey: string;
  type: "writeonly" | "recoverable";
  note: string | null;
  onClose: () => void;
};

export function EnvVarEditRow({ envVarId, variableKey, type, note, onClose }: EnvVarEditRowProps) {
  const isWriteonly = type === "writeonly";
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();

  const {
    register,
    handleSubmit,
    setValue,
    formState: { isSubmitting, errors },
  } = useForm<EditEnvVarFormValues>({
    resolver: zodResolver(editEnvVarSchema),
    defaultValues: {
      key: variableKey,
      value: "",
      description: note ?? "",
    },
  });

  useEffect(() => {
    if (isWriteonly) {
      return;
    }
    let cancelled = false;
    decryptMutation.mutateAsync({ envVarId }).then(
      (result) => {
        if (!cancelled) {
          setValue("value", result.value);
        }
      },
      () => {
        if (!cancelled) {
          toast.error("Failed to decrypt value");
        }
      },
    );
    return () => {
      cancelled = true;
    };
  }, [envVarId, isWriteonly, setValue, decryptMutation]);

  const onSubmit = useCallback(
    async (values: EditEnvVarFormValues) => {
      if (isWriteonly && !values.value) {
        onClose();
        return;
      }

      try {
        collection.envVars.update(envVarId, (draft) => {
          draft.key = values.key;
          draft.value = values.value;
          draft.description = values.description || null;
        });
        toast.success("Variable updated");
        onClose();
      } catch {
        toast.error("Failed to update variable");
      }
    },
    [envVarId, isWriteonly, onClose],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    },
    [onClose],
  );

  return (
    <div className="bg-gray-1 px-6 pb-6 pt-5 border-t border-grayA-4" onKeyDown={handleKeyDown}>
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-5">
        <FormInput
          label="Key"
          className="[&_input]:font-mono"
          placeholder="VARIABLE_NAME"
          error={errors.key?.message}
          readOnly={isWriteonly}
          {...register("key")}
        />
        <FormTextarea
          label={isWriteonly ? "New Value" : "Value"}
          className="[&_textarea]:font-mono"
          placeholder={
            isWriteonly
              ? "Enter new value to replace"
              : decryptMutation.isLoading
                ? "Decrypting..."
                : "value"
          }
          rows={3}
          disabled={decryptMutation.isLoading}
          {...register("value")}
        />
        <FormInput
          label="Note"
          className="[&_input]:text-sm"
          placeholder="Optional description for this variable..."
          {...register("description")}
        />
        <div className="flex items-center justify-end gap-2 pt-5 mt-1">
          <Button type="button" variant="outline" size="md" onClick={onClose} className="px-3">
            Cancel
          </Button>
          <Button
            type="submit"
            className="px-3"
            variant="primary"
            size="md"
            loading={isSubmitting}
            disabled={isSubmitting}
          >
            Save
          </Button>
        </div>
      </form>
    </div>
  );
}
