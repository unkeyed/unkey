"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Bolt, BoltSlash, Trash } from "@unkey/icons";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useState } from "react";

export type DrainStatus = "enabled" | "disabled" | "paused_by_failure";

export function StatusBadge({ status }: { status: DrainStatus }) {
  const label = status === "paused_by_failure" ? "Paused by failure" : status;
  return (
    <span className="flex items-center gap-2 whitespace-nowrap text-[13px] capitalize text-accent-12">
      <span
        className={cn(
          "size-2 shrink-0 rounded-full",
          status === "enabled"
            ? "bg-success-9"
            : status === "disabled"
              ? "bg-gray-9"
              : "bg-error-9",
        )}
      />
      {label}
    </span>
  );
}

export function DrainActions({
  drain,
}: {
  drain: {
    id: string;
    name: string;
    status: DrainStatus;
  };
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const utils = trpc.useUtils();
  const update = trpc.logdrain.update.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      toast.success("Log drain updated");
    },
    onError: (error) => toast.error(error.message),
  });
  const remove = trpc.logdrain.delete.useMutation({
    onSuccess: () => {
      utils.logdrain.list.invalidate();
      setConfirmDelete(false);
      toast.success("Log drain deleted");
    },
    onError: (error) => toast.error(error.message),
  });
  const nextStatus = drain.status === "enabled" ? "disabled" : "enabled";
  const menuItems: MenuItem[] = [
    {
      id: "toggle",
      label:
        drain.status === "paused_by_failure"
          ? "Resume log drain"
          : drain.status === "enabled"
            ? "Disable log drain"
            : "Enable log drain",
      icon:
        drain.status === "enabled" ? (
          <BoltSlash iconSize="md-medium" />
        ) : (
          <Bolt iconSize="md-medium" />
        ),
      disabled: update.isLoading,
      divider: true,
      onClick: () => update.mutate({ id: drain.id, status: nextStatus }),
    },
    {
      id: "delete",
      label: "Delete log drain",
      icon: <Trash iconSize="md-medium" className="text-error-11" />,
      className: "text-error-11 hover:bg-error-3 focus:bg-error-3",
      onClick: () => setConfirmDelete(true),
    },
  ];

  return (
    <div
      onClick={(event) => event.stopPropagation()}
      onKeyDown={(event) => event.stopPropagation()}
    >
      <TableActionPopover items={menuItems} />
      <DialogContainer
        isOpen={confirmDelete}
        onOpenChange={setConfirmDelete}
        title={`Delete ${drain.name}?`}
        subTitle="This stops delivery immediately and cannot be undone."
        footer={
          <div className="flex w-full gap-2">
            <Button className="flex-1" variant="outline" onClick={() => setConfirmDelete(false)}>
              Cancel
            </Button>
            <Button
              className="flex-1"
              variant="primary"
              loading={remove.isLoading}
              onClick={() => remove.mutate({ id: drain.id })}
            >
              Delete log drain
            </Button>
          </div>
        }
      >
        <p className="text-sm text-gray-10">
          Existing delivery history is retained, but this destination will no longer receive audit
          logs.
        </p>
      </DialogContainer>
    </div>
  );
}
