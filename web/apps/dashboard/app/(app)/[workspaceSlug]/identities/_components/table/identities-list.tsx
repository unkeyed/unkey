"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useIdentities } from "@/lib/identities-query";
import { routes } from "@/lib/navigation/routes";
import { shortenId } from "@/lib/shorten-id";
import { getErrorMessage } from "@/lib/unkey-client";
import type { Identity } from "@unkey/api/models/components";
import {
  Button,
  Empty,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListItem,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import Link from "next/link";
import { IconBookBookmarkOutline18, IconFingerprintOutline18 } from "nucleo-ui-outline-18";
import { parseAsString, useQueryState } from "nuqs";
import { useState } from "react";

const SKELETON_ROWS = 8;

function IdentityActionSkeleton() {
  return <div aria-hidden="true" className="size-8 shrink-0 animate-pulse rounded-md bg-grayA-3" />;
}

const IdentityActions = dynamic(
  () =>
    import("@/components/identities-table/components/identity-table-actions").then(
      (module) => module.IdentityTableActions,
    ),
  { loading: () => <IdentityActionSkeleton /> },
);

function IdentityRow({
  identity,
  href,
}: {
  identity: Identity;
  href: ReturnType<typeof routes.identities.detail>;
}) {
  const rateLimitCount = identity.ratelimits?.length ?? 0;

  return (
    <ResourceListItem className="group flex h-16 items-center gap-3 px-4 py-3 transition-colors hover:bg-grayA-2">
      <Link
        href={href}
        className="absolute inset-0 z-10 rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
        aria-label={`Identity ${identity.externalId}`}
      />
      <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-brandA-3">
        <IconFingerprintOutline18 className="size-4 text-brandA-11" />
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <span className="truncate font-medium text-accent-12 text-sm" title={identity.externalId}>
          {identity.externalId}
        </span>
        <span className="truncate font-mono text-gray-9 text-xs" title={identity.id}>
          {shortenId(identity.id)}
        </span>
      </div>
      <span className="hidden shrink-0 text-gray-10 text-xs sm:block">
        {rateLimitCount} {rateLimitCount === 1 ? "rate limit" : "rate limits"}
      </span>
      <div className="relative z-20">
        <IdentityActions identity={identity} />
      </div>
    </ResourceListItem>
  );
}

function IdentitiesSkeleton() {
  return (
    <ResourceListContent aria-busy="true">
      <output className="sr-only">Loading identities...</output>
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: SKELETON_ROWS }).map((_, index) => (
          <ResourceListItem
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            className="flex h-16 items-center gap-3 px-4 py-3"
          >
            <div className="size-8 shrink-0 animate-pulse rounded-md bg-grayA-3" />
            <div className="flex min-w-0 flex-1 flex-col gap-2">
              <div className="h-3 w-40 animate-pulse rounded-sm bg-grayA-3" />
              <div className="h-2.5 w-20 animate-pulse rounded-sm bg-grayA-3" />
            </div>
            <div className="hidden h-3 w-16 animate-pulse rounded-sm bg-grayA-3 sm:block" />
            <IdentityActionSkeleton />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

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
  const workspace = useWorkspaceNavigation();
  const [nextPageError, setNextPageError] = useState<unknown>();
  const {
    identities: identityRows,
    isLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isError,
    error,
    refetch,
  } = useIdentities({ search });
  const retry = () => {
    refetch().catch((error: unknown) => {
      console.error("Failed to retry identities query", error);
    });
  };
  const loadMore = async () => {
    try {
      await fetchNextPage({ throwOnError: true });
      setNextPageError(undefined);
    } catch (error) {
      setNextPageError(error);
    }
  };

  if (isLoading && identityRows.length === 0) {
    return <IdentitiesSkeleton />;
  }

  if (isError && identityRows.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex flex-col items-center gap-3 px-4 py-16 text-center">
          <span role="alert" className="text-gray-11 text-sm">
            {getErrorMessage(error, "We couldn't load identities.")}
          </span>
          <Button variant="outline" onClick={retry}>
            Retry
          </Button>
        </div>
      </ResourceListContent>
    );
  }

  if (identityRows.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center px-4 py-16">
          <Empty className="w-[400px] items-start">
            <Empty.Icon className="w-auto" />
            <Empty.Title>No Identities Found</Empty.Title>
            <Empty.Description className="text-left">
              {search
                ? "Try adjusting your search query"
                : "There are no identities yet. Create your first identity to get started."}
            </Empty.Description>
            {!search && (
              <Empty.Actions className="mt-4 justify-start">
                <Button
                  size="md"
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
              </Empty.Actions>
            )}
          </Empty>
        </div>
      </ResourceListContent>
    );
  }

  return (
    <ResourceListContent>
      <ResourceListBody aria-label="Identities">
        {identityRows.map((identity) => (
          <IdentityRow
            key={identity.id}
            identity={identity}
            href={routes.identities.detail({
              workspaceSlug: workspace.slug,
              identityId: identity.id,
            })}
          />
        ))}
      </ResourceListBody>
      {nextPageError ? (
        <ResourceListFooter>
          <div className="flex items-center gap-3">
            <span role="alert" className="text-error-11 text-sm">
              {getErrorMessage(nextPageError, "We couldn't load more identities.")}
            </span>
            <Button
              variant="outline"
              disabled={isFetchingNextPage}
              loading={isFetchingNextPage}
              onClick={loadMore}
            >
              Retry
            </Button>
          </div>
        </ResourceListFooter>
      ) : isError ? (
        <ResourceListFooter>
          <div className="flex items-center gap-3">
            <span role="alert" className="text-error-11 text-sm">
              {getErrorMessage(error, "We couldn't refresh identities.")}
            </span>
            <Button variant="outline" onClick={retry}>
              Retry
            </Button>
          </div>
        </ResourceListFooter>
      ) : hasNextPage ? (
        <ResourceListFooter className="py-6">
          <Button
            size="md"
            variant="outline"
            disabled={isFetchingNextPage}
            loading={isFetchingNextPage}
            onClick={loadMore}
          >
            Load more
          </Button>
        </ResourceListFooter>
      ) : null}
    </ResourceListContent>
  );
}
