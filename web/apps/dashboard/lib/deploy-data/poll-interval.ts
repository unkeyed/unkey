const POLL_IDLE_MS = 20_000;
const POLL_IDLE_MAX_MS = 60_000;
const POLL_SETTLING_MS = 5_000;
const POLL_AWAITING_MS = 2_000;

// Faster while a deployment of the scope is settling, and fastest while this tab waits on an
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
  const idleMs = Math.min(POLL_IDLE_MS * (1 + unchangedPolls), POLL_IDLE_MAX_MS);
  return Math.min(
    awaiting ? POLL_AWAITING_MS : idleMs,
    settling ? POLL_SETTLING_MS : idleMs,
    requestedMs ?? idleMs,
  );
}
