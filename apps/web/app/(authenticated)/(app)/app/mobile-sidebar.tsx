"use client";
import { Button } from "@/components/ui/button";
import { Sheet, SheetClose, SheetContent, SheetHeader, SheetTrigger } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { BookOpen, FileJson, Menu, Settings } from "lucide-react";
import Link from "next/link";
import { WorkspaceSwitcher } from "./team-switcher";

type Props = {
  className?: string;
};

export const MobileSideBar = ({ className }: Props) => {
  return (
    <div className={cn(className, "w-full")}>
      <Sheet>
        <div className="flex items-center justify-between w-full p-4">
          <SheetTrigger>
            <Menu className="w-6 h-6" />
          </SheetTrigger>
          <WorkspaceSwitcher />
        </div>
        <SheetHeader>
          <SheetClose />
        </SheetHeader>
        <SheetContent
          position="bottom"
          className="bg-white shadow dark:bg-gray-950 rounded-xl"
          size="lg"
        >
          <div className="flex flex-col w-full p-4 ">
            <h2 className="px-2 mb-2 text-lg font-semibold tracking-tight">Workspace</h2>
            <div className="space-y-1">
              <Link href="/app/apis">
                <SheetClose asChild>
                  <Button variant="ghost" className="justify-start w-full">
                    <FileJson className="w-4 h-4 mr-2" />
                    APIs
                  </Button>
                </SheetClose>
              </Link>

              <Link href="/app/keys">
                <SheetClose asChild>
                  <Button variant="ghost" className="justify-start w-full border-t">
                    <Settings className="w-4 h-4 mr-2" />
                    Settings
                  </Button>
                </SheetClose>
              </Link>

              <Link href="https://docs.unkey.dev" target="_blank">
                <Button variant="ghost" className="justify-start w-full py-2 border-t">
                  <BookOpen className="w-4 h-4 mr-2" />
                  Docs
                </Button>
              </Link>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};
