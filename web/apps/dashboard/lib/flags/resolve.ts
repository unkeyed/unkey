import * as flags from ".";

// Add a line per flag here when you declare one in ./index.ts so the
// FlagsProvider exposes it to client components. The Flags type is derived
// from this function's return shape, so any flag missing from this list will
// fail to type-check at every useFlag(key) call site.
export async function resolveAll() {
  const [helloWorld, deployBilling] = await Promise.all([
    flags.helloWorld(),
    flags.deployBilling(),
  ]);
  return { helloWorld, deployBilling };
}

export type Flags = Awaited<ReturnType<typeof resolveAll>>;
