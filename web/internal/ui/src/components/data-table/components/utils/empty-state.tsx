import { IconBookBookmarkOutline18 } from "@unkey/icons";
import { buttonVariants } from "../../../buttons/button";
import {
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
} from "../../../empty-state";

export const DataTableEmptyState = () => (
  <EmptyState frame="none">
    <EmptyStateHeader>
      <EmptyStateTitle>Nothing here yet</EmptyStateTitle>
      <EmptyStateDescription>
        Ready to get started? Check our documentation for a step-by-step guide.
      </EmptyStateDescription>
    </EmptyStateHeader>
    <EmptyStateActions>
      <a
        href="https://www.unkey.com/docs/introduction"
        target="_blank"
        rel="noopener noreferrer"
        className={buttonVariants({ variant: "outline", size: "md" })}
      >
        <IconBookBookmarkOutline18 />
        Documentation
      </a>
    </EmptyStateActions>
  </EmptyState>
);
