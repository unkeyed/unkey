"use client";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSidebar } from "@/components/ui/sidebar";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { signOut } from "@/lib/auth/utils";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Laptop2, MoonStars, Sun } from "@unkey/icons";
import { useTheme } from "next-themes";
import type React from "react";

export const UserButton: React.FC = () => {
  const { isMobile, state, openMobile } = useSidebar();
  const { data: user, isLoading } = trpc.user.getCurrentUser.useQuery();

  const { theme, setTheme } = useTheme();

  // When mobile sidebar is open, we want to show the full component
  const isCollapsed = (state === "collapsed" || isMobile) && !(isMobile && openMobile);

  // Get user display name
  const displayName = user?.fullName ?? user?.email ?? "";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "px-2 py-1 flex hover:bg-grayA-4 rounded-lg min-w-0",
          isCollapsed ? "justify-center size-8 p-0" : "justify-between gap-2 flex-grow h-8",
        )}
      >
        <div className={cn("flex items-center gap-2 overflow-hidden whitespace-nowrap")}>
          <Avatar className="size-5 rounded-full border border-grayA-6">
            {user?.avatarUrl ? (
              <AvatarImage src={user.avatarUrl} alt="Profile picture" className="rounded-full" />
            ) : null}
            <AvatarFallback
              className={cn("bg-gray-2 rounded-full", "size-5")}
            >
              {user ? (user?.fullName ?? "U").slice(0, 1).toUpperCase() : null}
            </AvatarFallback>
          </Avatar>
          {/* Show username when not collapsed OR when on mobile with sidebar open */}
          {!isCollapsed || (isMobile && openMobile) ? (
            isLoading ? (
              <div className="bg-gray-5 animate-pulse rounded-lg w-24 h-4" />
            ) : (
              <span className="overflow-hidden text-ellipsis text-accent-12 text-sm font-medium">
                {displayName}
              </span>
            )
          ) : null}
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        side="bottom"
        className="flex w-min-44 flex-col gap-2"
        align={isMobile ? "center" : "end"}
      >
        <DropdownMenuGroup className="w-full">
          <DropdownMenuLabel>Theme</DropdownMenuLabel>

          <Tabs value={theme} onValueChange={setTheme}>
            <TabsList className="w-full">
              <TabsTrigger className="w-full" value="light">
                <Sun className="size-4" />
              </TabsTrigger>
              <TabsTrigger className="w-full" value="dark">
                <MoonStars className="size-4" />
              </TabsTrigger>
              <TabsTrigger className="w-full" value="system">
                <Laptop2 className="size-4" />
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup className="w-full">
          <DropdownMenuItem
            asChild
            className="cursor-pointer"
            onClick={async () => {
              await signOut();
            }}
          >
            <span className="text-accent-12 text-sm font-medium">Sign out</span>
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
