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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { SignOutButton, useUser } from "@clerk/nextjs";
import { Book, ChevronRight, LogOut, Rocket, Settings } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type React from "react";
import { cn } from "@/lib/utils";
import { useSidebar } from "@/components/ui/sidebar";

export const UserButton: React.FC = () => {
  const { user } = useUser();
  const router = useRouter();
  const { isMobile, state, openMobile } = useSidebar();

  // When mobile sidebar is open, we want to show the full component
  const isCollapsed =
    (state === "collapsed" || isMobile) && !(isMobile && openMobile);

  if (!user) {
    return null;
  }

  // Get user display name
  const displayName =
    user.username ?? user.fullName ?? user.primaryEmailAddress?.emailAddress;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "flex items-center rounded-lg hover:bg-background-subtle hover:cursor-pointer text-content",
          isCollapsed
            ? "justify-center w-10 h-10 p-0"
            : "justify-between gap-2 p-2 w-auto lg:w-full h-12"
        )}
      >
        <div
          className={cn(
            "flex items-center gap-2 overflow-hidden whitespace-nowrap",
            isCollapsed ? "justify-center" : ""
          )}
        >
          <Avatar className="w-5 h-5">
            {user.imageUrl ? (
              <AvatarImage src={user.imageUrl} alt="Profile picture" />
            ) : null}
            <AvatarFallback
              className={cn(
                "bg-gray-100 border border-gray-500 rounded-md",
                "w-5 h-5"
              )}
            >
              {(user?.fullName ?? "U").slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          {/* Username - only one instance that's conditionally shown */}
          {!isCollapsed && (
            <Tooltip>
              <TooltipTrigger
                className={cn(
                  "w-full overflow-hidden text-ellipsis",
                  // On desktop: show on small/medium screens, hide on large screens
                  // On mobile with open sidebar: always show
                  !isMobile && "sm:inline lg:hidden"
                )}
                asChild
              >
                <span className="overflow-hidden text-ellipsis text-sm font-medium">
                  {displayName}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <span className="text-sm font-medium">{displayName}</span>
              </TooltipContent>
            </Tooltip>
          )}

          {/* Username on large screens */}
          {!isCollapsed && (
            <Tooltip>
              <TooltipTrigger
                className={cn(
                  "w-full overflow-hidden text-ellipsis",
                  // Only show on large screens on desktop
                  // On mobile with open sidebar: never show this one
                  "hidden",
                  !isMobile && "lg:inline"
                )}
                asChild
              >
                <span className="overflow-hidden text-ellipsis text-sm font-medium">
                  {displayName}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <span className="text-sm font-medium">{displayName}</span>
              </TooltipContent>
            </Tooltip>
          )}
        </div>

        {!isCollapsed ? <ChevronRight className="inline w-4 h-4" /> : null}
      </DropdownMenuTrigger>

      <DropdownMenuContent
        side="bottom"
        className="w-full max-w-xs md:w-96"
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
