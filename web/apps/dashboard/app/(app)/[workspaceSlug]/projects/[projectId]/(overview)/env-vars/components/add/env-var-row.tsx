import { Plus, Trash } from "@unkey/icons";
import { Button, FormInput, FormTextarea } from "@unkey/ui";
import type { ClipboardEvent, KeyboardEvent } from "react";
import { useCallback } from "react";
import type { Control, FieldErrors, UseFormRegister } from "react-hook-form";
import { useWatch } from "react-hook-form";
import { parseEnvText } from "../../hooks/use-drop-zone";
import type { EnvVarsFormValues } from "./schema";

type EnvVarRowProps = {
  index: number;
  isOnly: boolean;
  register: UseFormRegister<EnvVarsFormValues>;
  control: Control<EnvVarsFormValues>;
  onRemove: (index: number) => void;
  onPasteEntries: (index: number, entries: { key: string; value: string }[]) => void;
  onAdvanceRow: () => void;
  onRemoveAndFocusPrevious: (index: number) => void;
  isLast: boolean;
  errors?: FieldErrors<EnvVarsFormValues>["envVars"];
};

export const EnvVarRow = ({
  index,
  isOnly,
  register,
  control,
  onRemove,
  onPasteEntries,
  onAdvanceRow,
  onRemoveAndFocusPrevious,
  isLast,
  errors,
}: EnvVarRowProps) => {
  const fieldErrors = errors?.[index];
  const value = useWatch({ control, name: `envVars.${index}.value` });
  const hasSpaces = value?.trim().includes(" ");

  const handleKeyPaste = useCallback(
    (e: ClipboardEvent<HTMLInputElement>) => {
      const text = e.clipboardData.getData("text/plain");
      if (!text.includes("=")) {
        return;
      }
      const { entries } = parseEnvText(text);
      if (entries.length === 0) {
        return;
      }
      e.preventDefault();
      onPasteEntries(index, entries);
    },
    [index, onPasteEntries],
  );

  const handleKeyKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Backspace" && !isOnly && e.currentTarget.value === "") {
        e.preventDefault();
        onRemoveAndFocusPrevious(index);
      }
    },
    [index, isOnly, onRemoveAndFocusPrevious],
  );

  const handleValueKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      // Plain Enter inserts a newline so multi-line values (PEM keys, JSON) can
      // be entered. Cmd/Ctrl+Enter keeps the keyboard-driven flow of advancing
      // to / creating the next row.
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && isLast) {
        e.preventDefault();
        onAdvanceRow();
      }
    },
    [isLast, onAdvanceRow],
  );

  const keyRegistration = register(`envVars.${index}.key`);
  const valueRegistration = register(`envVars.${index}.value`);

  return (
    <div className="flex flex-col gap-3">
      {/* Key + Value + Delete side by side */}
      <div className="flex items-start gap-4">
        <FormInput
          label="Key"
          className="flex-1 [&_input]:font-mono"
          placeholder="CLIENT_KEY..."
          error={fieldErrors?.key?.message}
          {...keyRegistration}
          onPaste={handleKeyPaste}
          onKeyDown={handleKeyKeyDown}
        />
        <FormTextarea
          label="Value"
          className="flex-1 [&_textarea]:font-mono"
          placeholder="value"
          rows={1}
          error={fieldErrors?.value?.message}
          variant={!fieldErrors?.value && hasSpaces ? "warning" : undefined}
          description={!fieldErrors?.value && hasSpaces ? "Value contains spaces" : undefined}
          {...valueRegistration}
          onKeyDown={handleValueKeyDown}
        />
        {!isOnly && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="size-9 shrink-0 px-0 justify-center text-gray-11 hover:text-gray-12 hover:bg-grayA-3 rounded-lg mt-6.5"
            onClick={() => onRemove(index)}
          >
            <Trash iconSize="sm-regular" />
          </Button>
        )}
      </div>

      <details className="group">
        <summary className="w-fit text-[13px] text-gray-11 hover:text-gray-12 transition-colors cursor-pointer list-none [&::-webkit-details-marker]:hidden flex items-center gap-1.5 group">
          <span className="group-open:hidden flex items-center gap-2">
            <Plus
              iconSize="sm-medium"
              className="text-gray-9 group-hover:text-gray-12 transition-colors"
            />
            Add Note
          </span>
          <span className="hidden group-open:inline">Note</span>
        </summary>
        <div className="pt-1.5">
          <FormInput
            className="[&_input]:text-sm"
            placeholder="Optional description for this variable..."
            {...register(`envVars.${index}.description`)}
          />
        </div>
      </details>
    </div>
  );
};
