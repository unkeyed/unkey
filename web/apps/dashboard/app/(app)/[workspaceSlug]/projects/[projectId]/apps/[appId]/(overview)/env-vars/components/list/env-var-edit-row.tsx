"use client";

import { Switch } from "@/components/ui/switch";
import { collection } from "@/lib/collections";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, FormTextarea, InfoTooltip } from "@unkey/ui";
import { IconPlusOutline12 } from "nucleo-ui-outline-12";
import { IconCircleInfoOutline18 } from "nucleo-ui-outline-18";
import { type ClipboardEvent, useCallback } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { parseEnvText } from "../../hooks/use-drop-zone";

const editEnvVarSchema = z.object({
  key: envVarKeySchema,
  value: envVarValueSchema.or(z.literal("")),
  description: z.string().optional(),
  sensitive: z.boolean(),
});

type EditEnvVarFormValues = z.infer<typeof editEnvVarSchema>;

type EnvVarEditRowProps = {
  envVarId: string;
  value: string;
  variableKey: string;
  type: "writeonly" | "recoverable";
  note: string | null;
  onClose: () => void;
};

export function EnvVarEditRow({
  envVarId,
  value,
  variableKey,
  type,
  note,
  onClose,
}: EnvVarEditRowProps) {
  const isWriteonly = type === "writeonly";

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    control,
    formState: { isSubmitting, errors },
  } = useForm<EditEnvVarFormValues>({
    mode: "onChange",
    resolver: zodResolver(editEnvVarSchema),
    defaultValues: {
      key: variableKey,
      // The API never returns a writeonly value. The field starts empty, and a
      // blank value keeps the stored one.
      value: isWriteonly ? "" : value,
      description: note ?? "",
      sensitive: isWriteonly,
    },
  });

  const onSubmit = useCallback(
    async (values: EditEnvVarFormValues) => {
      if (isWriteonly && !values.value) {
        // A write replaces the whole variable and the API never returns a
        // sensitive value, so the note cannot be saved on its own.
        if ((values.description || "") !== (note ?? "")) {
          setError("value", {
            message: "Type the new value to save a change to a sensitive variable.",
          });
          return;
        }
        onClose();
        return;
      }

      collection.envVars.update(envVarId, (draft) => {
        draft.key = values.key;
        draft.value = values.value;
        draft.description = values.description || null;
        draft.type = values.sensitive ? "writeonly" : "recoverable";
      });
      onClose();
    },
    [envVarId, isWriteonly, note, setError, onClose],
  );

  const handleKeyPaste = useCallback(
    (e: ClipboardEvent<HTMLInputElement>) => {
      if (isWriteonly) {
        return;
      }
      const text = e.clipboardData.getData("text/plain");
      if (!text.includes("=")) {
        return;
      }
      const { entries } = parseEnvText(text);
      if (entries.length === 0) {
        return;
      }
      e.preventDefault();
      setValue("key", entries[0].key);
      setValue("value", entries[0].value);
    },
    [isWriteonly, setValue],
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
    <div className="bg-gray-1 px-12 pb-6 pt-5 border-t border-grayA-4" onKeyDown={handleKeyDown}>
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-5">
        <FormInput
          label="Key"
          className="[&_input]:font-mono"
          placeholder="VARIABLE_NAME"
          error={errors.key?.message}
          readOnly={isWriteonly}
          disabled={isWriteonly}
          title={isWriteonly ? "You cannot rename sensitive environment variables" : ""}
          {...register("key")}
          onPaste={handleKeyPaste}
        />
        <FormTextarea
          label="Value"
          rows={1}
          className="[&_textarea]:font-mono [&_textarea]:min-h-9 [&_textarea]:max-h-40 [&_textarea]:resize-y [&_textarea]:overflow-y-auto"
          placeholder={isWriteonly ? "Leave empty to keep the current value" : "value"}
          description={
            isWriteonly
              ? "A sensitive value cannot be shown. Type a new value to replace it, or leave this empty to keep the current one."
              : undefined
          }
          error={errors.value?.message}
          {...register("value")}
        />
        <details className="group" open={Boolean(note)}>
          <summary className="w-fit text-[13px] text-gray-11 hover:text-gray-12 transition-colors cursor-pointer list-none [&::-webkit-details-marker]:hidden flex items-center gap-1.5 group">
            <span className="group-open:hidden flex items-center gap-2">
              <IconPlusOutline12 className="text-gray-9 group-hover:text-gray-12 transition-colors" />
              Add Note
            </span>
            <span className="hidden group-open:inline">Note</span>
          </summary>
          <div className="pt-1.5">
            <FormInput
              className="[&_input]:text-sm"
              placeholder="Optional description for this variable..."
              {...register("description")}
            />
          </div>
        </details>
        {!isWriteonly && (
          <div className="flex items-center gap-3">
            <Controller
              control={control}
              name="sensitive"
              render={({ field }) => (
                <Switch checked={field.value} onCheckedChange={field.onChange} />
              )}
            />
            <span className="text-[13px] text-gray-12 font-normal">Sensitive</span>
            <InfoTooltip
              content="Permanently hides values after saving. This cannot be undone."
              position={{ side: "top" }}
              className="z-60"
              asChild
            >
              <span className="text-grayA-9">
                <IconCircleInfoOutline18 className="size-4" />
              </span>
            </InfoTooltip>
          </div>
        )}

        <div className="flex items-center justify-end gap-2 pt-5 mt-1">
          <Button type="button" variant="outline" size="md" onClick={onClose} className="px-3">
            Cancel
          </Button>
          <Button
            type="submit"
            className="px-3"
            variant="primary"
            size="md"
            loading={isSubmitting}
            disabled={isSubmitting}
          >
            Save
          </Button>
        </div>
      </form>
    </div>
  );
}
