"use client";

import { grantLabel } from "../../lib/grant-label";

type GrantListProps = {
  grants: readonly string[];
};

export function GrantList({ grants }: GrantListProps) {
  return (
    <ul className="flex list-none flex-col gap-3 p-0">
      {grants.map((grant) => {
        const { path, action } = grantLabel(grant);
        return (
          <li key={grant} className="rounded-lg border border-grayA-4 bg-white p-4 dark:bg-black">
            <span className="block truncate text-[13px] text-accent-12">
              {path === null ? null : (
                <span className="font-mono text-xs text-gray-10">{path} — </span>
              )}
              {action}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
