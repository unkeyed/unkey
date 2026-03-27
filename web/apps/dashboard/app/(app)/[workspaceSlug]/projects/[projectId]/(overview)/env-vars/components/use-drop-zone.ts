import { toast } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import type { UseFormGetValues, UseFormReset, UseFormTrigger } from "react-hook-form";
import type { EnvVarsFormValues } from "./schema";

const ENV_KEY_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

export const parseEnvText = (text: string): Array<{ key: string; value: string }> => {
  const lines = text.trim().split("\n");
  return lines
    .map((line) => {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) {
        return null;
      }

      const eqIndex = trimmed.indexOf("=");
      if (eqIndex === -1) {
        return null;
      }

      const key = trimmed.slice(0, eqIndex).trim();
      let value = trimmed.slice(eqIndex + 1).trim();

      if (
        (value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))
      ) {
        value = value.slice(1, -1);
      }

      if (!ENV_KEY_RE.test(key)) {
        return null;
      }

      return { key, value };
    })
    .filter((v): v is NonNullable<typeof v> => v !== null);
};

const isEnvFile = (file: File) =>
  file.name.endsWith(".env") || file.type === "text/plain" || file.type === "";

export function useDropZone(
  reset: UseFormReset<EnvVarsFormValues>,
  trigger: UseFormTrigger<EnvVarsFormValues>,
  getValues: UseFormGetValues<EnvVarsFormValues>,
) {
  const [isDragging, setIsDragging] = useState(false);
  const ref = useRef<HTMLFormElement>(null);

  const importParsed = useCallback(
    (parsed: Array<{ key: string; value: string }>) => {
      // Deduplicate within the batch — last occurrence wins (standard .env behavior)
      const deduped = new Map<string, { key: string; value: string }>();
      for (const row of parsed) {
        deduped.set(row.key, row);
      }

      const existing = getValues("envVars").filter((row) => row.key !== "");
      const existingKeys = new Set(existing.map((row) => row.key));

      const newRows = [...deduped.values()].filter((row) => !existingKeys.has(row.key));
      const skipped = deduped.size - newRows.length;

      if (newRows.length === 0) {
        toast.info("All variables already exist");
        return;
      }

      reset(
        { ...getValues(), envVars: [...existing, ...newRows.map((row) => ({ ...row, description: "" }))] },
        { keepDefaultValues: true },
      );

      if (skipped > 0) {
        toast.success(`Imported ${newRows.length} variable(s), ${skipped} duplicate(s) skipped`);
      } else {
        toast.success(`Imported ${newRows.length} variable(s)`);
      }
      trigger();
    },
    [getValues, reset, trigger],
  );

  const importText = useCallback(
    (text: string) => {
      const parsed = parseEnvText(text);
      if (parsed.length > 0) {
        importParsed(parsed);
      } else {
        toast.error("No valid environment variables found");
      }
    },
    [importParsed],
  );

  const importFile = useCallback(
    async (file: File) => {
      importText(await file.text());
    },
    [importText],
  );

  useEffect(() => {
    const dropZone = ref.current;
    if (!dropZone) {
      return;
    }

    const handlePaste = async (e: ClipboardEvent) => {
      const target = e.target;
      if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
        return;
      }

      const clipboardData = e.clipboardData;
      if (!clipboardData) {
        return;
      }

      const files = clipboardData.files;
      if (files.length > 0) {
        const file = files[0];
        if (isEnvFile(file)) {
          e.preventDefault();
          importText(await file.text());
          return;
        }
      }

      const text = clipboardData.getData("text/plain");
      if (text?.includes("\n") && text?.includes("=")) {
        e.preventDefault();
        importText(text);
      }
    };

    const handleDragEnter = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(true);
    };

    const handleDragOver = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
    };

    const handleDragLeave = (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const related = e.relatedTarget;
      if (e.currentTarget === dropZone && !(related instanceof Node && dropZone.contains(related))) {
        setIsDragging(false);
      }
    };

    const handleDrop = async (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);

      const files = e.dataTransfer?.files;
      if (!files || files.length === 0) {
        return;
      }

      const file = files[0];
      if (isEnvFile(file)) {
        importText(await file.text());
      } else {
        toast.error("Please drop a .env or text file");
      }
    };

    dropZone.addEventListener("paste", handlePaste);
    dropZone.addEventListener("dragenter", handleDragEnter);
    dropZone.addEventListener("dragover", handleDragOver);
    dropZone.addEventListener("dragleave", handleDragLeave);
    dropZone.addEventListener("drop", handleDrop);

    return () => {
      dropZone.removeEventListener("paste", handlePaste);
      dropZone.removeEventListener("dragenter", handleDragEnter);
      dropZone.removeEventListener("dragover", handleDragOver);
      dropZone.removeEventListener("dragleave", handleDragLeave);
      dropZone.removeEventListener("drop", handleDrop);
    };
  }, [importText]);

  return { ref, isDragging, importFile };
}
