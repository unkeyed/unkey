import type { ReactNode } from "react";

type DataListProps = {
  children: ReactNode;
};

export function DataList({ children }: DataListProps) {
  return <dl className="flex flex-col">{children}</dl>;
}

type DataRowProps = {
  label: string;
  children: ReactNode;
};

export function DataRow({ label, children }: DataRowProps) {
  return (
    <div className="flex items-start gap-6 border-t border-grayA-3 py-3 first:border-t-0">
      <dt className="w-32 shrink-0 text-[13px] leading-5 text-gray-9">{label}</dt>
      <dd className="min-w-0 flex-1 text-[13px] font-medium leading-5 text-accent-12">
        {children}
      </dd>
    </div>
  );
}
