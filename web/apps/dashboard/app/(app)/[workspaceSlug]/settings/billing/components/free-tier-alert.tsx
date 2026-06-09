"use client";
import { cn } from "@/lib/utils";
import Link from "next/link";
import type React from "react";
import { billingStripes } from "./billing-card";

export const FreeTierAlert: React.FC = () => {
  return (
    <div className={cn("w-full border border-grayA-4 px-5 py-6", billingStripes)}>
      <div className="flex flex-col gap-1.5">
        <span className="font-mono text-[11px] text-gray-9 uppercase tracking-wider">
          Free tier
        </span>
        <p className="font-medium text-gray-12 text-sm">You are on the Free tier.</p>
        <p className="text-[13px] text-gray-10 leading-snug">
          The Free tier includes 150k requests of free usage. To unlock additional usage and add
          team members, upgrade to Pro.{" "}
          <Link
            href="https://unkey.com/pricing"
            target="_blank"
            rel="noopener noreferrer"
            className="text-gray-12 underline underline-offset-2"
          >
            See pricing
          </Link>
        </p>
      </div>
    </div>
  );
};
