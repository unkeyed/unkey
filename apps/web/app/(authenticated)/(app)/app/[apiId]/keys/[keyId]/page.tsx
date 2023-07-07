import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { notFound } from "next/navigation";
import { getUsage } from "@/lib/tinybird";
import { env } from "@/lib/env";
import { fillRange } from "@/lib/utils";
import { ColumnChart } from "@/components/charts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export const revalidate = 0;

export default async function ApiPage(props: { params: { keyId: string } }) {
  const tenantId = getTenantId();

  const key = await db.query.keys.findFirst({
    where: eq(schema.keys.id, props.params.keyId),
    with: {
      workspace: true,
    },
  });
  if (!key || key.workspace.tenantId !== tenantId) {
    return notFound();
  }

  const usage = await getUsage({
    workspaceId: key.workspaceId,
    apiId: key.apiId,
    keyId: key.id,
  });

  const now = new Date();
  const year = now.getUTCFullYear();
  const month = now.getUTCMonth();
  const start = new Date(year, month, 0, 0, 0, 0, 0);
  const end = new Date(start.getTime());
  end.setUTCMonth(end.getUTCMonth() + 1);

  const usageOverTime = fillRange(
    usage.data.map(({ time, usage }) => ({ value: usage, time })),
    start.getTime(),
    end.getTime(),
    "1d",
  ).map(({ value, time }) => ({
    x: new Date(time).toUTCString(),
    y: value,
  }));

  return (
    <div>
      <Card>
        <CardHeader>
          <CardTitle>Usage in the last 30 days</CardTitle>
          <CardDescription>See when this key was verified</CardDescription>
        </CardHeader>
        <CardContent>
          <ColumnChart data={usageOverTime} />
        </CardContent>
      </Card>
    </div>
  );
}
