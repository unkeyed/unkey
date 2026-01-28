import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";
import { DisabledWrapper } from "../../../components/disabled-wrapper";
import type { DetailItem } from "./sections";

type DetailRowProps = {
  icon: ReactNode | null;
  label: string | null;
  children: ReactNode;
  disabled?: boolean;
  alignment?: "center" | "start";
};

export function DetailRow({
  icon,
  label,
  disabled,
  children,
  alignment = "center",
}: DetailRowProps) {
  const alignmentClass = alignment === "start" ? "items-start" : "items-center";

  // If both icon and label are missing, let children take full space
  if (!icon && !label) {
    return (
      <div className={`flex ${alignmentClass}`}>
        <div className="text-grayA-11 text-[13px] min-w-0 flex-1">{children}</div>
      </div>
    );
  }

  return (
    <DisabledWrapper disabled={Boolean(disabled)} tooltipContent="Resource metrics coming soon">
      <div className={cn("flex", alignmentClass)}>
        <div className="flex items-center gap-3 w-[135px]">
          {icon && (
            <div className="bg-grayA-3 size-[22px] rounded-md flex items-center justify-center">
              {icon}
            </div>
          )}
          {label && <span className="text-grayA-11 text-[13px]">{label}</span>}
        </div>
        <div className="text-grayA-11 text-[13px] min-w-0 flex-1">{children}</div>
      </div>
    </DisabledWrapper>
  );
}

type DetailSectionProps = {
  title: string;
  items: DetailItem[];
  disabled?: boolean;
  isFirst?: boolean;
};

export function DetailSection({ title, items, disabled, isFirst = false }: DetailSectionProps) {
  return (
    <div className={`px-4 ${isFirst ? "" : "mt-7"}`}>
      <DisabledWrapper disabled={Boolean(disabled)} tooltipContent={`${title} coming soon`}>
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
              disabled={item.disabled}
            >
              {item.content}
            </DetailRow>
          ))}
        </div>
      </DisabledWrapper>
    </div>
  );
}
