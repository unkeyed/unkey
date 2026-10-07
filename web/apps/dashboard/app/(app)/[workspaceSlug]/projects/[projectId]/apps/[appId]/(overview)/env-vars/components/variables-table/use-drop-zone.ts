import { toast } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { type EnvEntry, parseEnvText } from "../../env-file";

const isEnvFile = (file: File) =>
  file.name.endsWith(".env") || file.type === "text/plain" || file.type === "";

export function useDropZone<T extends HTMLElement>(onEntries: (entries: EnvEntry[]) => void) {
  const [isDragging, setIsDragging] = useState(false);
  const [dropZone, setDropZone] = useState<T | null>(null);
  const onEntriesRef = useRef(onEntries);
  useEffect(() => {
    onEntriesRef.current = onEntries;
  });

  const importText = useCallback((text: string) => {
    const { entries } = parseEnvText(text);
    if (entries.length > 0) {
      onEntriesRef.current(entries);
    } else {
      toast.error("No valid environment variables found");
    }
  }, []);

  const importFile = useCallback(
    async (file: File) => {
      importText(await file.text());
    },
    [importText],
  );

  useEffect(() => {
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
      if (
        e.currentTarget === dropZone &&
        !(related instanceof Node && dropZone.contains(related))
      ) {
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
        toast.error("Drop a .env or text file");
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
  }, [dropZone, importText]);

  return { ref: setDropZone, isDragging, importFile };
}
