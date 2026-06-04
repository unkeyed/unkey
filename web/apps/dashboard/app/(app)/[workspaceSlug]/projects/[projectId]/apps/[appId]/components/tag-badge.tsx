import { cn } from "@/lib/utils";
import { cva } from "class-variance-authority";
import type { HTMLAttributes, ReactNode } from "react";

const tagBadgeVariants = cva(
  "inline-flex items-center rounded-md px-2 text-xs leading-5 gap-1 shrink-0",
  {
    variants: {
      variant: {
        enabled: "text-successA-11 bg-successA-3",
        disabled: "text-warningA-11 bg-warningA-3",
        current: "text-feature-11 bg-feature-4",
        rolledBack: "text-warningA-11 bg-warningA-4",
        custom: "text-featureA-11 bg-featureA-3",
      },
    },
    defaultVariants: {
      variant: "current",
    },
  },
);

export type TagBadgeVariant = "enabled" | "disabled" | "current" | "rolledBack" | "custom";

type TagBadgeProps = HTMLAttributes<HTMLDivElement> & {
  variant?: TagBadgeVariant;
  icon?: ReactNode;
  text: string;
};

export const TagBadge = ({
  variant = "current",
  icon,
  text,
  className,
  ...props
}: TagBadgeProps) => {
  return (
    <div className={cn(tagBadgeVariants({ variant }), className)} {...props}>
      {icon && <span className="inline-flex items-center">{icon}</span>}
      <span className="font-medium">{text}</span>
    </div>
  );
};
