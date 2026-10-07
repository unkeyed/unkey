import { cn } from "cn";
import type { ComponentProps, ReactNode } from "react";

type NodeTone = "default" | "warning";

const NODE_TONE: Record<NodeTone, string> = {
  default:
    "bg-raised shadow-sm ring-1 ring-grayA-5 hover:ring-grayA-8 data-[active=true]:ring-grayA-8",
  warning: "bg-raised shadow-sm ring-1 ring-warning-7",
};

export function Node({
  tone = "default",
  active = false,
  children,
  ...rest
}: {
  tone?: NodeTone;
  active?: boolean;
  children: ReactNode;
} & Pick<ComponentProps<"div">, "onMouseEnter" | "onMouseLeave" | "title">) {
  return (
    <div
      {...rest}
      data-active={active}
      className={cn(
        "relative flex min-w-0 flex-col overflow-hidden rounded-lg transition-[border-color,box-shadow]",
        NODE_TONE[tone],
      )}
    >
      {children}
    </div>
  );
}

export function NodeHeader({
  leading,
  title,
  meta,
  right,
}: { leading: ReactNode; title: string; meta?: ReactNode; right?: ReactNode }) {
  return (
    <span className="flex min-w-0 items-center gap-1.5 px-3 py-2">
      {leading}
      <span className="shrink-0 text-xs font-medium text-gray-12">{title}</span>
      {meta != null && <span className="truncate font-mono text-2xs text-gray-10">{meta}</span>}
      {right != null && <span className="ml-auto flex shrink-0 items-center gap-1">{right}</span>}
    </span>
  );
}

type BadgeTone = "default" | "warning";

const BADGE: Record<BadgeTone, string> = {
  default: "border-grayA-5 bg-grayA-3 font-mono font-medium text-gray-12",
  warning: "border-warning-6 bg-warning-3 text-warning-11",
};

export function Badge({ tone = "default", children }: { tone?: BadgeTone; children: ReactNode }) {
  return (
    <span className={cn("rounded-sm border px-1 text-3xs leading-4", BADGE[tone])}>{children}</span>
  );
}

export function SpecGrid({ children }: { children: ReactNode }) {
  return (
    <span className="grid grid-cols-2 gap-x-3 gap-y-1 whitespace-nowrap border-t border-grayA-4 px-3 py-2 font-mono text-3xs">
      {children}
    </span>
  );
}

export function SpecItem({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: ReactNode;
}) {
  return (
    <span className="flex min-w-0 items-center gap-1" title={label}>
      <span className="shrink-0 text-gray-10 [&_svg]:size-3" aria-label={label}>
        {icon}
      </span>
      <span className="truncate text-gray-12">{value}</span>
    </span>
  );
}

export function FanOut({ targets }: { targets: number }) {
  const n = Math.max(targets, 1);
  return (
    <svg
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      className="pointer-events-none h-10 w-full"
      aria-hidden="true"
    >
      {Array.from({ length: n }, (_, i) => ((i + 0.5) / n) * 100).map((x) => (
        <path
          key={x}
          d={`M50,0 V50 H${x} V100`}
          fill="none"
          strokeWidth={1}
          strokeDasharray="3 3"
          vectorEffect="non-scaling-stroke"
          className="stroke-gray-7 animate-dash-flow motion-reduce:animate-none"
        />
      ))}
    </svg>
  );
}
