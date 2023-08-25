import { DesktopSidebar } from "./desktop-sidebar";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { redirect } from "next/navigation";
import { MobileSideBar } from "./mobile-sidebar";
import { UsageBanner } from "./banner";
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
      <div className="flex flex-col min-h-screen lg:flex-row bg-gradient-to-tl from-stone-200 to-stone-100 dark:from-neutral-950 dark:to-neutral-900">
        <UsageBanner />
        <DesktopSidebar workspace={workspace} className="hidden lg:block" />
        <MobileSideBar workspace={workspace} className="lg:hidden" />
        <div className="p-4 m-2 bg-white shadow dark:shadow-none lg:w-full lg:p-6 dark:bg-neutral-950 lg:ml-72 rounded-xl dark:rounded-none dark:m-0 dark:lg:ml-72">
          {children}
        </div>
      </div>
    </>
  );
}
