"use client";

import { PageContent } from "@/components/page-content";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import { Loader2 } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { SearchField } from "./filter";
import { useIdentities } from "./hooks/use-identities";
import { Navigation } from "./navigation";
import { Row } from "./row";

export function IdentitiesTanStack() {
  const [search] = useQueryState("search", parseAsString.withDefault(""));

  // Get current user to determine workspace
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships } = trpc.user.listMemberships.useQuery(user?.id as string, {
    enabled: !!user,
  });

  const currentWorkspace = memberships?.data?.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  // Use TanStack DB hook for identities
  const { identities, isLoading, error, hasMore, nextCursor } = useIdentities({
    workspaceId: currentWorkspace?.organization.id || "",
    search: search || undefined,
    enabled: !!currentWorkspace?.organization.id,
  });

  if (error) {
    return (
      <div>
        <Navigation />
        <PageContent>
          <div className="flex items-center justify-center h-32">
            <p className="text-content-subtle">Failed to load identities</p>
          </div>
        </PageContent>
      </div>
    );
  }

  return (
    <div>
      <Navigation />
      <PageContent>
        <div className="flex justify-between items-center mb-6">
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-semibold">Identities</h1>
            <span className="text-xs bg-primary/20 text-primary px-2 py-1 rounded">
              TanStack DB
            </span>
          </div>
        </div>

        <SearchField />

        <div className="flex flex-col gap-8 mb-20 mt-8">
          {isLoading ? (
            <Empty>
              <Empty.Title>
                <Loader2 className="w-4 h-4 animate-spin" />
              </Empty.Title>
            </Empty>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>External ID</TableHead>
                    <TableHead>Meta</TableHead>
                    <TableHead>Keys</TableHead>
                    <TableHead>Ratelimits</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {identities.map((identity) => {
                    return (
                      <Row
                        key={identity.id}
                        identity={{
                          id: identity.id,
                          externalId: identity.externalId,
                          meta: identity.meta ?? undefined,
                          // Note: TanStack DB version doesn't include ratelimits/keys count
                          // This would need to be enhanced if those counts are needed
                          ratelimits: [],
                          keys: [],
                        }}
                      />
                    );
                  })}
                </TableBody>
              </Table>

              {identities.length === 0 && !isLoading ? (
                <p className="flex items-center justify-center w-full text-content-subtle text-sm">
                  {search ? "No identities found matching your search" : "No identities found"}
                </p>
              ) : null}

              {hasMore && (
                <div className="flex justify-center mt-4">
                  <p className="text-content-subtle text-sm">
                    More results available (pagination not implemented in TanStack version yet)
                  </p>
                </div>
              )}
            </>
          )}
        </div>
      </PageContent>
    </div>
  );
}
