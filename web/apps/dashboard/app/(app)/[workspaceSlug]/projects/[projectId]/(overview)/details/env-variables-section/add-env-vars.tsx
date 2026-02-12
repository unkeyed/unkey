import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Plus, Trash } from "@unkey/icons";
import { Button, Input, toast } from "@unkey/ui";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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

  const isDoubleQuoted = value.startsWith('"');
  const isSingleQuoted = value.startsWith("'");

  if (isDoubleQuoted || isSingleQuoted) {
    const quote = isDoubleQuoted ? '"' : "'";
    const endQuoteIndex = value.indexOf(quote, 1);
    if (endQuoteIndex !== -1) {
      value = value.slice(1, endQuoteIndex);
    } else {
      value = value.slice(1);
    }
  } else {
    const commentIndex = value.indexOf(" #");
    if (commentIndex !== -1) {
      value = value.slice(0, commentIndex);
    }
  }

  return value.trim();
}

function parseEnvFile(content: string): EnvVarEntry[] {
  const lines = content.split("\n");
  const entries: EnvVarEntry[] = [];

  for (const line of lines) {
    const trimmed = line.trim();
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
  const keyInputRefs = useRef<Map<string, HTMLInputElement>>(new Map());
  const valueInputRefs = useRef<Map<string, HTMLInputElement>>(new Map());
  const [entries, setEntries] = useState<EnvVarEntry[]>([
    { id: crypto.randomUUID(), key: "", value: "", type: "recoverable" },
  ]);

  useEffect(() => {
    containerRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
    });
  }, []);

  const isSubmitting = createMutation.isLoading;

  const addEntry = () => {
    const newEntry = {
      id: crypto.randomUUID(),
      key: "",
      value: "",
      type: "recoverable" as const,
    };
    setEntries([...entries, newEntry]);
    setTimeout(() => {
      keyInputRefs.current.get(newEntry.id)?.focus();
    }, 0);
  };

  const handleKeyDown = (e: React.KeyboardEvent, entryId: string, field: "key" | "value") => {
    if (e.key === "Enter") {
      e.preventDefault();
      const entryIndex = entries.findIndex((entry) => entry.id === entryId);
      const isLastEntry = entryIndex === entries.length - 1;

      if (field === "value" && isLastEntry) {
        addEntry();
      } else if (field === "key") {
        valueInputRefs.current.get(entryId)?.focus();
      } else if (field === "value" && !isLastEntry) {
        const nextEntryId = entries[entryIndex + 1]?.id;
        if (nextEntryId) {
          keyInputRefs.current.get(nextEntryId)?.focus();
        }
      }
    } else if (e.key === "Escape") {
      onCancel();
    }
  };

  const removeEntry = (id: string) => {
    if (entries.length > 1) {
      setEntries(entries.filter((e) => e.id !== id));
      keyInputRefs.current.delete(id);
      valueInputRefs.current.delete(id);
    }
  };

  const updateEntry = (id: string, field: keyof EnvVarEntry, value: string | EnvVarType) => {
    const transformedValue =
      field === "key" && typeof value === "string" ? value.toUpperCase().replace(/ /g, "_") : value;
    setEntries(entries.map((e) => (e.id === id ? { ...e, [field]: transformedValue } : e)));
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>, entryId: string) => {
    const pastedText = e.clipboardData.getData("text");

    if (pastedText.includes("\n") && pastedText.includes("=")) {
      e.preventDefault();
      const parsed = parseEnvFile(pastedText);
      if (parsed.length > 0) {
        const currentIndex = entries.findIndex((e) => e.id === entryId);
        const newEntries = [...entries];
        newEntries.splice(currentIndex, 1, ...parsed);
        setEntries(newEntries);
      }
    }
  };

  const getErrors = useCallback(
    (entry: EnvVarEntry): { key?: string; value?: string } => {
      const errors: { key?: string; value?: string } = {};

      if (entry.key && !ENV_VAR_KEY_REGEX.test(entry.key)) {
        errors.key = "Must be UPPERCASE";
      } else if (entry.key && getExistingEnvVar(entry.key)) {
        errors.key = "Already exists";
      } else if (entry.key) {
        const duplicates = entries.filter((e) => e.key === entry.key);
        if (duplicates.length > 1) {
          errors.key = "Duplicate";
        }
      }

      return errors;
    },
    [entries, getExistingEnvVar],
  );

  const validEntries = useMemo(
    () =>
      entries.filter((e) => {
        const errors = getErrors(e);
        return e.key && e.value && !errors.key && !errors.value;
      }),
    [entries, getErrors],
  );

  const handleSave = async () => {
    if (validEntries.length === 0) {
      return;
    }

    const mutation = createMutation.mutateAsync({
      environmentId,
      variables: validEntries.map((e) => ({
        key: e.key,
        value: e.value,
        type: e.type,
      })),
    });

    toast.promise(mutation, {
      loading: "Adding environment variables...",
      success: `Added ${validEntries.length} environment variable${
        validEntries.length > 1 ? "s" : ""
      }`,
      error: (err) => ({
        message: "Failed to add environment variables",
        description: err.message || "Please try again",
      }),
    });

    try {
      await mutation;
      onSuccess();
    } catch {}
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
                  ref={(el) => {
                    if (el) {
                      keyInputRefs.current.set(entry.id, el);
                    }
                  }}
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
                ref={(el) => {
                  if (el) {
                    valueInputRefs.current.set(entry.id, el);
                  }
                }}
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
                variant="ghost"
                size="icon"
                onClick={() => removeEntry(entry.id)}
                disabled={entries.length === 1 || isSubmitting}
                className="h-[32px] w-[32px] text-gray-9 hover:text-gray-11 hover:bg-gray-3 shrink-0"
              >
                <Trash className="size-4" iconSize="md-medium" />
              </Button>
            </div>
          );
        })}
      </div>

      <div className="px-4 py-2 flex items-center justify-between border-t border-gray-4 sticky bottom-0 bg-gray-2">
        <Button size="sm" variant="ghost" onClick={addEntry} className="text-xs gap-1 px-3">
          <Plus className="!size-3" />
          Add more
        </Button>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={onCancel}
            disabled={isSubmitting}
            className="text-xs px-3"
          >
            Cancel
          </Button>
          <Button
            size="sm"
            variant="primary"
            onClick={handleSave}
            disabled={validEntries.length === 0 || isSubmitting}
            loading={isSubmitting}
            className="text-xs px-3"
          >
            Save {validEntries.length > 0 && `(${validEntries.length})`}
          </Button>
        </div>
      </div>
    </div>
  );
}
