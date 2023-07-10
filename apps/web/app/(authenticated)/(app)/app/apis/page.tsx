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
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

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
    console.log(unpaid)
    return(
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
             Team workspaces is a paid feature. Please add billing to your account to continue using it.
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
      <PageHeader title="Applications" description="Manage your APIs" />

      <Separator className="my-6" />

      <ul role="list" className="grid grid-cols-1 gap-x-6 gap-y-8 lg:grid-cols-3 xl:gap-x-8">
        <Card className="duration-500 hover:border-primary bg-muted">
          <CardHeader>
            <CardTitle>Create New API</CardTitle>
          </CardHeader>

          <CardContent />
          <CardFooter>
            <CreateApiButton isDisabled={workspace.tenantId.startsWith("org_") && workspace.plan === "free"} key="createApi" />
          </CardFooter>
        </Card>
        {apis.map((api) => (
          <Link key={api.id} href={`/app/${api.id}`}>
            <Card className="duration-500 hover:border-primary">
              <CardHeader>
                <CardTitle>{api.name}</CardTitle>
                <CardDescription>{api.id}</CardDescription>
              </CardHeader>

              <CardContent>
                <dl className="text-sm leading-6 divide-y divide-gray-100 ">
                  <div className="flex justify-between py-3 gap-x-4">
                    <dt className="text-gray-500">API Keys</dt>
                    <dd className="flex items-start gap-x-2">
                      <div className="font-medium text-gray-900">{api.keys.at(0)?.count ?? 0}</div>
                    </dd>
                  </div>
                </dl>
              </CardContent>
            </Card>
          </Link>
        ))}
      </ul>
    </div>
  )}
  </div>
  );
}
