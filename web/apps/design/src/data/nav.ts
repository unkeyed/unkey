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
    items: [{ name: "Skeleton", href: "/primitives/skeleton" }],
  },
];
