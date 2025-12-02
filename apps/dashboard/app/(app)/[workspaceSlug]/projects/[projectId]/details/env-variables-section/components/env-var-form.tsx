import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "@unkey/ui";
import { useForm } from "react-hook-form";
import { type EnvVar, type EnvVarFormData, EnvVarFormSchema } from "../types";
import { EnvVarInputs } from "./env-var-inputs";
import { EnvVarSaveActions } from "./env-var-save-actions";
import { EnvVarSecretSwitch } from "./env-var-secret-switch";

type EnvVarFormProps = {
  envVarId: string;
  initialData: EnvVarFormData;
  projectId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
  onSuccess: () => void;
  onCancel: () => void;
  excludeId?: string;
  autoFocus?: boolean;
  className?: string;
};

export function EnvVarForm({
  envVarId,
  initialData,
  projectId: _projectId,
  getExistingEnvVar,
  onSuccess,
  onCancel,
  excludeId,
  autoFocus = false,
  className = "w-full flex px-4 py-3 bg-gray-2 border-b border-gray-4 last:border-b-0",
}: EnvVarFormProps) {
  const updateMutation = trpc.deploy.envVar.update.useMutation();

  // Writeonly vars cannot have their key renamed
  const isWriteOnly = initialData.type === "writeonly";

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isValid },
  } = useForm<EnvVarFormData>({
    resolver: zodResolver(
      EnvVarFormSchema.superRefine((data, ctx) => {
        const existing = getExistingEnvVar(data.key, excludeId);
        if (existing) {
          ctx.addIssue({
            code: "custom",
            message: "Variable name already exists",
            path: ["key"],
          });
        }
      }),
    ),
    defaultValues: initialData,
  });

  const watchedType = watch("type");

  const handleSave = async (formData: EnvVarFormData) => {
    const mutation = updateMutation.mutateAsync({
      envVarId,
      key: isWriteOnly ? undefined : formData.key,
      value: formData.value,
      type: formData.type,
    });

    toast.promise(mutation, {
      loading: `Updating environment variable ${formData.key}...`,
      success: `Updated environment variable ${formData.key}`,
      error: (err) => ({
        message: "Failed to update environment variable",
        description: err.message || "Please try again",
      }),
    });

    try {
      await mutation;
      onSuccess();
    } catch {
    }
  };

  const isSubmitting = updateMutation.isLoading;

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && isValid && !isSubmitting) {
      handleSubmit(handleSave)();
    } else if (e.key === "Escape") {
      onCancel();
    }
  };

  return (
    <div className={className}>
      <form onSubmit={handleSubmit(handleSave)} className="w-full flex items-center gap-2">
        <EnvVarInputs
          register={register}
          setValue={setValue}
          errors={errors}
          isSecret={watchedType === "writeonly"}
          keyDisabled={isWriteOnly}
          onKeyDown={handleKeyDown}
          autoFocus={autoFocus}
        />
        <div className="flex items-center gap-2 ml-auto">
          <EnvVarSecretSwitch
            isSecret={watchedType === "writeonly"}
            onCheckedChange={(checked) =>
              setValue("type", checked ? "writeonly" : "recoverable", {
                shouldDirty: true,
                shouldValidate: true,
              })
            }
            disabled={isSubmitting || Boolean(envVarId)}
          />
          <EnvVarSaveActions
            isSubmitting={isSubmitting}
            save={{
              disabled: !isValid || isSubmitting,
            }}
            cancel={{
              disabled: isSubmitting,
              onClick: onCancel,
            }}
          />
        </div>
      </form>
    </div>
  );
}
