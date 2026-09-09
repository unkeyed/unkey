"use client";

import { Loading, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import {
  IconCloneOutline18,
  IconLayers3Outline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import { Suspense } from "react";
import { DeleteDialog } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_components/delete-dialog";
import { IdentifierDialog } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_components/identifier-dialog";
import type { OverrideDetails } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/types";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
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
  const workspace = useWorkspaceNavigation();

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
        icon: <IconLayers3Outline18 className="size-4" />,
        onClick: (e) => {
          e.stopPropagation();
          router.push(
            `${routes.ratelimits.logs({ workspaceSlug: workspace.slug, namespaceId })}?${getTimeParams()}`,
          );
        },
      },
      {
        id: "copy",
        label: "Copy identifier",
        icon: <IconCloneOutline18 className="size-4" />,
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
        icon: <IconPenWriting3Outline18 className="size-4 text-orange-11" />,
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
        icon: <IconTrashOutline18 className="size-4 text-error-10" />,
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

  return (
    <Suspense fallback={<Loading type="spinner" />}>
      <TableActionPopover items={menuItems} />
    </Suspense>
  );
};
