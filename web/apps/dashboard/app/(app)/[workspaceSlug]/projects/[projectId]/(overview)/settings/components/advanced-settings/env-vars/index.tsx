"use client";

import { collection } from "@/lib/collections";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Nodes2 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useEffect, useMemo, useRef } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { useProjectData } from "../../../../data-provider";
import { useEnvironmentSettings } from "../../../environment-provider";
import { FormSettingCard, resolveSaveState } from "../../shared/form-setting-card";
import { EnvVarRow } from "./env-var-row";
import { type EnvVarsFormValues, createEmptyRow, envVarsSchema } from "./schema";
import { useDecryptedValues } from "./use-decrypted-values";
import { useDropZone } from "./use-drop-zone";
import { computeEnvVarsDiff, groupByEnvironment, toTrpcType } from "./utils";

export const EnvVars = () => {
  const { projectId, environments } = useProjectData();
  const { settings } = useEnvironmentSettings();
  const defaultEnvironmentId = settings.environmentId;

  const { data: envVarData } = useLiveQuery(
    (q) => q.from({ v: collection.envVars }).where(({ v }) => eq(v.projectId, projectId)),
    [projectId],
  );

  const allVariables = useMemo(() => {
    if (!envVarData) {
      return [];
    }
    return [...envVarData]
      .sort((a, b) => b.createdAt - a.createdAt)
      .map((v) => ({
        id: v.id,
        key: v.key,
        type: v.type,
        environmentId: v.environmentId,
        createdAt: v.createdAt,
      }));
  }, [envVarData]);

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

  return (
    <EnvVarsForm
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
  const {
    register,
    handleSubmit,
    formState: { isValid, isSubmitting, errors, isDirty },
    control,
    reset,
    trigger,
    getValues,
  } = useForm<EnvVarsFormValues>({
    resolver: zodResolver(envVarsSchema),
    mode: "onChange",
    defaultValues,
  });

  const prevDefaultsRef = useRef(defaultValues);

  useEffect(() => {
    if (prevDefaultsRef.current !== defaultValues) {
      prevDefaultsRef.current = defaultValues;
      reset(defaultValues);
    }
  }, [defaultValues, reset]);

  const { ref, isDragging } = useDropZone(reset, trigger, getValues, defaultEnvironmentId);

  const { fields, prepend, remove } = useFieldArray({ control, name: "envVars" });

  const onSubmit = async (values: EnvVarsFormValues) => {
    const { toDelete, toCreate, toUpdate } = computeEnvVarsDiff(
      defaultValues.envVars,
      values.envVars,
    );

    const createsByEnv = groupByEnvironment(toCreate);

    toast.promise(
      Promise.all([
        ...toDelete.map(async (id) => {
          collection.envVars.delete(id);
        }),
        ...[...createsByEnv.entries()].map(async ([envId, vars]) => {
          for (const v of vars) {
            collection.envVars.insert({
              id: crypto.randomUUID(),
              environmentId: envId,
              projectId,
              key: v.key,
              value: v.value,
              type: toTrpcType(v.secret) as "recoverable" | "writeonly",
              description: null,
              createdAt: Date.now(),
            });
          }
        }),
        ...toUpdate.map(async (v) => {
          collection.envVars.update(v.id as string, (draft) => {
            draft.key = v.key;
            draft.value = v.value;
            draft.type = toTrpcType(v.secret) as "recoverable" | "writeonly";
          });
        }),
      ]),
      {
        loading: "Saving environment variable(s)...",
        success: `Saved ${toDelete.length + toCreate.length + toUpdate.length} variable(s)`,
        error: (err) => ({
          message: "Failed to save environment variable(s)",
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        }),
      },
    );
  };

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [isDecrypting, { status: "disabled", reason: "Decrypting values…" }],
    [!isValid, { status: "disabled", reason: "Fix validation errors above" }],
    [!isDirty, { status: "disabled", reason: "No changes to save" }],
  ]);

  const varCount = defaultValues.envVars.filter((v) => v.key !== "").length;
  const displayValue =
    varCount === 0 ? null : (
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
      saveState={saveState}
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
              isFirst={index === 0}
              isOnly={fields.length === 1}
              keyError={errors.envVars?.[index]?.key?.message}
              environmentError={errors.envVars?.[index]?.environmentId?.message}
              defaultEnvVars={defaultValues.envVars}
              environments={environments}
              control={control}
              register={register}
              trigger={trigger}
              onAdd={() => prepend(createEmptyRow(defaultEnvironmentId))}
              onRemove={() => {
                remove(index);
                trigger("envVars");
              }}
            />
          ))}
        </div>
      </div>
    </FormSettingCard>
  );
};
