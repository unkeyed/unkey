import Link from "next/link";

// import { Logo } from "@/components/logo";
import { DesktopSidebar } from "./DesktopSidebar";
// import { MobileNav } from "@/components/mobile-nav";
// import { MobileSidebar } from "./MobileSidebar";
import { getTenantId } from "@/lib/auth";
interface LayoutProps {
  children: React.ReactNode;
  params: {
    workspaceSlug: string;
  };
}

export default function Layout({ params, children }: LayoutProps) {
  const _workspaceId = getTenantId();

  return (
    <>
      <div className="bg-gradient-to-tl from-stone-200 to-stone-100 flex">
        <DesktopSidebar workspaceSlug={params.workspaceSlug} />

        {/* <MobileSidebar channels={channels.map((c) => ({ name: c.name }))} navigation={[]} /> */}

        <div className=" w-full rounded-xl m-2 bg-white p-8 shadow">{children}</div>
      </div>
    </>
  );
}
