import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useMemo, useState } from "react";
import type { ChangeEvent } from "react";
import {
  apiPermissions,
  appPermissions,
  environmentPermissions,
  projectPermissions,
  workspacePermissions,
} from "../permissions";
import { hasPermissionResults } from "../utils/permissions";

type ScopedItem = { id: string; name: string };

type UsePermissionSheetProps = {
  apis: ScopedItem[];
  projects: ScopedItem[];
  apps: ScopedItem[];
  environments: ScopedItem[];
  selectedPermissions: UnkeyPermission[];
  onChange?: (permissions: UnkeyPermission[]) => void;
  editMode?: boolean;
};

function collectPermissions(
  list: ScopedItem[],
  build: (id: string) => {
    [category: string]: { [action: string]: { permission: UnkeyPermission } };
  },
  skipId?: string,
): Set<UnkeyPermission> {
  const set = new Set<UnkeyPermission>();
  for (const item of list) {
    if (skipId !== undefined && item.id === skipId) {
      continue;
    }
    for (const category of Object.values(build(item.id))) {
      for (const entry of Object.values(category)) {
        set.add(entry.permission);
      }
    }
  }
  return set;
}

export function usePermissionSheet({
  apis,
  projects,
  apps,
  environments,
  selectedPermissions,
  onChange,
}: UsePermissionSheetProps) {
  const [searchValue, setSearchValue] = useState<string | undefined>(undefined);
  const [isProcessing, setIsProcessing] = useState(false);

  const workspacePermsSet = useMemo(
    () =>
      new Set<UnkeyPermission>(
        Object.values(workspacePermissions).flatMap((category) =>
          Object.values(category).map((item) => item.permission),
        ),
      ),
    [],
  );

  const handleSearchChange = useCallback((e: ChangeEvent<HTMLInputElement>) => {
    setIsProcessing(true);
    if (e.target.value === "") {
      setSearchValue(undefined);
    } else {
      setSearchValue(e.target.value);
    }
    setIsProcessing(false);
  }, []);

  const rebuildScopedPerms = useCallback(
    (
      opts: {
        apiSkipId?: string;
        projectSkipId?: string;
        appSkipId?: string;
        environmentSkipId?: string;
      },
      newPerms: UnkeyPermission[],
    ): UnkeyPermission[] => {
      const workspacePerms = selectedPermissions.filter((p) => workspacePermsSet.has(p));

      const apisSet = collectPermissions(apis, apiPermissions, opts.apiSkipId);
      const apiPerms = selectedPermissions.filter((p) => apisSet.has(p));

      const projectsSet = collectPermissions(projects, projectPermissions, opts.projectSkipId);
      const projectPerms = selectedPermissions.filter((p) => projectsSet.has(p));

      const appsSet = collectPermissions(apps, appPermissions, opts.appSkipId);
      const appPerms = selectedPermissions.filter((p) => appsSet.has(p));

      const environmentsSet = collectPermissions(
        environments,
        environmentPermissions,
        opts.environmentSkipId,
      );
      const environmentPerms = selectedPermissions.filter((p) => environmentsSet.has(p));

      return Array.from(
        new Set<UnkeyPermission>([
          ...workspacePerms,
          ...apiPerms,
          ...projectPerms,
          ...appPerms,
          ...environmentPerms,
          ...newPerms,
        ]),
      );
    },
    [selectedPermissions, apis, projects, apps, environments, workspacePermsSet],
  );

  const handleApiPermissionChange = useCallback(
    (apiId: string, permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }
      onChange(rebuildScopedPerms({ apiSkipId: apiId }, permissions));
    },
    [onChange, rebuildScopedPerms],
  );

  const handleProjectPermissionChange = useCallback(
    (projectId: string, permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }
      onChange(rebuildScopedPerms({ projectSkipId: projectId }, permissions));
    },
    [onChange, rebuildScopedPerms],
  );

  const handleAppPermissionChange = useCallback(
    (appId: string, permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }
      onChange(rebuildScopedPerms({ appSkipId: appId }, permissions));
    },
    [onChange, rebuildScopedPerms],
  );

  const handleEnvironmentPermissionChange = useCallback(
    (environmentId: string, permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }
      onChange(rebuildScopedPerms({ environmentSkipId: environmentId }, permissions));
    },
    [onChange, rebuildScopedPerms],
  );

  const handleWorkspacePermissionChange = useCallback(
    (permissions: UnkeyPermission[]) => {
      if (!onChange) {
        return;
      }

      const scopedPerms = selectedPermissions.filter((p) => !workspacePermsSet.has(p));
      onChange(Array.from(new Set<UnkeyPermission>([...permissions, ...scopedPerms])));
    },
    [selectedPermissions, onChange, workspacePermsSet],
  );

  const hasNoResults = useMemo(() => {
    const workspaceHasResults = hasPermissionResults(workspacePermissions, searchValue);
    const anyApiHasResults = apis.some((api) =>
      hasPermissionResults(apiPermissions(api.id), searchValue),
    );
    const anyProjectHasResults = projects.some((project) =>
      hasPermissionResults(projectPermissions(project.id), searchValue),
    );
    const anyAppHasResults = apps.some((app) =>
      hasPermissionResults(appPermissions(app.id), searchValue),
    );
    const anyEnvironmentHasResults = environments.some((environment) =>
      hasPermissionResults(environmentPermissions(environment.id), searchValue),
    );

    return (
      !workspaceHasResults &&
      !anyApiHasResults &&
      !anyProjectHasResults &&
      !anyAppHasResults &&
      !anyEnvironmentHasResults
    );
  }, [searchValue, apis, projects, apps, environments]);

  return {
    searchValue,
    isProcessing,
    hasNoResults,
    handleSearchChange,
    handleApiPermissionChange,
    handleProjectPermissionChange,
    handleAppPermissionChange,
    handleEnvironmentPermissionChange,
    handleWorkspacePermissionChange,
  };
}
