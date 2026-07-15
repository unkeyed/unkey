import type { ElementType, ReactNode } from "react";

export type ResolvedNavLink = {
  key: string;
  label: ReactNode;
  href: string;
  icon?: ElementType;
  isActive: boolean;
  disabled?: boolean;
  external?: boolean;
  separatorAbove?: boolean;
  tag?: ReactNode;
};
