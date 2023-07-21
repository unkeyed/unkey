import { PageHeader } from "@/components/dashboard/page-header";
import { CreateApiButton } from "@/components/dashboard/create-api";

import { getTenantId } from "@/lib/auth";
import { db, schema, eq, sql } from "@unkey/db";
import { redirect } from "next/navigation";
import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import { ApiList } from "./client";
import { Icons } from "@/components/ui/icons";

export const revalidate = 3;

export default async function TenantOverviewPage() {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: true,
    },
  });

  if (!workspace) {
    return redirect("/onboarding");
  }

  const apis = await Promise.all(
    workspace.apis.map(async (api) => ({
      id: api.id,
      name: api.name,
      keys: await db
        .select({ count: sql<number>`count(*)` })
        .from(schema.keys)
        .where(eq(schema.keys.apiId, api.id)),
    })),
  );
  const unpaid = workspace.tenantId.startsWith("org_") && workspace.plan === "free";
  return (
    <div className=" ">
      {unpaid ? (
        <div>
          <PageHeader title="Applications" description="Manage your APIs" />
          <Separator className="my-6" />
          <section className=" my-4 flex md:items-center gap-4 flex-col md:flex-row">
            <div className="flex h-10 flex-grow items-center gap-2 rounded-md border border-input bg-transparent px-3 py-2 text-sm focus-within:border-primary/40">
              <Icons.search size={18} />
              <input
                disabled
                className="disabled:cursor-not-allowed bg-transparent flex-grow disabled:opacity-50 placeholder:text-muted-foreground focus-visible:outline-none  "
                placeholder="Search.."
              />
            </div>
            <CreateApiButton disabled />
          </section>
          <div className="flex flex-col justify-center items-center mt-10  md:mt-24 px-4 space-y-6 border border-dashed rounded-lg min-h-[400px]">
            <Icons.payment size={80} />
            <h3 className="md:text-2xl tet-xl font-semibold text-center leading-none tracking-tight">
              Please add billing to your account
            </h3>
            <p className="text-gray-500 text-center text-sm md:text-base">
              Team workspaces is a paid feature. Please add billing to your account to continue
              using it.
            </p>
            <Link
              href="/app/stripe"
              target="_blank"
              className="px-4 py-2 mr-3 text-sm font-medium text-center text-white bg-gray-800 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-gray-300 dark:bg-gray-600 dark:hover:bg-gray-500 dark:focus:ring-gray-800"
            >
              Add billing
            </Link>
          </div>
        </div>
      ) : (
        <ApiList apis={apis} />
      )}
    </div>
  );
}
