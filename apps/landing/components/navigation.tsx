import { ChevronRight } from "lucide-react";
import Link from "next/link";

import { Logo } from "./svg";

export function Navigation() {
  const NavLink: React.FC<{ href: string; label: string }> = ({ href, label }) => {
    return (
      <Link href={href} className="text-white/50 hover:text-white duration-200 text-sm">
        {label}
      </Link>
    );
  };

  return (
    <nav className="bg-black flex items-center justify-between h-20">
      <div className="flex items-center justify-between gap-32">
        <div className="flex items-center gap-2">
          <Logo />
        </div>
        {/* Nav */}
        <ul className="flex items-center gap-8 justify-between">
          <NavLink href="/about" label="About" />
          <NavLink href="/blog" label="Blog" />
          <NavLink href="/pricing" label="Pricing" />
          <NavLink href="/changelog" label="Changelog" />
          <NavLink href="/docs" label="Docs" />
        </ul>
      </div>

      {/* Auth */}
      <div>
        <Link
          href="/app"
          className="shadow-md font-medium bg-white h-8 flex items-center border border-white px-4  rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
        >
          Log In <ChevronRight className="w-4 h-4" />
        </Link>
      </div>
    </nav>
  );
}
