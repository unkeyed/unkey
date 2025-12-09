import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { Clone } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

export const IdentityTableActions = ({ identity }: { identity: Identity }) => {
  const menuItems: MenuItem[] = useMemo(
    () => [
      {
        id: "copy-identity-id",
        label: "Copy identity ID",
        icon: <Clone iconSize="md-medium" />,
        onClick: () => {
          navigator.clipboard
            .writeText(identity.id)
            .then(() => {
              toast.success("Identity ID copied to clipboard");
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
      },
      {
        id: "copy-external-id",
        label: "Copy external ID",
        icon: <Clone iconSize="md-medium" />,
        onClick: () => {
          navigator.clipboard
            .writeText(identity.externalId)
            .then(() => {
              toast.success("External ID copied to clipboard");
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
      },
    ],
    [identity.id, identity.externalId],
  );

  return <TableActionPopover items={menuItems} />;
};
