export interface NavItem {
  name: string;
  href: string;
}

export interface NavSection {
  title: string;
  items: NavItem[];
}

export const nav: NavSection[] = [
  {
    title: "Primitives",
    items: [
      { name: "Alert banner", href: "/primitives/alert-banner" },
      { name: "Item", href: "/primitives/item" },
      { name: "Resource list", href: "/primitives/resource-list" },
      { name: "Skeleton", href: "/primitives/skeleton" },
    ],
  },
  {
    title: "Charts",
    items: [{ name: "Meter", href: "/charts/meter" }],
  },
  {
    title: "Patterns",
    items: [
      { name: "Layout", href: "/patterns/layout" },
      { name: "Resource list page", href: "/patterns/resource-list-page" },
    ],
  },
];
