"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SignOutButton, useUser } from "@clerk/nextjs";
import { Book, ChevronRight, LogOut, Monitor, Moon, Rocket, Sun, Zap } from "lucide-react";
import { useTheme } from "next-themes";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React from "react";
export const UserButton: React.FC = () => {
  const { user } = useUser();
  const { theme, setTheme } = useTheme();
  const router = useRouter();

  if (!user) {
    return null;
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="absolute inset-x-0 bottom-0 flex items-center justify-between gap-2 px-6 py-3 hover:bg-gray-200 hover:cursor-pointer">
        <div className="flex items-center gap-2">
          <Avatar className="w-8 h-8">
            {user.imageUrl ? (
              <AvatarImage src={user.imageUrl} alt={user.username ?? "Profile picture"} />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded-md">
              {(user?.fullName ?? "U").slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>

          <span className="text-sm font-semibold">{user.username}</span>
        </div>
        <ChevronRight className="w-4 h-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="right" className="w-96">
        <DropdownMenuGroup>
          <Link href="/onboarding">
            <DropdownMenuItem className="cursor-pointer">
              <Rocket className="w-4 h-4 mr-2" />
              <span>Onboarding</span>
            </DropdownMenuItem>
          </Link>
          <Link href="https://docs.unkey.dev" target="_blank">
            <DropdownMenuItem className="cursor-pointer">
              <Book className="w-4 h-4 mr-2" />
              <span>Docs</span>
            </DropdownMenuItem>
          </Link>
          <Link href="/app/stripe">
            <DropdownMenuItem className="cursor-pointer">
              <Zap className="w-4 h-4 mr-2" />
              <span>Plans & Billing</span>
            </DropdownMenuItem>
          </Link>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>Change Theme</DropdownMenuSubTrigger>
            <DropdownMenuPortal>
              <DropdownMenuSubContent>
                <DropdownMenuCheckboxItem
                  checked={theme === "light"}
                  onCheckedChange={() => setTheme("light")}
                >
                  <div className="flex items-center gap-2 ">
                    <Sun size={16} />
                    Light
                  </div>
                </DropdownMenuCheckboxItem>
                <DropdownMenuCheckboxItem
                  checked={theme === "dark"}
                  onCheckedChange={() => setTheme("dark")}
                >
                  <div className="flex items-center gap-2 ">
                    <Moon size={16} />
                    Dark
                  </div>
                </DropdownMenuCheckboxItem>
                <DropdownMenuCheckboxItem
                  checked={theme === "system"}
                  onCheckedChange={() => setTheme("system")}
                >
                  <div className="flex items-center gap-2 ">
                    <Monitor size={16} />
                    System
                  </div>
                </DropdownMenuCheckboxItem>
              </DropdownMenuSubContent>
            </DropdownMenuPortal>
          </DropdownMenuSub>
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
  );
};
