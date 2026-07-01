"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetOverlay,
  SheetPortal,
  SheetTitle,
} from "@/components/ui/sheet";
import type { UnkeyPermission } from "@unkey/rbac";
import { Button } from "@unkey/ui";
import { useMemo, useRef } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";
import { usePermissionSheet } from "../hooks/use-permission-sheet";
import { WORKSPACE_SCOPE } from "../permissions";
import { PermissionContentList } from "./permission-list";
import { SearchPermissions } from "./search-permissions";

type PermissionSheetProps = {
  apis: { id: string; name: string }[];
  projects: { id: string; name: string }[];
  apps: { id: string; name: string }[];
  environments: { id: string; name: string; appId: string }[];
  selectedPermissions: UnkeyPermission[];
  onChange: (permissions: UnkeyPermission[]) => void;
  loadMore?: () => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  editMode?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export const PermissionSheet = ({
  apis,
  projects,
  apps,
  environments,
  selectedPermissions,
  onChange,
  loadMore,
  hasNextPage,
  isFetchingNextPage,
  editMode = false,
  open,
  onOpenChange,
}: PermissionSheetProps) => {
  const inputRef = useRef<HTMLInputElement>(null);

  const {
    searchValue,
    isProcessing,
    hasNoResults,
    handleSearchChange,
    handleApiPermissionChange,
    handleProjectPermissionChange,
    handleAppPermissionChange,
    handleEnvironmentPermissionChange,
    handleWorkspacePermissionChange,
  } = usePermissionSheet({
    apis,
    projects,
    apps,
    environments,
    selectedPermissions,
    onChange,
    editMode,
  });

  const apiScopes = useMemo(
    () =>
      apis.map((api) => ({
        id: api.id,
        scope: { kind: "api" as const, id: api.id, name: api.name },
      })),
    [apis],
  );
  const projectScopes = useMemo(
    () =>
      projects.map((project) => ({
        id: project.id,
        scope: { kind: "project" as const, id: project.id, name: project.name },
      })),
    [projects],
  );
  const environmentsByApp = useMemo(() => {
    const map = new Map<string, { id: string; name: string }[]>();
    for (const environment of environments) {
      const existing = map.get(environment.appId);
      if (existing) {
        existing.push({ id: environment.id, name: environment.name });
      } else {
        map.set(environment.appId, [{ id: environment.id, name: environment.name }]);
      }
    }
    return map;
  }, [environments]);

  const appScopes = useMemo(
    () =>
      apps.map((app) => {
        const appEnvironments = environmentsByApp.get(app.id) ?? [];
        return {
          id: app.id,
          environmentIds: appEnvironments.map((environment) => environment.id),
          scope: {
            kind: "app" as const,
            id: app.id,
            name: app.name,
            environments: appEnvironments,
          },
        };
      }),
    [apps, environmentsByApp],
  );

  const orphanEnvironmentScopes = useMemo(() => {
    const appIds = new Set(apps.map((app) => app.id));
    return environments
      .filter((environment) => !appIds.has(environment.appId))
      .map((environment) => ({
        id: environment.id,
        scope: { kind: "environment" as const, id: environment.id, name: environment.name },
      }));
  }, [environments, apps]);

  return (
    <Sheet modal={true} open={open} onOpenChange={onOpenChange}>
      <SheetPortal>
        <SheetOverlay className="bg-black/30 backdrop-blur-xs" />
        <SheetContent
          disableClose={false}
          className="flex flex-col p-0 m-0 h-full gap-0 border-l border-l-gray-4 w-[420px] bg-gray-1 dark:bg-black overflow-hidden"
          side="right"
          overlay="transparent"
        >
          <SheetHeader className="flex flex-row min-w-full border-b border-gray-4 gap-2 shrink-0">
            <SheetTitle className="sr-only">Select Permissions</SheetTitle>
            <SearchPermissions
              isProcessing={isProcessing}
              search={searchValue}
              inputRef={inputRef}
              onChange={handleSearchChange}
            />
          </SheetHeader>
          <ScrollArea className="flex-1 min-h-0">
            <div className="flex flex-col gap-1 pt-2 pb-6">
              {hasNoResults ? (
                <p className="text-sm text-gray-10 ml-6 py-1.5 mt-1.5">
                  {ROOT_KEY_MESSAGES.UI.NO_RESULTS}
                </p>
              ) : (
                <>
                  <PermissionContentList
                    selected={selectedPermissions}
                    searchValue={searchValue}
                    key="workspace"
                    scope={WORKSPACE_SCOPE}
                    onPermissionChange={handleWorkspacePermissionChange}
                  />
                  {apiScopes.length > 0 && (
                    <p className="text-sm text-gray-10 ml-6 py-1.5 mb-2">
                      {ROOT_KEY_MESSAGES.UI.FROM_APIS}
                    </p>
                  )}
                  {apiScopes.map(({ id, scope }) => (
                    <PermissionContentList
                      selected={selectedPermissions}
                      searchValue={searchValue}
                      key={id}
                      scope={scope}
                      onPermissionChange={(permissions) =>
                        handleApiPermissionChange(id, permissions)
                      }
                    />
                  ))}
                  {projectScopes.length > 0 && (
                    <p className="text-sm text-gray-10 ml-6 py-1.5 mb-2">
                      {ROOT_KEY_MESSAGES.UI.FROM_PROJECTS}
                    </p>
                  )}
                  {projectScopes.map(({ id, scope }) => (
                    <PermissionContentList
                      selected={selectedPermissions}
                      searchValue={searchValue}
                      key={id}
                      scope={scope}
                      onPermissionChange={(permissions) =>
                        handleProjectPermissionChange(id, permissions)
                      }
                    />
                  ))}
                  {appScopes.length > 0 && (
                    <p className="text-sm text-gray-10 ml-6 py-1.5 mb-2">
                      {ROOT_KEY_MESSAGES.UI.FROM_APPS}
                    </p>
                  )}
                  {appScopes.map(({ id, environmentIds, scope }) => (
                    <PermissionContentList
                      selected={selectedPermissions}
                      searchValue={searchValue}
                      key={id}
                      scope={scope}
                      onPermissionChange={(permissions) =>
                        handleAppPermissionChange(id, environmentIds, permissions)
                      }
                    />
                  ))}
                  {orphanEnvironmentScopes.length > 0 && (
                    <p className="text-sm text-gray-10 ml-6 py-1.5 mb-2">
                      {ROOT_KEY_MESSAGES.UI.FROM_ENVIRONMENTS}
                    </p>
                  )}
                  {orphanEnvironmentScopes.map(({ id, scope }) => (
                    <PermissionContentList
                      selected={selectedPermissions}
                      searchValue={searchValue}
                      key={id}
                      scope={scope}
                      onPermissionChange={(permissions) =>
                        handleEnvironmentPermissionChange(id, permissions)
                      }
                    />
                  ))}
                </>
              )}
            </div>
          </ScrollArea>
          {hasNextPage && (
            <div className="shrink-0 bg-gray-1 dark:bg-black border-t border-gray-4 w-full py-4">
              <div className="flex flex-row justify-center items-center">
                <Button
                  className="mx-auto rounded-lg"
                  size="sm"
                  onClick={() => loadMore?.()}
                  disabled={!loadMore}
                  loading={isFetchingNextPage}
                >
                  {ROOT_KEY_MESSAGES.UI.LOAD_MORE}
                </Button>
              </div>
            </div>
          )}
        </SheetContent>
      </SheetPortal>
    </Sheet>
  );
};
