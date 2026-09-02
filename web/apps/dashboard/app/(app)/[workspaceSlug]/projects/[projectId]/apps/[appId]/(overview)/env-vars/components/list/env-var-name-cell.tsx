"use client";

import { Badge, InfoTooltip, toast } from "@unkey/ui";
import { IconNote3Outline18 } from "nucleo-ui-outline-18";
import { useCallback, useEffect, useRef, useState } from "react";
import { HighlightMatch } from "../shared/highlight-match";

type EnvVarNameCellProps = {
  value: string;
  variableKey: string;
  environmentName: string;
  note?: string | null;
  searchQuery: string;
  type: "writeonly" | "recoverable";
};

export const EnvVarNameCell = ({
  value,
  variableKey,
  environmentName,
  note,
  searchQuery,
  type,
}: EnvVarNameCellProps) => {
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
    <div className="flex items-center gap-3 px-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <InfoTooltip
            content={
              copied ? (
                "Copied!"
              ) : (
                <div className="flex flex-col gap-0.5">
                  <span className="font-mono break-all">{variableKey}</span>
                  <span className="text-gray-11">
                    {type === "recoverable" ? "Click to copy KEY=VALUE" : "Click to copy key"}
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
              className="font-mono font-normal text-[13px] text-accent-12 truncate leading-4 cursor-pointer hover:text-accent-11 transition-colors max-w-[250px] "
            >
              <HighlightMatch text={variableKey} query={searchQuery} />
            </button>
          </InfoTooltip>
          {type === "writeonly" && (
            <Badge
              className="px-1.5 py-0 rounded-md h-5 text-[11px] font-normal pointer-events-none"
              variant="warning"
            >
              Sensitive
            </Badge>
          )}
          {note && (
            <InfoTooltip content={note} position={{ side: "top" }}>
              <span className="shrink-0 text-gray-10">
                <IconNote3Outline18 className="size-4 mt-0.5" />
              </span>
            </InfoTooltip>
          )}
        </div>
        <div className="text-[13px] mt-1 text-gray-10 capitalize">{environmentName}</div>
      </div>
    </div>
  );
};
