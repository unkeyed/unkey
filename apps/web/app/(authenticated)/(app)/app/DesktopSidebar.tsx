import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  BarChart,
  BookOpen,
  Code,
  Database,
  FileJson,
  FileKey,
  Filter,
  FormInput,
  Home,
  Key,
  KeyRound,
  Settings,
} from "lucide-react";
import Link from "next/link";
import { ApiLink } from "./AppLink";
import { WorkspaceSwitcher } from "./TeamSwitcher";
type Props = {
  apis: {
    id: string;
    name: string;
  }[];
};

export const DesktopSidebar: React.FC<Props> = ({ apis }) => {
  return (
    <aside className="relative min-h-screen pb-12 border-r lg:inset-y-0 lg:z-50 lg:flex lg:w-72 lg:flex-col border-white/10">
      <Link
        href="/overview"
        className="flex items-center gap-2 px-8 py-6 text-2xl font-semibold tracking-tight duration-200 stroke-zinc-800 dark:text-zinc-200 dark:stroke-zinc-500 dark:hover:stroke-white hover:stroke-zinc-700 hover:text-zinc-700 dark:hover:text-white"
      >
        {/* <Logo className="w-8 h-8 duration-200 " /> */}
        Unkey.dev
      </Link>
      <div className="space-y-4">
        <div className="px-6 py-2">
          <h2 className="px-2 mb-2 text-lg font-semibold tracking-tight">Workspace</h2>
          <div className="space-y-1">
            {/* <Link href="/app">
              <Button variant="ghost" size="sm" className="justify-start w-full">
                <Home className="w-4 h-4 mr-2" />
                Overview
              </Button>
            </Link> */}
            <Link href="/app/apis">
              <Button variant="ghost" size="sm" className="justify-start w-full">
                <FileJson className="w-4 h-4 mr-2" />
                APIs
              </Button>
            </Link>

            <Button variant="ghost" disabled size="sm" className="justify-start w-full">
              <BarChart className="w-4 h-4 mr-2" />
              Audit
            </Button>

            <Link href="/app/keys">
              <Button variant="ghost" size="sm" className="justify-start w-full">
                <Settings className="w-4 h-4 mr-2" />
                Settings
              </Button>
            </Link>
            <Link href="https://docs.unkey.dev" target="_blank">
              <Button variant="ghost" size="sm" className="justify-start w-full">
                <BookOpen className="w-4 h-4 mr-2" />
                Docs
              </Button>
            </Link>
          </div>
        </div>

        <div className="py-2">
          <h2 className="relative px-8 text-lg font-semibold tracking-tight">Apis</h2>
          <ScrollArea className="h-[230px] px-4">
            <div className="p-2 space-y-1">
              {apis
                .sort((a, b) => a.name.localeCompare(b.name))
                .map((api) => (
                  <ApiLink key={api.id} name={api.name} href={`/app/${api.id}`} id={api.id} />
                ))}
            </div>
          </ScrollArea>
        </div>
      </div>
      <div className="absolute inset-x-0 mx-6 bottom-8">
        <WorkspaceSwitcher />
      </div>
    </aside>
  );
};
