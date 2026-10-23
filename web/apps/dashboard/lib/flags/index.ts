export type WorkspaceFlag = {
  readonly slug: string;
  readonly defaultValue: boolean;
};

export const helloWorld = {
  slug: "hello_world",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const deployBilling = {
  slug: "deploy_billing",
  defaultValue: true,
} as const satisfies WorkspaceFlag;

export const billingUIUpgrades = {
  slug: "billing_ui_upgrades",
  defaultValue: true,
} as const satisfies WorkspaceFlag;

export const showDarksoulsSuccessBanner = {
  slug: "show_darksouls_success_banner",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const projectsNav = {
  slug: "projects_nav",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const portalManagement = {
  slug: "portal_management",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const platformFeatures = {
  slug: "platform_features",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const appV2 = {
  slug: "app_v2",
  defaultValue: false,
} as const satisfies WorkspaceFlag;
