import Link from "next/link";

// import { Logo } from "@/components/logo";
import { DesktopSidebar } from "./DesktopSidebar";
// import { MobileNav } from "@/components/mobile-nav";
// import { MobileSidebar } from "./MobileSidebar";
import { getTenantId } from "@/lib/auth";

interface LayoutProps {
  children: React.ReactNode;
  params: {
    tenantSlug: string;
  };
}

export default function Layout({ params, children }: LayoutProps) {
  const _tenantId = getTenantId();

  return (
    <div className="bg-gray-100 flex">
      <DesktopSidebar navigation={[]} tenantSlug={params.tenantSlug} />

      {/* <MobileSidebar channels={channels.map((c) => ({ name: c.name }))} navigation={[]} /> */}

      <div className=" w-full rounded-xl m-2 bg-white p-4">{children}</div>
    </div>
  );
}
