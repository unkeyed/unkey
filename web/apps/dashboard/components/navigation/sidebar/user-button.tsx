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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { signOut } from "@/lib/auth/utils";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Laptop2, MoonStars, Sun } from "@unkey/icons";
import { useTheme } from "next-themes";
import type React from "react";

type UserButtonProps = {
  isCollapsed?: boolean;
  isMobile?: boolean;
  isMobileSidebarOpen?: boolean;
  className?: string;
};

export const UserButton: React.FC<UserButtonProps> = ({
  isCollapsed = false,
  isMobile = false,
  isMobileSidebarOpen = false,
  className,
}) => {
  const { data: user, isLoading } = trpc.user.getCurrentUser.useQuery();
  const { theme, setTheme } = useTheme();
  const queryClient = useQueryClient();

  const displayName = user?.fullName ?? user?.email ?? "";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "px-2 py-1 flex hover:bg-grayA-4 rounded-lg min-w-0",
          isCollapsed ? "justify-center size-8 p-0" : "justify-between gap-2 grow h-8",
          className,
        )}
      >
        <div className="flex items-center gap-2 overflow-hidden whitespace-nowrap">
          <Avatar className="size-5 rounded-full border border-grayA-6">
            {user?.avatarUrl ? (
              <AvatarImage src={user.avatarUrl} alt="Profile picture" className="rounded-full" />
            ) : null}
            <AvatarFallback className="bg-gray-2 rounded-full size-5">
              {user ? (user?.fullName ?? "U").slice(0, 1).toUpperCase() : null}
            </AvatarFallback>
          </Avatar>
          {!isCollapsed || (isMobile && isMobileSidebarOpen) ? (
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
      <DropdownMenuContent side="bottom" className="flex w-min-44 flex-col gap-2" align="start">
        {user?.email && (
          <>
            <DropdownMenuLabel className="font-normal">
              <span title={user.email} className="text-accent-11 text-xs truncate max-w-44 secret">
                {user.email}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
          </>
        )}
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
              queryClient.clear();
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
