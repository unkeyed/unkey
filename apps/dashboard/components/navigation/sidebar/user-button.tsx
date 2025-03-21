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
import { useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";
import { SignOutButton, useUser } from "@clerk/nextjs";
import { Book, ChevronRight, LogOut, Rocket, Settings } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type React from "react";

export const UserButton: React.FC = () => {
  const { user } = useUser();
  const router = useRouter();
  const { isMobile, state, openMobile } = useSidebar();

  // When mobile sidebar is open, we want to show the full component
  const isCollapsed = (state === "collapsed" || isMobile) && !(isMobile && openMobile);

  if (!user) {
    return null;
  }

  // Get user display name
  const displayName = user.username ?? user.fullName ?? user.primaryEmailAddress?.emailAddress;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "flex items-center rounded-lg hover:bg-background-subtle hover:cursor-pointer text-content",
          isCollapsed
            ? "justify-center w-10 h-10 p-0"
            : "justify-between gap-2 p-2 w-auto lg:w-full h-10",
        )}
      >
        <div
          className={cn(
            "flex items-center gap-2 overflow-hidden whitespace-nowrap",
            isCollapsed ? "justify-center" : "",
          )}
        >
          <Avatar className="w-6 h-6 rounded-full border border-grayA-6">
            {user.imageUrl ? (
              <AvatarImage src={user.imageUrl} alt="Profile picture" className="rounded-full" />
            ) : null}
            <AvatarFallback
              className={cn("bg-gray-2 border border-grayA-6 rounded-full", "w-6 h-6")}
            >
              {(user?.fullName ?? "U").slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          {/* Show username when not collapsed OR when on mobile with sidebar open */}
          {(!isCollapsed || (isMobile && openMobile)) && (
            <span className="overflow-hidden text-ellipsis text-sm font-medium">{displayName}</span>
          )}
        </div>
        {/* Show chevron when not collapsed OR when on mobile with sidebar open */}
        {(!isCollapsed || (isMobile && openMobile)) && <ChevronRight className="inline w-4 h-4" />}
      </DropdownMenuTrigger>
      <DropdownMenuContent
        side="bottom"
        className="w-full max-w-xs md:w-96 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg"
        align={isMobile ? "center" : "end"}
      >
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
