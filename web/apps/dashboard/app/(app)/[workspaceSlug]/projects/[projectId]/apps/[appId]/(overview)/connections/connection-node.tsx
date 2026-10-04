import { IconCubeOutline18 } from "@unkey/icons";
import { cn } from "@unkey/ui";
import type { ReactNode } from "react";
import {
  type ConnectionTargets,
  type Environment,
  type ListedConnection,
  describeTarget,
  isProduction,
} from "./connection-rules";

export function Group({
  icon,
  label,
  className,
  children,
}: {
  icon: ReactNode;
  label: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("relative flex min-w-0 flex-col gap-2", className)}>
      <span className="flex items-center gap-1.5 px-1 pb-0.5 text-xs text-gray-11">
        {icon}
        {label}
      </span>
      {children}
    </div>
  );
}

export function ConnectionNode({
  connection,
  environment,
  targets,
  selected,
  onSelect,
}: {
  connection: ListedConnection;
  environment: Environment;
  targets: ConnectionTargets;
  selected: boolean;
  onSelect: () => void;
}) {
  const description = describeTarget({
    connection,
    targetName: connection.targetAppName,
    callerEnvironment: environment,
    ...targets,
  });
  const targetLabel =
    connection.targetType === "automatic"
      ? isProduction(environment)
        ? "production"
        : "Same branch"
      : connection.targetType === "environment"
        ? (targets.environments.find((env) => env.id === connection.targetEnvironmentId)?.slug ??
          "Deleted environment")
        : connection.targetDeploymentId;
  return (
    <button
      type="button"
      data-wire-to={connection.id}
      data-connection={connection.id}
      aria-pressed={selected}
      aria-controls={selected ? "connection-details" : undefined}
      onClick={onSelect}
      className={cn(
        "flex w-full min-w-0 items-center gap-2 rounded-lg border bg-raised px-3 py-2 text-left shadow-xs transition-[border-color,box-shadow] hover:border-gray-8 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-10",
        selected && "border-gray-10 shadow-[0_0_0_3px_var(--color-grayA-3)]",
      )}
    >
      <IconCubeOutline18 className="size-4 shrink-0 text-gray-11" />
      <span
        className="min-w-0 flex-1 truncate text-sm font-medium"
        title={connection.targetAppName}
      >
        {connection.targetAppName}
      </span>
      <span
        className={cn(
          "max-w-[45%] truncate rounded bg-gray-3 px-1.5 py-0.5 text-xs text-gray-11",
          description.unavailable && "bg-warning-3 text-warning-11",
        )}
        title={description.text}
      >
        {targetLabel}
      </span>
    </button>
  );
}
