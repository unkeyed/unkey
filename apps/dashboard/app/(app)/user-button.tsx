"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Book, ChevronRight, LogOut, Rocket, Settings } from "lucide-react";
import Link from "next/link";

import { signOut } from "@/lib/auth/actions";
import { useUser } from "@/lib/auth/hooks";
import type React from "react";

export const UserButton: React.FC = () => {
  const { user } = useUser();
  if (!user) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between gap-2 p-2 w-auto lg:w-full h-12 rounded-lg hover:bg-background-subtle hover:cursor-pointer text-content ">
        <div className="flex items-center gap-2 overflow-hidden whitespace-nowrap">
          <Tooltip>
            <TooltipTrigger
              className="w-full overflow-hidden text-ellipsis hidden sm:inline lg:hidden"
              asChild
            >
              <span className="overflow-hidden text-ellipsis text-sm font-medium">
                {user.fullName ?? user.email}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <span className="text-sm font-medium">{user.fullName ?? user.email}</span>
            </TooltipContent>
          </Tooltip>
          <Avatar className="w-8 h-8 lg:w-5 lg:h-5">
            {user?.avatarUrl ? <AvatarImage src={user.avatarUrl} alt="Profile picture" /> : null}
            <AvatarFallback className="w-8 h-8 lg:w-5 lg:h-5 bg-gray-100 border border-gray-500 rounded-md">
              {(user.fullName ?? "U").slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>

          <Tooltip>
            <TooltipTrigger
              className="hidden lg:inline w-full overflow-hidden text-ellipsis"
              asChild
            >
              <span className="overflow-hidden text-ellipsis text-sm font-medium">
                {user.fullName ?? user.email}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <span className="text-sm font-medium">{user.fullName ?? user.email}</span>
            </TooltipContent>
          </Tooltip>
        </div>
        <ChevronRight className="hidden lg:inline w-4 h-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="bottom" className="w-full max-w-xs md:w-96">
        <DropdownMenuGroup>
          <Link href="/new">
            <DropdownMenuItem className="cursor-pointer">
              <Rocket className="w-4 h-4 mr-2" />
              <span>Onboarding</span>
            </DropdownMenuItem>
          </Link>
          <Link href="https://unkey.dev/docs" target="_blank">
            <DropdownMenuItem className="cursor-pointer">
              <Book className="w-4 h-4 mr-2" />
              <span>Docs</span>
            </DropdownMenuItem>
          </Link>
          <Link href="/settings/user">
            <DropdownMenuItem className="cursor-pointer">
              <Settings className="w-4 h-4 mr-2" />
              <span>Settings</span>
            </DropdownMenuItem>
          </Link>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem
            asChild
            className="cursor-pointer"
            onClick={async () => {
              await signOut();
            }}
          >
            <span>
              <LogOut className="w-4 h-4 mr-2" />
              Sign out
            </span>
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
