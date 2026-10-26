"use client";

import { getErrorMessage } from "@/lib/unkey-client";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";
import type { ComponentProps } from "react";

export function LoadError({
  title,
  error,
  frame,
  onRetry,
}: {
  title: string;
  error?: unknown;
  frame?: ComponentProps<typeof EmptyState>["frame"];
  onRetry: () => void;
}) {
  return (
    <EmptyState frame={frame}>
      <EmptyStateHeader role="alert">
        <EmptyStateTitle>{title}</EmptyStateTitle>
        <EmptyStateDescription>
          {getErrorMessage(error, "The request failed. Try again.")}
        </EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>
        <Button variant="primary" size="sm" onClick={onRetry}>
          Retry
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
