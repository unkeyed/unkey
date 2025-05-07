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
import { DeleteKey } from "./components/delete-key";
import { UpdateKeyStatus } from "./components/disable-key";
import { EditCredits } from "./components/edit-credits";
import { EditExpiration } from "./components/edit-expiration";
import { EditExternalId } from "./components/edit-external-id";
import { EditKeyName } from "./components/edit-key-name";
import { EditMetadata } from "./components/edit-metadata";
import { EditRatelimits } from "./components/edit-ratelimits";
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
      id: "delete-key",
      label: "Delete key",
      icon: <Trash size="md-regular" />,
      ActionComponent: (props) => <DeleteKey {...props} keyDetails={key} />,
    },
  ];
};
