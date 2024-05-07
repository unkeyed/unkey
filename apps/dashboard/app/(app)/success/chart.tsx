import { AreaChart } from "@/components/dashboard/charts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

export const revalidate = 60;

type Props = {
  title: string;
  description?: string;
  query: () => Promise<{ date: string; count: number }[]>;
};

export const Chart: React.FC<Props> = async ({ query, title, description }) => {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  const res = await query();

  const data: Array<{ x: string; y: number }> = [];

  if (res.length === 0) {
    return <div>No data</div>;
  }

  data.push({ x: res[0].date, y: Number(res[0].count) });
  for (let i = 1; i < res.length; i++) {
    data.push({ x: res[i].date, y: data[i - 1].y + Number(res[i].count) });
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <AreaChart data={data} timeGranularity="day" tooltipLabel="Total keys" />
      </CardContent>
    </Card>
  );
};
