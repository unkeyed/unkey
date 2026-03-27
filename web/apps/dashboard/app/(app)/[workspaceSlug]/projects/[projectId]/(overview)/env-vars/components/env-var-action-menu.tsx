"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useMemo } from "react";

type EnvVarActionMenuProps = {
  envVarId: string;
  variableKey: string;
  type: "writeonly" | "recoverable";
  onEdit: () => void;
};

export function EnvVarActionMenu({ envVarId, variableKey, type, onEdit }: EnvVarActionMenuProps) {
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();

  const menuItems = useMemo((): MenuItem[] => {
    return [
      {
        id: "edit",
        label: "Edit",
        icon: <PenWriting3 iconSize="md-regular" />,
        onClick: (e) => {
          e.stopPropagation();
          onEdit();
        },
      },
      {
        id: "delete",
        label: "Delete",
        icon: <Trash iconSize="md-regular" />,
        divider: true,
        onClick: (e) => {
          e.stopPropagation();
          try {
            collection.envVars.delete(envVarId);
            toast.success("Variable deleted");
          } catch {
            toast.error("Failed to delete variable");
          }
        },
      },
      {
        id: "copy",
        label: "Copy to Clipboard",
        icon: <Clone iconSize="md-regular" />,
        disabled: type === "writeonly",
        tooltip: type === "writeonly" ? "Write-only variables cannot be copied" : undefined,
        onClick: async (e) => {
          e.stopPropagation();
          try {
            const result = await decryptMutation.mutateAsync({ envVarId });
            navigator.clipboard.writeText(`${variableKey}=${result.value}`);
            toast.success("Copied to clipboard");
          } catch {
            toast.error("Failed to decrypt value");
          }
        },
      },
    ];
  }, [envVarId, variableKey, type, onEdit, decryptMutation]);

  return <TableActionPopover items={menuItems} />;
}
