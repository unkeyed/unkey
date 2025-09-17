"use client";

import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { trpc } from "@/lib/trpc/client";
import { Row } from "../row";

export const Results: React.FC<{ search: string; limit: number }> = (props) => {
  const {
    data: workspace,
    isLoading,
    error,
  } = trpc.identity.searchWithRelations.useQuery({
    search: props.search,
    limit: props.limit,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center w-full py-8">
        <p className="text-content-subtle text-sm">Loading identities...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center w-full py-8">
        <p className="text-content-subtle text-sm">
          Unable to load identities. Please try refreshing the page or contact support if the issue
          persists.
        </p>
      </div>
    );
  }

  if (!workspace) {
    return (
      <div className="flex items-center justify-center w-full py-8">
        <p className="text-content-subtle text-sm">No workspace found.</p>
      </div>
    );
  }

  return (
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
          {workspace.identities.map((identity) => {
            return (
              <Row
                key={identity.id}
                workspaceSlug={workspace.slug}
                identity={{
                  id: identity.id,
                  externalId: identity.externalId,
                  meta: identity.meta ?? undefined,
                  ratelimits: identity.ratelimits,
                  keys: identity.keys,
                  workspaceId: workspace.id,
                }}
              />
            );
          })}
        </TableBody>
      </Table>
      {workspace.identities.length === 0 ? (
        <p className="flex items-center justify-center w-full text-content-subtle text-sm">
          No identities found
        </p>
      ) : null}
    </>
  );
};
