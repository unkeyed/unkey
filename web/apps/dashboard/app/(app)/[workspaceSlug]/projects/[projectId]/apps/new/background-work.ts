export type WorkStatus =
  | { kind: "running" }
  | { kind: "done" }
  | { kind: "failed"; error: unknown };

type Task = () => Promise<unknown>;

type Lane = { tail: Promise<void>; failed: { task: Task; error: unknown } | null };

export type BackgroundWork = {
  /** Runs `task` after every earlier task queued under the same key. */
  run: (key: string, task: Task) => void;
  /** Runs every failed task again. */
  retry: () => void;
  /** Resolves once every queued task has succeeded; rejects with the first failure. */
  settled: () => Promise<void>;
  status: () => WorkStatus;
  subscribe: (listener: () => void) => () => void;
};

export const WORK_DONE: WorkStatus = { kind: "done" };
const running: WorkStatus = { kind: "running" };

export function createBackgroundWork(): BackgroundWork {
  const lanes = new Map<string, Lane>();
  const listeners = new Set<() => void>();
  let active = 0;
  let status = WORK_DONE;

  const refresh = () => {
    const failed = [...lanes.values()].find((lane) => lane.failed)?.failed ?? null;
    status = active > 0 ? running : failed ? { kind: "failed", error: failed.error } : WORK_DONE;
    for (const listener of listeners) {
      listener();
    }
  };

  const run = (key: string, task: Task) => {
    const lane = lanes.get(key) ?? { tail: Promise.resolve(), failed: null };
    lanes.set(key, lane);
    lane.failed = null;
    active++;
    lane.tail = lane.tail
      .then(task)
      .then(
        () => undefined,
        (error: unknown) => {
          lane.failed = { task, error };
        },
      )
      .finally(() => {
        active--;
        refresh();
      });
    refresh();
  };

  const subscribe = (listener: () => void) => {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  };

  return {
    run,
    retry: () => {
      for (const [key, lane] of lanes) {
        if (lane.failed) {
          run(key, lane.failed.task);
        }
      }
    },
    settled: () =>
      new Promise<void>((resolve, reject) => {
        const check = () => {
          if (status.kind === "running") {
            return;
          }
          unsubscribe();
          if (status.kind === "done") {
            resolve();
          } else {
            reject(status.error);
          }
        };
        const unsubscribe = subscribe(check);
        check();
      }),
    status: () => status,
    subscribe,
  };
}
