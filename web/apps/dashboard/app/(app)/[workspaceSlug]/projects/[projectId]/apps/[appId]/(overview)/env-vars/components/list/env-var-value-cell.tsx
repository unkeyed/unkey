"use client";

import { IconEyeOutline12, IconEyeSlashOutline18 } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { memo, useCallback, useEffect, useRef, useState } from "react";

const AUTO_HIDE_MS = 10_000;

type EnvVarValueCellProps = {
  value: string;
  type: "writeonly" | "recoverable";
};

export const EnvVarValueCell = memo(function EnvVarValueCell({
  value,
  type,
}: EnvVarValueCellProps) {
  const [visible, setVisible] = useState(false);
  const [copied, setCopied] = useState(false);
  const copyTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const hideTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const isWriteonly = type === "writeonly";

  useEffect(() => {
    return () => {
      clearTimeout(copyTimeoutRef.current);
      clearTimeout(hideTimeoutRef.current);
    };
  }, []);

  const startAutoHide = useCallback(() => {
    clearTimeout(hideTimeoutRef.current);
    hideTimeoutRef.current = setTimeout(() => {
      setVisible(false);
    }, AUTO_HIDE_MS);
  }, []);

  const handleToggleReveal = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();

      if (visible) {
        setVisible(false);
        clearTimeout(hideTimeoutRef.current);
        return;
      }

      setVisible(true);
      startAutoHide();
    },
    [visible, startAutoHide],
  );

  const handleCopy = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await navigator.clipboard.writeText(value);
        setCopied(true);
        toast.success("Copied to clipboard");
        clearTimeout(copyTimeoutRef.current);
        copyTimeoutRef.current = setTimeout(() => setCopied(false), 2000);
        startAutoHide();
      } catch {
        toast.error("Failed to copy to clipboard");
      }
    },
    [value, startAutoHide],
  );

  if (isWriteonly) {
    return null;
  }

  return (
    <div data-sentry-mask className="flex min-w-0 items-center gap-1">
      <div className="flex shrink-0 items-center">
        <button
          type="button"
          aria-label={visible ? "Hide value" : "Reveal value"}
          title={visible ? "Hide value" : "Reveal value"}
          onClick={handleToggleReveal}
          className="flex size-6 cursor-pointer items-center justify-center rounded-md text-gray-10 transition-colors hover:bg-grayA-3 hover:text-gray-12"
        >
          {visible ? <IconEyeSlashOutline18 className="size-3" /> : <IconEyeOutline12 />}
        </button>
      </div>
      {visible ? (
        <InfoTooltip content={copied ? "Copied!" : "Copy"} position={{ side: "top" }} asChild>
          <button
            type="button"
            onClick={handleCopy}
            className="min-w-0 cursor-pointer truncate rounded-md bg-grayA-3 px-1.5 py-0.5 font-mono text-xs text-gray-12"
          >
            {value}
          </button>
        </InfoTooltip>
      ) : (
        <span className="font-mono text-xs tracking-widest text-gray-11">••••••••••••</span>
      )}
    </div>
  );
});
