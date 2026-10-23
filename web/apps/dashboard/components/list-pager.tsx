"use client";

import type { ListFooterState } from "@/lib/loadable/list";
import { getErrorMessage } from "@/lib/unkey-client";
import { match } from "@unkey/match";
import { Button, ResourceListFooter } from "@unkey/ui";
import { cn } from "cn";

type ErrorReason = Extract<ListFooterState, { kind: "error" }>["reason"];

const ERROR_COPY: Record<ErrorReason, (noun: string) => string> = {
  refresh: (noun) => `We couldn't refresh ${noun}.`,
  loadMore: (noun) => `We couldn't load more ${noun}.`,
};

export function ListPager({
  state,
  noun,
  className,
}: {
  state: ListFooterState;
  noun: string;
  className?: string;
}) {
  return match(state)
    .with({ kind: "hidden" }, () => null)
    .with({ kind: "more" }, ({ onLoadMore }) => (
      <ResourceListFooter className={cn("py-6", className)}>
        <Button size="md" variant="outline" onClick={onLoadMore}>
          Load more
        </Button>
      </ResourceListFooter>
    ))
    .with({ kind: "loadingMore" }, () => (
      <ResourceListFooter className={cn("py-6", className)}>
        <Button size="md" variant="outline" disabled loading>
          Load more
        </Button>
      </ResourceListFooter>
    ))
    .with({ kind: "error" }, ({ reason, error, onRetry, retrying }) => (
      <ResourceListFooter className={className}>
        <div className="flex items-center gap-3">
          <span role="alert" className="text-error-11 text-sm">
            {getErrorMessage(error, ERROR_COPY[reason](noun))}
          </span>
          <Button
            size="md"
            variant="outline"
            onClick={onRetry}
            loading={retrying}
            disabled={retrying}
          >
            Retry
          </Button>
        </div>
      </ResourceListFooter>
    ))
    .exhaustive();
}
