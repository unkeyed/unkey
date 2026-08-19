"use client";

import { Eye, EyeSlash } from "@unkey/icons";
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
    <div className="flex items-center min-w-0">
      <div className="shrink-0 w-7 flex items-center justify-center">
        <button
          type="button"
          aria-label={visible ? "Click to hide" : "Click to reveal"}
          title={visible ? "Click to hide" : "Click to reveal"}
          onClick={handleToggleReveal}
          className="text-gray-10 hover:text-gray-11 transition-colors cursor-pointer hover:bg-gray-3 rounded-md px-1.5 py-0.5 h-[22px]"
        >
          {visible ? <EyeSlash iconSize="sm-regular" /> : <Eye iconSize="sm-regular" />}
        </button>
      </div>
      {visible ? (
        <InfoTooltip content={copied ? "Copied!" : "Click to copy"} position={{ side: "top" }}>
          <button
            type="button"
            onClick={handleCopy}
            className="font-mono bg-gray-3 px-1.5 py-0.5 truncate text-[13px] text-accent-12 cursor-pointer transition-colors min-w-0 rounded-md h-5.5 max-w-70"
          >
            {value}
          </button>
        </InfoTooltip>
      ) : (
        <span className="font-mono text-[13px] text-gray-11">••••••••••••</span>
      )}
    </div>
  );
});
