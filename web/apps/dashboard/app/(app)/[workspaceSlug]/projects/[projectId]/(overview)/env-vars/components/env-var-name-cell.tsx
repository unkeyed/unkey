"use client";

import { trpc } from "@/lib/trpc/client";
import { Note3 } from "@unkey/icons";
import { Badge, InfoTooltip, toast } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { HighlightMatch } from "./highlight-match";

type EnvVarNameCellProps = {
  envVarId: string;
  variableKey: string;
  environmentName: string;
  note?: string | null;
  searchQuery: string;
  type: "writeonly" | "recoverable";
};

export const EnvVarNameCell = ({
  envVarId,
  variableKey,
  environmentName,
  note,
  searchQuery,
  type,
}: EnvVarNameCellProps) => {
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
          <InfoTooltip
            content={
              copied
                ? "Copied!"
                : type === "recoverable"
                  ? "Click to copy KEY=VALUE"
                  : "Click to copy key"
            }
            position={{ side: "top" }}
            asChild
          >
            <button
              type="button"
              onClick={handleCopy}
              className="font-mono font-medium text-[13px] text-accent-12 truncate leading-4 cursor-pointer hover:text-accent-11 transition-colors max-w-[250px] "
            >
              <HighlightMatch text={variableKey} query={searchQuery} />
            </button>
          </InfoTooltip>
          {type === "writeonly" && (
            <Badge
              className="px-1.5 py-0 rounded-md h-5 text-[11px] font-medium pointer-events-none"
              variant="warning"
            >
              Sensitive
            </Badge>
          )}
          {note && (
            <InfoTooltip content={note} position={{ side: "top" }}>
              <span className="shrink-0 text-gray-10">
                <Note3 iconSize="md-medium" className="mt-0.5" />
              </span>
            </InfoTooltip>
          )}
        </div>
        <div className="text-[13px] mt-1 text-gray-10 capitalize">{environmentName}</div>
      </div>
    </div>
  );
};
