"use client";
import { Empty } from "@unkey/ui";
import Link from "next/link";
import type React from "react";

export const FreeTierAlert: React.FC = () => {
  return (
    <Empty className="border border-gray-4 rounded-xl">
      <Empty.Title>You are on the Free tier.</Empty.Title>
      <Empty.Description>
        The Free tier includes 150k requests of free usage.
        <br />
        To unlock additional usage and add team members, upgrade to Pro.{" "}
        <Link href="https://unkey.com/pricing" target="_blank" className="underline text-info-11">
          See Pricing
        </Link>
      </Empty.Description>
    </Empty>
  );
};
