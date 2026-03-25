import { collection } from "@/lib/collections";
import { FormCombobox } from "@/components/ui/form-combobox";
import { Switch } from "@/components/ui/switch";
import { zodResolver } from "@hookform/resolvers/zod";
import { DoubleChevronRight, Plus } from "@unkey/icons";
import { Button, InfoTooltip, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { Controller, useFieldArray, useForm } from "react-hook-form";
import { useProjectData } from "../../data-provider";
import { EnvVarRow } from "./env-var-row";
import { type EnvVarsFormValues, createEmptyEntry, envVarsSchema } from "./schema";
import { useDropZone } from "./use-drop-zone";
import { expandToFlatRecords, groupByEnvironment, toTrpcType } from "./utils";

type AddEnvVarExpandableProps = {
  tableDistanceToTop: number;
  isOpen: boolean;
  onClose: () => void;
};

export const AddEnvVarExpandable = ({
  tableDistanceToTop,
  isOpen,
  onClose,
}: AddEnvVarExpandableProps) => {
  const panelRef = useRef<HTMLDivElement>(null);
  const { projectId, environments } = useProjectData();
  const allEnvironmentIds = useMemo(() => environments.map((e) => e.id), [environments]);

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
    defaultValues: {
      envVars: [createEmptyEntry()],
      environmentIds: allEnvironmentIds,
      secret: false,
    },
  });

  const { fields, append, remove } = useFieldArray({ control, name: "envVars" });
  const { ref: formRef, isDragging } = useDropZone(reset, trigger, getValues);

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

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handleClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        onClose();
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen, onClose]);

  useEffect(() => {
    if (!isOpen) {
      reset({
        envVars: [createEmptyEntry()],
        environmentIds: allEnvironmentIds,
        secret: false,
      });
    }
  }, [isOpen, reset, allEnvironmentIds]);

  const handleAdd = useCallback(() => {
    append(createEmptyEntry());
  }, [append]);

  const handleRemove = useCallback(
    (index: number) => {
      remove(index);
      debouncedTrigger();
    },
    [remove, debouncedTrigger],
  );

  const onSubmit = async (values: EnvVarsFormValues) => {
    const nonEmpty = values.envVars.filter((v) => v.key !== "" && v.value !== "");
    if (nonEmpty.length === 0) {
      return;
    }

    const flatRecords = expandToFlatRecords(nonEmpty, values.environmentIds, values.secret);
    const byEnv = groupByEnvironment(flatRecords);

    toast.promise(
      Promise.all(
        [...byEnv.entries()].map(async ([envId, vars]) => {
          for (const v of vars) {
            collection.envVars.insert({
              id: crypto.randomUUID(),
              environmentId: envId,
              projectId,
              key: v.key,
              value: v.value,
              type: toTrpcType(v.secret) as "recoverable" | "writeonly",
              description: v.description || null,
              createdAt: Date.now(),
            });
          }
        }),
      ),
      {
        loading: "Saving environment variable(s)...",
        success: `Added ${flatRecords.length} variable(s)`,
        error: (err) => ({
          message: "Failed to add environment variable(s)",
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        }),
      },
    );

    reset({
      envVars: [createEmptyEntry()],
      environmentIds: allEnvironmentIds,
      secret: false,
    });
    onClose();
  };

  return (
    <div className="flex">
      <div
        ref={panelRef}
        className={cn(
          "fixed right-3 bg-gray-1 border border-grayA-4 rounded-xl w-175 overflow-hidden z-50",
          "transition-all duration-300 ease-out",
          "shadow-md",
          isOpen ? "translate-x-0 opacity-100" : "translate-x-full opacity-0",
        )}
        style={{
          top: `${tableDistanceToTop + 12}px`,
          height: `calc(100vh - ${tableDistanceToTop + 24}px)`,
          willChange: isOpen ? "transform, opacity" : "auto",
        }}
      >
        <div className="h-full flex flex-col">
          {/* Header */}
          <div className="flex items-start justify-between border-b border-grayA-4 px-6 py-4">
            <div className="flex flex-col">
              <span className="text-gray-12 font-medium text-base leading-8">
                Add Environment Variable
              </span>
              <span className="text-gray-9 text-[13px] leading-5">
                Set a key-value pair for your project.
              </span>
            </div>
            <InfoTooltip
              content="Close"
              asChild
              position={{
                side: "bottom",
                align: "end",
              }}
            >
              <Button variant="ghost" size="icon" onClick={onClose} className="mt-0.5">
                <DoubleChevronRight
                  iconSize="lg-medium"
                  className="text-gray-8 transition-transform duration-300 ease-out group-hover:text-gray-12"
                />
              </Button>
            </InfoTooltip>
          </div>

          {/* Form content */}
          <div
            className={cn(
              "transition-all duration-500 ease-out flex-1 min-h-0",
              isOpen ? "translate-x-0 opacity-100" : "translate-x-6 opacity-0",
            )}
            style={{
              transitionDelay: isOpen ? "150ms" : "0ms",
            }}
          >
            <form
              ref={formRef}
              onSubmit={handleSubmit(onSubmit)}
              className="h-full flex flex-col relative"
            >
              {/* Drop zone overlay */}
              <div
                className={cn(
                  "absolute inset-4 rounded-lg border-2 border-dotted pointer-events-none transition-colors z-10",
                  isDragging ? "border-successA-9" : "border-transparent",
                )}
              />

              {/* Scrollable content */}
              <div className="flex-1 overflow-y-auto px-6">
                <p className="text-xs text-gray-11 pt-5 pb-6">
                  Drag & drop your{" "}
                  <span className="font-mono font-medium text-gray-12">.env</span> file or paste{" "}
                  <span className="font-mono font-medium text-gray-12">env</span> vars (⌘V /
                  Ctrl+V)
                </p>

                {/* Variable entries */}
                <div className="flex flex-col gap-7">
                  {fields.map((field, index) => (
                    <EnvVarRow
                      key={field.id}
                      index={index}
                      isOnly={fields.length === 1}
                      isLast={index === fields.length - 1}
                      keyError={errors.envVars?.[index]?.key?.message}
                      register={register}
                      onRemove={handleRemove}
                    />
                  ))}
                </div>

                {/* Add Another */}
                <div className="flex pt-5 pb-4">
                  <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
                    <Plus iconSize="sm-regular" />
                    Add Another
                  </Button>
                </div>

                {/* Separator */}
                <div className="border-t border-grayA-4 pt-6 pb-4 space-y-5">
                  {/* Environment multi-select (shared) */}
                  <Controller
                    control={control}
                    name="environmentIds"
                    render={({ field }) => {
                      const allSelected = allEnvironmentIds.every((id) =>
                        field.value.includes(id),
                      );
                      const displayText =
                        allSelected || field.value.length === 0
                          ? "All Environments"
                          : environments
                            .filter((e) => field.value.includes(e.id))
                            .map((e) => e.slug)
                            .join(", ");

                      return (
                        <FormCombobox
                          label="Environment"
                          options={[
                            { label: "All Environments", value: "__all__" },
                            ...environments.map((env) => ({
                              label: env.slug,
                              value: env.id,
                            })),
                          ]}
                          value=""
                          closeOnSelect={false}
                          onSelect={(envId) => {
                            if (envId === "__all__") {
                              field.onChange(allSelected ? [] : allEnvironmentIds);
                            } else {
                              const next = field.value.includes(envId)
                                ? field.value.filter((id: string) => id !== envId)
                                : [...field.value, envId];
                              field.onChange(next);
                            }
                            trigger("environmentIds");
                          }}
                          placeholder={displayText}
                          error={errors.environmentIds?.message}
                        />
                      );
                    }}
                  />

                  {/* Sensitive toggle (shared) */}
                  <div className="flex items-center gap-2 mt-4">
                    <Controller
                      control={control}
                      name="secret"
                      render={({ field }) => (
                        <Switch
                          className="h-6 w-10 data-[state=checked]:bg-success-9 data-[state=checked]:ring-2 data-[state=checked]:ring-successA-5 data-[state=unchecked]:bg-grayA-1 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-3 [&>span]:h-[18px] [&>span]:w-[18px]"
                          thumbClassName="h-[18px] w-[18px] data-[state=unchecked]:bg-grayA-12 data-[state=checked]:bg-white data-[state=checked]:translate-x-[17px]"
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      )}
                    />
                    <span className="text-[13px] text-gray-11">Sensitive</span>
                  </div>
                </div>
              </div>

              {/* Sticky save footer */}
              <div className="border-t border-grayA-4 bg-gray-1 px-6 py-4 flex justify-end">
                <Button
                  type="submit"
                  variant="primary"
                  size="md"
                  loading={isSubmitting}
                  disabled={!isValid || !isDirty || isSubmitting}
                >
                  Save Variables
                </Button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
};

