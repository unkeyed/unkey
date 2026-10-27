"use client";
import * as React from "react";
import { toast } from "../components/toaster";

const COPIED_MS = 2000;

export function useCopyToClipboard(value: string, toastMessage?: string) {
  const [copiedAt, setCopiedAt] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (copiedAt === null) {
      return;
    }
    const timer = setTimeout(() => setCopiedAt(null), COPIED_MS);
    return () => clearTimeout(timer);
  }, [copiedAt]);

  const copy = React.useCallback(async () => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success("Copied to clipboard", { description: toastMessage });
      setCopiedAt(Date.now());
    } catch (e) {
      toast.error("Failed to copy to clipboard", {
        description: e instanceof Error ? e.message : "Unknown error",
      });
    }
  }, [value, toastMessage]);

  return { copied: copiedAt !== null, copy };
}
