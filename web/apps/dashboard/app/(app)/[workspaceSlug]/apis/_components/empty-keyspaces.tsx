"use client";

import { IconBookBookmarkOutline18, IconNodesOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";
import { CreateApiButton } from "./create-api-button";

export function EmptyKeyspaces({
  workspaceSlug,
  isNewApi,
}: {
  workspaceSlug: string;
  isNewApi: boolean;
}) {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconNodesOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>Create your first keyspace</EmptyStateTitle>
        <EmptyStateDescription>
          You haven't created any keyspaces yet. Create one to get started.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>
        <CreateApiButton defaultOpen={isNewApi} workspaceSlug={workspaceSlug} />
        <a
          href="https://www.unkey.com/docs/platform/apis/overview"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Button variant="outline" size="md">
            <IconBookBookmarkOutline18 />
            Read the docs
          </Button>
        </a>
      </EmptyStateActions>
    </EmptyState>
  );
}
