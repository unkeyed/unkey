import { toast } from "@/components/ui/toaster";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type Field<T> = {
  label: string;
  description: (content: NonNullable<T>) => ReactNode;
  content: T | null;
  tooltipContent: ReactNode;
  tooltipSuccessMessage: string;
  className?: string;
};

type Props<T extends unknown[]> = {
  fields: { [K in keyof T]: Field<T[K]> };
  className?: string;
};
//This function ensures that content is not nil, and if it's an object or array, it has some content.
const isNonEmpty = (content: unknown): boolean => {
  if (Array.isArray(content)) {
    return content.length > 0;
  }
  if (typeof content === "object" && content !== null) {
    return Object.keys(content).length > 0;
  }
  return Boolean(content);
};

export const RequestResponseDetails = <T extends unknown[]>({ fields, className }: Props<T>) => {
  return (
    <div className={cn("font-sans", className)}>
      {fields.map(
        (field, index) =>
          isNonEmpty(field.content) && (
            // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
            <TooltipProvider key={index}>
              <Tooltip>
                <TooltipTrigger
                  className={cn(
                    "flex w-full justify-between border-border border-solid pr-3 py-[10px] items-center",
                    index !== fields.length - 1 && "border-b",
                    field.className,
                  )}
                  onClick={() => {
                    navigator.clipboard.writeText(
                      typeof field.content === "object"
                        ? JSON.stringify(field.content)
                        : String(field.content),
                    );
                    toast.success(field.tooltipSuccessMessage);
                  }}
                >
                  <span className="text-sm text-content/65">{field.label}</span>
                  {field.description(field.content as NonNullable<T[number]>)}
                </TooltipTrigger>
                <TooltipContent side="left">{field.tooltipContent}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ),
      )}
    </div>
  );
};
