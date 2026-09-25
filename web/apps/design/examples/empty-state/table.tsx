import { IconSquareBulletListOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";

export default function TableEmptyState() {
  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <div className="flex items-center gap-4 border-b px-4 py-2.5 text-gray-9 text-xs">
        <div className="flex-1">Name</div>
        <div className="w-24">Status</div>
        <div className="w-28">Created</div>
      </div>
      <EmptyState frame="none">
        <EmptyStateIcon>
          <IconSquareBulletListOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>No deployments</EmptyStateTitle>
          <EmptyStateDescription>
            Deployments appear here once this project ships its first build.
          </EmptyStateDescription>
        </EmptyStateHeader>
        <EmptyStateActions>
          <Button variant="primary" size="md">
            New deployment
          </Button>
        </EmptyStateActions>
      </EmptyState>
    </div>
  );
}
