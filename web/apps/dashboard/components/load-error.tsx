"use client";

import type { LoadFailure } from "@/lib/loadable/loadable";
import { getErrorMessage } from "@/lib/unkey-client";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";
import type { ComponentProps, ReactNode } from "react";

export function LoadError({
  title,
  error,
  frame,
  onRetry,
  retrying = false,
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
        <Button variant="primary" size="sm" onClick={onRetry} loading={retrying}>
          Retry
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}

export function EmptyOrLoadError({
  failure,
  title,
  children,
}: {
  failure: LoadFailure | null;
  title: string;
  children: ReactNode;
}) {
  if (!failure) {
    return children;
  }
  return (
    <LoadError
      title={title}
      error={failure.error}
      frame="none"
      onRetry={failure.retry}
      retrying={failure.retrying}
    />
  );
}
