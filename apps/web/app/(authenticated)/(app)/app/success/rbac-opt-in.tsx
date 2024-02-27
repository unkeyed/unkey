import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db, schema, sql } from "@/lib/db";
import { notFound } from "next/navigation";

export const revalidate = 60;

export async function RbacOptIn() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  const optedIn = await db
    .select({ count: sql<string>`count(*)` })
    .from(schema.workspaces)
    .where(sql`beta_features->>'$.rbac' = 'true' `)
    .then((res) => res.at(0)?.count ?? "0")
    .catch((err) => {
      console.error(err);
      throw err;
    });

  return (
    <Card className="flex flex-col w-full h-fit">
      <CardHeader>
        <CardTitle>RBAC Opt In</CardTitle>
        <CardDescription>How many workspaces have opted into the RBAC alpha</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="mt-2 text-2xl font-semibold leading-none tracking-tight">{optedIn}</div>
        <div className="mt-4" />
      </CardContent>
    </Card>
  );
}
