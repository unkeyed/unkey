type ReasonCopy = { label: string; body: string; fatalWaiting?: true };

const CANT_PULL: ReasonCopy = {
  label: "Can't pull image",
  body: "Check the image name, tag, and registry access.",
  fatalWaiting: true,
};

const CANT_START: ReasonCopy = {
  label: "Couldn't start",
  body: "Check your start command and entrypoint.",
};

const EXIT_REASONS = {
  CrashLoopBackOff: {
    label: "Keeps crashing",
    body: "Your app exits right after it starts. Check your logs.",
    fatalWaiting: true,
  },
  ImagePullBackOff: CANT_PULL,
  ErrImagePull: CANT_PULL,
  InvalidImageName: {
    label: "Invalid image name",
    body: "Check the image name in your build settings.",
    fatalWaiting: true,
  },
  CreateContainerConfigError: {
    label: "Config error",
    body: "An environment variable or secret is missing.",
    fatalWaiting: true,
  },
  OOMKilled: {
    label: "Out of memory",
    body: "Your app hit its memory limit. Fix a leak or raise the limit.",
  },
  Error: {
    label: "Exited with error",
    body: "Check your logs for the cause.",
  },
  ContainerCannotRun: CANT_START,
  StartError: CANT_START,
  Completed: {
    label: "Exited cleanly",
    body: "Your app stopped on its own. A service should keep running.",
  },
  ContainerStatusUnknown: {
    label: "Lost contact",
    body: "The server it ran on went away.",
  },
} satisfies Record<string, ReasonCopy>;

type KnownReason = keyof typeof EXIT_REASONS;

function isKnownReason(reason: string): reason is KnownReason {
  return Object.hasOwn(EXIT_REASONS, reason);
}

export function reasonCopy(reason: string | null): ReasonCopy | null {
  return reason !== null && isKnownReason(reason) ? EXIT_REASONS[reason] : null;
}

export function isFatalWaitingReason(reason: string | null): reason is string {
  return reasonCopy(reason)?.fatalWaiting === true;
}
