import type { ReactNode } from "react";
import type { DetailItem } from "./sections";

type DetailRowProps = {
  icon: ReactNode;
  label: string;
  children: ReactNode;
  alignment?: "center" | "start";
};

function DetailRow({
  icon,
  label,
  children,
  alignment = "center",
}: DetailRowProps) {
  const alignmentClass = alignment === "start" ? "items-start" : "items-center";

  return (
    <div className={`flex ${alignmentClass}`}>
      <div className="flex items-center gap-3 w-[135px]">
        <div className="bg-grayA-3 size-[22px] rounded-md flex items-center justify-center">
          {icon}
        </div>
        <span className="text-grayA-11 text-[13px]">{label}</span>
      </div>
      <div className="text-grayA-11 text-sm min-w-0 flex-1">{children}</div>
    </div>
  );
}

type DetailSectionProps = {
  title: string;
  items: DetailItem[];
  isFirst?: boolean;
};

export function DetailSection({
  title,
  items,
  isFirst = false,
}: DetailSectionProps) {
  return (
    <div className={`px-4 ${isFirst ? "" : "mt-7"}`}>
      <div className="flex items-center gap-3">
        <div className="text-gray-9 text-sm flex-shrink-0">{title}</div>
        <div className="h-px bg-grayA-3 w-full" />
      </div>
      <div className="mt-5" />
      <div className="flex flex-col gap-3.5">
        {items.map((item, index) => (
          <DetailRow
            key={`${item.label}-${index}`}
            icon={item.icon}
            label={item.label}
            alignment={item.alignment}
          >
            {item.content}
          </DetailRow>
        ))}
      </div>
    </div>
  );
}
