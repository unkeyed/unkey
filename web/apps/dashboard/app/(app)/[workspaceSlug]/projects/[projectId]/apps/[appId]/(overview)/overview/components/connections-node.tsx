"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import type { ListView } from "@/lib/loadable/list";
import type { DeploymentConnectionSummary } from "@unkey/api/models/components";
import { IconLinkOutline18, IconPlusOutline12 } from "@unkey/icons";
import { match } from "@unkey/match";
import type { Route } from "next";
import {
  LabeledRow,
  LinkRow,
  MoreRow,
  NODE_ROW_LIMIT,
  Node,
  NodeError,
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
  const addRow = (
    <LinkRow href={addHref} icon={<IconPlusOutline12 className="size-2.5" />} className="mt-1">
      Add connection
    </LinkRow>
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
      >
        {addRow}
      </NodeError>
    ))
    .with({ status: "empty" }, () => (
      <Node link={link}>
        <NodeHeader icon={<IconLinkOutline18 />} name="Connections" count={0} />
        <Rows>
          <span className="text-gray-10">None in this deployment</span>
          {addRow}
        </Rows>
      </Node>
    ))
    .with({ status: "ready" }, ({ rows }) => (
      <Node link={link}>
        <NodeHeader
          icon={<IconLinkOutline18 />}
          name="Connections"
          count={`${rows.length}${truncated ? "+" : ""}`}
        />
        <Rows>
          {rows.slice(0, NODE_ROW_LIMIT).map((connection) => (
            <LabeledRow
              key={connection.id}
              name={connection.targetAppName ?? "Deleted app"}
              label={targetLabel(connection)}
            />
          ))}
          <MoreRow count={rows.length - NODE_ROW_LIMIT} truncated={truncated} href={href} />
          {addRow}
        </Rows>
      </Node>
    ))
    .exhaustive();
}
