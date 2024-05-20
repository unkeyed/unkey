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

export default async function SemanticCachePage() {
  const { data } = await getAllSemanticCacheLogs({ limit: 10 });
  return (
    <div>
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching"
      />
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
    </div>
  );
}
