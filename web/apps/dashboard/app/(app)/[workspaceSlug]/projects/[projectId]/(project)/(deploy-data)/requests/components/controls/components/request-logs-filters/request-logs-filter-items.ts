import type { FilterItemOption } from "@/components/logs/checkbox/filters-popover";

export type RequestLogsStatusRangeOption = {
  id: number;
  status: number;
  display: string;
  label: string;
  color: string;
  checked: boolean;
};

export const statusRangeOptions: RequestLogsStatusRangeOption[] = [
  { id: 1, status: 200, display: "2xx", label: "Success", color: "bg-success-9", checked: false },
  { id: 2, status: 300, display: "3xx", label: "Redirect", color: "bg-info-8", checked: false },
  { id: 3, status: 400, display: "4xx", label: "Warning", color: "bg-warning-8", checked: false },
  { id: 4, status: 500, display: "5xx", label: "Error", color: "bg-error-9", checked: false },
];

export type RequestLogsMethodOption = {
  id: number;
  method: string;
  checked: boolean;
};

export const methodOptions: RequestLogsMethodOption[] = [
  { id: 1, method: "GET", checked: false },
  { id: 2, method: "POST", checked: false },
  { id: 3, method: "PUT", checked: false },
  { id: 4, method: "DELETE", checked: false },
  { id: 5, method: "PATCH", checked: false },
];

const statusSearchOptions: FilterItemOption<number>[] = statusRangeOptions.map((status) => ({
  value: status.status,
  label: status.label,
  description: status.display,
}));

const methodSearchOptions: FilterItemOption<string>[] = methodOptions.map((method) => ({
  value: method.method,
  label: method.method,
}));

export type RequestLogsDeploymentSearchSource = {
  id: string;
  gitBranch: string | null;
};

function requestLogsDeploymentSearchOptions(
  deployments: readonly RequestLogsDeploymentSearchSource[],
): FilterItemOption<string>[] {
  return deployments.map((deployment) => {
    const branch = deployment.gitBranch?.trim() || null;
    return {
      value: deployment.id,
      label: branch ?? deployment.id,
      description: branch === null ? undefined : deployment.id,
      keywords: branch === null ? [deployment.id] : [deployment.id, branch],
    };
  });
}

type RequestLogsSearchableItem<T extends { id: string }> = T & {
  options?: FilterItemOption<string | number>[];
};

export function createRequestLogsFilterItems<T extends { id: string }>(
  items: readonly T[],
  deployments: readonly RequestLogsDeploymentSearchSource[],
): RequestLogsSearchableItem<T>[] {
  const deploymentOptions = requestLogsDeploymentSearchOptions(deployments);
  return items.map((item) => {
    switch (item.id) {
      case "status":
        return { ...item, options: statusSearchOptions };
      case "methods":
        return { ...item, options: methodSearchOptions };
      case "deploymentId":
        return { ...item, options: deploymentOptions };
      default:
        return item;
    }
  });
}
