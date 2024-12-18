import { OptIn } from "@/components/opt-in";
import { PageContent } from "@/components/page-content";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Empty } from "@unkey/ui";
import { Loader2 } from "lucide-react";
import { unstable_cache as cache } from "next/cache";
import { redirect } from "next/navigation";
import { parseAsInteger, parseAsString } from "nuqs/server";
import { Suspense } from "react";
import { SearchField } from "./filter";
import { Navigation } from "./navigation";
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

  const tenantId = await getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.tenantId, tenantId),
  });

  if (!workspace) {
    return redirect("/auth/sign-in");
  }

  if (!workspace.betaFeatures.identities) {
    return <OptIn title="Identities" description="Identities are in beta" feature="identities" />;
  }

  return (
    <div>
      <Navigation />
      <PageContent>
        <SearchField />
        <div className="flex flex-col gap-8 mb-20 mt-8">
          <Suspense
            fallback={
              <Empty>
                <Empty.Title>
                  <Loader2 className="w-4 h-4 animate-spin" />
                </Empty.Title>
              </Empty>
            }
          >
            <Results search={search ?? ""} limit={limit ?? 10} />
          </Suspense>
        </div>
      </PageContent>
    </div>
  );
}

const Results: React.FC<{ search: string; limit: number }> = async (props) => {
  const tenantId = await getTenantId();

  const getData = cache(
    async () =>
      db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
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
