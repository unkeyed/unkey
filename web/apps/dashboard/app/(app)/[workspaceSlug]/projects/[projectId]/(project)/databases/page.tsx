"use client";

import { LoadError } from "@/components/load-error";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { listOf } from "@/lib/loadable/list";
import { readyOr } from "@/lib/loadable/loadable";
import { routes } from "@/lib/navigation/routes";
import { IconDatabaseOutline18, IconPlusOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderTitle,
  ResourceList,
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { parseAsString, useQueryState } from "nuqs";
import type { ReactNode } from "react";
import { DatabasePanel } from "./database-panel";
import { DatabasesTable, DatabasesTableSkeleton } from "./databases-table";
import { useDeleteDatabase, useProjectDatabases } from "./use-project-databases";

export default function DatabasesPage() {
  const workspace = useWorkspaceNavigation();
  const { projectId, databases } = useProjectDatabases();
  const remove = useDeleteDatabase();
  const [databaseId, setDatabaseId] = useQueryState(
    "database",
    parseAsString.withOptions({ history: "replace" }),
  );
  const scope = { workspaceSlug: workspace.slug, projectId };
  const createButton = (
    <Button
      size="md"
      variant="primary"
      render={<Link href={routes.projects.databases.new(scope)} />}
    >
      <IconPlusOutline18 />
      Create database
    </Button>
  );

  return (
    <PageContainer className="flex-1">
      <PageHeader>
        <PageHeaderContent>
          <PageHeaderTitle>Databases</PageHeaderTitle>
        </PageHeaderContent>
        <PageHeaderActions>{createButton}</PageHeaderActions>
      </PageHeader>
      <PageBody className="flex-1">
        {match(listOf(databases))
          .with({ status: "loading" }, () => (
            <ResourceList>
              <DatabasesTableSkeleton />
            </ResourceList>
          ))
          .with({ status: "failed" }, ({ error, retry }) => (
            <ResourceList>
              <LoadError
                title="Could not load databases"
                error={error}
                frame="none"
                onRetry={retry}
              />
            </ResourceList>
          ))
          .with({ status: "empty" }, () => <DatabasesEmpty action={createButton} />)
          .with({ status: "ready" }, ({ rows }) => (
            <ResourceList>
              <DatabasesTable databases={rows} onSelect={setDatabaseId} />
            </ResourceList>
          ))
          .exhaustive()}
        <DatabasePanel
          db={readyOr(databases, []).find((db) => db.id === databaseId)}
          onClose={() => setDatabaseId(null)}
          onDelete={(db) => {
            if (remove.isLoading) {
              return;
            }
            setDatabaseId(null);
            remove.mutate(db.id, {
              onError: (error) => {
                toast.error(
                  error instanceof Error ? error.message : "Could not delete the database.",
                );
              },
            });
          }}
        />
      </PageBody>
    </PageContainer>
  );
}

function DatabasesEmpty({ action }: { action: ReactNode }) {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconDatabaseOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>No databases yet</EmptyStateTitle>
        <EmptyStateDescription>
          Create a Postgres or MySQL database for this project.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <EmptyStateActions>{action}</EmptyStateActions>
    </EmptyState>
  );
}
