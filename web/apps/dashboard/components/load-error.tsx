"use client";

import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";

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
