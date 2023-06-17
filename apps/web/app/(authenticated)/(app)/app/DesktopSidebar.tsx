import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { BarChart, Database, FileKey, Filter, FormInput, Home } from "lucide-react";
import Link from "next/link";
import { AppLink } from "./AppLink";
import { TeamSwitcher } from "./TeamSwitcher";
type Props = {
  workspaceSlug: string;
};

export const DesktopSidebar: React.FC<Props> = ({ workspaceSlug }) => {
  const apps: {
    name: string;
    slug: string;
  }[] = [];

  return (
    <aside className="relative min-h-screen pb-12 border-r fixed lg:inset-y-0 lg:z-50 lg:flex lg:w-72 lg:flex-col border-white/10">
      <Link
        href="/overview"
        className="flex items-center gap-2 px-8 py-6 text-2xl font-semibold tracking-tight duration-200 stroke-zinc-800 dark:text-zinc-200 dark:stroke-zinc-500 dark:hover:stroke-white hover:stroke-zinc-700 hover:text-zinc-700 dark:hover:text-white"
      >
        {/* <Logo className="w-8 h-8 duration-200 " /> */}
        Unkey.dev
      </Link>
      <div className="space-y-4">
        <div className="px-6 py-2">
          <h2 className="px-2 mb-2 text-lg font-semibold tracking-tight">{/* Events */}</h2>
          <div className="space-y-1">
            <Link href="/app">
              <Button variant="ghost" size="sm" className="justify-start w-full">
                <Home className="w-4 h-4 mr-2" />
                Overview
              </Button>
            </Link>
            <Link href="/app/apis">
              <Button variant="ghost" size="sm" className="justify-start w-full">
                <FileKey className="w-4 h-4 mr-2" />
                APIs
              </Button>
            </Link>

            <Button variant="ghost" disabled size="sm" className="justify-start w-full">
              <BarChart className="w-4 h-4 mr-2" />
              Audit
            </Button>
          </div>
        </div>
        <div className="py-2">
          <h2 className="relative px-8 text-lg font-semibold tracking-tight">Apps</h2>
          <ScrollArea className="h-[230px] px-4">
            <div className="p-2 space-y-1">
              {apps
                .sort((a, b) => a.name.localeCompare(b.name))
                .map((app) => (
                  <AppLink key={app.name} href={`/apps/${app.name}`} slug={app.slug} />
                ))}
            </div>
          </ScrollArea>
        </div>
      </div>
      <div className="absolute inset-x-0  bottom-8 mx-6">
        <TeamSwitcher />
      </div>
    </aside>
  );
};
