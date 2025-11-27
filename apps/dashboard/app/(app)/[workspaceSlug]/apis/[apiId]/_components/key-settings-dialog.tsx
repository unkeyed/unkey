"use client";
import { TableActionPopover } from "@/components/logs/table-action.popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useTRPC } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { Gear } from "@unkey/icons";
import { getKeysTableActionItems } from "../keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover.constants";

import { useQueryClient } from "@tanstack/react-query";

interface KeySettingsDialogProps {
  keyData: KeyDetails;
}

export const KeySettingsDialog = ({ keyData }: KeySettingsDialogProps) => {
  const queryClient = useQueryClient();
  const trpc = useTRPC();
  const items = getKeysTableActionItems(keyData, queryClient, trpc);

  return (
    <TableActionPopover items={items}>
      <NavbarActionButton>
        <Gear />
        Settings
      </NavbarActionButton>
    </TableActionPopover>
  );
};
