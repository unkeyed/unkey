import { AppSidebar } from "@/components/app-sidebar"
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile"
import { SidebarProvider } from "@/components/ui/sidebar"
import { getTenantId } from "@/lib/auth"
import { db } from "@/lib/db"
import { Empty } from "@unkey/ui"
import Link from "next/link"
import { redirect } from "next/navigation"
import { UsageBanner } from "./banner"

interface LayoutProps {
  children: React.ReactNode
}

export default async function Layout({ children }: LayoutProps) {
  const tenantId = getTenantId()
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
    with: {
      apis: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
      },
      quota: true,
    },
  })

  if (!workspace) {
    return redirect("/apis")
  }

  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-base-12 lg:flex-row">
      <UsageBanner workspace={workspace} />
      <SidebarProvider>
        <div className="flex flex-1 overflow-hidden">
          {/* Desktop Sidebar */}
          <AppSidebar
            workspace={workspace}
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
          </div>
        </div>
      </SidebarProvider>
    </div>
  )
}
