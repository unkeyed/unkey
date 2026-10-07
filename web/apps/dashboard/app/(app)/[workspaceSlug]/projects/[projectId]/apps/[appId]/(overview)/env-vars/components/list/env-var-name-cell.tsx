"use client";

import { IconNote3Outline18 } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { HighlightMatch } from "../shared/highlight-match";

type EnvVarNameCellProps = {
  value: string;
  variableKey: string;
  note?: string | null;
  searchQuery: string;
  type: "writeonly" | "recoverable";
};

export function EnvVarNameCell({
  value,
  variableKey,
  note,
  searchQuery,
  type,
}: EnvVarNameCellProps) {
  const [copied, setCopied] = useState(false);
  const copyTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      clearTimeout(copyTimeoutRef.current);
    };
  }, []);

  const handleCopy = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      const text = type === "recoverable" ? `${variableKey}=${value}` : variableKey;
      try {
        await navigator.clipboard.writeText(text);
        setCopied(true);
        toast.success("Copied to clipboard");
        clearTimeout(copyTimeoutRef.current);
        copyTimeoutRef.current = setTimeout(() => setCopied(false), 2000);
      } catch {
        toast.error("Failed to copy variable");
      }
    },
    [variableKey, type, value],
  );

  return (
    <div className="flex min-w-0 items-center gap-1.5">
      <InfoTooltip
        content={
          copied ? (
            "Copied!"
          ) : (
            <div className="flex flex-col gap-0.5">
              <span className="font-mono break-all">{variableKey}</span>
              <span className="opacity-75">
                {type === "recoverable" ? "Copy KEY=VALUE" : "Copy key"}
              </span>
            </div>
          )
        }
        position={{ side: "top" }}
        asChild
      >
        <button
          type="button"
          onClick={handleCopy}
          className="min-w-0 cursor-pointer truncate text-left font-mono text-sm text-gray-12 transition-colors hover:text-gray-11"
        >
          <HighlightMatch text={variableKey} query={searchQuery} />
        </button>
      </InfoTooltip>
      {note && (
        <InfoTooltip content={note} position={{ side: "top" }}>
          <span className="shrink-0 text-gray-10">
            <IconNote3Outline18 className="size-3.5" />
          </span>
        </InfoTooltip>
      )}
    </div>
  );
}
