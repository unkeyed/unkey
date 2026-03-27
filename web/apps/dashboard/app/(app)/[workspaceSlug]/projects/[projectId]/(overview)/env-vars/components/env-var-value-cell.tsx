"use client";

import { trpc } from "@/lib/trpc/client";
import { CircleLock, Eye, EyeSlash } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { memo, useCallback, useEffect, useRef, useState } from "react";

type EnvVarValueCellProps = {
  envVarId: string;
  type: "writeonly" | "recoverable";
};

export const EnvVarValueCell = memo(function EnvVarValueCell({
  envVarId,
  type,
}: EnvVarValueCellProps) {
  const [visible, setVisible] = useState(false);
  const [copied, setCopied] = useState(false);
  const [decryptedValue, setDecryptedValue] = useState<string>();
  const copyTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const isWriteonly = type === "writeonly";

  useEffect(() => {
    return () => {
      clearTimeout(copyTimeoutRef.current);
    };
  }, []);

  const handleToggleReveal = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();

      if (visible) {
        setVisible(false);
        return;
      }

      if (decryptedValue !== undefined) {
        setVisible(true);
        return;
      }

      try {
        const result = await decryptMutation.mutateAsync({ envVarId });
        setDecryptedValue(result.value);
        setVisible(true);
      } catch {
        toast.error("Failed to decrypt value");
      }
    },
    [visible, decryptedValue, envVarId, decryptMutation],
  );

  const handleCopy = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      if (decryptedValue === undefined) {
        return;
      }
      navigator.clipboard.writeText(decryptedValue);
      setCopied(true);
      toast.success("Copied to clipboard");
      clearTimeout(copyTimeoutRef.current);
      copyTimeoutRef.current = setTimeout(() => setCopied(false), 2000);
    },
    [decryptedValue],
  );

  return (
    <div className="flex items-center min-w-0">
      <div className="shrink-0 w-7 flex items-center justify-center">
        {isWriteonly ? (
          <CircleLock iconSize="sm-regular" className="text-gray-11" />
        ) : (
          <button
            type="button"
            aria-label={visible ? "Click to hide" : "Click to reveal"}
            title={visible ? "Click to hide" : "Click to reveal"}
            onClick={handleToggleReveal}
            disabled={decryptMutation.isLoading}
            className="text-gray-10 hover:text-gray-11 transition-colors cursor-pointer hover:bg-gray-3 rounded-md px-1.5 py-0.5 h-[22px]"
          >
            {decryptMutation.isLoading ? (
              <span className="text-[11px] text-gray-9">...</span>
            ) : visible ? (
              <EyeSlash iconSize="sm-regular" />
            ) : (
              <Eye iconSize="sm-regular" />
            )}
          </button>
        )}
      </div>
      {!isWriteonly && visible && decryptedValue !== undefined ? (
        <InfoTooltip content={copied ? "Copied!" : "Click to copy"} position={{ side: "top" }}>
          <button
            type="button"
            onClick={handleCopy}
            className="font-mono max-w-90 bg-gray-3 px-1.5 py-0.5 truncate text-[13px] text-accent-12 cursor-pointer hover:text-accent-11 transition-colors min-w-0 rounded-md h-[22px]"
          >
            {decryptedValue}
          </button>
        </InfoTooltip>
      ) : (
        <span className="font-mono text-[13px] text-gray-11">••••••••••••</span>
      )}
    </div>
  );
});
