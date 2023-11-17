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
import { SignOutButton, useUser } from "@clerk/nextjs";
import { Book, ChevronRight, LogOut, Rocket, Settings } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React from "react";
import { AppVersion } from "./AppVersion";
export const UserButton: React.FC = () => {
  const { user } = useUser();
  const router = useRouter();

  if (!user) {
    return null;
  }
  return (
    <div className="absolute inset-x-0 bottom-0">
      <DropdownMenu>
        <DropdownMenuTrigger className="w-full flex items-center justify-between gap-2 px-6 py-3 hover:bg-gray-200 dark:hover:bg-gray-800 hover:cursor-pointer">
          <div className="flex items-center gap-2">
            <Avatar className="w-8 h-8">
              {user.imageUrl ? (
                <AvatarImage
                  src={user.imageUrl}
                  alt={user.username ?? user.fullName ?? "Profile picture"}
                />
              ) : null}
              <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded-md">
                {(user?.fullName ?? "U").slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>

            <span className="text-sm font-semibold">
              {user.username ?? user.fullName ?? user.primaryEmailAddress?.emailAddress}
            </span>
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
            <Link href="/app/settings/user">
              <DropdownMenuItem className="cursor-pointer">
                <Settings className="w-4 h-4 mr-2" />
                <span>Settings</span>
              </DropdownMenuItem>
            </Link>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <SignOutButton signOutCallback={() => router.push("/auth/sign-in")}>
              <DropdownMenuItem asChild className="cursor-pointer">
                <span>
                  <LogOut className="w-4 h-4 mr-2" />
                  Sign out
                </span>
              </DropdownMenuItem>
            </SignOutButton>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <AppVersion />
    </div>
  );
};
