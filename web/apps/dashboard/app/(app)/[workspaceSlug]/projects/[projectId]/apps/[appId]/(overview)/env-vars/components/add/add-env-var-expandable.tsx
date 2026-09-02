import { zodResolver } from "@hookform/resolvers/zod";
import {
  Button,
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelDescription,
  SlidePanelHeader,
  SlidePanelTitle,
  toast,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import {
  IconChevronDownOutline18,
  IconCircleInfoOutline18,
  IconCloudUploadOutline18,
  IconPlusOutline18,
} from "nucleo-ui-outline-18";
import { type ChangeEvent, useCallback, useEffect, useRef } from "react";
import { Controller, useFieldArray } from "react-hook-form";
import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { Switch } from "@/components/ui/switch";
import { usePersistedForm } from "@/hooks/use-persisted-form";
import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { collection } from "@/lib/collections";
import {
  listExistingKeys,
  setVariables,
  type VariableInput,
} from "@/lib/collections/deploy/env-vars";
import { trackSave } from "@/lib/collections/deploy/environment-settings";
import { getErrorMessage } from "@/lib/unkey-client";
import { useDropZone } from "../../hooks/use-drop-zone";
import { EnvVarRow } from "./env-var-row";
import { createEmptyEntry, type EnvVarsFormValues, envVarsSchema, findConflicts } from "./schema";

type AddEnvVarExpandableProps = {
  projectId: string;
  appId: string;
  isOpen: boolean;
  onClose: () => void;
};

export const AddEnvVarExpandable = ({
  projectId,
  appId,
  isOpen,
  onClose,
}: AddEnvVarExpandableProps) => {
  const { environments } = useProjectData();

  const {
    register,
    handleSubmit,
    formState: { isSubmitting, errors },
    control,
    reset,
    trigger,
    getValues,
    setValue,
    setFocus,
    setError,
    clearPersistedData,
    saveCurrentValues,
    loadSavedValues,
  } = usePersistedForm<EnvVarsFormValues>(
    `env-vars-add-${appId}`,
    {
      resolver: zodResolver(envVarsSchema),
      mode: "onChange",
      defaultValues: {
        envVars: [createEmptyEntry()],
        environmentId: "__all__",
        secret: false,
      },
    },
    // Values may be secrets (the "Sensitive" toggle), so the draft must never
    // be written to session/local storage in plaintext.
    "memory",
  );

  const { fields, append, remove } = useFieldArray({ control, name: "envVars" });

  const handlePasteEntries = useCallback(
    (index: number, entries: { key: string; value: string }[]) => {
      if (entries.length === 0) {
        return;
      }
      const [first, ...rest] = entries;
      setValue(`envVars.${index}.key`, first.key);
      setValue(`envVars.${index}.value`, first.value);
      if (rest.length > 0) {
        append(rest.map((e) => ({ key: e.key, value: e.value, description: "" })));
      }
      trigger("envVars");
    },
    [setValue, append, trigger],
  );

  const handleRemoveAndFocusPrevious = useCallback(
    (index: number) => {
      if (index === 0) {
        return;
      }
      remove(index);
      requestAnimationFrame(() => {
        setFocus(`envVars.${index - 1}.value`);
      });
    },
    [remove, setFocus],
  );

  const handleAdvanceRow = useCallback(() => {
    append(createEmptyEntry());
    requestAnimationFrame(() => {
      setFocus(`envVars.${fields.length}.key`);
    });
  }, [append, setFocus, fields.length]);
  const { ref: formRef, isDragging, importFile } = useDropZone(reset, trigger, getValues);
  const fileInputRef = useRef<HTMLInputElement>(null);

  usePreventLeave(isOpen);

  useEffect(
    function purgeLegacyPersistedDraft() {
      // Earlier versions persisted this draft (possibly containing secrets) to
      // sessionStorage, which survives page reloads. Remove any leftovers.
      sessionStorage.removeItem(`env-vars-add-${appId}`);
    },
    [appId],
  );

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

    const allEnvIds = environments.map((e) => e.id);
    const targetEnvIds = values.environmentId === "__all__" ? allEnvIds : [values.environmentId];

    const existing = await listExistingKeys(projectId, appId, targetEnvIds);
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

    const kind = values.secret ? ("writeonly" as const) : ("recoverable" as const);
    const variables: VariableInput[] = nonEmpty.map((entry) => ({
      key: entry.key,
      value: entry.value,
      kind,
      description: entry.description || undefined,
    }));

    try {
      // This writes outside the collection, so call trackSave here to show
      // the pending-redeploy banner.
      await trackSave(
        Promise.all(targetEnvIds.map((envId) => setVariables(projectId, appId, envId, variables))),
      );
      toast.success(`Added ${variables.length * targetEnvIds.length} variable(s)`);
    } catch (err) {
      toast.error("Failed to create environment variables", {
        description: getErrorMessage(err),
      });
      return;
    } finally {
      // A rejection can still leave variables written, since each environment
      // and each part commits on its own.
      await collection.envVars.utils.refetch().catch(() => {});
    }

    clearPersistedData();
    reset({
      envVars: [createEmptyEntry()],
      environmentId: "__all__",
      secret: false,
    });
    onClose();
  };

  return (
    <SlidePanel isOpen={isOpen} onClose={onClose}>
      <SlidePanelHeader>
        <div className="flex flex-col gap-0.5">
          <SlidePanelTitle>Add Environment Variable</SlidePanelTitle>
          <SlidePanelDescription>Set a key-value pair for your app.</SlidePanelDescription>
        </div>
        <SlidePanelCloseButton className="mt-0.5" />
      </SlidePanelHeader>

      <SlidePanelContent>
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
                <IconCloudUploadOutline18 className="text-success-11" />
              </div>
              <div className="flex flex-col items-center gap-1">
                <span className="text-sm font-medium text-success-11">Drop your .env file</span>
                <span className="text-xs text-success-10">
                  We'll parse and import your variables
                </span>
              </div>
            </div>
          </div>

          <div className="flex-1 overflow-y-auto pt-6">
            <div className="flex flex-col gap-4 px-6">
              {fields.map((field, index) => (
                <EnvVarRow
                  key={field.id}
                  index={index}
                  isOnly={fields.length === 1}
                  isLast={index === fields.length - 1}
                  register={register}
                  onRemove={remove}
                  onPasteEntries={handlePasteEntries}
                  onAdvanceRow={handleAdvanceRow}
                  onRemoveAndFocusPrevious={handleRemoveAndFocusPrevious}
                  errors={errors.envVars}
                />
              ))}
            </div>

            <div className="flex py-6 px-6">
              <Button
                type="button"
                variant="outline"
                size="md"
                className="font-medium"
                onClick={() => append(createEmptyEntry())}
              >
                <IconPlusOutline18 />
                Add Another
              </Button>
            </div>
          </div>

          <div className="border-t border-grayA-4">
            <div className="px-6 py-6 space-y-6">
              <Controller
                control={control}
                name="environmentId"
                render={({ field }) => (
                  <fieldset className="flex flex-col gap-1.5 border-0 m-0 p-0">
                    <label htmlFor="environment-select" className="text-gray-11 text-[13px]">
                      Environment
                    </label>
                    <Select
                      value={field.value}
                      onValueChange={field.onChange}
                      items={[
                        { value: "__all__", label: "All Environments" },
                        ...environments.map((env) => ({ value: env.id, label: env.slug })),
                      ]}
                    >
                      <SelectTrigger
                        id="environment-select"
                        className="capitalize"
                        rightIcon={<IconChevronDownOutline18 className="size-4 absolute right-2" />}
                      >
                        <SelectValue placeholder="Select environment" />
                      </SelectTrigger>
                      <SelectContent className="z-60">
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
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  )}
                />
                <span className="text-[13px] text-gray-12 font-medium">Sensitive</span>
                <InfoTooltip
                  content="Permanently hides values after saving. Use for API keys and secrets."
                  position={{ side: "top" }}
                  className="z-60"
                  asChild
                >
                  <span className="text-grayA-9">
                    <IconCircleInfoOutline18 className="size-4" />
                  </span>
                </InfoTooltip>
              </div>
            </div>
          </div>

          <div className="border-t border-gray-4 bg-white dark:bg-black px-6 py-5 flex items-center justify-between">
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
                <IconCloudUploadOutline18 className="size-3" />
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
      </SlidePanelContent>
    </SlidePanel>
  );
};
