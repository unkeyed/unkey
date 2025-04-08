import { AppSidebar } from "@/components/navigation/sidebar/app-sidebar";
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile";
import { SidebarProvider } from "@/components/ui/sidebar";
import { getIsImpersonator, getOrgId } from "@/lib/auth";
import { db } from "@/lib/db";
import { Empty } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";

interface LayoutProps {
  children: React.ReactNode;
}

export default async function Layout({ children }: LayoutProps) {
  const orgId = await getOrgId();
  const isImpersonator = await getIsImpersonator();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
      quotas: true,
    },
  });

  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-base-12 lg:flex-row">
      <SidebarProvider>
        <div className="flex flex-1 overflow-hidden">
          {/* Desktop Sidebar */}
          <AppSidebar
            workspace={{ ...workspace, quotas: workspace.quotas! }}
            className="bg-gray-1 border-grayA-4"
          />

          {/* Main content area */}
          <div className="flex-1 overflow-auto">
            <div
              className="isolate bg-base-12 w-full overflow-x-auto flex flex-col items-center"
              id="layout-wrapper"
            >
              {/* Mobile sidebar at the top of content */}
              <SidebarMobile workspace={workspace} />

              <div className="w-full">
                {workspace.enabled ? (
                  children
                ) : (
                  <div className="flex items-center justify-center w-full h-full">
                    <Empty>
                      <Empty.Icon />
                      <Empty.Title>This workspace is disabled</Empty.Title>
                      <Empty.Description>
                        Contact{" "}
                        <Link
                          href={`mailto:support@unkey.dev?body=workspaceId: ${workspace.id}`}
                          className="underline"
                        >
                          support@unkey.dev
                        </Link>
                      </Empty.Description>
                    </Empty>
                  </div>
                )}
              </div>
            </div>
            {isImpersonator ? (
              <div className="fixed top-0 inset-x-0 z-50 flex justify-center  border-t-2 border-error-9">
                <div className="bg-error-9  flex -mt-1 font-mono items-center gap-2 text-white text-xs rounded-b overflow-hidden shadow-lg select-none pointer-events-none px-1.5 py-0.5">
                  Impersonation Mode. Do not change anything and log out after you are done.
                </div>
              </div>
            ) : null}
          </div>
        </div>
      </SidebarProvider>
    </div>
  );
}
