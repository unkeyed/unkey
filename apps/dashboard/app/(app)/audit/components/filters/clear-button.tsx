"use client";

import { Button } from "@unkey/ui";
import Link from "next/link";

export const ClearButton = () => {
  return (
    <Link href="/audit">
      <Button className="flex items-center h-8 gap-2 bg-transparent">Clear</Button>
    </Link>
  );
};
