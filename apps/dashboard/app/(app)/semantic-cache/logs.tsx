import { PageHeader } from "@/components/dashboard/page-header";
import { Separator } from "@/components/ui/separator";

import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { getAllSemanticCacheLogs } from "@/lib/tinybird";

export const formatDate = (timestamp: string | number | Date): string => {
  const date = new Date(timestamp);
  const options: Intl.DateTimeFormatOptions = {
    year: "numeric",
    month: "long",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  };
  return date.toLocaleDateString("en-US", options);
};

export default async function SemanticCachePage() {
  const { data } = await getAllSemanticCacheLogs({ limit: 10 });
  const _transformedData = data.map((log) => {
    const isCacheHit = log.cache > 0;
    return {
      x: log.timestamp,
      y: isCacheHit ? 1 : 0, // Assuming cache > 0 indicates a cache hit
      category: isCacheHit ? "cache hit" : "cache miss",
    };
  });

  return (
    <div>
      <Separator className="my-6" />
      <h1 className="font-medium">Logs</h1>
      <Table className="mt-4">
        <TableCaption>View real-time logs from the semantic cache.</TableCaption>
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
              <TableCell className="font-medium">{formatDate(data.timestamp)}</TableCell>
              <TableCell>{data.model}</TableCell>
              <TableCell>{data.cache === 0 ? "Miss" : "Hit"}</TableCell>
              <TableCell>{data.query}</TableCell>
              <TableCell>{data.response}</TableCell>
              <TableCell>{data.requestId}</TableCell>
              <TableCell>{data.timing}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
