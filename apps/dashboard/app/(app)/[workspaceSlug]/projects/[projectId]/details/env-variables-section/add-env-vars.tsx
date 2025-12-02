import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Plus, Trash } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import { EnvVarSecretSwitch } from "./components/env-var-secret-switch";
import { ENV_VAR_KEY_REGEX, type EnvVar, type EnvVarType } from "./types";

type EnvVarEntry = {
  id: string;
  key: string;
  value: string;
  type: EnvVarType;
};

type AddEnvVarsProps = {
  environmentId: string;
  getExistingEnvVar: (key: string, excludeId?: string) => EnvVar | undefined;
  onCancel: () => void;
  onSuccess: () => void;
};

function cleanEnvValue(raw: string): string {
  let value = raw.trim();

  // Check if value is quoted
  const isDoubleQuoted = value.startsWith('"');
  const isSingleQuoted = value.startsWith("'");

  if (isDoubleQuoted || isSingleQuoted) {
    const quote = isDoubleQuoted ? '"' : "'";
    const endQuoteIndex = value.indexOf(quote, 1);
    if (endQuoteIndex !== -1) {
      // Extract content between quotes
      value = value.slice(1, endQuoteIndex);
    } else {
      // No closing quote, just remove the opening one
      value = value.slice(1);
    }
  } else {
    // Unquoted value - remove trailing comments
    const commentIndex = value.indexOf(" #");
    if (commentIndex !== -1) {
      value = value.slice(0, commentIndex);
    }
    // Also check for # without space (less common but valid)
    const hashIndex = value.indexOf("#");
    if (hashIndex !== -1 && !value.slice(0, hashIndex).includes(" ")) {
      // Only trim if # appears to be a comment, not part of the value
      // This is a heuristic - if there's no space before #, it might be part of value
    }
  }

  return value.trim();
}

function parseEnvFile(content: string): EnvVarEntry[] {
  const lines = content.split("\n");
  const entries: EnvVarEntry[] = [];

  for (const line of lines) {
    const trimmed = line.trim();
    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const eqIndex = trimmed.indexOf("=");
    if (eqIndex === -1) {
      continue;
    }

    const key = trimmed.slice(0, eqIndex).trim();
    const value = cleanEnvValue(trimmed.slice(eqIndex + 1));

    const upperKey = key.toUpperCase();
    if (upperKey && ENV_VAR_KEY_REGEX.test(upperKey)) {
      entries.push({
        id: crypto.randomUUID(),
        key: upperKey,
        value,
        type: "recoverable",
      });
    }
  }

  return entries;
}

