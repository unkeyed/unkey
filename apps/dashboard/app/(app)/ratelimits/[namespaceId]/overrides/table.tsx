import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { clickhouse } from "@/lib/clickhouse";
import { Button } from "@unkey/ui";
import { ChevronRight, Minus } from "lucide-react";
import ms from "ms";
import Link from "next/link";

type Props = {
  workspaceId: string;
  namespaceId: string;
  ratelimits: {
    id: string;
    identifier: string;
    limit: number;
    duration: number;
    async: boolean | null;
  }[];
};

export const Overrides: React.FC<Props> = async ({ workspaceId, namespaceId, ratelimits }) => {
  return (
    <Table className="no-scrollbar mt-4">
      <TableHeader>
        <TableRow>
          <TableHead>Identifier</TableHead>
          <TableHead>Limits</TableHead>
          <TableHead>Async</TableHead>
          <TableHead>Last used</TableHead>
          <TableHead />
        </TableRow>
      </TableHeader>
      <TableBody>
        {ratelimits.map((rl) => (
          <TableRow key={rl.id}>
            <TableCell>
              <div className="flex flex-col items-start ">
                <span className="text-sm text-content">{rl.identifier}</span>
                <pre className="text-xs text-content-subtle">{rl.id}</pre>
              </div>
            </TableCell>

            <TableCell className="flex items-center gap-2">
              <Badge variant="secondary">{Intl.NumberFormat().format(rl.limit)} requests</Badge>
              <span className="text-content-subtle">/</span>
              <Badge variant="secondary">{ms(rl.duration)}</Badge>
            </TableCell>
            <TableCell>
              {rl.async === null ? (
                <Minus className="w-4 h-4 text-content-subtle" />
              ) : (
                <Badge variant="secondary">{rl.async ? "async" : "sync"}</Badge>
              )}
            </TableCell>
            <TableCell>
              <LastUsed
                workspaceId={workspaceId}
                namespaceId={namespaceId}
                identifier={rl.identifier}
              />
            </TableCell>
            <TableCell className="flex justify-end">
              <Link href={`/ratelimits/${namespaceId}/overrides/${rl.id}`}>
                <Button variant="ghost">
                  <ChevronRight className="w-4 h-4" />
                </Button>
              </Link>
            </TableCell>
            {/* </Link> */}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};

const LastUsed: React.FC<{
  workspaceId: string;
  namespaceId: string;
  identifier: string;
}> = async ({ workspaceId, namespaceId, identifier }) => {
  const lastUsed = await clickhouse.ratelimits.latest({
    workspaceId,
    namespaceId,
    identifier: [identifier],
    limit: 1,
  });

  const unixMilli = lastUsed.val?.at(0)?.time;
  if (unixMilli) {
    return <span className="text-sm text-content-subtle">{ms(Date.now() - unixMilli)} ago</span>;
  }
  return <Minus className="w-4 h-4 text-content-subtle" />;
};
