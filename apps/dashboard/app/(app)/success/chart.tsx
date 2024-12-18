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
  t0: number;
};

export const Chart: React.FC<Props> = async ({ t0, query, title, description }) => {
  const tenantId = await getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });

  if (!workspace?.features.successPage) {
    return notFound();
  }

  const res = await query();

  const data: Array<{ x: string; y: number }> = [];

  if (res.length === 0) {
    return <div>No data</div>;
  }

  const lookup = res.reduce(
    (acc, { date, count }) => {
      acc[date] = count;
      return acc;
    },
    {} as Record<string, number>,
  );

  const t = new Date(t0);
  t.setUTCHours(0, 0, 0, 0);
  let sum = 0;
  while (t < new Date()) {
    const date = t.toISOString().split("T")[0];
    const added = lookup[date];
    if (added) {
      sum += added;
    }
    data.push({ x: date, y: sum });

    t.setUTCDate(t.getUTCDate() + 1);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="relative h-40">
        <AreaChart
          padding={[8, 40, 64, 40]}
          data={data}
          timeGranularity="day"
          tooltipLabel="Total keys"
        />
      </CardContent>
    </Card>
  );
};
