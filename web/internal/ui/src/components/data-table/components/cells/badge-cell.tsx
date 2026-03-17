import { Badge } from "../../../badge";

export interface BadgeCellProps {
  children: React.ReactNode;
  variant?: "primary" | "secondary" | "success" | "warning" | "error" | "blocked";
  className?: string;
}

/**
 * Generic badge cell component
 */
export function BadgeCell({ children, variant = "primary", className }: BadgeCellProps) {
  return (
    <Badge variant={variant} className={className}>
      {children}
    </Badge>
  );
}
