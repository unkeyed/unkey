import { PageHeader } from "@/components/PageHeader";
import { CreateApiButton } from "./CreateAPI";

import { getTenantId } from "@/lib/auth";
import { db, schema, eq, sql } from "@unkey/db";
import { redirect } from "next/navigation";
import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ApiList } from "./client";

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
    }))
  );
  const unpaid =
    workspace.tenantId.startsWith("org_") && workspace.plan === "free";
  return (
    <div>
      {unpaid ? (
        <div>
          <PageHeader title="Applications" description="Manage your APIs" />

          <Separator className="my-6" />

          <div className="flex justify-center items-center">
            <Card className="duration-500 hover:border-primary bg-muted w-3xl ">
              <CardHeader>
                <CardTitle>Please add billing to your account</CardTitle>
              </CardHeader>

              <CardContent>
                <p className="text-gray-500">
                  Team workspaces is a paid feature. Please add billing to your
                  account to continue using it.
                </p>
              </CardContent>
              <CardFooter>
                <Link
                  href="/app/stripe"
                  target="_blank"
                  className="px-4 py-2 mr-3 text-sm font-medium text-center text-white bg-gray-800 rounded-lg hover:bg-gray-500 focus:ring-4 focus:outline-none focus:ring-gray-300 dark:bg-gray-600 dark:hover:bg-gray-500 dark:focus:ring-gray-800"
                >
                  Add billing
                </Link>
              </CardFooter>
            </Card>
          </div>
        </div>
      ) : (
        <div>
          <ApiList apis={apis} />
        </div>
      )}
    </div>
  );
}
