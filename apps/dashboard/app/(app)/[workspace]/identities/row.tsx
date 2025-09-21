"use client";

import { TableCell, TableRow } from "@/components/ui/table";
import { useRouter } from "next/navigation";
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
  workspaceSlug: string;
};

function RowComponent(props: Props) {
  const { identity, workspaceSlug } = props;
  const router = useRouter();

  const detailsUrl = useMemo(() => {
    const encodedWorkspaceId = encodeURIComponent(workspaceSlug);
    const encodedId = encodeURIComponent(identity.id);
    return `/${encodedWorkspaceId}/identities/${encodedId}`;
  }, [workspaceSlug, identity.id]);

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

  const handleRowClick = useCallback(() => {
    router.push(detailsUrl);
  }, [router, detailsUrl]);

  return (
    <TableRow
      className="group cursor-pointer"
      onClick={handleRowClick}
      onMouseEnter={handlePrefetch}
      onFocus={handlePrefetch}
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          handleRowClick();
        }
      }}
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
    </TableRow>
  );
}

export const Row = memo(RowComponent);
