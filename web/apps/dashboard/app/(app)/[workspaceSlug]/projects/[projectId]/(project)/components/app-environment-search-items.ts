import type { FilterSearchItem } from "@/components/logs/checkbox/filters-popover";
import {
  type AppEnvironmentSelection,
  isEntireAppSelected,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "./app-environment-selection";

type AppOption = {
  appId: string;
  name: string;
};

type EnvironmentOption = {
  id: string;
  slug: string;
};

type CreateAppEnvironmentSearchItemsParams = {
  apps: readonly AppOption[];
  environmentsByAppId: ReadonlyMap<string, readonly EnvironmentOption[]>;
  selection: AppEnvironmentSelection;
  onSelectionChange: (selection: AppEnvironmentSelection) => void;
};

export function createAppEnvironmentSearchItems({
  apps,
  environmentsByAppId,
  selection,
  onSelectionChange,
}: CreateAppEnvironmentSearchItemsParams): FilterSearchItem[] {
  return apps.flatMap((app) => {
    const appEnvironments = environmentsByAppId.get(app.appId) ?? [];
    const environmentIds = appEnvironments.map((environment) => environment.id);

    return [
      {
        kind: "option",
        id: `app:${app.appId}`,
        label: app.name,
        path: ["App"],
        keywords: [app.appId, "all environments"],
        checked: isEntireAppSelected(selection, app.appId, environmentIds),
        onSelect: () => onSelectionChange(toggleAppSelection(selection, app.appId, environmentIds)),
      },
      ...appEnvironments.map(
        (environment): FilterSearchItem => ({
          kind: "option",
          id: `environment:${environment.id}`,
          label: environment.slug,
          path: ["App", app.name],
          keywords: [environment.id, app.appId, app.name],
          checked: selection.appIds.has(app.appId) || selection.environmentIds.has(environment.id),
          onSelect: () =>
            onSelectionChange(
              toggleEnvironmentSelection(selection, app.appId, environment.id, environmentIds),
            ),
        }),
      ),
    ];
  });
}
