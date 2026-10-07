import type { Environment, EnvironmentKind } from "@/lib/collections/deploy/environments";
import { IconCloudOutline18, IconEyeOutline18 } from "@unkey/icons";
import { cn } from "cn";

const ENVIRONMENT_ICONS: Record<EnvironmentKind, typeof IconCloudOutline18> = {
  production: IconCloudOutline18,
  preview: IconEyeOutline18,
};

export function EnvironmentIcon({
  kind,
  className,
}: { kind: EnvironmentKind; className?: string }) {
  const Icon = ENVIRONMENT_ICONS[kind];
  return <Icon className={cn("size-3.5 shrink-0", className)} />;
}

export function EnvironmentLabel({
  environment,
  className,
}: {
  environment: Pick<Environment, "slug" | "kind"> | undefined;
  className?: string;
}) {
  if (!environment) {
    return <span className="text-gray-9">Unknown</span>;
  }
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 whitespace-nowrap font-mono text-sm text-gray-11",
        className,
      )}
    >
      <EnvironmentIcon kind={environment.kind} />
      <span className="capitalize">{environment.slug}</span>
    </span>
  );
}
