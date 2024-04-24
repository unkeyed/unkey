import { PageHeader } from "@/components/dashboard/page-header";
import { CreateApiButton } from "./create-api-button";

import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { Search } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { ApiList } from "./client";

export const dynamic = "force-dynamic";
export const runtime = "edge";

export default async function ApisOverviewPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAt),
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const apis = await Promise.all(
    workspace.apis.map(async (api) => ({
      id: api.id,
      name: api.name,
      keys: await db
        .select({ count: sql<number>`count(*)` })
        .from(schema.keys)
        .where(and(eq(schema.keys.keyAuthId, api.keyAuthId!), isNull(schema.keys.deletedAt))),
    })),
  );

  const unpaid = workspace.tenantId.startsWith("org_") && workspace.plan === "free";

  return (
    <div className="">
      {unpaid ? (
        <div>
          <PageHeader title="Applications" description="Manage your APIs" />
          <Separator className="my-6" />
          <section className="flex flex-col gap-4 my-4 md:flex-row md:items-center">
            <div className="flex items-center flex-grow h-8 gap-2 px-3 py-2 text-sm bg-transparent border rounded-md border-border focus-within:border-primary/40">
              <Search className="w-4 h-4" />
              <input
                disabled
                className="flex-grow bg-transparent placeholder:text-content-subtle focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 "
                placeholder="Search.."
              />
            </div>
            <CreateApiButton disabled />
          </section>
          <div className="mt-10 flex min-h-[400px] flex-col items-center  justify-center space-y-6 rounded-lg border border-dashed px-4 md:mt-24">
            <h3 className="text-xl font-semibold leading-none tracking-tight text-center md:text-2xl">
              Upgrade your plan
            </h3>
            <p className="text-sm text-center text-gray-500 md:text-base">
              Team workspaces is a paid feature. Please switch to a paid plan to continue using it.
            </p>
            <Link
              href="/settings/billing"
              className="px-4 py-2 mr-3 text-sm font-medium text-center text-white bg-gray-800 rounded-lg hover:bg-gray-500 focus:outline-none focus:ring-4 focus:ring-gray-300 dark:bg-gray-600 dark:hover:bg-gray-500 dark:focus:ring-gray-800"
            >
              Subscribe
            </Link>
          </div>
        </div>
      ) : (
        <ApiList apis={apis} />
      )}
    </div>
  );
}
