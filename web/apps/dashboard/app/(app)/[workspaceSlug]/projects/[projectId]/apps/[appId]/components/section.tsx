import type { ReactNode } from "react";

export function SectionHeader({
  icon,
  title,
  rightAction,
}: {
  icon: ReactNode;
  title: string;
  // Optional slot rendered right-aligned in the header, used for
  // per-section actions (e.g. "Cancel deployment" on the deployment
  // detail page while a build is in flight).
  rightAction?: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-2.5 mb-4 px-2">
      <div className="flex items-center gap-2.5 min-w-0">
        {icon}
        <div className="text-accent-12 font-medium text-[13px] leading-4">{title}</div>
      </div>
      {rightAction}
    </div>
  );
}

export function Section({ children }: { children: ReactNode }) {
  return <div className="flex flex-col">{children}</div>;
}
