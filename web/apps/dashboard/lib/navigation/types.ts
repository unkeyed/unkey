import type { ElementType, ReactNode } from "react";

export type ResolvedNavLink = {
  key: string;
  label: ReactNode;
  href: string;
  icon?: ElementType;
  isActive: boolean;
  disabled?: boolean;
  external?: boolean;
  tag?: ReactNode;
};

export type SidebarAction = {
  key: string;
  label: string;
  icon?: ElementType;
  href?: string;
  onClick?: () => void;
  disabled?: boolean;
};

export type SidebarContent = {
  back?: { label: string; href: string };
  links: ResolvedNavLink[];
  actions?: SidebarAction[];
};
