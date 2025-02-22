"use client";
import { createWorkspaceNavigation, resourcesNavigation } from "@/app/(app)/workspace-navigations";
import { Feedback } from "@/components/dashboard/feedback-component";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetClose, SheetContent, SheetHeader, SheetTrigger } from "@/components/ui/sheet";
import type { Workspace } from "@/lib/db";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { Menu } from "lucide-react";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { WorkspaceSwitcher } from "./team-switcher";
import { UserButton } from "./user-button";

type Props = {
  className?: string;
  workspace: Workspace & {
    apis: {
      id: string;
      name: string;
    }[];
  };
};

export const MobileSideBar = ({ className, workspace }: Props) => {
  const segments = useSelectedLayoutSegments() ?? [];
  const workspaceNavigation = createWorkspaceNavigation(workspace, segments);

  return (
    <div className={cn(className, "w-full")}>
      <Sheet>
        <div className="flex items-center justify-between w-full p-4 gap-6">
          <div className={cn(className, "w-96 flex items-center justify-between py-4 gap-6")}>
            <SheetTrigger>
              <Menu className="w-6 h-6 " />
            </SheetTrigger>
            <WorkspaceSwitcher workspace={workspace} />
          </div>
          <UserButton />
        </div>
        <SheetHeader>
          <SheetClose />
        </SheetHeader>
        <SheetContent side="bottom" className="bg-white shadow dark:bg-gray-950 rounded-xl">
          <div className="flex flex-col w-full p-4 ">
            <h2 className="px-2 mb-2 text-lg font-semibold tracking-tight">Workspace</h2>
            <div className="space-y-1">
              {workspaceNavigation.map((item) => (
                <Link href={`${item.href}`} key={item.label}>
                  <SheetClose asChild>
                    <Button variant="ghost" className="justify-start w-full">
                      <item.icon className="w-4 h-4 mr-2" />
                      {item.label}
                    </Button>
                  </SheetClose>
                </Link>
              ))}
            </div>

            <Separator className="my-2" />
            <h2 className="px-2 mb-2 text-lg font-semibold tracking-tight">Resources</h2>
            <div className="space-y-1">
              {resourcesNavigation.map((item) => (
                <Link href={`${item.href}`} target="_blank" key={item.label}>
                  <SheetClose asChild>
                    <Button variant="ghost" className="justify-start w-full">
                      <item.icon className="w-4 h-4 mr-2" />
                      {item.label}
                    </Button>
                  </SheetClose>
                </Link>
              ))}
              <Feedback />
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};
