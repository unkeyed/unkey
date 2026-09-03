import * as flags from ".";

// Add a line per flag here when you declare one in ./index.ts so the
// FlagsProvider exposes it to client components. The Flags type is derived
// from this function's return shape, so any flag missing from this list will
// fail to type-check at every useFlag(key) call site.
export async function resolveAll() {
  const [
    helloWorld,
    deployBilling,
    billingUIUpgrades,
    showDarksoulsSuccessBanner,
    logdrains,
    deployAnomalyAlerts,
    portalManagement,
    projectsNav,
  ] = await Promise.all([
    flags.helloWorld(),
    flags.deployBilling(),
    flags.billingUIUpgrades(),
    flags.showDarksoulsSuccessBanner(),
    flags.logdrains(),
    flags.deployAnomalyAlerts(),
    flags.portalManagement(),
    flags.projectsNav(),
  ]);
  return {
    helloWorld,
    deployBilling,
    billingUIUpgrades,
    showDarksoulsSuccessBanner,
    logdrains,
    deployAnomalyAlerts,
    portalManagement,
    projectsNav,
  };
}

export type Flags = Awaited<ReturnType<typeof resolveAll>>;
