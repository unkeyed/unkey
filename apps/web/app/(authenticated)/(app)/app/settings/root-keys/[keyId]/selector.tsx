"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ChevronsUpDown } from "lucide-react";
import Link from "next/link";
import { useSelectedLayoutSegment } from "next/navigation";
import React from "react";

type Props = {
  keyId: string;
};

export const Selector: React.FC<Props> = ({ keyId }) => {
  const segment = useSelectedLayoutSegment();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="w-[180px] capitalize flex h-8 truncate items-center justify-between hover:border-primary focus:border-primary rounded-md border border-border bg-transparent px-3 py-2 text-sm  placeholder:text-content-subtle   disabled:cursor-not-allowed disabled:opacity-50">
        {segment}
        <ChevronsUpDown className="w-4 h-4 opacity-50" />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <Link href={`/app/settings/root-keys/${keyId}/permissions`}>
          <DropdownMenuItem>Permissions</DropdownMenuItem>
        </Link>
        <Link href={`/app/settings/root-keys/${keyId}/history`}>
          <DropdownMenuItem>History</DropdownMenuItem>
        </Link>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
