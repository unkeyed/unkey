export type WorkspaceFlag = {
  readonly slug: string;
  readonly defaultValue: boolean;
};

export const helloWorld = {
  slug: "hello-world",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const deployBilling = {
  slug: "deploy-billing",
  defaultValue: true,
} as const satisfies WorkspaceFlag;

export const billingUIUpgrades = {
  slug: "billing-ui-upgrades",
  defaultValue: true,
} as const satisfies WorkspaceFlag;

export const showDarksoulsSuccessBanner = {
  slug: "show-darksouls-success-banner",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const projectsNav = {
  slug: "projects-nav",
  defaultValue: false,
} as const satisfies WorkspaceFlag;

export const portalManagement = {
  slug: "portal-management",
  defaultValue: false,
} as const satisfies WorkspaceFlag;
