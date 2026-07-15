export type ExitReason =
  | "OOMKilled"
  | "Error"
  | "ContainerCannotRun"
  | "Completed"
  | "ImagePullBackOff"
  | "CrashLoopBackOff"
  | string; // fallback for unknown reasons

export type StatusReason = "CrashLoopBackOff" | "ImagePullBackOff" | "Pending" | string; // fallback for unknown status reasons

export type LastExit = {
  restartCount: number;
  exitCode: number | null;
  signal: number | null;
  reason: ExitReason | null;
  finishedAt: number | null;
  statusReason: StatusReason | null;
};
