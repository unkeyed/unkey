import { redirect } from "next/navigation";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Scan } from "lucide-react";
import { unstable_cache as cache } from "next/cache";
import { parseAsInteger, parseAsString } from "nuqs/server";
import { SearchField } from "./filter";
import { Row } from "./row";
type Props = {
  searchParams: {
    search?: string;
    limit?: string;
  };
};

export default async function Page(props: Props) {
  const search = parseAsString.withDefault("").parse(props.searchParams.search ?? "");
  const limit = parseAsInteger.withDefault(10).parse(props.searchParams.limit ?? "10");

  const tenantId = getTenantId();

  const getData = cache(
    async () =>
      db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
        with: {
          identities: {
            where: (table, { or, like }) =>
              or(like(table.externalId, `%${search}%`), like(table.id, `%${search}%`)),

            limit: limit!,
            orderBy: (table, { asc }) => asc(table.id),

            with: {
              ratelimits: {
                columns: {
                  id: true,
                },
              },
              keys: {
                columns: {
                  id: true,
                },
              },
            },
          },
        },
      }),
    [`${tenantId}-${search}-${limit}`],
  );

  const workspace = await getData();

  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="flex flex-col gap-8">
      <SearchField />

      <div className="flex flex-col gap-8 mb-20 ">
        {workspace.identities.length === 0 ? (
          <EmptyPlaceholder>
            <EmptyPlaceholder.Icon>
              <Scan />
            </EmptyPlaceholder.Icon>
            <EmptyPlaceholder.Title>No Identities found</EmptyPlaceholder.Title>
            <EmptyPlaceholder.Description>
              Create an identity or change your search.
            </EmptyPlaceholder.Description>
          </EmptyPlaceholder>
        ) : (
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
                    identity={{
                      id: identity.id,
                      externalId: identity.externalId,
                      meta: identity.meta ?? undefined,
                      ratelimits: identity.ratelimits,
                      keys: identity.keys,
                    }}
                  />
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  );
}
