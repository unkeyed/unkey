"use client";
import { Button } from "@/components/ui/button";
import { BarChart, BookOpen, FileJson, Settings } from "lucide-react";
import Link from "next/link";
import { WorkspaceSwitcher } from "./team-switcher";
import { Workspace } from "@unkey/db";
import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";

type Props = {
  workspace: Workspace;
};

export const MobileSideBar = ({ workspace }: Props) => {
  const [hideNav, setHideNav] = useState(true);
  return (
    <div className=" ">
      <div className=" md:hidden p-4 flex items-center justify-between dark:bg-zinc-900 bg-zinc-100">
        {/* rome-ignore lint/a11y/useKeyWithClickEvents: <explanation> */}
        <svg
          width="24"
          height="24"
          viewBox="0 0 16 16"
          fill="lch(79.412% 1.095 265.851 / 1)"
          className=" fill-zinc-900 dark:fill-zinc-100"
          onClick={() => setHideNav(!hideNav)}
        >
          <title>sidebar</title>
          <path d="M15 5.25A3.25 3.25 0 0 0 11.75 2h-7.5A3.25 3.25 0 0 0 1 5.25v5.5A3.25 3.25 0 0 0 4.25 14h7.5A3.25 3.25 0 0 0 15 10.75v-5.5Zm-3.5 7.25H7v-9h4.5a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2Zm-6 0H4.25a1.75 1.75 0 0 1-1.75-1.75v-5.5c0-.966.784-1.75 1.75-1.75H5.5v9Z" />
        </svg>
        <WorkspaceSwitcher workspace={workspace} />
      </div>
      <AnimatePresence>
        {!hideNav && (
          <motion.aside
            initial={{
              x: "-100%",
              opacity: 0,
            }}
            animate={{
              x: "0%",
              opacity: 1,
            }}
            exit={{
              x: "-100%",
              opacity: 1,
            }}
            transition={{
              ease: "easeInOut",
            }}
            className="fixed h-screen pb-12 border-r lg:inset-y-0 w-4/6 lg:flex-col bg-gradient-to-tr from-zinc-200 to-zinc-100 dark:from-zinc-800 dark:to-zinc-950 border-white/10 z-50"
          >
            <div className="  flex flex-col w-full p-4">
              <h2 className="px-2 mb-2 text-lg font-semibold tracking-tight">
                Workspace
              </h2>
              <div className="space-y-1">
                <Link href="/app/apis">
                  <Button variant="ghost" className="justify-start w-full">
                    <FileJson className="w-4 h-4 mr-2" />
                    APIs
                  </Button>
                </Link>

                <Button
                  variant="ghost"
                  disabled
                  className="justify-start w-full border-t"
                >
                  <BarChart className="w-4 h-4 mr-2" />
                  Audit
                </Button>

                <Link href="/app/keys">
                  <Button
                    variant="ghost"
                    className="justify-start w-full border-t"
                  >
                    <Settings className="w-4 h-4 mr-2" />
                    Settings
                  </Button>
                </Link>
                <Link href="https://docs.unkey.dev" target="_blank">
                  <Button
                    variant="ghost"
                    className="justify-start w-full border-t py-2"
                  >
                    <BookOpen className="w-4 h-4 mr-2" />
                    Docs
                  </Button>
                </Link>
              </div>
            </div>
          </motion.aside>
        )}
      </AnimatePresence>
    </div>
  );
};
