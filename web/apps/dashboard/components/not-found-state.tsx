"use client";
import { routes } from "@/lib/navigation/routes";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  PageBody,
  PageContainer,
} from "@unkey/ui";
import { useRouter } from "next/navigation";

type NotFoundStateProps = {
  title?: string;
  description?: string;
};

export function NotFoundState({
  title = "404 Not Found",
  description = "We couldn't find the page that you're looking for!",
}: NotFoundStateProps) {
  const router = useRouter();

  return (
    <PageContainer>
      <PageBody>
        <EmptyState>
          <EmptyStateHeader>
            <EmptyStateTitle>{title}</EmptyStateTitle>
            <EmptyStateDescription>{description}</EmptyStateDescription>
          </EmptyStateHeader>
          <EmptyStateActions>
            <Button
              variant="default"
              onClick={() => {
                router.push(routes.workspaces.root());
              }}
            >
              Go Back
            </Button>
          </EmptyStateActions>
        </EmptyState>
      </PageBody>
    </PageContainer>
  );
}
