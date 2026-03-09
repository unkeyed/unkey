import type { ReactNode } from "react";

export function SectionHeader({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="flex items-center gap-2.5 mb-4 px-2">
      {icon}
      <div className="text-accent-12 font-medium text-[13px] leading-4">{title}</div>
    </div>
  );
}

export function Section({ children }: { children: ReactNode }) {
  return <div className="flex flex-col">{children}</div>;
}
