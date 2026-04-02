import { P, match } from "@unkey/match";

type StepData = {
  startedAt: number;
  endedAt: number | null;
  duration: number | null;
  completed: boolean;
  error: string | null;
};

type DeploymentStepContext = {
  step: StepData | null | undefined;
  now: number;
  isFailed: boolean;
  skippable: boolean;
  implicitlyComplete?: boolean;
  completedMessage: string;
  inProgressMessage: string;
  waitingMessage: string;
};

type DeploymentStepResolution = {
  status: "pending" | "started" | "completed" | "error" | "skipped";
  description: string;
  duration: number | undefined;
};

export function resolveDeploymentStep(ctx: DeploymentStepContext): DeploymentStepResolution {
  const {
    step,
    now,
    isFailed,
    skippable,
    implicitlyComplete,
    completedMessage,
    inProgressMessage,
    waitingMessage,
  } = ctx;

  const duration = step ? (step.endedAt ?? now) - step.startedAt : undefined;

  return match(step)
    .returnType<DeploymentStepResolution>()
    .with(P.nullish, () => Boolean(implicitlyComplete), () => ({
      status: "completed", description: completedMessage, duration: undefined,
    }))
    .with(P.nullish, () => skippable && isFailed, () => ({
      status: "skipped", description: "Skipped", duration: undefined,
    }))
    .with(P.nullish, () => ({
      status: "pending", description: waitingMessage, duration: undefined,
    }))
    .when(
      (s) => Boolean(s?.error),
      (s) => ({ status: "error", description: s?.error ?? "", duration }),
    )
    .when(
      (s) => Boolean(s?.completed),
      () => ({ status: "completed", description: completedMessage, duration }),
    )
    .otherwise(() => ({ status: "started", description: inProgressMessage, duration }));
}
