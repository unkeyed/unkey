export type SaveState =
  | { status: "ready" }
  | { status: "disabled"; reason?: string }
  | { status: "saving" };

export function firstMatchingSaveState(
  checks: ReadonlyArray<readonly [boolean, SaveState]>,
): SaveState {
  for (const [condition, state] of checks) {
    if (condition) {
      return state;
    }
  }
  return { status: "ready" };
}

export function formSaveState({
  isSubmitting,
  isValid,
  isDirty,
  blockedReason,
}: {
  isSubmitting: boolean;
  isValid: boolean;
  isDirty: boolean;
  blockedReason?: string;
}): SaveState {
  return firstMatchingSaveState([
    [isSubmitting, { status: "saving" }],
    [blockedReason !== undefined, { status: "disabled", reason: blockedReason }],
    [!isValid, { status: "disabled", reason: "Fix the invalid fields to save" }],
    [!isDirty, { status: "disabled" }],
  ]);
}

export type GroupSave<T> =
  | { status: "clean" }
  | { status: "saving" }
  | { status: "blocked"; reasons: string[] }
  | { status: "ready"; submit: T[]; dirty: number };

export function resolveGroupSave<T extends { dirty: boolean; saveState: SaveState }>(
  members: ReadonlyArray<T>,
): GroupSave<T> {
  if (members.some((member) => member.saveState.status === "saving")) {
    return { status: "saving" };
  }
  const dirty = members.filter((member) => member.dirty);
  if (dirty.length === 0) {
    return { status: "clean" };
  }
  const submit = dirty.filter((member) => member.saveState.status === "ready");
  if (submit.length > 0) {
    return { status: "ready", submit, dirty: dirty.length };
  }
  const reasons = [
    ...new Set(
      dirty.flatMap((member) =>
        member.saveState.status === "disabled" && member.saveState.reason
          ? [member.saveState.reason]
          : [],
      ),
    ),
  ];
  return { status: "blocked", reasons };
}
