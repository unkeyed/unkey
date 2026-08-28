"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { cn } from "@/lib/utils";
import type { inferRouterOutputs } from "@trpc/server";
import { Bolt, BoltSlash, Trash } from "@unkey/icons";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useState } from "react";

export type DrainStatus = inferRouterOutputs<Router>["logdrain"]["list"][number]["status"];

export function StatusBadge({ status }: { status: DrainStatus }) {
  const label =
    status === "running"
      ? "Running"
      : status === "paused_by_user"
        ? "Paused by user"
        : "Paused by failure";
  return (
    <span className="flex items-center gap-2 whitespace-nowrap text-[13px] text-accent-12">
      <span
        className={cn(
          "size-2 shrink-0 rounded-full",
          status === "running"
            ? "bg-success-9"
            : status === "paused_by_user"
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
  const nextStatus = drain.status === "running" ? "paused_by_user" : "running";
  const menuItems: MenuItem[] = [
    {
      id: "toggle",
      label:
        drain.status === "paused_by_failure"
          ? "Resume log drain"
          : drain.status === "running"
            ? "Pause log drain"
            : "Resume log drain",
      icon:
        drain.status === "running" ? (
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
