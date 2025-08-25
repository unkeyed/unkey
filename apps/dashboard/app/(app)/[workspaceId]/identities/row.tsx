"use client";

import { TableCell, TableRow } from "@/components/ui/table";
import Link from "next/link";
import { memo, useCallback, useMemo, useRef } from "react";

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

function RowComponent(props: Props) {
  const { identity } = props;

  const detailsUrl = useMemo(() => {
    const encodedWorkspaceId = encodeURIComponent(identity.workspaceId);
    const encodedId = encodeURIComponent(identity.id);
    return `/${encodedWorkspaceId}/identities/${encodedId}`;
  }, [identity.workspaceId, identity.id]);

  const prefetchTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const hasPrefetchedRef = useRef(false);

  const handlePrefetch = useCallback(() => {
    if (hasPrefetchedRef.current) {
      return;
    }

    if (prefetchTimeoutRef.current) {
      clearTimeout(prefetchTimeoutRef.current);
    }

    prefetchTimeoutRef.current = setTimeout(() => {
      // Use Link's built-in prefetch behavior instead of manual router.prefetch
      hasPrefetchedRef.current = true;
    }, 100); // 100ms debounce
  }, []);

  return (
    <TableRow className="group">
      <Link
        href={detailsUrl}
        prefetch={false}
        scroll={false}
        onMouseEnter={handlePrefetch}
        onFocus={handlePrefetch}
        className="contents"
        aria-label={`View details for identity ${identity.externalId}`}
      >
        <TableCell className="group-hover:bg-muted/50 transition-colors">
          <span className="font-mono text-xs text-content">{identity.externalId}</span>
        </TableCell>
        <TableCell className="flex flex-col gap-1 group-hover:bg-muted/50 transition-colors">
          <pre className="text-xs text-content font-mono">
            {JSON.stringify(identity.meta, null, 2)}
          </pre>
        </TableCell>

        <TableCell className="font-mono group-hover:bg-muted/50 transition-colors">
          {identity.keys.length}
        </TableCell>

        <TableCell className="font-mono group-hover:bg-muted/50 transition-colors">
          {identity.ratelimits.length}
        </TableCell>
      </Link>
    </TableRow>
  );
}

export const Row = memo(RowComponent);
