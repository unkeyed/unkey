import { toast } from "@/components/ui/toaster";
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
  Trash,
} from "@unkey/icons";
import { UpdateKeyStatus } from "./components/disable-key";
import { EditKeyName } from "./components/edit-key-name";
import { EditRemainingUses } from "./components/edit-remaining-uses";
import type { MenuItem } from "./keys-table-action.popover";

export const getKeysTableActionItems = (key: KeyDetails): MenuItem[] => {
  return [
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
      id: "edit-owner-id",
      label: "Edit Owner ID...",
      icon: <ArrowOppositeDirectionY size="md-regular" />,
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
      id: "edit-remaining-uses",
      label: "Edit remaining uses...",
      icon: <ChartPie size="md-regular" />,
      ActionComponent: (props) => <EditRemainingUses {...props} keyDetails={key} />,
    },
    {
      id: "edit-ratelimit",
      label: "Edit ratelimit...",
      icon: <Gauge size="md-regular" />,
      onClick: () => {},
    },
    {
      id: "edit-expiration",
      label: "Edit expiration...",
      icon: <CalendarClock size="md-regular" />,
      onClick: () => {},
    },
    {
      id: "edit-metadata",
      label: "Edit metadata...",
      icon: <Code size="md-regular" />,
      onClick: () => {},
      divider: true,
    },
    {
      id: "delete-key",
      label: "Delete key",
      icon: <Trash size="md-regular" />,
      onClick: () => {},
    },
  ];
};
