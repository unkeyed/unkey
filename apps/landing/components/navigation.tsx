import { ChevronRight } from "lucide-react";
import Link from "next/link";

import { Logo } from "./svg";

export function Navigation() {
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
    <nav className="bg-black flex items-center justify-between h-20 pt-12 bg-red-500">
      <div className="flex items-center justify-between gap-32">
        <div className="flex items-center gap-2">
          <Logo />
        </div>
        <ul className="flex items-center gap-8 justify-between">
          <NavLink href="/about" label="About" />
          <NavLink href="/blog" label="Blog" />
          <NavLink href="/pricing" label="Pricing" />
          <NavLink href="/changelog" label="Changelog" />
          <NavLink href="/docs" label="Docs" />
        </ul>
      </div>
      <div className="flex">
        <Link
          href="/auth/sign-up"
          className="text-white/60 text-sm flex items-center justify-center px-3 mr-3 h-8 py-2 gap-2 duration-150 hover:text-white"
        >
          Create Account
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
