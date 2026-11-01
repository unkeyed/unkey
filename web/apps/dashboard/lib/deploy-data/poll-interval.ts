const IDLE_POLL_MS = 20_000;
const IDLE_POLL_MAX_MS = 60_000;
const SETTLING_POLL_MS = 5_000;
const AWAITING_POLL_MS = 2_000;

// Faster while a held row is moving, and fastest while this tab waits on an
// action of its own. A page may ask for a faster idle cadence. Idle polls that
// change nothing back off, since a page left open mostly sees no deploys.
export function deploymentsPollIntervalMs({
  awaiting,
  settling,
  requestedMs,
  unchangedPolls,
}: {
  awaiting: boolean;
  settling: boolean;
  requestedMs: number | null;
  unchangedPolls: number;
}): number {
  const idleMs = Math.min(IDLE_POLL_MS * (1 + unchangedPolls), IDLE_POLL_MAX_MS);
  return Math.min(
    awaiting ? AWAITING_POLL_MS : idleMs,
    settling ? SETTLING_POLL_MS : idleMs,
    requestedMs ?? idleMs,
  );
}
