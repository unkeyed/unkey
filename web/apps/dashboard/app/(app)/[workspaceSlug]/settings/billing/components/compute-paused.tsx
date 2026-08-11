export function pausedBody(budgetLabel?: string): string {
  const cap = budgetLabel ? `your ${budgetLabel} spend budget` : "your spend budget";
  return `Workloads stopped because you reached ${cap}. Raise or remove the budget to start them again.`;
}
