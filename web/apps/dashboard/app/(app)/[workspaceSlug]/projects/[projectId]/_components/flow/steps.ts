export type StepStatus = "done" | "current" | "todo";

export function stepStatus(index: number, at: number): StepStatus {
  if (index < at) {
    return "done";
  }
  return index === at ? "current" : "todo";
}

export function stepLabel(index: number, total: number): string {
  return `Step ${index + 1} of ${total}`;
}
