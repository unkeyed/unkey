"use client";

import { collection } from "@/lib/collections";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Nodes2 } from "@unkey/icons";
import { FormInput, toast } from "@unkey/ui";
import { useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
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

const ROW_HEIGHT = 44; // h-9 (36px) + gap-2 (8px)
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

  const [searchQuery, setSearchQuery] = useState("");
  const deferredSearch = useDeferredValue(searchQuery);

  const filteredIndices = useMemo(() => {
    if (!deferredSearch) {
      return fields.map((_, i) => i);
    }
    const query = deferredSearch.toLowerCase();
    return fields.reduce<number[]>((acc, _, i) => {
      const values = getValues(`envVars.${i}`);
      if (values?.key?.toLowerCase().includes(query)) {
        acc.push(i);
      }
      return acc;
    }, []);
  }, [fields, deferredSearch, getValues]);

  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const triggerTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const debouncedTrigger = useCallback(() => {
    clearTimeout(triggerTimerRef.current);
    triggerTimerRef.current = setTimeout(() => {
      trigger("envVars");
    }, 100);
  }, [trigger]);

  useEffect(() => {
    return () => clearTimeout(triggerTimerRef.current);
  }, []);

  const virtualizer = useVirtualizer({
    count: filteredIndices.length,
    getScrollElement: () => scrollContainerRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 5,
  });

  const handleAdd = useCallback(() => {
    prepend(createEmptyRow(defaultEnvironmentId));
  }, [prepend, defaultEnvironmentId]);

  const handleRemove = useCallback(
    (index: number) => {
      remove(index);
      debouncedTrigger();
    },
    [remove, debouncedTrigger],
  );

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
      contentRef={scrollContainerRef}
      className={cn("relative", isDragging && "bg-primary/5")}
      stickyHeader={
        fields.length > 1 ? (
          <div className="flex flex-col gap-2">
            <p className="text-xs text-gray-11">
              Drag & drop your <span className="font-mono font-medium text-gray-12">.env</span> file
              or paste <span className="font-mono font-medium text-gray-12">env</span> vars (⌘V /
              Ctrl+V)
            </p>
            <FormInput
              placeholder="Search by key..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="[&_input]:h-8 [&_input]:text-xs mb-2"
            />
          </div>
        ) : undefined
      }
    >
      <div
        className={cn(
          "absolute inset-2 rounded-lg border-2 border-dotted pointer-events-none transition-colors",
          isDragging ? "border-successA-9" : "border-transparent",
        )}
      />
      <div className="flex flex-col gap-2 w-full">
        {fields.length <= 1 && (
          <p className="text-xs text-gray-11 mb-2">
            Drag & drop your <span className="font-mono font-medium text-gray-12">.env</span> file
            or paste <span className="font-mono font-medium text-gray-12">env</span> vars (⌘V /
            Ctrl+V)
          </p>
        )}

        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <span className="w-[120px] shrink-0 text-[13px] text-gray-11">Environment</span>
            <span className="flex-1 text-[13px] text-gray-11">Key</span>
            <span className="flex-1 text-[13px] text-gray-11">Value</span>
            <span className="w-16 text-left text-[13px] text-gray-11">Sensitive</span>
            <div className="w-16 shrink-0" />
          </div>

          <div
            className="relative w-full min-h-[200px]"
            style={{ height: virtualizer.getTotalSize() }}
          >
            {filteredIndices.length === 0 && searchQuery ? (
              <div className="absolute inset-0 flex items-center justify-center">
                <p className="text-sm text-gray-9">
                  No variables matching &ldquo;{searchQuery}&rdquo;
                </p>
              </div>
            ) : (
              virtualizer.getVirtualItems().map((virtualRow) => {
                const fieldIndex = filteredIndices[virtualRow.index];
                const field = fields[fieldIndex];
                const index = fieldIndex;
                return (
                  <div
                    key={field.id}
                    className="absolute left-0 w-full"
                    style={{
                      height: ROW_HEIGHT,
                      transform: `translateY(${virtualRow.start}px)`,
                    }}
                  >
                    <EnvVarRow
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
                      onAdd={handleAdd}
                      onRemove={handleRemove}
                    />
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>
    </FormSettingCard>
  );
};
