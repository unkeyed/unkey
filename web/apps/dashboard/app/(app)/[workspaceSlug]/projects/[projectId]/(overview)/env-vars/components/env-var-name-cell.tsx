"use client";

import { CircleInfo } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { HighlightMatch } from "./highlight-match";

type EnvVarNameCellProps = {
  variableKey: string;
  environmentName: string;
  note?: string | null;
  searchQuery: string;
};

export const EnvVarNameCell = memo(function EnvVarNameCell({
  variableKey,
  environmentName,
  note,
  searchQuery,
}: EnvVarNameCellProps) {
  const [copied, setCopied] = useState(false);
  const copyTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      clearTimeout(copyTimeoutRef.current);
    };
  }, []);

  const handleCopy = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      navigator.clipboard.writeText(variableKey);
      setCopied(true);
      toast.success("Copied to clipboard");
      clearTimeout(copyTimeoutRef.current);
      copyTimeoutRef.current = setTimeout(() => setCopied(false), 2000);
    },
    [variableKey],
  );

  return (
    <div className="flex items-center gap-3 px-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <InfoTooltip content={copied ? "Copied!" : "Click to copy"} position={{ side: "top" }}>
            <button
              type="button"
              onClick={handleCopy}
              className="font-mono font-medium text-[13px] text-accent-12 truncate leading-4 cursor-pointer hover:text-accent-11 transition-colors max-w-[250px] "
            >
              <HighlightMatch text={variableKey} query={searchQuery} />
            </button>
          </InfoTooltip>
          {note && (
            <InfoTooltip content={note} position={{ side: "top" }}>
              <span className="shrink-0 text-gray-9">
                <CircleInfo iconSize="md-regular" />
              </span>
            </InfoTooltip>
          )}
        </div>
        <div className="text-[13px] mt-1 text-gray-10 capitalize">{environmentName}</div>
      </div>
    </div>
  );
});
