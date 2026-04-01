"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown } from "@unkey/icons";
import {
  Button,
  FormInput,
  FormTextarea,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useCallback, useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";

const editEnvVarSchema = z.object({
  key: z.string().min(1, "Variable name is required"),
  value: z.string(),
  environmentId: z.string().min(1, "Environment is required"),
  description: z.string().optional(),
});

type EditEnvVarFormValues = z.infer<typeof editEnvVarSchema>;

type EnvVarEditRowProps = {
  envVarId: string;
  variableKey: string;
  type: "writeonly" | "recoverable";
  environmentId: string;
  note: string | null;
  onClose: () => void;
};

export function EnvVarEditRow({
  envVarId,
  environmentId,
  variableKey,
  type,
  note,
  onClose,
}: EnvVarEditRowProps) {
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const { environments } = useProjectData();

  const isWriteonly = type === "writeonly";

  const {
    register,
    handleSubmit,
    setValue,
    control,
    formState: { isSubmitting, errors },
  } = useForm<EditEnvVarFormValues>({
    resolver: zodResolver(editEnvVarSchema),
    defaultValues: {
      key: variableKey,
      value: "",
      environmentId,
      description: note ?? "",
    },
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: decryptMutation is not stable
  useEffect(
    function decryptValue() {
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
    },
    [envVarId, isWriteonly, setValue],
  );

  const onSubmit = useCallback(
    async (values: EditEnvVarFormValues) => {
      if (isWriteonly && !values.value) {
        onClose();
        return;
      }

      collection.envVars.update(envVarId, (draft) => {
        draft.key = values.key;
        draft.value = values.value;
        draft.description = values.description || null;
        draft.environmentId = values.environmentId;
      });
      onClose();
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
          disabled={isWriteonly}
          title={isWriteonly ? "You cannot rename sensitive environment variables" : ""}
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
        <Controller
          control={control}
          defaultValue={environmentId}
          name="environmentId"
          render={({ field }) => (
            <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
              <label htmlFor="environment-select" className="text-gray-11 text-[13px]">
                Environment
              </label>
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger
                  id="environment-select"
                  className="capitalize"
                  rightIcon={<ChevronDown className="absolute right-2" iconSize="md-medium" />}
                >
                  <SelectValue placeholder="Select environment" />
                </SelectTrigger>
                <SelectContent className="z-[60]">
                  {environments.map((env) => (
                    <SelectItem key={env.id} value={env.id} className="capitalize">
                      {env.slug}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </fieldset>
          )}
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
