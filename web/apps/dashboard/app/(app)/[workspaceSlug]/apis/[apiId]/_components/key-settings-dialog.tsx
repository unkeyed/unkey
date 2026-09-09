"use client";

import { getKeysTableActionItems } from "@/components/api-keys-table/components/actions/keys-table-action.popover.constants";
import { TableActionPopover } from "@/components/logs/table-action.popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { IconGearOutline18 } from "nucleo-ui-outline-18";

interface KeySettingsDialogProps {
  keyData: KeyDetails;
  apiId?: string;
  keyspaceId?: string | null;
}

export const KeySettingsDialog = ({ keyData, apiId, keyspaceId }: KeySettingsDialogProps) => {
  const trpcUtils = trpc.useUtils();
  const items = getKeysTableActionItems(keyData, trpcUtils, { apiId, keyspaceId });

  return (
    <TableActionPopover items={items}>
      <NavbarActionButton>
        <IconGearOutline18 />
        Settings
      </NavbarActionButton>
    </TableActionPopover>
  );
};
