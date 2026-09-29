"use client";
import { EmptyState, EmptyStateDescription, EmptyStateHeader, EmptyStateTitle } from "@unkey/ui";

export function PoliciesError() {
  return (
    <div className="border border-errorA-4 bg-errorA-2 rounded-lg overflow-hidden">
      <div className="flex items-center justify-center py-16 px-4">
        <EmptyState frame="none">
          <EmptyStateHeader>
            <EmptyStateTitle className="text-error-11">Failed to load policies</EmptyStateTitle>
            <EmptyStateDescription>
              Something went wrong while loading policies. Try refreshing the page.
            </EmptyStateDescription>
          </EmptyStateHeader>
        </EmptyState>
      </div>
    </div>
  );
}
