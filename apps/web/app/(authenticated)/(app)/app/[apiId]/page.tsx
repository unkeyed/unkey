import { PageHeader } from "@/components/PageHeader";
import { getTenantId } from "@/lib/auth";
import { db, schema, eq } from "@unkey/db";
import { redirect } from "next/navigation";
import { DeleteApiButton } from "./DeleteApi";
import { Separator } from "@/components/ui/separator";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { CopyButton } from "@/components/CopyButton";
import { Button } from "@/components/ui/button";
import { getUsage, Tinybird } from "@unkey/tinybird";
import { env } from "@/lib/env";
import { fillRange } from "@/lib/utils";
import { ColumnChart } from "@/components/charts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
export default async function ApiPage(props: { params: { apiId: string } }) {
  const tenantId = getTenantId();

  const api = await db.query.apis.findFirst({
    where: eq(schema.apis.id, props.params.apiId),
    with: {
      workspace: true,
    },
  });
  if (!api || api.workspace.tenantId !== tenantId) {
    return redirect("/onboarding");
  }

  const usage = await getUsage(new Tinybird({ token: env.TINYBIRD_TOKEN }))({
    workspaceId: api.workspaceId,
    apiId: api.id,
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
          <CardDescription>This includes all key verifications in this API</CardDescription>
        </CardHeader>
        <CardContent>
          <ColumnChart data={usageOverTime} />
        </CardContent>
      </Card>
    </div>
  );
}
