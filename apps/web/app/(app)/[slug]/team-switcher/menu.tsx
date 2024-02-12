"use client";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Check, ChevronsUpDown, Plus } from "lucide-react";
import React, { useMemo, useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

import { Workspace } from "@unkey/db";
import Link from "next/link";

type Props = {
  current: {
    id: string;
    name: string;
  };
  workspaces: {
    id: string;
    name: string;
  }[];
};

export const Menu: React.FC<Props> = ({ current, workspaces }): JSX.Element => {
  console.log({ workspaces });
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between w-full h-12 gap-2 px-2 overflow-hidden rounded-md hover:bg-gray-200 dark:hover:bg-gray-800 hover:cursor-pointer whitespace-nowrap ring-0 focus:ring-0 focus:outline-none ">
        <div className="flex items-center gap-2 overflow-hidden whitespace-nowrap">
          <Avatar className="w-6 h-6">
            {/* {current?.imageUrl ? (
              <AvatarImage src={currentOrg.imageUrl} alt={currentOrg.name ?? "Profile picture"} />
            ) : user?.imageUrl ? (
              <AvatarImage
                src={user.imageUrl}
                alt={user?.username ?? user?.fullName ?? "Profile picture"}
              />
            ) : null} */}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {current.name.slice(0, 2).toUpperCase() ?? "P"}
            </AvatarFallback>
          </Avatar>

          <Tooltip>
            <TooltipTrigger className="w-full overflow-hidden text-ellipsis">
              <span className="overflow-hidden text-sm font-semibold text-ellipsis">
                {current.name}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <span className="text-sm font-semibold">{current.name}</span>
            </TooltipContent>
          </Tooltip>
        </div>

        <ChevronsUpDown className="hidden w-3 h-3 shrink-0 md:block" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        side="left"
        className="absolute left-0 w-96 lg:left-8 top-8 max-sm:left-0"
      >
        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          {workspaces.map((ws) => (
            <DropdownMenuItem
              key={ws.id}
              className="flex items-center justify-between"
              // onClick={() => changeOrg(membership.organization.id)}
            >
              <span className={ws.id === current.id ? "font-semibold" : undefined}> {ws.name}</span>
              {ws.id === current.id ? <Check className="w-4 h-4" /> : null}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />

          <DropdownMenuItem>
            <Link href="/new" className="flex items-center">
              <Plus className="w-4 h-4 mr-2" />
              <span>Create Workspace</span>
            </Link>
          </DropdownMenuItem>
          {/* {membership?.role === "admin" ? (
            <Link href="/app/settings/team">
              <DropdownMenuItem>
                <UserPlus className="w-4 h-4 mr-2 " />
                <span className="cursor-pointer">Invite Member</span>
              </DropdownMenuItem>
            </Link>
          ) : null} */}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
