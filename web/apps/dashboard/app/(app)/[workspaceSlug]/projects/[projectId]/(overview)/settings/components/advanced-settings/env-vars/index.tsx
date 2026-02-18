"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeSlash, Nodes2, Plus, Trash } from "@unkey/icons";
import { Button, FormCheckbox, FormInput, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useMemo, useState } from "react";
import {
  type Control,
  Controller,
  type UseFormRegister,
  useFieldArray,
  useForm,
  useWatch,
} from "react-hook-form";
import { useProjectData } from "../../../../data-provider";
import { FormSettingCard } from "../../shared/form-setting-card";
import { EMPTY_ROW, type EnvVarsFormValues, envVarsSchema } from "./schema";
import { useDropZone } from "./use-drop-zone";

const toTrpcType = (secret: boolean) => (secret ? "writeonly" : "recoverable");

export const EnvVars = () => {
  const { projectId, environments } = useProjectData();
  const environmentId = environments[0]?.id;
  const envSlug = environments[0]?.slug;

  const { data } = trpc.deploy.envVar.list.useQuery({ projectId }, { enabled: Boolean(projectId) });

  const envData = envSlug ? data?.[envSlug] : undefined;

  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const [decryptedValues, setDecryptedValues] = useState<Record<string, string>>({});
  const [isDecrypting, setIsDecrypting] = useState(false);

  // biome-ignore lint/correctness/useExhaustiveDependencies:  its already stable
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
        decryptMutation.mutateAsync({ envVarId: v.id }).then((r) => [v.id, r.value] as const),
      ),
    )
      .then((entries) => {
        setDecryptedValues(Object.fromEntries(entries));
        setIsDecrypting(false);
      })
      .catch(() => setIsDecrypting(false));
  }, [envData]);

  const defaultValues = useMemo<EnvVarsFormValues>(() => {
    if (!envData || envData.variables.length === 0) {
      return { envVars: [{ ...EMPTY_ROW }] };
    }
    return {
      envVars: envData.variables.map((v) => ({
        id: v.id,
        key: v.key,
        value: v.type === "writeonly" ? "" : (decryptedValues[v.id] ?? ""),
        secret: v.type === "writeonly",
      })),
    };
  }, [envData, decryptedValues]);

  return (
    <EnvVarsForm
      defaultValues={defaultValues}
      environmentId={environmentId}
      projectId={projectId}
      isDecrypting={isDecrypting}
    />
  );
};

type EnvVarsFormProps = {
  defaultValues: EnvVarsFormValues;
  environmentId: string | undefined;
  projectId: string;
  isDecrypting: boolean;
};

const EnvVarsForm: React.FC<EnvVarsFormProps> = ({
  defaultValues,
  environmentId,
  projectId,
  isDecrypting,
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors },
    control,
    reset,
  } = useForm<EnvVarsFormValues>({
    resolver: zodResolver(envVarsSchema),
    mode: "onChange",
    defaultValues,
  });

  const { ref, isDragging } = useDropZone(reset);

  useEffect(() => {
    reset(defaultValues);
  }, [reset, defaultValues]);

  const { fields, append, remove } = useFieldArray({ control, name: "envVars" });

  const currentEnvVars = useWatch({ control, name: "envVars" });

  const hasChanges = JSON.stringify(currentEnvVars) !== JSON.stringify(defaultValues.envVars);

  const createMutation = trpc.deploy.envVar.create.useMutation();
  const updateMutation = trpc.deploy.envVar.update.useMutation();
  const deleteMutation = trpc.deploy.envVar.delete.useMutation();

  const isSaving =
    createMutation.isLoading ||
    updateMutation.isLoading ||
    deleteMutation.isLoading ||
    isSubmitting;

  const onSubmit = async (values: EnvVarsFormValues) => {
    if (!environmentId) {
      return;
    }

    const originalVars = defaultValues.envVars.filter((v) => v.id);
    const originalIds = new Set(originalVars.map((v) => v.id as string));
    const originalMap = new Map(originalVars.map((v) => [v.id as string, v]));

    const currentIds = new Set(values.envVars.filter((v) => v.id).map((v) => v.id as string));

    const toDelete = [...originalIds].filter((id) => !currentIds.has(id));

    const toCreate = values.envVars.filter((v) => !v.id && v.key !== "" && v.value !== "");

    const toUpdate = values.envVars.filter((v) => {
      if (!v.id) {
        return false;
      }
      const original = originalMap.get(v.id);
      if (!original) {
        return false;
      }
      if (v.value === "") {
        return false;
      }
      return v.key !== original.key || v.value !== original.value || v.secret !== original.secret;
    });

    try {
      await Promise.all([
        ...toDelete.map(async (id) => {
          const key = originalMap.get(id)?.key ?? id;
          try {
            return await deleteMutation.mutateAsync({ envVarId: id });
          } catch (err) {
            throw new Error(`"${key}": ${err instanceof Error ? err.message : "Failed to delete"}`);
          }
        }),
        ...(toCreate.length > 0
          ? [
              createMutation.mutateAsync({
                environmentId,
                variables: toCreate.map((v) => ({
                  key: v.key,
                  value: v.value,
                  type: toTrpcType(v.secret),
                })),
              }),
            ]
          : []),
        ...toUpdate.map((v) =>
          updateMutation
            .mutateAsync({
              envVarId: v.id as string,
              key: v.key,
              value: v.value,
              type: toTrpcType(v.secret),
            })
            .catch((err) => {
              throw new Error(
                `"${v.key}": ${err instanceof Error ? err.message : "Failed to update"}`,
              );
            }),
        ),
      ]);

      utils.deploy.envVar.list.invalidate({ projectId });
      toast.success("Environment variables saved");
    } catch (err) {
      toast.error("Failed to save environment variables", {
        description:
          err instanceof Error
            ? err.message
            : "An unexpected error occurred. Please try again or contact support@unkey.com",
      });
    }
  };

  const displayValue = (() => {
    const vars = defaultValues.envVars.filter((v) => v.key !== "");
    if (vars.length === 0) {
      return <span className="text-gray-11 font-normal">None</span>;
    }
    if (vars.length === 1) {
      return <span className="font-medium text-gray-12 font-mono text-xs">{vars[0].key}</span>;
    }
    return <span className="text-gray-11 font-normal">{vars.length} variables</span>;
  })();

  return (
    <FormSettingCard
      icon={<Nodes2 className="text-gray-12" iconSize="xl-medium" />}
      title="Environment Variables"
      description="Set environment variables available at runtime. Changes apply on next deploy."
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSaving && !isDecrypting && hasChanges && Boolean(environmentId)}
      isSaving={isSaving}
      ref={ref}
      className={cn("relative", isDragging && "bg-primary/5")}
    >
      <div
        className={cn(
          "absolute inset-2 rounded-lg border-2 border-dotted pointer-events-none transition-colors",
          isDragging ? "border-successA-9" : "border-transparent",
        )}
      />
      <div className="flex flex-col gap-2 w-[520px]">
        <p className="text-xs text-gray-11 mb-2">
          Drag & drop your <span className="font-mono font-medium text-gray-12">.env</span> file or
          paste <span className="font-mono font-medium text-gray-12">env</span> vars (⌘V / Ctrl+V)
        </p>

        <div className="flex items-center gap-2">
          <span className="flex-1 text-[13px] text-gray-11">Key</span>
          <span className="flex-1 text-[13px] text-gray-11">Value</span>
          <span className="w-9 text-center text-[13px] text-gray-11">Sensitive</span>
          <div className="w-16 shrink-0" />
        </div>

        {fields.map((field, index) => {
          const isLast = index === fields.length - 1;
          const isOnly = fields.length === 1;
          const keyError = errors.envVars?.[index]?.key?.message;
          const isPreviouslyAdded =
            index < defaultValues.envVars.length && defaultValues.envVars[index].key !== "";

          return (
            <EnvVarRow
              key={field.id}
              index={index}
              isLast={isLast}
              isOnly={isOnly}
              keyError={keyError}
              isPreviouslyAdded={isPreviouslyAdded}
              isSecret={currentEnvVars[index]?.secret ?? false}
              control={control}
              register={register}
              onAdd={() => append({ ...EMPTY_ROW })}
              onRemove={() => remove(index)}
            />
          );
        })}
      </div>
    </FormSettingCard>
  );
};

