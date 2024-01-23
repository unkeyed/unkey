import { Drawer, DrawerClose, DrawerContent, DrawerTrigger } from "@/components/ui/drawer";
import { X } from "lucide-react";
import Link from "next/link";

function MobileNavLink({ href, label }: { href: string; label: string }) {
  return (
    <Link
      href={href}
      className="text-white/50 hover:text-white duration-200 text-lg font-medium tracking-[0.07px] border-white"
    >
      {label}
    </Link>
  );
}

export function Menu() {
  return (
    <div className="text-white">
      <Drawer>
        <DrawerTrigger asChild>
          <button type="button">Open drawer</button>
        </DrawerTrigger>
        <DrawerContent>
          <div className="mx-auto w-full ml-20 pt-20 antialiased relative">
            <div className="text-white absolute right-[110px] top-[0px]">
              <DrawerClose>
                {" "}
                <X className="h-6 w-6 text-white/50" />
              </DrawerClose>
            </div>
            <ul className="flex flex-col text-white space-y-20">
              <MobileNavLink href="/about" label="About" />
              <MobileNavLink href="/blog" label="Blog" />
              <MobileNavLink href="/pricing" label="Pricing" />
              <MobileNavLink href="/changelog" label="Changelog" />
              <MobileNavLink href="/docs" label="Docs" />
            </ul>
          </div>
        </DrawerContent>
      </Drawer>
    </div>
  );
}
