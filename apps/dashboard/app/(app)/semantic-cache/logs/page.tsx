import { PageHeader } from "@/components/dashboard/page-header";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { getAllSemanticCacheLogs } from "@/lib/tinybird";

export const formatDate = (timestamp: string | number | Date): string => {
  const date = new Date(timestamp);
  const options: Intl.DateTimeFormatOptions = {
    month: "long",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  };
  return date.toLocaleDateString("en-US", options);
};

export default async function SemanticCacheLogsPage() {
  const { data } = await getAllSemanticCacheLogs({ limit: 10 });

  return (
    <div className="mt-4 ml-1">
      <h1 className="font-medium">Logs</h1>
      <Table className="mt-4">
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Model</TableHead>
            <TableHead>Cache status</TableHead>
            <TableHead>Query</TableHead>
            <TableHead>Request ID</TableHead>
            <TableHead>Request timing</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((data) => (
            <Dialog>
              <DialogTrigger asChild>
                <TableRow key={data.requestId}>
                  <TableCell className="font-medium">{formatDate(data.timestamp)}</TableCell>
                  <TableCell>{data.model}</TableCell>
                  <TableCell>{data.cache === 0 ? "Miss" : "Hit"}</TableCell>
                  <TableCell>{data.query}</TableCell>
                  <TableCell>{data.requestId}</TableCell>
                  <TableCell>{data.timing}</TableCell>
                </TableRow>
              </DialogTrigger>
              <DialogContent className="translate-x-0 translate-y-0 right-0 top-0 left-[unset] h-screen">
                {data.response}
              </DialogContent>
            </Dialog>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
