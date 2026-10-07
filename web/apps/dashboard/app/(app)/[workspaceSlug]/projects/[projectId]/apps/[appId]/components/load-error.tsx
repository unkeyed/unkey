"use client";

import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";

/**
 * A failed query collection load ends ready with no rows, so pages check the
 * collection's error before they show an empty state. Retry loads it again.
 */
export function LoadError({
  title,
  onRetry,
}: {
  title: string;
  onRetry: () => void;
}) {
  return (
    <EmptyState>
      <EmptyStateHeader>
        <EmptyStateTitle>{title}</EmptyStateTitle>
        <EmptyStateDescription>The request failed. Try again.</EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>
        <Button variant="primary" size="md" onClick={onRetry}>
          Retry
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
