"use client";

import { ListViews } from "@/components/list-views";
import { ResourceSearchInput, useListSearch } from "@/components/resource-search-input";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { useShowMore } from "@/lib/loadable/list";
import { readyOr } from "@/lib/loadable/loadable";
import { routes } from "@/lib/navigation/routes";
import { IconDatabaseOutline18, IconPlusOutline18 } from "@unkey/icons";
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
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { parseAsString, useQueryState } from "nuqs";
import type { ReactNode } from "react";
import { DatabaseCard, DatabaseCardSkeleton } from "./database-card";
import { DatabasePanel } from "./database-panel";
import { DATABASE_TABLE, DatabaseRow } from "./databases-table";
import { useDeleteDatabase, useProjectDatabases } from "./use-project-databases";

export default function DatabasesPage() {
  const workspace = useWorkspaceNavigation();
  const { projectId, databases } = useProjectDatabases();
  const remove = useDeleteDatabase();
  const [databaseId, setDatabaseId] = useQueryState(
    "database",
    parseAsString.withOptions({ history: "replace" }),
  );
  const search = useListSearch();
  const needle = search.toLowerCase();
  const list = useShowMore({
    all: readyOr(databases, []),
    isLoading: databases.status === "loading",
    matches: (rows) => rows.filter((db) => db.name.toLowerCase().includes(needle)),
    search,
    pageSize: GRID_PAGE_SIZE,
  });
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
        <ListViews
          list={list}
          viewStorageKey="databases-view"
          noun="databases"
          failedTitle="Could not load databases"
          empty={<DatabasesEmpty action={createButton} />}
          controls={<ResourceSearchInput label="Search databases" placeholder="Search databases" />}
          table={DATABASE_TABLE}
          rowKey={(db) => db.id}
          renderCard={(db) => <DatabaseCard db={db} onSelect={setDatabaseId} />}
          renderRow={(db) => <DatabaseRow db={db} onSelect={setDatabaseId} />}
          cardSkeleton={<DatabaseCardSkeleton />}
        />
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
