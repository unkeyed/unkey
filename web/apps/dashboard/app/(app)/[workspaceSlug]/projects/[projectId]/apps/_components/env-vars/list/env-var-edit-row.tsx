"use client";

import { Switch } from "@/components/ui/switch";
import { collection } from "@/lib/collections";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconCircleInfoOutline18 } from "@unkey/icons";
import { Button, FormInput, FormTextarea, InfoTooltip, useReportUnsavedChanges } from "@unkey/ui";
import { type ClipboardEvent, useCallback } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { pastedEntries } from "../env-file";

const editRecoverableSchema = z.object({
  key: envVarKeySchema,
  value: envVarValueSchema,
  description: z.string().optional(),
  sensitive: z.boolean(),
});

const editWriteonlySchema = editRecoverableSchema.extend({
  value: envVarValueSchema.or(z.literal("")),
});

type EditEnvVarFormValues = z.infer<typeof editWriteonlySchema>;

export function EnvVarEditRow({ envVar, onClose }: { envVar: EnvVar; onClose: () => void }) {
  const { id: envVarId, value, key: variableKey, description: note } = envVar;
  const isWriteonly = envVar.type === "writeonly";

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    control,
    formState: { errors, isDirty },
  } = useForm<EditEnvVarFormValues>({
    mode: "onChange",
    resolver: zodResolver(isWriteonly ? editWriteonlySchema : editRecoverableSchema),
    defaultValues: {
      key: variableKey,
      value: isWriteonly ? "" : value,
      description: note ?? "",
      sensitive: isWriteonly,
    },
  });
  useReportUnsavedChanges(isDirty);

  const onSubmit = useCallback(
    (values: EditEnvVarFormValues) => {
      if (isWriteonly && !values.value) {
        if ((values.description || "") !== (note ?? "")) {
          setError("value", {
            message: "Type the new value to save a change to a sensitive variable.",
          });
          return;
        }
        if (values.key === variableKey) {
          onClose();
          return;
        }
      }

      collection.envVars.update(envVarId, (draft) => {
        draft.key = values.key;
        draft.value = values.value;
        draft.description = values.description || null;
        draft.type = values.sensitive ? "writeonly" : "recoverable";
      });
      onClose();
    },
    [envVarId, isWriteonly, note, variableKey, setError, onClose],
  );

  const handleKeyPaste = useCallback(
    (e: ClipboardEvent<HTMLInputElement>) => {
      const [entry] = pastedEntries(e.clipboardData.getData("text/plain"));
      if (!entry) {
        return;
      }
      e.preventDefault();
      setValue("key", entry.key);
      setValue("value", entry.value);
    },
    [setValue],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    },
    [onClose],
  );

  return (
    <div className="bg-raised border-t" onKeyDown={handleKeyDown}>
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col">
        <div className="flex flex-col gap-5 px-4 pt-5 pb-5">
          <FormInput
            label="Key"
            className="[&_input]:font-mono"
            placeholder="VARIABLE_NAME"
            error={errors.key?.message}
            {...register("key")}
            onPaste={handleKeyPaste}
          />
          <FormTextarea
            label="Value"
            rows={1}
            className="[&_textarea]:font-mono [&_textarea]:min-h-24 [&_textarea]:max-h-60 [&_textarea]:resize-y [&_textarea]:overflow-y-auto"
            placeholder={isWriteonly ? "Leave empty to keep the current value" : "value"}
            error={errors.value?.message}
            {...register("value")}
          />
          <FormInput
            label="Note (optional)"
            placeholder="Where to rotate it, or who to contact"
            {...register("description")}
          />
          {!isWriteonly && (
            <div className="flex items-center gap-3">
              <Controller
                control={control}
                name="sensitive"
                render={({ field }) => (
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                )}
              />
              <span className="text-sm text-gray-12 font-medium">Sensitive</span>
              <InfoTooltip
                content="Hides the value after saving. This cannot be undone."
                position={{ side: "top" }}
                asChild
              >
                <span className="text-grayA-9">
                  <IconCircleInfoOutline18 className="size-3.5" />
                </span>
              </InfoTooltip>
            </div>
          )}
        </div>
        <div className="flex items-center justify-end gap-3 border-t border-grayA-4 bg-grayA-2 px-4 py-3">
          <Button type="button" variant="outline" size="sm" onClick={onClose} className="px-3">
            Cancel
          </Button>
          <Button type="submit" className="px-3" variant="primary" size="sm">
            Save
          </Button>
        </div>
      </form>
    </div>
  );
}
