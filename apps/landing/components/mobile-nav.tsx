import { ChevronDown, ChevronRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { LogoMobile } from "./svg";
export function MobileNav() {
  const NavLink: React.FC<{ href: string; label: string }> = ({ href, label }) => {
    return (
      <Link
        href={href}
        className="text-white/50 hover:text-white duration-200 text-sm tracking-[0.07px]"
      >
        {label}
      </Link>
    );
  };

  return (
    <nav className="md:hidden bg-black flex items-center justify-between h-20 pt-12">
      <div className="flex items-center justify-between sm:gap-16 lg:gap-32">
        {/* TODO: replace with SVG logo – currently setting display:hidden on desktop SVG will also hide mobile SVG */}
        <Image src="/logo.png" alt="Unkey logo" width={75} height={32} />
        <button
          type="button"
          className="text-white/60 text-sm flex items-center justify-center px-3 mr-3 h-8 py-2 gap-2 duration-150 hover:text-white"
        >
          Menu
          <ChevronDown className="w-4 h-4" />
        </button>
      </div>
      <div className="flex">
        <Link
          href="/auth/sign-up"
          className="text-white/60 text-sm flex items-center justify-center px-3 mr-3 h-8 py-2 gap-2 duration-150 hover:text-white"
        >
          Sign Up
          <ChevronRight className="w-4 h-4" />
        </Link>
        <Link
          href="/app"
          className="shadow-md font-medium text-sm bg-white h-8 flex items-center border border-white pl-4 pr-2.5 py-1 rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
        >
          Log In <ChevronRight className="w-4 h-4" />
        </Link>
      </div>
    </nav>
  );
}
