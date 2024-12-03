import { toast } from "@/components/ui/toaster";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
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
  if (content === undefined || content === null) {
    return false;
  }

  if (Array.isArray(content)) {
    return content.some((item) => item !== null && item !== undefined);
  }

  if (typeof content === "object" && content !== null) {
    return Object.values(content).some(
      (value) => value !== null && value !== undefined
    );
  }

  if (typeof content === "string") {
    return content.trim().length > 0;
  }

  return Boolean(content);
};

export const RequestResponseDetails = <T extends unknown[]>({
  fields,
  className,
}: Props<T>) => {
  const handleClick = (field: Field<unknown>) => {
    try {
      const text =
        typeof field.content === "object"
          ? JSON.stringify(field.content)
          : String(field.content);

      navigator.clipboard
        .writeText(text)
        .then(() => {
          toast.success(field.tooltipSuccessMessage);
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
  return (
    <div className={cn("font-sans", className)}>
      {fields.map(
        (field, index) =>
          isNonEmpty(field.content) && (
            <TooltipProvider key={`${field.label}-${index}`}>
              <Tooltip>
                <TooltipTrigger
                  className={cn(
                    "flex w-full justify-between border-border border-solid pr-3 py-[10px] items-center",
                    index !== fields.length - 1 && "border-b",
                    field.className
                  )}
                  onClick={() => handleClick(field)}
                >
                  <span className="text-sm text-content/65">{field.label}</span>
                  {field.description(field.content as NonNullable<T[number]>)}
                </TooltipTrigger>
                <TooltipContent side="left">
                  {field.tooltipContent}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )
      )}
    </div>
  );
};
