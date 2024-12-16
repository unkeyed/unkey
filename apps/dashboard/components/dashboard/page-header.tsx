import { cn } from "@/lib/utils";
import type React from "react";

type Props = {
  title: React.ReactNode;
  description?: string;
  /**
   * A set of components displayed in the top right
   * null components are filtered out
   */
  actions?: React.ReactNode[];
  /**
   * Additional classes to be applied to the root element
   */
  className?: string;
};

export const PageHeader: React.FC<Props> = ({ title, description, actions, className }) => {
  const actionRows: React.ReactNode[][] = [];
  if (actions) {
    for (let i = 0; i < actions.length; i += 3) {
      actionRows.push(actions.slice(i, i + 3));
    }
  }

  return (
    <div
      className={cn(
        "flex flex-col items-start justify-between w-full gap-2 mb-4 md:items-center md:flex-row md:gap-4",
        className,
      )}
    >
      <div className="space-y-1 truncate">
        <h1 className="text-2xl font-semibold tracking-tight truncate">{title}</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400 truncate">{description}</p>
      </div>
      {actionRows.map((row, i) => (
        <ul
          key={i.toString()}
          className="flex flex-wrap items-center justify-end gap-2 md:gap-4 md:flex-nowrap"
        >
          {row.map((action, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: I got nothing better right now
            <li key={i}>{action}</li>
          ))}
        </ul>
      ))}
    </div>
  );
};