type EnvVarRowProps = {
  index: number;
  isLast: boolean;
  isOnly: boolean;
  keyError: string | undefined;
  isPreviouslyAdded: boolean;
  isSecret: boolean;
  control: Control<EnvVarsFormValues>;
  register: UseFormRegister<EnvVarsFormValues>;
  onAdd: () => void;
  onRemove: () => void;
};

const EnvVarRow = ({
  index,
  isLast,
  isOnly,
  keyError,
  isPreviouslyAdded,
  isSecret,
  control,
  register,
  onAdd,
  onRemove,
}: EnvVarRowProps) => {
  const [isVisible, setIsVisible] = useState(false);

  const inputType = isPreviouslyAdded ? (isVisible ? "text" : "password") : "text";

  const eyeButton =
    isPreviouslyAdded && !isSecret ? (
      <button
        type="button"
        className="text-gray-9 hover:text-gray-11 transition-colors"
        onClick={() => setIsVisible((v) => !v)}
        tabIndex={-1}
      >
        {isVisible ? <EyeSlash iconSize="sm-regular" /> : <Eye iconSize="sm-regular" />}
      </button>
    ) : undefined;

  return (
    <div className="flex items-start gap-2">
      <FormInput
        className="flex-1 [&_input]:h-9  [&_input]:font-mono"
        placeholder="MY_VAR"
        error={keyError}
        {...register(`envVars.${index}.key`)}
      />
      <FormInput
        className="flex-1 [&_input]:h-9  [&_input]:font-mono"
        placeholder={isPreviouslyAdded && isSecret ? "sensitive" : "value"}
        type={inputType}
        rightIcon={eyeButton}
        {...register(`envVars.${index}.value`)}
      />
      <div className="bg-grayA-3 size-9 flex items-center justify-center rounded-lg">
        <Controller
          control={control}
          name={`envVars.${index}.secret`}
          render={({ field }) => (
            <FormCheckbox
              disabled={isPreviouslyAdded && isSecret}
              className="bg-white data-[state=checked]:bg-white data-[state=unchecked]:bg-white rounded"
              checked={field.value}
              onCheckedChange={field.onChange}
            />
          )}
        />
      </div>
      <div className="relative w-16 h-7 mt-1 shrink-0">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={`absolute left-0 w-7 px-0 justify-center text-error-11 hover:text-error-11 transition-opacity duration-150 ${isOnly ? "opacity-0 pointer-events-none" : "opacity-100"}`}
          onClick={onRemove}
        >
          <Trash iconSize="sm-regular" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={`absolute left-0 w-7 px-0 justify-center transition-all duration-150 ${isOnly ? "translate-x-0" : "translate-x-9"} ${isLast ? "opacity-100" : "opacity-0 pointer-events-none"}`}
          onClick={onAdd}
        >
          <Plus iconSize="sm-regular" />
        </Button>
      </div>
    </div>
  );
};
