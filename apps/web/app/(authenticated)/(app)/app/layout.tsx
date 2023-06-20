import Link from "next/link";

// import { Logo } from "@/components/logo";
import { DesktopSidebar } from "./DesktopSidebar";
// import { MobileNav } from "@/components/mobile-nav";
// import { MobileSidebar } from "./MobileSidebar";
import { getTenantId } from "@/lib/auth";
import { db, eq, schema } from "@unkey/db";
import { redirect } from "next/navigation";
interface LayoutProps {
  children: React.ReactNode;
  params: {
    workspaceSlug: string;
  };
}

export default async function Layout({ params, children }: LayoutProps) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: eq(schema.workspaces.tenantId, tenantId),
    with: {
      apis: true
    }
  })
  if (!workspace){
    return redirect("/app")
  }

  return (
    <>
      <div className="flex bg-gradient-to-tl from-stone-200 to-stone-100">
        <DesktopSidebar  apis={workspace.apis} />

        {/* <MobileSidebar channels={channels.map((c) => ({ name: c.name }))} navigation={[]} /> */}

        <div className="w-full p-8 m-2 bg-white shadow rounded-xl">{children}</div>
      </div>
    </>
  );
}
