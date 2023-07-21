import { DesktopSidebar } from "@/app/(authenticated)/(app)/app/desktop-sidebar";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@unkey/db";
import { redirect } from "next/navigation";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
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
      <div className="flex min-h-screen bg-gradient-to-tl from-stone-200 to-stone-100">
        <DesktopSidebar workspace={workspace} />
        <div className="w-full m-2 bg-white shadow ml-72 rounded-xl">
          <ScrollArea className="max-h-screen p-4 m-4 overflow-y-auto ">
            {children}
            <ScrollBar />
          </ScrollArea>
        </div>
      </div>
    </>
  );
}
