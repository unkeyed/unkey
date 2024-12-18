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
import { initiateSignOut } from "../auth/actions";
// import { SignOutButton, useUser } from "@clerk/nextjs";
import { Book, ChevronRight, LogOut, Rocket, Settings } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type React from "react";
import { useEffect } from "react";
import { useUser } from "@/lib/auth/hooks/useUser";
export const UserButton: React.FC = () => {

  const router = useRouter();
  const { user, isLoading: userLoading } = useUser();
  // Handle authentication check in an effect
  useEffect(() => {
    if (!userLoading && !user) {
      router.push("/auth/sign-in");
    }
  }, [user, userLoading, router]);

  async function handleSignOut() {
    return await initiateSignOut();
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between gap-2 p-2 w-full h-12 rounded-[0.625rem] hover:bg-background-subtle hover:cursor-pointer text-content">
        <div className="flex items-center gap-2 whitespace-nowrap overflow-hidden">
          <Avatar className="w-5 h-5">
            {user?.avatarUrl ? <AvatarImage src={user.avatarUrl} alt="Profile picture" /> : null}
            <AvatarFallback className=" w-5 h-5 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded-md">
              {(user?.fullName ?? "U").slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>

          <Tooltip>
            <TooltipTrigger asChild>
              <span className="overflow-hidden text-ellipsis text-sm font-medium">
                {user?.fullName ?? user?.email}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <span className="text-sm font-medium">
                {user?.fullName ?? user?.email}
              </span>
            </TooltipContent>
          </Tooltip>
        </div>
        <ChevronRight className="w-4 h-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="right" className="w-96">
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
            <DropdownMenuItem asChild className="cursor-pointer" onClick={handleSignOut}>
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
