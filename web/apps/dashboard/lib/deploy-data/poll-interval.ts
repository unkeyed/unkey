const IDLE_POLL_MS = 20_000;
const SETTLING_POLL_MS = 5_000;
const AWAITING_POLL_MS = 2_000;

// Faster while a held row is moving, and fastest while this tab waits on an
// action of its own. A page may ask for a faster idle cadence.
export function deploymentsPollIntervalMs({
  awaiting,
  settling,
  requestedMs,
}: {
  awaiting: boolean;
  settling: boolean;
  requestedMs: number | null;
}): number {
  return Math.min(
    awaiting ? AWAITING_POLL_MS : IDLE_POLL_MS,
    settling ? SETTLING_POLL_MS : IDLE_POLL_MS,
    requestedMs ?? IDLE_POLL_MS,
  );
}
