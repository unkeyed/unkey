import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { OverviewTooltip } from "@unkey/ui";
import type { ReactNode } from "react";

type Field<T> = {
  label: string;
  description: (content: NonNullable<T>) => ReactNode;
  content: T | null;
  tooltipContent?: ReactNode;
  tooltipSuccessMessage?: string;
  className?: string;
  skipTooltip?: boolean;
};

type Props<T extends unknown[]> = {
  fields: { [K in keyof T]: Field<T[K]> };
  className?: string;
};

const isNonEmpty = (content: unknown): boolean => {
  if (content === undefined || content === null) {
    return false;
  }

  if (Array.isArray(content)) {
    return content.some((item) => item !== null && item !== undefined);
  }

  if (typeof content === "object" && content !== null) {
    return Object.values(content).some((value) => value !== null && value !== undefined);
  }

  if (typeof content === "string") {
    return content.trim().length > 0;
  }

  return Boolean(content);
};

export const RequestResponseDetails = <T extends unknown[]>({ fields, className }: Props<T>) => {
  const handleClick = (field: Field<unknown>) => {
    try {
      const text =
        typeof field.content === "object" ? JSON.stringify(field.content) : String(field.content);
      navigator.clipboard
        .writeText(text)
        .then(() => {
          if (field.tooltipSuccessMessage) {
            toast.success(field.tooltipSuccessMessage);
          }
        })
        .catch((error) => {
          console.error("Failed to copy to clipboard:", error);
          toast.error("Failed to copy to clipboard");
        });
    } catch (error) {
      console.error("Error preparing content for clipboard:", error);
      toast.error("Failed to prepare content for clipboard");
    }
  };

  const renderField = (field: Field<T[number]>, index: number) => {
    const baseContent = (
      // biome-ignore lint/a11y/useKeyWithClickEvents: no need
      <div
        className={cn(
          "flex w-full justify-between border-border border-solid pr-3 py-3 items-center cursor-pointer",
          "border-b",
          field.className,
        )}
        onClick={!field.skipTooltip ? () => handleClick(field) : undefined}
      >
        <span className="text-accent-9 text-[13px] lg:no-wrap text-left">{field.label}</span>
        <span className="text-accent-12 text-right w-3/4">
          {field.description(field.content as NonNullable<T[number]>)}
        </span>
      </div>
    );

    if (field.skipTooltip) {
      return baseContent;
    }

    return (
      <OverviewTooltip
        position={{ side: "top" }}
        key={`${field.label}-${index}`}
        variant="secondary"
        content={field.tooltipContent}
        triggerClassName="w-full flex flex-row"
      >
        {baseContent}
      </OverviewTooltip>
    );
  };

  return (
    <div className={cn("font-sans", className)}>
      {fields.map(
        (field, index) =>
          isNonEmpty(field.content) && (
            <div key={`${field.label}-${index}`}>{renderField(field, index)}</div>
          ),
      )}
    </div>
  );
};
