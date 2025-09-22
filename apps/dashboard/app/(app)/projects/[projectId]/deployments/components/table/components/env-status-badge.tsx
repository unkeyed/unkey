import { cn } from "@/lib/utils";
import { cva } from "class-variance-authority";
import type { VariantProps } from "class-variance-authority";
import type { HTMLAttributes, ReactNode } from "react";

const statusBadgeVariants = cva(
  "inline-flex items-center rounded-md px-2 text-xs leading-5 gap-1",
  {
    variants: {
      variant: {
        enabled: "text-successA-11 bg-successA-3",
        disabled: "text-warningA-11 bg-warningA-3",
        live: "text-feature-11 bg-feature-4",
        rolledBack: "text-warningA-11 bg-warningA-3",
      },
    },
    defaultVariants: {
      variant: "live",
    },
  },
);

interface EnvStatusBadgeProps extends HTMLAttributes<HTMLDivElement> {
  variant?: VariantProps<typeof statusBadgeVariants>["variant"];
  icon?: ReactNode;
  text: string;
}

export const EnvStatusBadge = ({
  variant,
  icon,
  text,
  className,
  ...props
}: EnvStatusBadgeProps) => {
  return (
    <div className={cn(statusBadgeVariants({ variant }), className)} {...props}>
      {icon && <span className="inline-flex items-center">{icon}</span>}
      <span className="font-medium">{text}</span>
    </div>
  );
};
