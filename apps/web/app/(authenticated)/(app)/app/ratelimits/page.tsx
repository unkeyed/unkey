import { PageHeader } from "@/components/dashboard/page-header";

import { CopyButton } from "@/components/dashboard/copy-button";
import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";
import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { BookOpen, Scan } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { RatelimitCard } from "./card";
import { CreateNamespaceButton } from "./create-namespace-button";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function RatelimitOverviewPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAt),
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const snippet = `curl -XPOST 'https://api.unkey.dev/v1/ratelimit.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

  return (
    <div>
      <PageHeader
        title="Ratelimits"
        description="Manage your ratelimit namespaces"
        actions={[<CreateNamespaceButton key="create-namespace" />]}
      />
      <Separator className="my-6" />

      {workspace.ratelimitNamespaces.length > 0 ? (
        <ul className="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-2 ">
          {workspace.ratelimitNamespaces.map((namespace) => (
            <Link key={namespace.id} href={`/app/ratelimits/${namespace.id}`}>
              <Suspense fallback={null}>
                <RatelimitCard namespace={namespace} workspace={workspace} />
              </Suspense>
            </Link>
          ))}
        </ul>
      ) : (
        <EmptyPlaceholder className="my-4 ">
          <EmptyPlaceholder.Icon>
            <Scan />
          </EmptyPlaceholder.Icon>
          <EmptyPlaceholder.Title>No Namespaces found</EmptyPlaceholder.Title>
          <EmptyPlaceholder.Description>
            You haven&apos;t created any Namespaces yet. Create one by performing a limit request as
            shown below.
          </EmptyPlaceholder.Description>

          <Code className="flex items-start gap-8 p-4 my-8 text-xs text-left">
            {snippet}
            <CopyButton value={snippet} />
          </Code>

          <div className="flex flex-col items-center justify-center gap-2 md:flex-row">
            <Link href="/docs" target="_blank">
              <Button variant="secondary" className="items-center w-full gap-2 ">
                <BookOpen className="w-4 h-4 " />
                Read the docs
              </Button>
            </Link>
          </div>
        </EmptyPlaceholder>
      )}
    </div>
  );
}
