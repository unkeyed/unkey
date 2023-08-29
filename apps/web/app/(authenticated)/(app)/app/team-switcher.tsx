"use client";

import { Loading } from "@/components/dashboard/loading";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import {
  Book,
  Check,
  ChevronsUpDown,
  LogOut,
  Monitor,
  Moon,
  Plus,
  Rocket,
  Sun,
  Zap,
} from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import { SignOutButton, useOrganization, useOrganizationList, useUser } from "@clerk/nextjs";

import type { Workspace } from "@/lib/db";
import { DropdownMenuSub } from "@radix-ui/react-dropdown-menu";
import { useTheme } from "next-themes";

type Props = {
  workspace: Workspace;
};

export const WorkspaceSwitcher: React.FC<Props> = ({ workspace }): JSX.Element => {
  const { setActive, organizationList } = useOrganizationList();
  const { organization: currentOrg, membership } = useOrganization();
  const { user } = useUser();
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  async function changeOrg(orgId: string | null) {
    if (!setActive) {
      return;
    }
    try {
      setLoading(true);
      await setActive({
        organization: orgId,
      });
    } finally {
      setLoading(false);
    }
  }
  const { setTheme, theme } = useTheme();
  return (
    <DropdownMenu>
      {loading ? (
        <Loading />
      ) : (
        <DropdownMenuTrigger className="flex items-center justify-between gap-4 px-2 py-1 rounded lg:w-full hover:bg-gray-100 dark:hover:bg-gray-800">
          <div className="flex flex-row-reverse items-center justify-start w-full gap-4 lg:flex-row ">
            <Avatar className="w-8 h-8 lg:w-10 lg:h-10">
              {user?.imageUrl ? (
                <AvatarImage src={user.imageUrl} alt={user.fullName ?? "Profile picture"} />
              ) : null}
              <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded-md">
                {(currentOrg?.slug ?? user?.fullName ?? "").slice(0, 2).toUpperCase() ?? "P"}
              </AvatarFallback>
            </Avatar>
            <div className="flex flex-row-reverse items-center gap-4 lg:gap-1 lg:items-start lg:flex-col">
              <span className="text-ellipsis overflow-hidden whitespace-nowrap max-w-[8rem]">
                {currentOrg?.name ?? "Personal"}
              </span>
              <PlanBadge plan={workspace.plan} />
            </div>
          </div>
          <ChevronsUpDown className="hidden w-4 h-4 md:block" />
        </DropdownMenuTrigger>
      )}
      <DropdownMenuContent className="w-full lg:w-56" align="end" forceMount>
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
          <DropdownMenuLabel>Switch Workspace</DropdownMenuLabel>

          <DropdownMenuItem
            onClick={() => changeOrg(null)}
            className={cn("flex items-center justify-between", {
              "bg-gray-100 dark:bg-gray-700 dark:text-gray-100 cursor-pointer": currentOrg === null,
            })}
          >
            <span>Personal</span>
            {currentOrg === null ? <Check className="w-4 h-4" /> : null}
          </DropdownMenuItem>

          <ScrollArea className="max-h-64">
            {organizationList?.map((org) => (
              <DropdownMenuItem
                key={org.organization.id}
                onClick={() => changeOrg(org.organization.id)}
                className={cn("flex items-center justify-between", {
                  "bg-gray-100 dark:bg-gray-700 dark:text-gray-100 cursor-pointer":
                    currentOrg?.slug === org.organization.slug,
                })}
              >
                <span>{org.organization.name}</span>
                {currentOrg?.slug === org.organization.slug ? <Check className="w-4 h-4" /> : null}
              </DropdownMenuItem>
            ))}
            <ScrollBar />
          </ScrollArea>
        </DropdownMenuGroup>

        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <Link href="/new">
            <DropdownMenuItem>
              <Plus className="w-4 h-4 mr-2 " />
              <span className="cursor-pointer">Create Workspace</span>
            </DropdownMenuItem>
          </Link>
          {membership?.role === "admin" ? (
            <Link href="/app/team/invite">
              <DropdownMenuItem>
                <Plus className="w-4 h-4 mr-2 " />
                <span className="cursor-pointer">Invite Member</span>
              </DropdownMenuItem>
            </Link>
          ) : null}
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

const PlanBadge: React.FC<{ plan: Workspace["plan"] }> = ({ plan }) => {
  return (
    <span
      className={cn(" inline-flex items-center  font-medium py-0.5 text-xs uppercase  rounded-md", {
        "text-gray-800 dark:text-gray-300": plan === "free",
        "text-primary-foreground  bg-primary px-2 border border-primary-500": plan === "pro",
        "text-white bg-black px-2 border border-black": plan === "enterprise",
        "text-red-600 bg-red-100 px-2 border border-red-500": !plan,
      })}
    >
      {(plan ?? "N/A").toUpperCase()}
    </span>
  );
};
