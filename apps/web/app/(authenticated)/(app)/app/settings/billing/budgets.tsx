import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { Budget } from "@unkey/db";
import { BudgetsTable } from "./budgets-table";
import { CreateBudgetButton } from "./create-budget-button";

export type BudgetsSectionProps = {
  currentBilling: number;
  budgets: Budget[];
};

export function BudgetsSection({ currentBilling, budgets }: BudgetsSectionProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex justify-between items-center">
          Budgets
          <CreateBudgetButton />
        </CardTitle>
        <CardDescription>
          Set custom budgets that alert you when your costs and usage exceed your budgeted amount.
        </CardDescription>
      </CardHeader>

      <CardContent>
        <BudgetsTable currentBilling={currentBilling} budgets={budgets} />
      </CardContent>
    </Card>
  );
}
