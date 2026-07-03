import * as flags from ".";

// Add a line per flag here when you declare one in ./index.ts so the
// FlagsProvider exposes it to client components. The Flags type is derived
// from this function's return shape, so any flag missing from this list will
// fail to type-check at every useFlag(key) call site.
export async function resolveAll() {
  const [helloWorld, newNavigation, deletionRecoveryPage, deployBilling, appOverview] =
    await Promise.all([
      flags.helloWorld(),
      flags.newNavigation(),
      flags.deletionRecoveryPage(),
      flags.deployBilling(),
      flags.appOverview(),
    ]);
  return { helloWorld, newNavigation, deletionRecoveryPage, deployBilling, appOverview };
}

export type Flags = Awaited<ReturnType<typeof resolveAll>>;
