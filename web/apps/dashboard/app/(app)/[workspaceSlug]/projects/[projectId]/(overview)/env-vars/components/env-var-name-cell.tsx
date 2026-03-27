"use client";

import { trpc } from "@/lib/trpc/client";
import { CircleInfo } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import { HighlightMatch } from "./highlight-match";

type EnvVarNameCellProps = {
  envVarId: string;
  variableKey: string;
  environmentName: string;
  note?: string | null;
  searchQuery: string;
  type: "writeonly" | "recoverable";
};

export const EnvVarNameCell = memo(function EnvVarNameCell({
  envVarId,
  variableKey,
  environmentName,
  note,
  searchQuery,
  type,
}: EnvVarNameCellProps) {
  const [copied, setCopied] = useState(false);
  const copyTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();

  useEffect(() => {
    return () => {
      clearTimeout(copyTimeoutRef.current);
    };
  }, []);

  const handleCopy = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        if (type === "recoverable") {
          const result = await decryptMutation.mutateAsync({ envVarId });
          navigator.clipboard.writeText(`${variableKey}=${result.value}`);
        } else {
          navigator.clipboard.writeText(variableKey);
        }
        setCopied(true);
        toast.success("Copied to clipboard");
        clearTimeout(copyTimeoutRef.current);
        copyTimeoutRef.current = setTimeout(() => setCopied(false), 2000);
      } catch {
        toast.error("Failed to copy variable");
      }
    },
    [variableKey, type, envVarId, decryptMutation],
  );

  return (
    <div className="flex items-center gap-3 px-4">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <InfoTooltip content={copied ? "Copied!" : type === "recoverable" ? "Click to copy KEY=VALUE" : "Click to copy key"} position={{ side: "top" }}>
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
                <CircleInfo iconSize="lg-regular" className="mt-0.5" />
              </span>
            </InfoTooltip>
          )}
        </div>
        <div className="text-[13px] mt-1 text-gray-10 capitalize">{environmentName}</div>
      </div>
    </div>
  );
});
