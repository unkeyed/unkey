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
  retrying,
}: {
  title: string;
  error?: unknown;
  frame?: ComponentProps<typeof EmptyState>["frame"];
  onRetry: () => void;
  retrying?: boolean;
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
        <Button
          variant="primary"
          size="md"
          onClick={onRetry}
          loading={retrying}
          disabled={retrying}
        >
          Retry
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
