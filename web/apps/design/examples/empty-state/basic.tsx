import { IconCubeOutline18 } from "@unkey/icons";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";

export default function BasicEmptyState() {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconCubeOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>No projects yet</EmptyStateTitle>
        <EmptyStateDescription>
          You haven't created any projects yet. Get started by creating your first project.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>
        <Button variant="primary" size="md">
          Create project
        </Button>
        <Button variant="outline" size="md">
          Read the docs
        </Button>
      </EmptyStateActions>
    </EmptyState>
  );
}
