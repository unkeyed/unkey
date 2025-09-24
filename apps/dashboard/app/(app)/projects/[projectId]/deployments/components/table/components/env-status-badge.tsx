import { cn } from "@/lib/utils";
import { InfoTooltip } from "@unkey/ui";
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
        rolledBack: "text-warningA-11 bg-warningA-4",
      },
    },
    defaultVariants: {
      variant: "live",
    },
  },
);

const tooltipContent = {
  enabled: "This environment is enabled and ready to receive deployments.",
  disabled: "This environment is disabled and cannot receive deployments.",
  live: "This environment is currently receiving live traffic.",
  rolledBack: "This environment was previously live but has been rolled back.",
} as const;

type EnvStatusBadgeProps = HTMLAttributes<HTMLDivElement> & {
  variant?: VariantProps<typeof statusBadgeVariants>["variant"];
  icon?: ReactNode;
  text: string;
};

export const EnvStatusBadge = ({
  variant = "live",
  icon,
  text,
  className,
  ...props
}: EnvStatusBadgeProps) => {
  return (
    <InfoTooltip
      content={tooltipContent[variant as Exclude<typeof variant, null>]}
      variant="inverted"
    >
      <div className={cn(statusBadgeVariants({ variant }), className)} {...props}>
        {icon && <span className="inline-flex items-center">{icon}</span>}
        <span className="font-medium">{text}</span>
      </div>
    </InfoTooltip>
  );
};
