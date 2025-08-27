import { OptIn } from "@/components/opt-in";
import { PageContent } from "@/components/page-content";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getAuth } from "@/lib/auth";
import { db } from "@/lib/db";
import { Empty } from "@unkey/ui";
import { Loader2 } from "lucide-react";
import { unstable_cache as cache } from "next/cache";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { z } from "zod";
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

const searchParamsSchema = z.object({
  search: z.string().optional(),
  limit: z.string().regex(/^\d+$/).optional(),
});

export default async function Page(props: Props) {
  const validatedParams = searchParamsSchema.parse(props.searchParams);
  const search = validatedParams.search ?? "";
  const limit = validatedParams.limit ? Number.parseInt(validatedParams.limit, 10) : DEFAULT_LIMIT;

  const { orgId } = await getAuth();

  let workspace: Awaited<ReturnType<typeof db.query.workspaces.findFirst>>;
  try {
    workspace = await db.query.workspaces.findFirst({
      where: (table, { eq }) => eq(table.orgId, orgId),
    });
  } catch (error) {
    console.error({
      message: "Failed to fetch workspace for identities page",
      orgId,
      error: error instanceof Error ? error.message : String(error),
      stack: error instanceof Error ? error.stack : undefined,
    });

    // Redirect to sign-in page on database failure
    return redirect("/auth/sign-in");
  }

  if (!workspace) {
    return redirect("/auth/sign-in");
  }

  if (!workspace.betaFeatures.identities) {
    return <OptIn title="Identities" description="Identities are in beta" feature="identities" />;
  }

  return (
    <div>
      <Navigation workspaceSlug={workspace.slug ?? ""} />
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
  const { orgId } = await getAuth();
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

  let workspace: Awaited<ReturnType<typeof getData>>;
  try {
    workspace = await getData();
  } catch (error) {
    console.error({
      message: "Failed to fetch workspace data for identities results",
      orgId,
      search: props.search,
      limit: props.limit,
      error: error instanceof Error ? error.message : String(error),
      stack: error instanceof Error ? error.stack : undefined,
    });

    // Return an error state instead of crashing
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
    return redirect("/new");
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
                workspaceSlug={workspace.slug ?? ""}
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
