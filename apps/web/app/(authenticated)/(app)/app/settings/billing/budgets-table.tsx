"use client";

import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Check, Minus, X } from "lucide-react";
import type { BudgetsSectionProps } from "./budgets";
import { DeleteBudgetButton } from "./delete-budget-button";
import { EditBudgetButton } from "./edit-budget-button";

export const BudgetsTable: React.FC<BudgetsSectionProps> = ({ budgets, currentBilling }) => {
  return (
    <Table>
      {budgets.length === 0 ? <TableCaption>No budgets found</TableCaption> : null}
      <TableHeader>
        <TableRow>
          <TableHead>Enabled</TableHead>
          <TableHead>Name</TableHead>
          <TableHead>Budget</TableHead>
          <TableHead>Amount used</TableHead>
          <TableHead>Current vs. budgeted</TableHead>
          <TableHead />
        </TableRow>
      </TableHeader>
      <TableBody>
        {budgets.map((budget, i) => {
          const usage = Math.floor((currentBilling / budget.fixedAmount) * 10000) / 100;

          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
            <TableRow key={i}>
              <TableCell>{budget.enabled ? <Check /> : <X />}</TableCell>
              <TableCell className="w-fit">
                {budget.name ? (
                  <Badge variant="secondary">{budget.name}</Badge>
                ) : (
                  <Minus className="w-4 h-4 text-gray-300" />
                )}
              </TableCell>
              <TableCell>${budget.fixedAmount}</TableCell>
              <TableCell>${currentBilling}</TableCell>
              <TableCell>
                <div className="flex items-center gap-2 text-content-subtle text-xs max-w-40">
                  <Progress value={usage} className="h-2" />
                  {`${usage}%`}
                </div>
              </TableCell>
              <TableCell className="flex items-center gap-2">
                <EditBudgetButton budget={budget} />
                <DeleteBudgetButton budgetId={budget.id} />
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
};
