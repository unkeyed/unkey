import { PageHeader } from "@/components/dashboard/page-header";
import { CreateApiButton } from "./create-api-button";

import { Separator } from "@/components/ui/separator";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema, sql } from "@/lib/db";
import { Search } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { ApiList } from "./client";

export const revalidate = 3;
export const runtime = "edge";
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
        .where(eq(schema.keys.keyAuthId, api.keyAuthId!)),
    })),
  );
  const unpaid = workspace.tenantId.startsWith("org_") && workspace.plan === "free";
  return (
    <div className="">
      {unpaid ? (
        <div>
          <PageHeader title="Applications" description="Manage your APIs" />
          <Separator className="my-6" />
          <section className="flex flex-col gap-4 my-4 md:items-center md:flex-row">
            <div className="flex items-center flex-grow h-8 gap-2 px-3 py-2 text-sm bg-transparent border rounded-md border-border focus-within:border-primary/40">
              <Search className="w-4 h-4" />
              <input
                disabled
                className="flex-grow bg-transparent disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-content-subtle focus-visible:outline-none "
                placeholder="Search.."
              />
            </div>
            <CreateApiButton disabled />
          </section>
          <div className="flex flex-col justify-center items-center mt-10  md:mt-24 px-4 space-y-6 border border-dashed rounded-lg min-h-[400px]">
            <h3 className="text-xl font-semibold leading-none tracking-tight text-center md:text-2xl">
              Please add billing to your account
            </h3>
            <p className="text-sm text-center text-gray-500 md:text-base">
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
