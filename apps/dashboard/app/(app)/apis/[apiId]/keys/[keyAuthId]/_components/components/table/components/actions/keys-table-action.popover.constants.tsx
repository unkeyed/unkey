import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import {
  ArrowOppositeDirectionY,
  Ban,
  CalendarClock,
  ChartPie,
  Check,
  Clone,
  Code,
  Gauge,
  PenWriting3,
  Tag,
  Trash,
} from "@unkey/icons";
import { DeleteKey } from "./components/delete-key";
import { UpdateKeyStatus } from "./components/disable-key";
import { EditCredits } from "./components/edit-credits";
import { EditExpiration } from "./components/edit-expiration";
import { EditExternalId } from "./components/edit-external-id";
import { EditKeyName } from "./components/edit-key-name";
import { EditMetadata } from "./components/edit-metadata";
import { EditRatelimits } from "./components/edit-ratelimits";
import { KeyRbacDialog } from "./components/edit-rbac";
import { KeysTableActionPopover, type MenuItem } from "./keys-table-action.popover";

type KeysTableActionsProps = {
  keyData: KeyDetails;
};

export const KeysTableActions = ({ keyData: key }: KeysTableActionsProps) => {
  const trpcUtils = trpc.useUtils();

  const keysTableActionItems: MenuItem[] = [
    {
      id: "override",
      label: "Edit key name...",
      icon: <PenWriting3 size="md-regular" />,
      ActionComponent: (props) => <EditKeyName {...props} keyDetails={key} />,
    },
    {
      id: "copy",
      label: "Copy key ID",
      className: "mt-1",
      icon: <Clone size="md-regular" />,
      onClick: () => {
        navigator.clipboard
          .writeText(key.id)
          .then(() => {
            toast.success("Key ID copied to clipboard");
          })
          .catch((error) => {
            console.error("Failed to copy to clipboard:", error);
            toast.error("Failed to copy to clipboard");
          });
      },
      divider: true,
    },
    {
      id: "edit-external-id",
      label: "Edit External ID...",
      icon: <ArrowOppositeDirectionY size="md-regular" />,
      ActionComponent: (props) => <EditExternalId {...props} keyDetails={key} />,
      divider: true,
    },
    {
      id: key.enabled ? "disable-key" : "enable-key",
      label: key.enabled ? "Disable Key..." : "Enable Key...",
      icon: key.enabled ? <Ban size="md-regular" /> : <Check size="md-regular" />,
      ActionComponent: (props) => <UpdateKeyStatus {...props} keyDetails={key} />,
      divider: true,
    },
    {
      id: "edit-credits",
      label: "Edit credits...",
      icon: <ChartPie size="md-regular" />,
      ActionComponent: (props) => <EditCredits {...props} keyDetails={key} />,
    },
    {
      id: "edit-ratelimit",
      label: "Edit ratelimit...",
      icon: <Gauge size="md-regular" />,
      ActionComponent: (props) => <EditRatelimits {...props} keyDetails={key} />,
    },
    {
      id: "edit-expiration",
      label: "Edit expiration...",
      icon: <CalendarClock size="md-regular" />,
      ActionComponent: (props) => <EditExpiration {...props} keyDetails={key} />,
    },
    {
      id: "edit-metadata",
      label: "Edit metadata...",
      icon: <Code size="md-regular" />,
      ActionComponent: (props) => <EditMetadata {...props} keyDetails={key} />,
      divider: true,
    },
    {
      id: "edit-rbac",
      label: "Manage roles and permissions...",
      icon: <Tag size="md-regular" />,
      ActionComponent: (props) => (
        <KeyRbacDialog
          {...props}
          existingKey={{
            id: key.id,
            // Those permissionId and roleIds are being derived from prefetched tRPC call
            permissionIds: [],
            roleIds: [],
            name: key.name ?? undefined,
          }}
        />
      ),
      prefetch: async () => {
        await trpcUtils.key.connectedRolesAndPerms.prefetch({
          keyId: key.id,
        });
      },
      divider: true,
    },
    {
      id: "delete-key",
      label: "Delete key",
      icon: <Trash size="md-regular" />,
      ActionComponent: (props) => <DeleteKey {...props} keyDetails={key} />,
    },
  ];

  return <KeysTableActionPopover items={keysTableActionItems} />;
};