export function AddEnvVars({
  environmentId,
  getExistingEnvVar,
  onCancel,
  onSuccess,
}: AddEnvVarsProps) {
  const createMutation = trpc.deploy.envVar.create.useMutation();
  const containerRef = useRef<HTMLDivElement>(null);
  const [entries, setEntries] = useState<EnvVarEntry[]>([
    { id: crypto.randomUUID(), key: "", value: "", type: "recoverable" },
  ]);

  // Scroll into view when component mounts
  useEffect(() => {
    containerRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, []);

  const isSubmitting = createMutation.isLoading;

  const addEntry = () => {
    const newEntry = { id: crypto.randomUUID(), key: "", value: "", type: "recoverable" as const };
    setEntries([...entries, newEntry]);
    // Focus the new entry's key input after render
    setTimeout(() => {
      const inputs = containerRef.current?.querySelectorAll<HTMLInputElement>(
        'input[placeholder="KEY_NAME"]',
      );
      inputs?.[inputs.length - 1]?.focus();
    }, 0);
  };

  const handleKeyDown = (e: React.KeyboardEvent, entryId: string, field: "key" | "value") => {
    if (e.key === "Enter") {
      e.preventDefault();
      const entryIndex = entries.findIndex((entry) => entry.id === entryId);
      const isLastEntry = entryIndex === entries.length - 1;

      if (field === "value" && isLastEntry) {
        // On last entry's value field, create a new row
        addEntry();
      } else if (field === "key") {
        // Move focus to value field
        const valueInput = (e.target as HTMLInputElement)
          .closest(".flex")
          ?.querySelector<HTMLInputElement>('input[placeholder="Variable value"]');
        valueInput?.focus();
      } else if (field === "value" && !isLastEntry) {
        // Move focus to next row's key field
        const allRows = containerRef.current?.querySelectorAll<HTMLDivElement>(
          ".flex.items-center.gap-2",
        );
        const nextRow = allRows?.[entryIndex + 1];
        nextRow?.querySelector<HTMLInputElement>('input[placeholder="KEY_NAME"]')?.focus();
      }
    } else if (e.key === "Escape") {
      onCancel();
    }
  };

  const removeEntry = (id: string) => {
    if (entries.length > 1) {
      setEntries(entries.filter((e) => e.id !== id));
    }
  };

  const updateEntry = (id: string, field: keyof EnvVarEntry, value: string | EnvVarType) => {
    // Auto-uppercase the key field and replace spaces with underscores
    const transformedValue =
      field === "key" && typeof value === "string" ? value.toUpperCase().replace(/ /g, "_") : value;
    setEntries(entries.map((e) => (e.id === id ? { ...e, [field]: transformedValue } : e)));
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>, entryId: string) => {
    const pastedText = e.clipboardData.getData("text");

    // Check if it looks like an .env file (multiple lines with = signs)
    if (pastedText.includes("\n") && pastedText.includes("=")) {
      e.preventDefault();
      const parsed = parseEnvFile(pastedText);
      if (parsed.length > 0) {
        // Replace the current entry and add new ones
        const currentIndex = entries.findIndex((e) => e.id === entryId);
        const newEntries = [...entries];
        newEntries.splice(currentIndex, 1, ...parsed);
        setEntries(newEntries);
      }
    }
  };

  const getErrors = (entry: EnvVarEntry): { key?: string; value?: string } => {
    const errors: { key?: string; value?: string } = {};

    if (entry.key && !ENV_VAR_KEY_REGEX.test(entry.key)) {
      errors.key = "Must be UPPERCASE";
    } else if (entry.key && getExistingEnvVar(entry.key)) {
      errors.key = "Already exists";
    } else if (entry.key) {
      // Check for duplicates within the current entries
      const duplicates = entries.filter((e) => e.key === entry.key);
      if (duplicates.length > 1) {
        errors.key = "Duplicate";
      }
    }

    return errors;
  };

  const validEntries = entries.filter((e) => {
    const errors = getErrors(e);
    return e.key && e.value && !errors.key && !errors.value;
  });

  const handleSave = async () => {
    if (validEntries.length === 0) {
      return;
    }

    try {
      await createMutation.mutateAsync({
        environmentId,
        variables: validEntries.map((e) => ({
          key: e.key,
          value: e.value,
          type: e.type,
        })),
      });
      onSuccess();
    } catch (error) {
      console.error("Failed to add env vars:", error);
    }
  };

  return (
    <div ref={containerRef} className="bg-gray-2 border-b border-gray-4">
      <div className="px-4 py-2 border-b border-gray-4">
        <span className="text-xs text-gray-11">
          Add variables (paste .env file content to bulk import)
        </span>
      </div>

      <div className="max-h-48 overflow-y-auto">
        {entries.map((entry) => {
          const errors = getErrors(entry);
          return (
            <div
              key={entry.id}
              className="flex items-center gap-2 px-4 py-3 border-b border-gray-4 last:border-b-0"
            >
              <div className="w-[108px]">
                <Input
                  type="text"
                  placeholder="KEY_NAME"
                  value={entry.key}
                  onChange={(e) => updateEntry(entry.id, "key", e.target.value)}
                  onKeyDown={(e) => handleKeyDown(e, entry.id, "key")}
                  onPaste={(e) => handlePaste(e, entry.id)}
                  className={cn(
                    "min-h-[32px] text-xs w-[108px] font-mono uppercase",
                    errors.key && "border-red-6 focus:border-red-7",
                  )}
                  autoComplete="off"
                  spellCheck={false}
                />
              </div>
              <span className="text-gray-9 text-xs px-1">=</span>
              <Input
                type={entry.type === "writeonly" ? "password" : "text"}
                placeholder="Variable value"
                value={entry.value}
                onChange={(e) => updateEntry(entry.id, "value", e.target.value)}
                onKeyDown={(e) => handleKeyDown(e, entry.id, "value")}
                onPaste={(e) => handlePaste(e, entry.id)}
                className="min-h-[32px] text-xs flex-1 font-mono"
                autoComplete={entry.type === "writeonly" ? "new-password" : "off"}
                spellCheck={false}
              />
              <EnvVarSecretSwitch
                isSecret={entry.type === "writeonly"}
                onCheckedChange={(checked) =>
                  updateEntry(entry.id, "type", checked ? "writeonly" : "recoverable")
                }
                disabled={isSubmitting}
              />
              <Button
                size="icon"
                variant="ghost"
                onClick={() => removeEntry(entry.id)}
                disabled={entries.length === 1}
                className="size-6 text-gray-9 hover:text-gray-11"
              >
                <Trash className="!size-3" />
              </Button>
            </div>
          );
        })}
      </div>

      <div className="px-4 py-2 flex items-center justify-between border-t border-gray-4 sticky bottom-0 bg-gray-2">
        <Button size="sm" variant="ghost" onClick={addEntry} className="text-xs text-gray-11 gap-1">
          <Plus className="!size-3" />
          Add more
        </Button>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" onClick={onCancel} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            size="sm"
            variant="primary"
            onClick={handleSave}
            disabled={validEntries.length === 0 || isSubmitting}
            loading={isSubmitting}
          >
            Save {validEntries.length > 0 && `(${validEntries.length})`}
          </Button>
        </div>
      </div>
    </div>
  );
}
