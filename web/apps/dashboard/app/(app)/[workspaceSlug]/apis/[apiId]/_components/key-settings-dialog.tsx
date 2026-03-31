"use client";

import { getKeysTableActionItems } from "@/components/api-keys-table/components/actions/keys-table-action.popover.constants";
import { TableActionPopover } from "@/components/logs/table-action.popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { Gear } from "@unkey/icons";

interface KeySettingsDialogProps {
  keyData: KeyDetails;
}

export const KeySettingsDialog = ({ keyData }: KeySettingsDialogProps) => {
  const trpcUtils = trpc.useUtils();
  const items = getKeysTableActionItems(keyData, trpcUtils);

  return (
    <TableActionPopover items={items}>
      <NavbarActionButton>
        <Gear />
        Settings
      </NavbarActionButton>
    </TableActionPopover>
  );
};
