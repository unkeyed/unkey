export interface NavItem {
  label: string;
  href: string;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export const nav: NavGroup[] = [
  {
    label: "Getting Started",
    items: [{ label: "Introduction", href: "/" }],
  },
  {
    label: "Foundations",
    items: [
      { label: "Colors", href: "/colors" },
      { label: "Icons", href: "/icons" },
    ],
  },
  {
    label: "Components",
    items: [
      { label: "Alert", href: "/components/alert" },
      { label: "Badge", href: "/components/badge" },
      { label: "Button", href: "/components/button" },
      { label: "Card", href: "/components/card" },
      { label: "CircleProgress", href: "/components/circle-progress" },
      { label: "Code", href: "/components/code" },
      { label: "CopyButton", href: "/components/copy-button" },
      { label: "DateTime", href: "/components/date-time" },
      { label: "Dialog", href: "/components/dialog" },
      { label: "Drawer", href: "/components/drawer" },
      { label: "Drover", href: "/components/drover" },
      { label: "Empty", href: "/components/empty" },
      { label: "Id", href: "/components/id" },
      { label: "InfoTooltip", href: "/components/info-tooltip" },
      { label: "InlineLink", href: "/components/inline-link" },
      { label: "KeyboardButton", href: "/components/keyboard-button" },
      { label: "Loading", href: "/components/loading" },
      { label: "Popover", href: "/components/popover" },
      { label: "RefreshButton", href: "/components/refresh-button" },
      { label: "Separator", href: "/components/separator" },
      { label: "SettingsCard", href: "/components/settings-card" },
      { label: "SlidePanel", href: "/components/slide-panel" },
      { label: "Slider", href: "/components/slider" },
      { label: "StepWizard", href: "/components/step-wizard" },
      { label: "Tabs", href: "/components/tabs" },
      { label: "TimestampInfo", href: "/components/timestamp-info" },
      { label: "Toaster", href: "/components/toaster" },
      { label: "Tooltip", href: "/components/tooltip" },
      { label: "VisibleButton", href: "/components/visible-button" },
      { label: "VisuallyHidden", href: "/components/visually-hidden" },
    ],
  },
];

export const flatNav: NavItem[] = nav.flatMap((g) => g.items);
