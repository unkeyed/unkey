import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { type EnvVar, type EnvVarFormData, EnvVarFormSchema } from "../types";
import { EnvVarInputs } from "./env-var-inputs";
import { EnvVarSaveActions } from "./env-var-save-actions";
import { EnvVarSecretSwitch } from "./env-var-secret-switch";

type EnvVarFormProps = {
  initialData: EnvVarFormData;
  projectId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
  onSuccess: () => void;
  onCancel: () => void;
  excludeId?: string;
  autoFocus?: boolean;
  decrypted?: boolean;
  className?: string;
};

export function EnvVarForm({
  initialData,
  projectId,
  getExistingEnvVar,
  onSuccess,
  onCancel,
  excludeId,
  decrypted,
  autoFocus = false,
  className = "w-full flex px-4 py-3 bg-gray-2 border-b border-gray-4 last:border-b-0",
}: EnvVarFormProps) {
  console.debug(projectId);
  // TODO: Add mutations when available
  // const upsertMutation = trpc.deploy.project.envs.upsert.useMutation();

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isValid, isSubmitting },
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

  const handleSave = async (_formData: EnvVarFormData) => {
    try {
      // TODO: Call tRPC upsert when available
      // await upsertMutation.mutateAsync({
      //   projectId,
      //   id: excludeId, // Will be undefined for new entries
      //   ...formData
      // });

      // Mock successful save for now
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Invalidate to refresh data
      // await trpcUtils.deploy.project.envs.getEnvs.invalidate({ projectId });

      onSuccess();
    } catch (error) {
      console.error("Failed to save env var:", error);
      throw error; // Re-throw to let form handle the error state
    }
  };

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
          errors={errors}
          isSecret={watchedType === "secret"}
          decrypted={decrypted}
          onKeyDown={handleKeyDown}
          autoFocus={autoFocus}
        />
        <div className="flex items-center gap-2 ml-auto">
          <EnvVarSecretSwitch
            isSecret={watchedType === "secret"}
            onCheckedChange={(checked) =>
              setValue("type", checked ? "secret" : "env", {
                shouldDirty: true,
                shouldValidate: true,
              })
            }
            disabled={isSubmitting}
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
