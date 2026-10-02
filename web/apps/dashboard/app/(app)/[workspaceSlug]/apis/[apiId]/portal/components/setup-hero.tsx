"use client";

import { IconWindowLayoutOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";

export function SetupHero({ onEnable }: { onEnable: () => void }) {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconWindowLayoutOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>Customer portal</EmptyStateTitle>
        <EmptyStateDescription>
          An Unkey-hosted portal that allows your customers to manage their keys themselves.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>
        <Button variant="primary" size="md" onClick={onEnable}>
          Enable Customer portal
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
