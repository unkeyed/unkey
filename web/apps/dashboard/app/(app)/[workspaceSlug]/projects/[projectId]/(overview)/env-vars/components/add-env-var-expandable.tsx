import { Switch } from "@/components/ui/switch";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { CircleInfo, CloudUp, DoubleChevronRight, Plus } from "@unkey/icons";
import {
  Button,
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SlidePanel,
  toast,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { type ChangeEvent, useCallback, useEffect, useRef } from "react";
import { Controller, useFieldArray } from "react-hook-form";
import { useProjectData } from "../../data-provider";
import { EnvVarRow } from "./env-var-row";
import { type EnvVarsFormValues, createEmptyEntry, envVarsSchema, findConflicts } from "./schema";
import { useDropZone } from "./use-drop-zone";

import { usePreventLeave } from "@/hooks/use-prevent-leave";

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
  const { projectId, environments } = useProjectData();

  const { data: existingEnvVars } = useLiveQuery(
    (q) => q.from({ v: collection.envVars }).where(({ v }) => eq(v.projectId, projectId)),
    [projectId],
  );

  const {
    register,
    handleSubmit,
    formState: { isSubmitting, errors },
    control,
    reset,
    trigger,
    getValues,
    setFocus,
    setError,
    clearPersistedData,
    saveCurrentValues,
    loadSavedValues,
  } = usePersistedForm<EnvVarsFormValues>(
    `env-vars-add-${projectId}`,
    {
      resolver: zodResolver(envVarsSchema),
      mode: "onSubmit",
      defaultValues: {
        envVars: [createEmptyEntry()],
        environmentId: "__all__",
        secret: false,
      },
    },
    "session",
  );

  const { fields, append, remove } = useFieldArray({ control, name: "envVars" });
  const { ref: formRef, isDragging, importFile } = useDropZone(reset, trigger, getValues);
  const fileInputRef = useRef<HTMLInputElement>(null);

  usePreventLeave(isOpen);

  useEffect(
    function persistFormState() {
      if (isOpen) {
        loadSavedValues();
      } else {
        saveCurrentValues();
      }
    },
    [isOpen, loadSavedValues, saveCurrentValues],
  );

  const handleFileImport = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (file) {
        importFile(file);
      }
      e.target.value = "";
    },
    [importFile],
  );

  const onInvalid = useCallback(() => {
    const envVarErrors = errors.envVars;
    if (envVarErrors && Array.isArray(envVarErrors)) {
      const firstErrorIndex = envVarErrors.findIndex((e) => e != null);
      if (firstErrorIndex !== -1) {
        setFocus(`envVars.${firstErrorIndex}.key`);
      }
    }
  }, [errors.envVars, setFocus]);

  const onSubmit = async (values: EnvVarsFormValues) => {
    const nonEmpty = values.envVars.filter((v) => v.key !== "" && v.value !== "");
    if (nonEmpty.length === 0) {
      return;
    }

    const existing = (existingEnvVars ?? []).map((v) => ({
      key: v.key,
      environmentId: v.environmentId,
    }));
    const allEnvIds = environments.map((e) => e.id);
    const conflicts = findConflicts(nonEmpty, values.environmentId, existing, allEnvIds);

    if (conflicts.length > 0) {
      for (const idx of conflicts) {
        // Map back to the original form index
        const originalIdx = values.envVars.findIndex(
          (v) => v.key === nonEmpty[idx].key && v.value === nonEmpty[idx].value,
        );
        if (originalIdx !== -1) {
          setError(`envVars.${originalIdx}.key`, {
            message: "Variable already exists in this environment",
          });
        }
      }
      return;
    }

    const targetEnvIds =
      values.environmentId === "__all__" ? environments.map((e) => e.id) : [values.environmentId];
    const type = values.secret ? "writeonly" : "recoverable";
    const flatRecords = nonEmpty.flatMap((entry) =>
      targetEnvIds.map((envId) => ({
        key: entry.key,
        value: entry.value,
        description: entry.description,
        environmentId: envId,
      })),
    );

    for (const v of flatRecords) {
      collection.envVars.insert({
        id: crypto.randomUUID(),
        environmentId: v.environmentId,
        projectId,
        key: v.key,
        value: v.value,
        type,
        description: v.description || null,
        updatedAt: Date.now(),
      });
    }
    toast.success(`Added ${flatRecords.length} variable(s)`);

    clearPersistedData();
    reset({
      envVars: [createEmptyEntry()],
      environmentId: "__all__",
      secret: false,
    });
    onClose();
  };

  return (
    <SlidePanel.Root isOpen={isOpen} onClose={onClose} topOffset={tableDistanceToTop}>
      <SlidePanel.Header>
        <div className="flex flex-col">
          <span className="text-gray-12 font-medium text-base leading-8">
            Add Environment Variable
          </span>
          <span className="text-gray-11 text-[13px] leading-5">
            Set a key-value pair for your project.
          </span>
        </div>
        <SlidePanel.Close
          aria-label="Close panel"
          className="mt-0.5 inline-flex items-center justify-center size-9 rounded-md hover:bg-grayA-3 transition-colors"
        >
          <DoubleChevronRight
            iconSize="lg-medium"
            className="text-gray-10 transition-transform duration-300 ease-out group-hover:text-gray-12"
          />
        </SlidePanel.Close>
      </SlidePanel.Header>

      <SlidePanel.Content>
        <form
          ref={formRef}
          onSubmit={handleSubmit(onSubmit, onInvalid)}
          className="h-full flex flex-col relative"
        >
          {/* Drop zone overlay */}
          <div
            className={cn(
              "absolute inset-0 rounded-lg pointer-events-none z-10 flex items-center justify-center transition-all duration-200",
              isDragging ? "bg-successA-2 opacity-100" : "opacity-0",
            )}
          >
            <div
              className={cn(
                "absolute inset-4 rounded-lg border-2 border-dashed transition-all duration-200",
                isDragging ? "border-successA-8 scale-100" : "border-transparent scale-[0.98]",
              )}
            />
            <div
              className={cn(
                "flex flex-col items-center gap-3 transition-all duration-200",
                isDragging ? "opacity-100 scale-100" : "opacity-0 scale-95",
              )}
            >
              <div className="size-12 rounded-xl bg-successA-3 flex items-center justify-center">
                <CloudUp className="text-success-11" />
              </div>
              <div className="flex flex-col items-center gap-1">
                <span className="text-sm font-medium text-success-11">Drop your .env file</span>
                <span className="text-xs text-success-10">
                  We'll parse and import your variables
                </span>
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto pt-6 bg-grayA-2">
            <div className="flex flex-col gap-4 px-8">
              {fields.map((field, index) => (
                <EnvVarRow
                  key={field.id}
                  index={index}
                  isOnly={fields.length === 1}
                  register={register}
                  onRemove={remove}
                  errors={errors.envVars}
                />
              ))}
            </div>

            <div className="flex py-6 px-8">
              <Button
                type="button"
                variant="outline"
                size="md"
                className="font-medium"
                onClick={() => append(createEmptyEntry())}
              >
                <Plus iconSize="sm-regular" />
                Add Another
              </Button>
            </div>
          </div>

          <div className="border-t border-grayA-4">
            <div className="px-8 py-6 space-y-6">
              <Controller
                control={control}
                name="environmentId"
                render={({ field }) => (
                  <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                    <label htmlFor="environment-select" className="text-gray-11 text-[13px]">
                      Environment
                    </label>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger id="environment-select" className="capitalize">
                        <SelectValue placeholder="Select environment" />
                      </SelectTrigger>
                      <SelectContent className="z-[60]">
                        <SelectItem value="__all__">All Environments</SelectItem>
                        {environments.map((env) => (
                          <SelectItem key={env.id} value={env.id} className="capitalize">
                            {env.slug}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {errors.environmentId?.message && (
                      <p className="text-error-11 text-[13px]">{errors.environmentId.message}</p>
                    )}
                  </fieldset>
                )}
              />

              <div className="flex items-center gap-3 pt-6">
                <Controller
                  control={control}
                  name="secret"
                  render={({ field }) => (
                    <Switch
                      className="data-[state=checked]:bg-success-9 data-[state=checked]:ring-2 data-[state=checked]:ring-successA-5 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-3"
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  )}
                />
                <span className="text-[13px] text-gray-12 font-medium">Sensitive</span>
                <InfoTooltip
                  content="Permanently hides values after saving. Use for API keys and secrets."
                  position={{ side: "top" }}
                  className="z-[60]"
                >
                  <span className="text-grayA-9">
                    <CircleInfo iconSize="md-regular" />
                  </span>
                </InfoTooltip>
              </div>
            </div>
          </div>

          <div className="border-t border-grayA-4 bg-gray-1 px-8 py-5 flex items-center justify-between">
            <div className="hidden md:flex items-center gap-3">
              <input
                ref={fileInputRef}
                type="file"
                accept=".env,.txt,text/plain"
                className="hidden"
                onChange={handleFileImport}
              />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => fileInputRef.current?.click()}
              >
                <CloudUp iconSize="sm-regular" />
                Import <span className="font-medium">.env</span>
              </Button>
              <span className="text-[13px] text-gray-11">
                or drag & drop / paste (⌘V) your .env
              </span>
            </div>
            <Button
              type="submit"
              variant="primary"
              size="md"
              className="px-3"
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              Save
            </Button>
          </div>
        </form>
      </SlidePanel.Content>
    </SlidePanel.Root>
  );
};
