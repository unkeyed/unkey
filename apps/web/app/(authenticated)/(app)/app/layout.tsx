import { DesktopSidebar } from "@/components/desktop-sidebar";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@unkey/db";
import { redirect } from "next/navigation";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { MobileSideBar } from "../../../../components/mobile-sidebar";
interface LayoutProps {
  children: React.ReactNode;
  params: {
    workspaceSlug: string;
  };
}

export default async function Layout({ children }: LayoutProps) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: true,
    },
  });
  if (!workspace) {
    return redirect("/app/apis");
  }

  return (
    <>
      <div className="flex flex-col md:flex-row min-h-screen bg-gradient-to-tl">
        <DesktopSidebar workspace={workspace} className=" hidden md:block" />
        <MobileSideBar workspace={workspace} />
        <div className="w-full md:m-2 bg-gradient-to-br from-white to-zinc-100 dark:from-zinc-900 dark:to-zinc-950 shadow md:ml-72 rounded-xl">
          <ScrollArea className="max-h-screen p-4 m-4 overflow-y-auto ">
            {children}
            <ScrollBar />
          </ScrollArea>
        </div>
      </div>
    </>
  );
}
