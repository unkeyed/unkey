import { cn } from "@/lib/utils";
import { cva } from "class-variance-authority";
import type { VariantProps } from "class-variance-authority";
import type { HTMLAttributes, ReactNode } from "react";

const statusBadgeVariants = cva(
  "inline-flex items-center rounded-md px-[7px] text-[10px] leading-[20px] uppercase h-[22px] gap-1",
  {
    variants: {
      variant: {
        enabled: "text-success-11 bg-success-3",
        disabled: "text-warning-11 bg-warning-3",
        locked: "text-feature-11 bg-feature-3",
      },
    },
    defaultVariants: {
      variant: "enabled",
    },
  },
);

interface StatusBadgeProps extends HTMLAttributes<HTMLDivElement> {
  variant?: VariantProps<typeof statusBadgeVariants>["variant"];
  icon?: ReactNode;
  text: string;
}

export const StatusBadge = ({ variant, icon, text, className, ...props }: StatusBadgeProps) => {
  return (
    <div className={cn(statusBadgeVariants({ variant }), className)} {...props}>
      {icon && <span className="inline-flex items-center">{icon}</span>}
      <span>{text}</span>
    </div>
  );
};
