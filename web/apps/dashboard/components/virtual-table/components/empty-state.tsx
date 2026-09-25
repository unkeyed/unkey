import { IconBookBookmarkOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "@unkey/ui";

export const VirtualTableEmptyState = () => (
  <EmptyState frame="none">
    <EmptyStateHeader>
      <EmptyStateTitle>Nothing here yet</EmptyStateTitle>
      <EmptyStateDescription>
        Ready to get started? Check our documentation for a step-by-step guide.
      </EmptyStateDescription>
    </EmptyStateHeader>
    <EmptyStateActions>
      <a href="https://www.unkey.com/docs" target="_blank" rel="noopener noreferrer">
        <Button variant="outline">
          <IconBookBookmarkOutline18 />
          Documentation
        </Button>
      </a>
    </EmptyStateActions>
  </EmptyState>
);
