"use client";

import { DeleteDialog } from "@/app/(app)/ratelimits/[namespaceId]/_components/delete-dialog";

import { IdentifierDialog } from "@/app/(app)/ratelimits/[namespaceId]/_components/identifier-dialog";
import type { OverrideDetails } from "@/app/(app)/ratelimits/[namespaceId]/types";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Clone, Layers3, PenWriting3, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsTableAction = ({
  identifier,
  namespaceId,
  overrideDetails,
}: {
  identifier: string;
  namespaceId: string;
  overrideDetails?: OverrideDetails | null;
}) => {
  const router = useRouter();
  const { filters } = useFilters();

  const getTimeParams = () => {
    const timeFilters = filters.filter((f) => ["startTime", "endTime", "since"].includes(f.field));
    const params = new URLSearchParams({
      identifiers: `contains:${identifier}`,
    });

    const timeMap = {
      startTime: timeFilters.find((f) => f.field === "startTime")?.value,
      endTime: timeFilters.find((f) => f.field === "endTime")?.value,
      since: timeFilters.find((f) => f.field === "since")?.value,
    };

    Object.entries(timeMap).forEach(([key, value]) => {
      if (value) {
        params.append(key, value.toString());
      }
    });

    return params.toString();
  };

  const getLogsTableActionItems = (): MenuItem[] => {
    return [
      {
        id: "logs",
        label: "Go to logs",
        icon: <Layers3 size="md-regular" />,
        onClick: (e) => {
          e.stopPropagation();
          router.push(`/ratelimits/${namespaceId}/logs?${getTimeParams()}`);
        },
      },
      {
        id: "copy",
        label: "Copy identifier",
        icon: <Clone size="md-regular" />,
        onClick: (e) => {
          e.stopPropagation();
          navigator.clipboard
            .writeText(identifier)
            .then(() => {
              toast.success("Copied to clipboard", {
                description: identifier,
              });
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
      },
      {
        id: "override",
        label: overrideDetails ? "Update Override" : "Override Identifier",
        icon: <PenWriting3 size="md-regular" className="text-orange-11" />,
        className: "text-orange-11 hover:bg-orange-2 focus:bg-orange-3",
        ActionComponent: (props) => (
          <IdentifierDialog
            overrideDetails={overrideDetails}
            namespaceId={namespaceId}
            identifier={identifier}
            isModalOpen={props.isOpen}
            onOpenChange={(open) => !open && props.onClose()}
          />
        ),
        divider: true,
      },
      {
        id: "delete",
        label: "Delete Override",
        icon: <Trash size="md-regular" className="text-error-10" />,
        className: overrideDetails?.overrideId
          ? "text-error-10 hover:bg-error-3 focus:bg-error-3"
          : "text-error-10 cursor-not-allowed bg-error-3",
        disabled: !overrideDetails?.overrideId,
        ActionComponent: (props) =>
          overrideDetails?.overrideId ? (
            <DeleteDialog
              isModalOpen={props.isOpen}
              onOpenChange={(open) => !open && props.onClose()}
              overrideId={overrideDetails.overrideId}
              identifier={identifier}
            />
          ) : undefined,
      },
    ];
  };

  const menuItems = getLogsTableActionItems();

  return <TableActionPopover items={menuItems} />;
};
