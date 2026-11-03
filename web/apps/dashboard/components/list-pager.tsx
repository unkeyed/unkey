"use client";

import { getErrorMessage } from "@/lib/unkey-client";
import { match } from "@unkey/match";
import {
  Button,
  FOOTER_ERROR_COPY,
  type ListFooterState,
  ResourceListFooter,
  type ResourceListFooterVariant,
  ResourceListShowMore,
} from "@unkey/ui";

export function ListPager({
  state,
  noun,
  variant,
}: {
  state: ListFooterState;
  noun: string;
  variant?: ResourceListFooterVariant;
}) {
  return match(state)
    .with({ kind: "hidden" }, () => null)
    .with({ kind: "more" }, ({ onLoadMore }) => (
      <ResourceListFooter variant={variant}>
        <ResourceListShowMore onClick={onLoadMore} />
      </ResourceListFooter>
    ))
    .with({ kind: "loadingMore" }, () => (
      <ResourceListFooter variant={variant}>
        <ResourceListShowMore loading />
      </ResourceListFooter>
    ))
    .with({ kind: "error" }, ({ reason, error, onRetry, retrying }) => (
      <ResourceListFooter variant={variant}>
        <div className="flex items-center gap-3">
          <span role="alert" className="text-error-11 text-sm">
            {getErrorMessage(error, FOOTER_ERROR_COPY[reason](noun))}
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
