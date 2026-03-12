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

  if (!step) {
    if (implicitlyComplete) {
      return { status: "completed", description: completedMessage, duration: undefined };
    }
    if (skippable && isFailed) {
      return { status: "skipped", description: "Skipped", duration: undefined };
    }
    return { status: "pending", description: waitingMessage, duration: undefined };
  }

  const duration = (step.endedAt ?? now) - step.startedAt;

  if (step.error) {
    return { status: "error", description: step.error, duration };
  }

  if (step.completed) {
    return { status: "completed", description: completedMessage, duration };
  }

  return { status: "started", description: inProgressMessage, duration };
}
