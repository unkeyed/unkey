"use client";

import type { inferRouterOutputs } from "@trpc/server";
import { match } from "@unkey/match";
import { Button, DialogContainer, toast } from "@unkey/ui";
import {
  IconBoltOutline18,
  IconBoltSlashOutline18,
  IconEarthOutline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import { useState } from "react";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import { cn } from "@/lib/utils";
import { AxiomLogo } from "./axiom-logo";

export type DrainKind = inferRouterOutputs<Router>["logdrain"]["list"][number]["kind"];
export type DrainStatus = inferRouterOutputs<Router>["logdrain"]["list"][number]["status"];

export function SinkType({ kind }: { kind: DrainKind }) {
  return match(kind)
    .with("http", () => (
      <>
        <span className="flex size-5 shrink-0 items-center justify-center rounded-sm bg-grayA-3 text-gray-11">
          <IconEarthOutline18 className="size-3" />
        </span>
        <span>HTTP</span>
      </>
    ))
    .with("axiom", () => (
      <>
        <span className="flex size-5 shrink-0 items-center justify-center rounded-sm bg-grayA-3 text-gray-11">
          <AxiomLogo className="size-3.5" />
        </span>
        <span>Axiom</span>
      </>
    ))
    .exhaustive();
}

export function StatusBadge({ status }: { status: DrainStatus }) {
  const label =
    status === "running"
      ? "Running"
      : status === "paused_by_user"
        ? "Paused"
        : "Paused after failures";
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
          ? "Resume delivery"
          : drain.status === "running"
            ? "Pause delivery"
            : "Resume delivery",
      icon:
        drain.status === "running" ? (
          <IconBoltSlashOutline18 className="size-4" />
        ) : (
          <IconBoltOutline18 className="size-4" />
        ),
      disabled: update.isLoading,
      divider: true,
      onClick: () => update.mutate({ id: drain.id, status: nextStatus }),
    },
    {
      id: "delete",
      label: "Delete log drain",
      icon: <IconTrashOutline18 className="size-4 text-error-11" />,
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
        subTitle="Unkey will stop all future deliveries and delete this log drain."
        footer={
          <div className="flex w-full gap-2">
            <Button className="flex-1" variant="outline" onClick={() => setConfirmDelete(false)}>
              Keep log drain
            </Button>
            <Button
              className="flex-1"
              variant="primary"
              color="danger"
              loading={remove.isLoading}
              onClick={() => remove.mutate({ id: drain.id })}
            >
              Delete log drain
            </Button>
          </div>
        }
      >
        <p className="text-sm text-gray-10">You cannot undo this action.</p>
      </DialogContainer>
    </div>
  );
}
