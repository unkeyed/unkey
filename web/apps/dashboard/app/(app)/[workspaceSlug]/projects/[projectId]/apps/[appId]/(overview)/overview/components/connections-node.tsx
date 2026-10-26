"use client";

import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { IconLinkOutline18, IconTriangleWarningOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, Button, Skeleton } from "@unkey/ui";
import type { Route } from "next";
import Link from "next/link";
import { Node, NodeHeader, Rows } from "./canvas/nodes";

export function ConnectionsNode({
  query,
  href,
}: {
  query: {
    data: inferRouterOutputs<Router>["deploy"]["deployment"]["listConnections"] | undefined;
    isLoading: boolean;
    isError: boolean;
    isFetching: boolean;
    refetch: () => void;
  };
  href: Route;
}) {
  if (query.isLoading) {
    return <Skeleton aria-label="Loading connections" className="h-28 w-full rounded-lg" />;
  }

  return (
    <Node>
      <NodeHeader
        icon={<IconLinkOutline18 />}
        name="Connections"
        right={
          !query.isError && query.data ? (
            <Badge variant="count">
              {query.data.connections.length}
              {query.data.truncated && "+"}
            </Badge>
          ) : undefined
        }
      />
      <Rows>
        {query.isError ? (
          <span role="alert" className="flex items-center gap-2">
            <IconTriangleWarningOutline18 className="size-3 shrink-0 text-error-9" />
            <span className="text-gray-11">Couldn't load connections</span>
            <Button
              variant="ghost"
              size="sm"
              className="ml-auto"
              onClick={() => query.refetch()}
              loading={query.isFetching}
            >
              Retry
            </Button>
          </span>
        ) : (
          <>
            {query.data?.connections.map((connection) => (
              <span key={connection.id} className="flex min-w-0 items-center gap-2">
                <span className="truncate text-gray-12">
                  {connection.targetAppName ?? "Deleted app"}
                </span>
                <span className="ml-auto min-w-0 truncate text-gray-10">
                  {match(connection)
                    .with({ targetType: "automatic" }, () => "Automatic")
                    .with({ targetType: "environment" }, (target) =>
                      target.targetEnvironmentSlug
                        ? `Environment: ${target.targetEnvironmentSlug}`
                        : "Deleted environment",
                    )
                    .with(
                      { targetType: "deployment" },
                      (target) => `Pinned: ${target.targetDeploymentId}`,
                    )
                    .exhaustive()}
                </span>
              </span>
            ))}
            <span className="text-gray-10">
              {query.data?.connections.length
                ? "Saved in this deployment"
                : "No connections saved in this deployment"}
            </span>
            {query.data?.truncated && <span className="text-gray-10">Showing first 5</span>}
          </>
        )}
      </Rows>
      <Rows>
        <Link
          href={href}
          className="self-start text-gray-11 hover:text-gray-12 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
        >
          View connections
        </Link>
      </Rows>
    </Node>
  );
}
