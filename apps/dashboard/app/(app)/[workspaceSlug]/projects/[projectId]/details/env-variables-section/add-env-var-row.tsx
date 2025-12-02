import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EnvVarInputs } from "./components/env-var-inputs";
import { EnvVarSaveActions } from "./components/env-var-save-actions";
import { EnvVarSecretSwitch } from "./components/env-var-secret-switch";
import { type EnvVar, type EnvVarFormData, EnvVarFormSchema } from "./types";

type AddEnvVarRowProps = {
  environmentId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
  onCancel: () => void;
  onSuccess: () => void;
};

export function AddEnvVarRow({
  environmentId,
  getExistingEnvVar,
  onCancel,
  onSuccess,
}: AddEnvVarRowProps) {
  const createMutation = trpc.deploy.envVar.create.useMutation();

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isValid },
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
      type: "recoverable",
    },
  });

  const watchedType = watch("type");
  const isSubmitting = createMutation.isLoading;

  const handleSave = async (formData: EnvVarFormData) => {
    try {
      await createMutation.mutateAsync({
        environmentId,
        variables: [
          {
            key: formData.key,
            value: formData.value,
            type: formData.type,
          },
        ],
      });
      onSuccess();
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
          setValue={setValue}
          errors={errors}
          isSecret={watchedType === "writeonly"}
          onKeyDown={handleKeyDown}
          autoFocus
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
