import { IconChevronRightOutline12, IconCubeOutline18 } from "@unkey/icons";
import { Badge, cn } from "@unkey/ui";
import type { ReactNode } from "react";
import {
  type ConnectionTargets,
  type Environment,
  type ListedConnection,
  connectionHostVariable,
  describeTarget,
  isProduction,
} from "./connection-rules";

export function Group({
  label,
  className,
  children,
}: {
  label: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("relative flex min-w-0 flex-col gap-3", className)}>
      <span className="px-1 text-xs text-gray-10">{label}</span>
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
  onHighlight,
}: {
  connection: ListedConnection;
  environment: Environment;
  targets: ConnectionTargets;
  selected: boolean;
  onSelect: () => void;
  onHighlight: (id: string | null) => void;
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
  const variable = connectionHostVariable(connection.name);
  return (
    <button
      type="button"
      data-wire-to={connection.id}
      data-connection={connection.id}
      aria-pressed={selected}
      aria-controls={selected ? "connection-details" : undefined}
      onClick={onSelect}
      onPointerEnter={() => onHighlight(connection.id)}
      onPointerLeave={() => onHighlight(null)}
      onFocus={() => onHighlight(connection.id)}
      onBlur={() => onHighlight(null)}
      className={cn(
        "group relative flex w-full min-w-0 flex-col overflow-hidden rounded-lg bg-raised text-left shadow-sm ring-1 ring-grayA-5 hover:ring-gray-8 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-gray-10",
        selected && "ring-gray-10 hover:ring-gray-10",
      )}
    >
      <span className="flex w-full min-w-0 items-center gap-2.5 px-3 py-3">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-grayA-3 text-gray-11">
          <IconCubeOutline18 className="size-4" />
        </span>
        <span
          className="min-w-0 flex-1 truncate text-sm font-medium text-gray-12"
          title={connection.targetAppName}
        >
          {connection.targetAppName}
        </span>
        <Badge
          variant={description.unavailable ? "warning" : "code"}
          className="max-w-[45%] truncate"
          title={description.text}
        >
          {targetLabel}
        </Badge>
      </span>
      <span className="flex w-full min-w-0 items-center justify-between gap-2 border-t border-grayA-4 bg-grayA-2 px-3 py-2 text-gray-10 group-hover:text-gray-12">
        <code className="min-w-0 flex-1 truncate text-2xs" title={variable}>
          {variable}
        </code>
        <IconChevronRightOutline12 className="shrink-0 transition-transform motion-safe:group-hover:translate-x-0.5 motion-reduce:transition-none" />
      </span>
    </button>
  );
}
