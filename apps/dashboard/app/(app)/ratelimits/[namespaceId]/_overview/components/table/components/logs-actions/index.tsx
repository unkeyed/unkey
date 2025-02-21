import { DeleteDialog } from "@/app/(app)/ratelimits/[namespaceId]/_components/delete-dialog";
import { IdentifierDialog } from "@/app/(app)/ratelimits/[namespaceId]/_components/identifier-dialog";
import {
  type MenuItem,
  TableActionPopover,
} from "@/app/(app)/ratelimits/[namespaceId]/_components/table-action-popover";
import type { OverrideDetails } from "@/app/(app)/ratelimits/[namespaceId]/types";
import { toast } from "@/components/ui/toaster";
import { Clone, Layers3, PenWriting3, Trash } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { useState } from "react";
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
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const router = useRouter();
  const { filters } = useFilters();

  // Keep the time params logic
  const timeFilters = filters.filter((f) => ["startTime", "endTime", "since"].includes(f.field));

  const getTimeParams = () => {
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

  const items: MenuItem[] = [
    {
      id: "logs",
      label: "Go to logs",
      icon: <Layers3 size="md-regular" />,
      onClick(e) {
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
        navigator.clipboard.writeText(identifier);
        toast.success("Copied to clipboard", {
          description: identifier,
        });
      },
    },
    {
      id: "override",
      label: "Override Identifier",
      icon: <PenWriting3 size="md-regular" />,
      className: "text-orange-11 hover:bg-orange-2 focus:bg-orange-3",
      onClick: (e) => {
        e.stopPropagation();
        setIsModalOpen(true);
      },
    },
    {
      id: "delete",
      label: "Delete Override",
      icon: <Trash size="md-regular" />,
      className: "text-error-11 hover:bg-error-3 focus:bg-error-3",
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteModalOpen(true);
      },
    },
  ];

  return (
    <>
      <TableActionPopover items={items} />
      <IdentifierDialog
        overrideDetails={overrideDetails}
        namespaceId={namespaceId}
        identifier={identifier}
        isModalOpen={isModalOpen}
        onOpenChange={setIsModalOpen}
      />
      {overrideDetails?.overrideId && (
        <DeleteDialog
          isModalOpen={isDeleteModalOpen}
          onOpenChange={setIsDeleteModalOpen}
          overrideId={overrideDetails.overrideId}
          identifier={identifier}
        />
      )}
    </>
  );
};
