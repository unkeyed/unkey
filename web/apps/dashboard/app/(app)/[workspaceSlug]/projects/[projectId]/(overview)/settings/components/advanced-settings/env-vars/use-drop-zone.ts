import { toast } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import type { UseFormReset } from "react-hook-form";
import type { EnvVarsFormValues } from "./schema";

const parseEnvText = (text: string): Array<{ key: string; value: string; secret: boolean }> => {
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

      if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) {
        return null;
      }

      return { key, value, secret: false };
    })
    .filter((v): v is NonNullable<typeof v> => v !== null);
};

export function useDropZone(reset: UseFormReset<EnvVarsFormValues>, defaultEnvironmentId: string) {
  const [isDragging, setIsDragging] = useState(false);
  const ref = useRef<HTMLFormElement>(null);

  useEffect(() => {
    const dropZone = ref.current;
    if (!dropZone) {
      return;
    }

    const handlePaste = async (e: ClipboardEvent) => {
      const clipboardData = e.clipboardData;
      if (!clipboardData) {
        return;
      }

      const files = clipboardData.files;
      if (files.length > 0) {
        const file = files[0];
        if (file.name.endsWith(".env") || file.type === "text/plain" || file.type === "") {
          e.preventDefault();
          const text = await file.text();
          const parsed = parseEnvText(text);
          if (parsed.length > 0) {
            reset(
              {
                envVars: parsed.map((row) => ({
                  ...row,
                  environmentId: defaultEnvironmentId,
                })),
              },
              { keepDefaultValues: true },
            );
            toast.success(`Imported ${parsed.length} variable(s)`);
          } else {
            toast.error("No valid environment variables found");
          }
          return;
        }
      }

      const text = clipboardData.getData("text/plain");
      if (text?.includes("\n") && text?.includes("=")) {
        e.preventDefault();
        const parsed = parseEnvText(text);
        if (parsed.length > 0) {
          reset(
            {
              envVars: parsed.map((row) => ({
                ...row,
                environmentId: defaultEnvironmentId,
              })),
            },
            { keepDefaultValues: true },
          );
          toast.success(`Imported ${parsed.length} variable(s)`);
        } else {
          toast.error("No valid environment variables found");
        }
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
      if (e.currentTarget === dropZone && !dropZone.contains(e.relatedTarget as Node)) {
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
      if (file.name.endsWith(".env") || file.type === "text/plain" || file.type === "") {
        const text = await file.text();
        const parsed = parseEnvText(text);
        if (parsed.length > 0) {
          reset(
            {
              envVars: parsed.map((row) => ({
                ...row,
                environmentId: defaultEnvironmentId,
              })),
            },
            { keepDefaultValues: true },
          );
          toast.success(`Imported ${parsed.length} variable(s)`);
        } else {
          toast.error("No valid environment variables found");
        }
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
  }, [reset, defaultEnvironmentId]);

  return { ref, isDragging };
}
