"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { ENVIRONMENT_KINDS, type EnvironmentKind } from "@/lib/collections/deploy/environments";
import {
  IconCubeOutline18,
  IconDotsOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
  IconTriangleWarningOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, ConfirmPopover, InfoTooltip, ResourceListItem, ResourceListRow } from "@unkey/ui";
import { useRef, useState } from "react";
import { connectionHostVariable } from "../../connection-rules";
import { type ConnectionTargets, describeTarget, targetLabel } from "../../connection-rules";
import type { ConnectionEnvs, MergedConnection } from "../../model/merge";

export function ConnectionsListHeader() {
  return (
    <div className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
      <span>App</span>
      <span>Variable</span>
      <span>Environments</span>
      <span />
    </div>
  );
}

export function ConnectionRow({
  connection,
  envs,
  targets,
  onEdit,
  onRemove,
}: {
  connection: MergedConnection;
  envs: ConnectionEnvs;
  targets: ConnectionTargets;
  onEdit: (targetAppId: string) => void;
  onRemove: (connection: MergedConnection) => void;
}) {
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const [isRemoveConfirmOpen, setIsRemoveConfirmOpen] = useState(false);
  const variable = connectionHostVariable(connection.name);
  const edit = () => onEdit(connection.targetAppId);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <IconPenWriting3Outline18 className="size-3.5" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        edit();
      },
    },
    {
      id: "remove",
      label: "Remove",
      icon: <IconTrashOutline18 className="size-3.5" />,
      onClick: (e) => {
        e.stopPropagation();
        setIsRemoveConfirmOpen(true);
      },
    },
  ];

  return (
    <ResourceListItem>
      <ResourceListRow
        label={`Edit ${connection.targetAppName} connection`}
        onActivate={edit}
        className="group grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 text-left"
      >
        <span className="flex min-w-0 items-center gap-2">
          <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-11" />
          <span className="truncate text-sm font-medium text-gray-12">
            {connection.targetAppName}
          </span>
        </span>
        <code className="truncate font-mono text-xs text-gray-11">{variable}</code>
        <span className="flex items-center gap-4">
          {ENVIRONMENT_KINDS.map((kind) => (
            <EnvBadge key={kind} badge={envBadge(kind, connection, envs, targets)} />
          ))}
        </span>
        <span className="relative flex justify-end">
          <TableActionPopover items={menuItems}>
            <Button
              ref={menuButtonRef}
              variant="outline"
              aria-label={`Actions for ${connection.targetAppName} connection`}
              className="size-5 rounded-sm border-transparent [&_svg]:size-3 group-hover:border-strong"
            >
              <IconDotsOutline18 className="text-gray-11 group-hover:text-gray-12" />
            </Button>
          </TableActionPopover>
        </span>
      </ResourceListRow>
      <ConfirmPopover
        isOpen={isRemoveConfirmOpen}
        onOpenChange={setIsRemoveConfirmOpen}
        onConfirm={() => onRemove(connection)}
        triggerRef={menuButtonRef}
        title={`Remove the ${connection.targetAppName} connection?`}
        description={`New deployments will not include this connection or ${variable} in any environment. Existing deployments keep their saved connection.`}
        confirmButtonText="Remove connection"
        cancelButtonText="Keep connection"
        variant="danger"
        popoverProps={{ align: "end" }}
      />
    </ResourceListItem>
  );
}

type EnvBadgeState = { kind: EnvironmentKind; slug: string } & (
  | { type: "off" }
  | { type: "connected"; target: string; text: string }
  | { type: "unavailable"; target: string; text: string }
);

function envBadge(
  kind: EnvironmentKind,
  connection: MergedConnection,
  envs: ConnectionEnvs,
  targets: ConnectionTargets,
): EnvBadgeState {
  const slug = envs[kind]?.slug ?? kind;
  const row = connection.byEnv[kind];
  if (!row) {
    return { kind, slug, type: "off" };
  }
  const callerEnvironment = { kind };
  const target = targetLabel({
    connection: row,
    callerEnvironment,
    environments: targets.environments,
  });
  const { text, unavailable } = describeTarget({
    connection: row,
    targetName: connection.targetAppName,
    callerEnvironment,
    ...targets,
  });
  return { kind, slug, type: unavailable ? "unavailable" : "connected", target, text };
}

function EnvBadge({ badge }: { badge: EnvBadgeState }) {
  const environment = { slug: badge.slug, kind: badge.kind };
  return match(badge)
    .with({ type: "connected" }, ({ target, text }) => (
      <InfoTooltip
        asChild
        position={{ side: "top" }}
        content={<TargetTip target={target} text={text} />}
      >
        <span className="relative flex">
          <EnvironmentLabel environment={environment} />
        </span>
      </InfoTooltip>
    ))
    .with({ type: "unavailable" }, ({ target, text }) => (
      <InfoTooltip
        asChild
        position={{ side: "top" }}
        content={<TargetTip target={target} text={text} />}
      >
        <span className="relative flex items-center gap-1 rounded-sm bg-warningA-3 px-1 text-warningA-11">
          <IconTriangleWarningOutline12 className="size-3 shrink-0" />
          <EnvironmentLabel environment={environment} className="text-warningA-11" />
        </span>
      </InfoTooltip>
    ))
    .with({ type: "off" }, () => (
      <InfoTooltip
        asChild
        position={{ side: "top" }}
        content="New deployments here do not get this connection. Open it to turn it on."
      >
        <span className="relative flex items-center gap-1.5">
          <EnvironmentLabel environment={environment} className="text-gray-9" />
          <span className="rounded-sm border border-grayA-4 px-1 text-2xs leading-4 text-gray-10">
            Off
          </span>
        </span>
      </InfoTooltip>
    ))
    .exhaustive();
}

function TargetTip({ target, text }: { target: string; text: string }) {
  return (
    <span className="flex max-w-64 flex-col gap-0.5">
      <span className="font-medium">Target: {target}</span>
      <span>{text}</span>
    </span>
  );
}
