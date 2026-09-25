"use client";
import { EmptyState, EmptyStateDescription, EmptyStateHeader, EmptyStateTitle } from "@unkey/ui";
import Link from "next/link";
import type React from "react";

export const FreeTierAlert: React.FC = () => {
  return (
    <EmptyState>
      <EmptyStateHeader>
        <EmptyStateTitle>You are on the Free tier.</EmptyStateTitle>
        <EmptyStateDescription>
          The Free tier includes 150k requests of free usage.
          <br />
          To unlock additional usage and add team members, upgrade to Pro.{" "}
          <Link
            href="https://unkey.com/pricing"
            target="_blank"
            rel="noopener noreferrer"
            className="underline text-info-11"
          >
            See Pricing
          </Link>
        </EmptyStateDescription>
      </EmptyStateHeader>
    </EmptyState>
  );
};
