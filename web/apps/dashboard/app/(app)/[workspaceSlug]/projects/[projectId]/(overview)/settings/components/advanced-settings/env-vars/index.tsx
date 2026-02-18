"use client";

import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Nodes2 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { useProjectData } from "../../../../data-provider";
import { FormSettingCard } from "../../shared/form-setting-card";
import { EnvVarRow } from "./env-var-row";
import { type EnvVarsFormValues, createEmptyRow, envVarsSchema } from "./schema";
import { useDecryptedValues } from "./use-decrypted-values";
import { useDropZone } from "./use-drop-zone";
import { computeEnvVarsDiff, groupByEnvironment, toTrpcType } from "./utils";

export const EnvVars = () => {
  const { projectId, environments } = useProjectData();

  const defaultEnvironmentId =
    environments.find((e) => e.slug === "production")?.id ?? environments[0]?.id;

  const { data } = trpc.deploy.envVar.list.useQuery({ projectId }, { enabled: Boolean(projectId) });

  const allVariables = useMemo(() => {
    if (!data) {
      return [];
    }
    return environments.flatMap((env) => {
      const envData = data[env.slug];
      if (!envData) {
        return [];
      }
      return envData.variables.map((v) => ({
        ...v,
        environmentId: env.id,
      }));
    });
  }, [data, environments]);

  const { decryptedValues, isDecrypting } = useDecryptedValues(allVariables);

  const defaultValues = useMemo<EnvVarsFormValues>(() => {
    if (allVariables.length === 0) {
      return { envVars: [createEmptyRow(defaultEnvironmentId)] };
    }
    return {
      envVars: allVariables.map((v) => ({
        id: v.id,
        environmentId: v.environmentId,
        key: v.key,
        value: v.type === "writeonly" ? "" : (decryptedValues[v.id] ?? ""),
        secret: v.type === "writeonly",
      })),
    };
  }, [allVariables, decryptedValues, defaultEnvironmentId]);

  const formKey = useMemo(() => {
    const varIds = allVariables.map((v) => v.id).join("-") || "empty";
    const decryptedIds = Object.keys(decryptedValues).sort().join("-") || "none";
    return `${varIds}:${decryptedIds}`;
  }, [allVariables, decryptedValues]);

  if (!defaultEnvironmentId) {
    return null;
  }

  return (
    <EnvVarsForm
      key={formKey}
      defaultValues={defaultValues}
      defaultEnvironmentId={defaultEnvironmentId}
      environments={environments}
      projectId={projectId}
      isDecrypting={isDecrypting}
    />
  );
};

const EnvVarsForm = ({
  defaultValues,
  defaultEnvironmentId,
  environments,
  projectId,
  isDecrypting,
}: {
  defaultValues: EnvVarsFormValues;
  defaultEnvironmentId: string;
  environments: { id: string; slug: string }[];
  projectId: string;
  isDecrypting: boolean;
}) => {
  const utils = trpc.useUtils();

  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors, isDirty },
    control,
    reset,
  } = useForm<EnvVarsFormValues>({
    resolver: zodResolver(envVarsSchema),
    mode: "onChange",
    defaultValues,
  });

  const { ref, isDragging } = useDropZone(reset, defaultEnvironmentId);
  const { fields, append, remove } = useFieldArray({ control, name: "envVars" });

  const createMutation = trpc.deploy.envVar.create.useMutation();
  const updateMutation = trpc.deploy.envVar.update.useMutation();
  const deleteMutation = trpc.deploy.envVar.delete.useMutation();

  const isSaving =
    createMutation.isLoading ||
    updateMutation.isLoading ||
    deleteMutation.isLoading ||
    isSubmitting;

  const onSubmit = async (values: EnvVarsFormValues) => {
    const { toDelete, toCreate, toUpdate, originalMap } = computeEnvVarsDiff(
      defaultValues.envVars,
      values.envVars,
    );

    const createsByEnv = groupByEnvironment(toCreate);

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
        ...[...createsByEnv.entries()].map(([envId, vars]) =>
          createMutation.mutateAsync({
            environmentId: envId,
            variables: vars.map((v) => ({
              key: v.key,
              value: v.value,
              type: toTrpcType(v.secret),
            })),
          }),
        ),
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

  const varCount = defaultValues.envVars.filter((v) => v.key !== "").length;
  const displayValue =
    varCount === 0 ? (
      <span className="text-gray-11 font-normal">None</span>
    ) : (
      <div className="space-x-1">
        <span className="font-medium text-gray-12">{varCount}</span>
        <span className="text-gray-11 font-normal">variable{varCount !== 1 ? "s" : ""}</span>
      </div>
    );

  return (
    <FormSettingCard
      icon={<Nodes2 className="text-gray-12" iconSize="xl-medium" />}
      title="Environment Variables"
      description="Set environment variables available at runtime. Changes apply on next deploy."
      displayValue={displayValue}
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSaving && !isDecrypting && isDirty}
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
      <div className="flex flex-col gap-2 w-full">
        <p className="text-xs text-gray-11 mb-2">
          Drag & drop your <span className="font-mono font-medium text-gray-12">.env</span> file or
          paste <span className="font-mono font-medium text-gray-12">env</span> vars (⌘V / Ctrl+V)
        </p>

        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <span className="w-[120px] shrink-0 text-[13px] text-gray-11">Environment</span>
            <span className="flex-1 text-[13px] text-gray-11">Key</span>
            <span className="flex-1 text-[13px] text-gray-11">Value</span>
            <span className="w-16 text-left text-[13px] text-gray-11">Sensitive</span>
            <div className="w-16 shrink-0" />
          </div>

          {fields.map((field, index) => (
            <EnvVarRow
              key={`${field.id}-${index}`}
              index={index}
              isLast={index === fields.length - 1}
              isOnly={fields.length === 1}
              keyError={errors.envVars?.[index]?.key?.message}
              environmentError={errors.envVars?.[index]?.environmentId?.message}
              defaultEnvVars={defaultValues.envVars}
              environments={environments}
              control={control}
              register={register}
              onAdd={() => append(createEmptyRow(defaultEnvironmentId))}
              onRemove={() => remove(index)}
            />
          ))}
        </div>
      </div>
    </FormSettingCard>
  );
};
