"use client";

import { TableCell, TableRow } from "@/components/ui/table";
import { useRouter } from "next/navigation";

type Props = {
  identity: {
    id: string;
    externalId: string;
    meta?: Record<string, unknown>;
    ratelimits: Array<{
      id: string;
    }>;
    keys: Array<{
      id: string;
    }>;
    workspaceId: string;
  };
};
export const Row: React.FC<Props> = ({ identity }) => {
  const detailsUrl = `/${identity.workspaceId}/identities/${identity.id}`;
  const router = useRouter();
  router.prefetch(detailsUrl);
  return (
    <TableRow
      className="hover:cursor-pointer"
      onClick={() => {
        router.push(detailsUrl);
      }}
    >
      <TableCell>
        <span className="font-mono text-xs text-content">{identity.externalId}</span>
      </TableCell>
      <TableCell className="flex flex-col gap-1">
        <pre className="text-xs text-content font-mono">
          {JSON.stringify(identity.meta, null, 2)}
        </pre>
      </TableCell>

      <TableCell className="font-mono">{identity.keys.length}</TableCell>

      <TableCell className="font-mono">{identity.ratelimits.length}</TableCell>
    </TableRow>
  );
};
