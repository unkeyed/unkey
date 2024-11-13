import { useCallback, useEffect, useRef, useState } from "react";

const DEFAULT_TIMEOUT = 3000;

export const useCopyToClipboard = (
  timeout = DEFAULT_TIMEOUT,
): [boolean, (value: string | ClipboardItem) => Promise<void>] => {
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [copied, setCopied] = useState(false);

  const clearTimer = () => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
  };

  const writeToClipboard = async (value: string | ClipboardItem) => {
    const isClipboardAvailable =
      typeof navigator !== "undefined" && navigator.clipboard !== undefined;

    if (!isClipboardAvailable) {
      throw new Error("Clipboard API is not supported in this browser");
    }

    if (typeof value === "string") {
      await navigator.clipboard.writeText(value);
    } else if (value instanceof ClipboardItem) {
      await navigator.clipboard.write([value]);
    }
  };

  const handleTimeout = () => {
    if (Number.isFinite(timeout) && timeout >= 0) {
      timer.current = setTimeout(() => setCopied(false), timeout);
    } else {
      console.warn(`Invalid timeout value; defaulting to ${DEFAULT_TIMEOUT}ms`);
      timer.current = setTimeout(() => setCopied(false), DEFAULT_TIMEOUT);
    }
  };

  const copyToClipboard = useCallback(
    async (value: string | ClipboardItem) => {
      clearTimer();
      try {
        await writeToClipboard(value);
        setCopied(true);
        handleTimeout();
      } catch (error) {
        console.warn("Failed to copy to clipboard. ", error);
        throw error; // Propagate error for higher-level handling
      }
    },
    [timeout],
  );

  // Cleanup the timer when the component unmounts
  useEffect(() => {
    return () => clearTimer();
  }, []);

  return [copied, copyToClipboard];
};
