"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import type { ListView } from "@/lib/loadable/list";
import type { DeploymentConnectionSummary } from "@unkey/api/models/components";
import { IconCubeOutline18, IconLinkOutline18, IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import type { Route } from "next";
import {
  LabeledRow,
  MoreRow,
  NODE_ROW_LIMIT,
  Node,
  NodeError,
  NodeFooter,
  NodeHeader,
  NodeSkeleton,
  Rows,
} from "./canvas/nodes";

type Connection = DeploymentConnectionSummary;

function targetLabel(connection: Connection): string {
  return match(connection)
    .with({ targetType: "automatic" }, () => "Production")
    .with({ targetType: "environment" }, (target) =>
      target.targetEnvironmentSlug
        ? environmentName(target.targetEnvironmentSlug)
        : "Deleted environment",
    )
    .with({ targetType: "deployment" }, () => "Pinned")
    .exhaustive();
}

export function ConnectionsNode({
  view,
  truncated,
  href,
  addHref,
}: {
  view: ListView<Connection>;
  truncated: boolean;
  href: Route;
  addHref: Route;
}) {
  const link = { href, label: "Connections" };
  const footer = (
    <NodeFooter href={addHref} icon={<IconPlusOutline18 />}>
      Add connection
    </NodeFooter>
  );
  return match(view)
    .with({ status: "loading" }, () => <NodeSkeleton label="Loading connections" />)
    .with({ status: "failed" }, ({ retry, retrying }) => (
      <NodeError
        icon={<IconLinkOutline18 />}
        name="Connections"
        link={link}
        onRetry={retry}
        retrying={retrying}
        footer={footer}
      />
    ))
    .with({ status: "empty" }, () => (
      <Node link={link} footer={footer}>
        <NodeHeader icon={<IconLinkOutline18 />} name="Connections" />
        <Rows>
          <span className="text-gray-10">None in this deployment</span>
        </Rows>
      </Node>
    ))
    .with({ status: "ready" }, ({ rows }) => (
      <Node link={link} footer={footer}>
        <NodeHeader icon={<IconLinkOutline18 />} name="Connections" />
        <Rows>
          {rows.slice(0, NODE_ROW_LIMIT).map((connection) => (
            <LabeledRow
              key={connection.id}
              icon={<IconCubeOutline18 />}
              name={connection.targetAppName ?? "Deleted app"}
              label={targetLabel(connection)}
            />
          ))}
          <MoreRow count={rows.length - NODE_ROW_LIMIT} truncated={truncated} href={href} />
        </Rows>
      </Node>
    ))
    .exhaustive();
}
