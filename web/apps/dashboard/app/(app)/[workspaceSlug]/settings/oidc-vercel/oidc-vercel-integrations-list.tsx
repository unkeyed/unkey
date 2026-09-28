"use client";

import type { Router } from "@/lib/trpc/routers";
import type { UseTRPCQueryResult } from "@trpc/react-query/shared";
import type { inferRouterOutputs } from "@trpc/server";
import { IconCircleCheckOutline12, IconCircleXmarkOutline12 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  Skeleton,
} from "@unkey/ui";
import { useState } from "react";
import { OIDCVercelIntegrationPanel } from "./oidc-vercel-integration-panel";
import { VercelMark } from "./vercel-mark";

type Integration = inferRouterOutputs<Router>["oidcVercel"]["list"][number];
type IntegrationsQuery = UseTRPCQueryResult<Integration[], unknown>;

export function OIDCVercelIntegrationsList({
  query,
  onCreate,
}: {
  query: IntegrationsQuery;
  onCreate: () => void;
}) {
  const [selected, setSelected] = useState<Integration | null>(null);

  if (query.isLoading) {
    return (
      <ResourceListContent aria-busy="true">
        <ResourceListBody aria-hidden="true">
          {Array.from({ length: 3 }).map((_, index) => (
            <ResourceListItem
              // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static
              key={index}
              className="flex items-center gap-3 px-4 py-3"
            >
              <Skeleton className="size-8 rounded-xl" />
              <Skeleton className="h-4 w-40" />
              <Skeleton className="ml-auto h-4 w-28" />
            </ResourceListItem>
          ))}
        </ResourceListBody>
      </ResourceListContent>
    );
  }

  if (query.isError) {
    return (
      <ResourceListContent>
        <div className="flex flex-col items-center gap-3 px-4 py-16 text-center">
          <span role="alert" className="text-sm text-gray-11">
            We couldn't load Vercel OIDC integrations.
          </span>
          <Button variant="outline" onClick={() => query.refetch()}>
            Retry
          </Button>
        </div>
      </ResourceListContent>
    );
  }

  if (!query.data.length) {
    return (
      <EmptyState>
        <EmptyStateHeader>
          <EmptyStateTitle>Connect Vercel without a root key</EmptyStateTitle>
          <EmptyStateDescription>
            Trust one Vercel project and environment, then assign the permissions its deployments
            can use.
          </EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button variant="primary" onClick={onCreate}>
            Add Integration
          </Button>
        </EmptyStateActions>
      </EmptyState>
    );
  }

  return (
    <>
      <ResourceListContent aria-live="polite">
        <ResourceListBody aria-label="Vercel OIDC integrations">
          {query.data.map((integration) => (
            <ResourceListItem key={integration.id}>
              <button
                type="button"
                className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-grayA-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
                onClick={() => setSelected(integration)}
              >
                <span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-grayA-3 text-gray-12">
                  <VercelMark className="size-4" />
                </span>
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-sm font-medium text-gray-12">
                    {integration.name}
                  </span>
                  <span className="truncate text-xs text-gray-9">
                    {integration.vercelTeamSlug}/{integration.vercelProjectName} ·{" "}
                    {integration.vercelEnvironment}
                  </span>
                </span>
                <span className="flex items-center gap-1.5 text-xs text-gray-10">
                  {integration.enabled ? (
                    <IconCircleCheckOutline12 className="text-success-11" />
                  ) : (
                    <IconCircleXmarkOutline12 className="text-gray-9" />
                  )}
                  {integration.enabled ? "Enabled" : "Disabled"}
                </span>
                <span className="w-24 text-right text-xs text-gray-9">
                  {integration.permissions.length} permission
                  {integration.permissions.length === 1 ? "" : "s"}
                </span>
              </button>
            </ResourceListItem>
          ))}
        </ResourceListBody>
      </ResourceListContent>
      <OIDCVercelIntegrationPanel
        key={selected?.id ?? "closed"}
        isOpen={selected !== null}
        onClose={() => setSelected(null)}
        integration={selected ?? undefined}
      />
    </>
  );
}
