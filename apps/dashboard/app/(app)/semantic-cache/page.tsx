import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { StackedColumnChart } from "@/components/dashboard/charts";
import { Card } from "@/components/ui/card";
import { getAllSemanticCacheLogs } from "@/lib/tinybird";
import { type Interval, IntervalSelect } from "../apis/[apiId]/select";

export default async function SemanticCachePage(props: {
  searchParams: {
    interval?: Interval;
  };
}) {
  const interval = props.searchParams.interval ?? "7d";

  const { data } = await getAllSemanticCacheLogs({});
  return (
    <div>
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching"
      />
      <Separator className="my-6" />
      <h1 className="font-medium">Logs</h1>
      <p className="text-sm text-gray-500">View real-time logs from the semantic cache.</p>
      <Table className="mt-4">
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Model</TableHead>
            <TableHead>Cache status</TableHead>
            <TableHead>Query</TableHead>
            <TableHead>Response</TableHead>
            <TableHead>Request ID</TableHead>
            <TableHead>Request timing</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((data) => (
            <TableRow key={data.requestId}>
              <TableCell className="font-medium">{data.timestamp}</TableCell>
              <TableCell>{data.model}</TableCell>
              <TableCell>{data.cache}</TableCell>
              <TableCell>{data.query}</TableCell>
              <TableCell>{data.response}</TableCell>
              <TableCell>{data.requestId}</TableCell>
              <TableCell>{data.timing}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Separator className="my-6" />
      <div className="flex flex-col">
        <div className="flex items-center justify-between">
          <h1 className="font-medium mb-4">Analytics</h1>

          <div>
            <IntervalSelect defaultSelected={interval} />
          </div>
        </div>
      </div>
      {/* <StackedColumnChart
        colors={["primary", "warn", "danger"]}
        data={data}
        timeGranularity={
          granularity >= 1000 * 60 * 60 * 24 * 30
            ? "month"
            : granularity >= 1000 * 60 * 60 * 24
              ? "day"
              : "hour"
        }
      /> */}
    </div>
  );
}
