// import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";

// import { Input } from "@/components/ui/input";
// import { Label } from "@/components/ui/label";
// import { Separator } from "@/components/ui/separator";

// import {
//   Table,
//   TableBody,
//   TableCaption,
//   TableCell,
//   TableHead,
//   TableHeader,
//   TableRow,
// } from "@/components/ui/table";

import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { getAllSemanticCacheLogs } from "@/lib/tinybird";
import { redirect } from "next/navigation";
import { IntervalSelect } from "../../apis/[apiId]/select";
import { LogsTable } from "./table";

// const formatDate = (timestamp: string | number | Date): string => {
//   const date = new Date(timestamp);
//   const options: Intl.DateTimeFormatOptions = {
//     month: "long",
//     day: "numeric",
//     hour: "2-digit",
//     minute: "2-digit",
//     second: "2-digit",
//   };
//   return date.toLocaleDateString("en-US", options);
// };

export function getInterval(interval: string) {
  const now = new Date();
  console.info({ interval });
  let _timestamp = 0;

  switch (interval) {
    case "24h":
      _timestamp = now.getTime() - 24 * 60 * 60 * 1000; // 24 hours in milliseconds
      break;
    case "7d":
      // Get the start of the day 7 days ago
      _timestamp = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7).getTime();
      break;
    case "30d":
      // Get the start of the day 30 days ago
      _timestamp = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 30).getTime();
      break;
    case "90d":
      // Get the start of the day 90 days ago
      _timestamp = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 90).getTime();
      break;
    default:
      _timestamp = now.getTime() - 24 * 60 * 60 * 1000; // 24 hours in milliseconds
      break;
  }

  return _timestamp;
}

export default async function SemanticCacheLogsPage({
  searchParams,
}: { searchParams: { interval?: string } }) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      llmGateways: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  const gatewayId = workspace?.llmGateways[0]?.id;

  if (!gatewayId) {
    return redirect("/semantic-cache/new");
  }

  const interval = getInterval(searchParams.interval || "7d");

  const { data } = await getAllSemanticCacheLogs({
    gatewayId,
    workspaceId: workspace?.id,
    limit: 1000,
    interval,
  });

  return <LogsTable data={data} workspace={workspace} />;
}
