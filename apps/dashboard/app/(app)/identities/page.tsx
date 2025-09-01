import { OptIn } from "@/components/opt-in";
import { PageContent } from "@/components/page-content";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getAuthOrRedirect } from "@/lib/auth";
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

export const dynamic = "force-dynamic";

type Props = {
  searchParams: {
    search?: string;
    limit?: string;
  };
};

const DEFAULT_LIMIT = 100;
export default async function Page(props: Props) {
  const search = parseAsString.withDefault("").parse(props.searchParams.search ?? "");
  const limit = parseAsInteger
    .withDefault(DEFAULT_LIMIT)
    .parse(props.searchParams.limit ?? DEFAULT_LIMIT.toString());

  const { orgId } = await getAuthOrRedirect();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.orgId, orgId),
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
            <Results search={search ?? ""} limit={limit ?? DEFAULT_LIMIT} />
          </Suspense>
        </div>
      </PageContent>
    </div>
  );
}

const Results: React.FC<{ search: string; limit: number }> = async (props) => {
  const { orgId } = await getAuthOrRedirect();
  const getData = cache(
    async () =>
      db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
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
    [`${orgId}-${props.search}-${props.limit}`],
  );

  const workspace = await getData();

  if (!workspace) {
    redirect("/new");
  }

  if (props.search) {
    // If we have an exact match, we want to display it at the very top
    const exactMatchIndex = workspace.identities.findIndex(
      ({ id, externalId }) => props.search === id || props.search === externalId,
    );
    if (exactMatchIndex > 0) {
      workspace.identities.unshift(workspace.identities.splice(exactMatchIndex, 1)[0]);
    }
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
