import { redirect } from "next/navigation";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Loader2 } from "lucide-react";
import { unstable_cache as cache } from "next/cache";
import { parseAsInteger, parseAsString } from "nuqs/server";
import { Suspense } from "react";
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

  return (
    <div className="flex flex-col gap-8">
      <SearchField />

      <div className="flex flex-col gap-8 mb-20 ">
        <Suspense
          fallback={
            <EmptyPlaceholder>
              <EmptyPlaceholder.Title>
                <Loader2 className="w-4 h-4 animate-spin" />
              </EmptyPlaceholder.Title>
            </EmptyPlaceholder>
          }
        >
          <Results search={search ?? ""} limit={limit ?? 10} />
        </Suspense>
      </div>
    </div>
  );
}

const Results: React.FC<{ search: string; limit: number }> = async (props) => {
  const tenantId = getTenantId();

  const getData = cache(
    async () =>
      db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
        with: {
          identities: {
            where: (table, { or, like }) =>
              or(like(table.externalId, `%${props.search}%`), like(table.id, `%${props.search}%`)),

            limit: props.limit,
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
    [`${tenantId}-${props.search}-${props.limit}`],
  );

  const workspace = await getData();

  if (!workspace) {
    return redirect("/new");
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
      {workspace.identities.length === 0 ? (
        <p className="flex items-center justify-center w-full text-content-subtle text-sm">
          No identities found
        </p>
      ) : null}
    </>
  );
};
