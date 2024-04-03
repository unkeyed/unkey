"use client";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { UnkeyFooterLogo, UnkeyLogo } from "./footer-svgs";

type NavLink = {
  title: string;
  href: string;
};
const navigation = [
  {
    title: "Company",
    links: [
      { title: "About", href: "/about" },
      { title: "Blog", href: "/blog" },
      { title: "Changelog", href: "/changelog" },
      { title: "Templates", href: "/templates" },
      {
        title: "Analytics",
        href: "https://us.posthog.com/shared/HwZNjaKOLtgtpj6djuSo3fgOqrQm0Q?whitelabel",
      },
      {
        title: "Source Code",
        href: "https://github.com/unkeyed/unkey",
      },
      {
        title: "Docs",
        href: "https://unkey.dev/docs",
      },
    ],
  },
  {
    title: "Connect",
    links: [
      { title: "X(Twitter)", href: "https://twitter.com/unkeydev" },
      { title: "GitHub", href: "https://github.com/unkeyed" },
      { title: "OSS Friends", href: "/oss-friends" },
      {
        title: "Book a Call",
        href: "https://cal.com/team/unkey/unkey-chat??utm_source=banner&utm_campaign=oss",
      },
    ],
  },
  {
    title: "Legal",
    links: [
      { title: "Terms of Service", href: "/policies/terms" },
      { title: "Privacy Policy", href: "/policies/privacy" },
    ],
  },
] satisfies Array<{ title: string; links: Array<NavLink> }>;

const Column: React.FC<{ title: string; links: Array<NavLink>; className?: string }> = ({
  title,
  links,
  className,
}) => {
  return (
    <ul className={cn("flex flex-col gap-8 md:gap-12 text-left", className)}>
      <span className="text-sm font-medium tracking-wider text-white font-display">{title}</span>
      <ul className="flex flex-col gap-4 md:gap-8">
        {links.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            target={link.href.startsWith("https://") ? "_blank" : undefined}
            rel={link.href.startsWith("https://") ? "noopener noreferrer" : undefined}
            className="text-sm font-normal transition hover:text-white/40 text-white/70"
          >
            {link.title}
          </Link>
        ))}
      </ul>
    </ul>
  );
};

export function Footer() {
  return (
    <div className="border-t border-white/20">
      <footer className="container relative grid grid-cols-1 gap-8 pt-8 mx-auto overflow-hidden lg:gap-16 sm:grid-cols-3 xl:grid-cols-5 sm:pt-12 md:pt-16 lg:pt-24 xl:pt-32">
        <div className="flex flex-col items-start cols-span-1 sm:col-span-3 xl:col-span-2">
          <UnkeyLogo />
          <div className="mt-8 text-sm font-normal leading-6 text-white/60">
            Build better APIs faster.
          </div>
          <div className="text-sm font-normal leading-6 text-white/40">
            Unkeyed, Inc. {new Date().getUTCFullYear()}
          </div>
        </div>

        {navigation.map(({ title, links }) => (
          <Column key={title} title={title} links={links} className="col-span-1" />
        ))}
      </footer>
      <div className="flex justify-center w-full mt-8 lg:mt-16">
        <UnkeyFooterLogo />
      </div>
    </div>
  );
}
