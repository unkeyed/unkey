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
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { User } from "@unkey/icons";
import { useTheme } from "next-themes";
import Link from "next/link";

type UserButtonProps = {
  isCollapsed?: boolean;
  isMobile?: boolean;
  isMobileSidebarOpen?: boolean;
  className?: string;
};

const THEMES = [
  { value: "system", label: "System" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
] as const;

export function UserButton({ isCollapsed = false, className }: UserButtonProps) {
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const workspace = useWorkspaceNavigation();
  const { theme, setTheme } = useTheme();
  const queryClient = useQueryClient();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "px-2 py-1 flex hover:bg-grayA-4 rounded-lg min-w-0 cursor-pointer",
          isCollapsed ? "justify-center size-8 p-0" : "justify-between gap-2 grow h-8",
          className,
        )}
      >
        <div className="flex items-center gap-2 overflow-hidden whitespace-nowrap">
          <Avatar className="size-6 rounded-full border border-grayA-6">
            {user?.avatarUrl && <AvatarImage src={user.avatarUrl} alt="Profile picture" />}
            <AvatarFallback name={user?.email ?? "Username"} />
          </Avatar>
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="bottom" align="end" className="w-60 p-0">
        {user?.email && (
          <div className="border-b border-grayA-4 px-3 py-2.5">
            <span
              title={user.email}
              className="secret block truncate text-[13px] font-medium text-accent-12"
            >
              {user.email}
            </span>
          </div>
        )}
        <DropdownMenuGroup className="p-1">
          <DropdownMenuItem
            className="cursor-pointer gap-2 px-2 py-1.5 font-medium text-accent-12"
            render={
              <Link href={routes.account.overview({ workspaceSlug: workspace.slug })}>
                <User className="size-4 shrink-0 text-gray-11" iconSize="sm-regular" />
                Account settings
              </Link>
            }
          />
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="mx-0" />
        <DropdownMenuGroup className="p-1">
          <DropdownMenuLabel className="px-2">Theme</DropdownMenuLabel>
          <DropdownMenuRadioGroup value={theme ?? "system"} onValueChange={setTheme}>
            {THEMES.map((option) => (
              <DropdownMenuRadioItem
                key={option.value}
                value={option.value}
                className="cursor-pointer rounded-md py-1.5 pl-8 font-medium text-accent-12"
              >
                {option.label}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
        <DropdownMenuSeparator className="mx-0" />
        <DropdownMenuGroup className="p-1">
          <DropdownMenuItem
            className="cursor-pointer gap-2 px-2 py-1.5 font-medium text-accent-12"
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
