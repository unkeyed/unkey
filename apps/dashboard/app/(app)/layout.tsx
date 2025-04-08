import { AppSidebar } from "@/components/navigation/sidebar/app-sidebar";
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
          <AppSidebar workspace={workspace} className="bg-gray-1 border-grayA-4" />

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
            {isImpersonator && (
              <div className="absolute bottom-0 right-0 size-fit z-10 pb-2 pr-2 center">
                <Alert variant="alert">
                  <AlertTitle className="text-base font-semibold text-center">
                    Impersonating User
                  </AlertTitle>
                  <AlertDescription className="text-center">
                    Do not make changes, and log out when you're done
                  </AlertDescription>
                </Alert>
              </div>
            )}
          </div>
        </div>
      </SidebarProvider>
    </div>
  );
}
