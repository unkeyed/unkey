import { toast } from "@unkey/ui";
import { type ClipboardEvent, type DragEvent, useState } from "react";
import { type EnvEntry, parseEnvText } from "../env-file";

const isEnvFile = (file: File) =>
  file.name.endsWith(".env") || file.type === "text/plain" || file.type === "";

const stop = (e: DragEvent) => {
  e.preventDefault();
  e.stopPropagation();
};

export function useDropZone(onEntries: (entries: EnvEntry[]) => void) {
  const [isDragging, setIsDragging] = useState(false);

  const importText = (text: string) => {
    const { entries } = parseEnvText(text);
    if (entries.length > 0) {
      onEntries(entries);
    } else {
      toast.error("No valid environment variables found");
    }
  };

  const importFile = async (file: File | undefined) => {
    if (!file) {
      return;
    }
    if (!isEnvFile(file)) {
      toast.error("Drop a .env or text file");
      return;
    }
    importText(await file.text());
  };

  const dropZoneProps = {
    onDragEnter: (e: DragEvent) => {
      stop(e);
      setIsDragging(true);
    },
    onDragOver: stop,
    onDragLeave: (e: DragEvent) => {
      stop(e);
      const related = e.relatedTarget;
      if (!(related instanceof Node && e.currentTarget.contains(related))) {
        setIsDragging(false);
      }
    },
    onDrop: (e: DragEvent) => {
      stop(e);
      setIsDragging(false);
      importFile(e.dataTransfer.files[0]);
    },
    onPaste: (e: ClipboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) {
        return;
      }
      const file = e.clipboardData.files[0];
      const text = e.clipboardData.getData("text/plain");
      if (file && isEnvFile(file)) {
        e.preventDefault();
        importFile(file);
      } else if (text.includes("\n") && text.includes("=")) {
        e.preventDefault();
        importText(text);
      }
    },
  };

  return { isDragging, importFile, dropZoneProps };
}
