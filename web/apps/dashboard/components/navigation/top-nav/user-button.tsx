"use client";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { signOut } from "@/lib/auth/utils";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { useQueryClient } from "@tanstack/react-query";
import {
  IconLaptop2Outline18,
  IconMoonStarsOutline18,
  IconSunOutline18,
  IconUserOutline18,
} from "@unkey/icons";
import { useTheme } from "next-themes";
import Link from "next/link";

const THEMES = [
  { value: "system", label: "System", icon: IconLaptop2Outline18 },
  { value: "light", label: "Light", icon: IconSunOutline18 },
  { value: "dark", label: "Dark", icon: IconMoonStarsOutline18 },
] as const;

export function UserButton() {
  const { user } = useWorkspace();
  const workspace = useWorkspaceNavigation();
  const { theme, setTheme } = useTheme();
  const queryClient = useQueryClient();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="Account menu"
        className="group/user flex shrink-0 cursor-pointer rounded-full"
      >
        <Avatar className="size-6 rounded-full border border-input transition-colors group-hover/user:border-strong group-data-[popup-open]/user:border-strong">
          {user?.avatarUrl && <AvatarImage src={user.avatarUrl} alt="Profile picture" />}
          <AvatarFallback name={user?.email ?? "Username"} />
        </Avatar>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="bottom" align="end" className="w-56 p-0">
        {user?.email && (
          <DropdownMenuGroup className="border-b px-2 py-2">
            <DropdownMenuLabel
              title={user.email}
              className="secret block truncate px-0 py-0 text-[13px] text-gray-12"
            >
              {user.email}
            </DropdownMenuLabel>
          </DropdownMenuGroup>
        )}
        <DropdownMenuGroup className="p-1">
          <DropdownMenuItem
            className="h-8 cursor-pointer gap-2 px-2 text-[13px] font-medium text-gray-12"
            render={
              <Link href={routes.account.overview({ workspaceSlug: workspace.slug })}>
                <IconUserOutline18 className="size-4 shrink-0 text-gray-11" />
                Account settings
              </Link>
            }
          />
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="mx-0" />
        <DropdownMenuGroup className="p-1">
          <DropdownMenuLabel className="px-2">Theme</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            aria-label="Theme"
            value={theme ?? "system"}
            onValueChange={setTheme}
          >
            {THEMES.map(({ value, label, icon: Icon }) => (
              <DropdownMenuRadioItem
                key={value}
                value={value}
                className="h-8 cursor-pointer px-2 text-[13px] font-medium text-gray-12"
              >
                <Icon className="size-4 shrink-0 text-gray-11" />
                {label}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="mx-0" />
        <DropdownMenuGroup className="p-1">
          <DropdownMenuItem
            className="h-8 cursor-pointer gap-2 px-2 text-[13px] font-medium text-gray-12"
            onClick={async () => {
              queryClient.clear();
              await signOut();
            }}
          >
            Sign out
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
