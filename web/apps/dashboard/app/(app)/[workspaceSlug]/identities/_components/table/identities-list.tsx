"use client";

import { ListNoMatch, ListTableRow, ListTableView, listTable } from "@/components/list-views";
import { LoadError } from "@/components/load-error";
import { useProjectScope } from "@/hooks/use-project-scope";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useIdentities } from "@/lib/identities-query";
import { searchedListOf, useInfiniteList } from "@/lib/loadable/list";
import { routes } from "@/lib/navigation/routes";
import type { Identity } from "@unkey/api/models/components";
import { IconBookBookmarkOutline18, IconFingerprintOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceListContent,
  Skeleton,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import Link from "next/link";
import { parseAsString, useQueryState } from "nuqs";

const IdentityActions = dynamic(
  () =>
    import("@/components/identities-table/components/identity-table-actions").then(
      (module) => module.IdentityTableActions,
    ),
  { loading: () => <Skeleton className="size-5 rounded-sm" /> },
);

const IDENTITIES_TABLE = listTable({
  minWidth: 640,
  columns: [
    { id: "identity", header: "Identity", width: "minmax(0,1.6fr)", skeleton: "title" },
    { id: "id", header: "ID", width: "minmax(0,1.4fr)", skeleton: "text" },
    { id: "ratelimits", header: "Rate limits", width: "minmax(0,0.8fr)", skeleton: "text" },
    { id: "actions", header: null, width: "32px", skeleton: "none" },
  ],
});

export function IdentitiesList() {
  const [search] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );
  const normalizedSearch = search.trim();

  return <IdentityResults key={normalizedSearch} search={normalizedSearch} />;
}

function IdentityResults({ search }: { search: string }) {
  const view = searchedListOf(
    useInfiniteList(useIdentities({ search }), (page) => page.identities),
    search,
  );

  return match(view)
    .with({ status: "failed" }, ({ error, retry }) => (
      <ResourceListContent>
        <LoadError title="Could not load identities" error={error} frame="none" onRetry={retry} />
      </ResourceListContent>
    ))
    .with({ status: "empty" }, () => <IdentitiesEmpty />)
    .with({ status: "noMatch" }, () => <ListNoMatch noun="identities" search={search} />)
    .with({ status: "loading" }, { status: "ready" }, (list) => (
      <ListTableView
        list={list}
        noun="identities"
        table={IDENTITIES_TABLE}
        rowKey={(identity) => identity.id}
        renderRow={(identity) => <IdentityRow identity={identity} />}
      />
    ))
    .exhaustive();
}

function IdentityRow({ identity }: { identity: Identity }) {
  const workspace = useWorkspaceNavigation();
  const scope = useProjectScope();
  const rateLimitCount = identity.ratelimits?.length ?? 0;

  return (
    <ListTableRow
      table={IDENTITIES_TABLE}
      label={`View identity ${identity.externalId}`}
      link={
        <Link
          href={routes.identities.detail({
            workspaceSlug: workspace.slug,
            ...scope,
            identityId: identity.id,
          })}
        />
      }
      cells={{
        identity: (
          <span className="flex min-w-0 items-center gap-2.5">
            <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised">
              <IconFingerprintOutline18 className="size-3 text-gray-12" />
            </span>
            <span data-sentry-mask className="truncate font-medium text-sm">
              {identity.externalId}
            </span>
          </span>
        ),
        id: <span className="truncate font-mono text-gray-11">{identity.id}</span>,
        ratelimits: (
          <span className="text-gray-11">
            {rateLimitCount}
            <span className="sr-only">{rateLimitCount === 1 ? " rate limit" : " rate limits"}</span>
          </span>
        ),
        actions: (
          <span className="relative z-10 flex justify-end">
            <IdentityActions identity={identity} />
          </span>
        ),
      }}
    />
  );
}

function IdentitiesEmpty() {
  return (
    <ResourceListContent>
      <EmptyState frame="none">
        <EmptyStateIcon>
          <IconFingerprintOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>No Identities Found</EmptyStateTitle>
          <EmptyStateDescription>
            There are no identities yet. Create your first identity to get started.
          </EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button
            size="sm"
            variant="outline"
            render={
              <Link
                href="https://www.unkey.com/docs/platform/identities/overview"
                target="_blank"
                rel="noopener noreferrer"
              />
            }
          >
            <IconBookBookmarkOutline18 />
            Learn about Identities
          </Button>
        </EmptyStateActions>
      </EmptyState>
    </ResourceListContent>
  );
}
