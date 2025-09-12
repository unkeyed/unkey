import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EnvVarInputs } from "./components/env-var-inputs";
import { EnvVarSaveActions } from "./components/env-var-save-actions";
import { EnvVarSecretSwitch } from "./components/env-var-secret-switch";
import { type EnvVar, type EnvVarFormData, EnvVarFormSchema } from "./types";

type AddEnvVarRowProps = {
  projectId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
  onCancel: () => void;
};

export function AddEnvVarRow({ projectId, getExistingEnvVar, onCancel }: AddEnvVarRowProps) {
  const trpcUtils = trpc.useUtils();

  // TODO: Add mutation when available
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
        const existing = getExistingEnvVar(data.key);
        if (existing) {
          ctx.addIssue({
            code: "custom",
            message: "Variable name already exists",
            path: ["key"],
          });
        }
      }),
    ),
    defaultValues: {
      key: "",
      value: "",
      type: "env",
    },
  });

  const watchedType = watch("type");

  const handleSave = async (_formData: EnvVarFormData) => {
    try {
      // TODO: Call tRPC upsert when available
      // await upsertMutation.mutateAsync({
      //   projectId,
      //   ...formData
      // });

      // Mock successful save for now
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Invalidate to refresh data
      await trpcUtils.deploy.project.envs.getEnvs.invalidate({ projectId });

      onCancel(); // Close the add form
    } catch (error) {
      console.error("Failed to add env var:", error);
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
    <div className="w-full flex px-4 py-3 bg-gray-2 border-b border-gray-4 last:border-b-0">
      <form onSubmit={handleSubmit(handleSave)} className="w-full flex items-center gap-2 h-12">
        <EnvVarInputs
          register={register}
          errors={errors}
          isSecret={watchedType === "secret"}
          onKeyDown={handleKeyDown}
          autoFocus
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
