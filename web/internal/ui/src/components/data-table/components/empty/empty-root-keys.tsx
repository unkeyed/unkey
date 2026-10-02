import { IconBookBookmarkOutline18 } from "@unkey/icons";
import { buttonVariants } from "../../../buttons/button";
import {
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "../../../empty-state";

export function EmptyRootKeys() {
  return (
    <div className="w-full flex justify-center items-center h-full">
      <EmptyState frame="none" className="w-[400px] items-start text-left">
        <EmptyStateIcon>
          <IconBookBookmarkOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader className="items-start">
          <EmptyStateTitle>No Root Keys found</EmptyStateTitle>
          <EmptyStateDescription className="text-left">
            There are no Root Keys configured yet. Create your first Root Key to start managing
            permissions and access control.
          </EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <a
            href="https://www.unkey.com/docs/security/overview#root-keys"
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ variant: "outline" })}
          >
            <IconBookBookmarkOutline18 />
            Learn about Root Keys
          </a>
        </EmptyStateActions>
      </EmptyState>
    </div>
  );
}
