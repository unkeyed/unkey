import { DesktopSidebar } from "./desktop-sidebar";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
import { MobileSideBar } from "./mobile-sidebar";
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
        <div className="w-full px-4 md:px-8 md:m-2 bg-gradient-to-br bg-zinc-100 dark:bg-zinc-900 shadow md:ml-72 rounded-xl">
          {children}
        </div>
      </div>
    </>
  );
}
