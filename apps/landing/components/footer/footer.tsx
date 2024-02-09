"use client";
import Link from "next/link";
import { UnkeyFooterLogo, UnkeyLogoSmall } from "./footer-svgs";
import { socialMediaProfiles } from "./social-media";
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
    links: socialMediaProfiles,
  },
  {
    title: "Legal",
    links: [
      { title: "Terms of Service", href: "/policies/terms" },
      { title: "Privacy Policy", href: "/policies/privacy" },
    ],
  },
];

function CompanyInfo() {
  return (
    <div className="flex flex-col">
      <UnkeyLogoSmall />
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.5)] mt-8">
        Seriously Fast API Authentication.
      </div>
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.3)]">
        Unkeyed, Inc. 2023
      </div>
    </div>
  );
}

function Navigation() {
  return (
    <nav>
      <ul className="flex flex-auto gap-16">
        {navigation.map((section) => (
          <li key={section.title}>
            <div className="text-sm font-medium tracking-wider text-white font-display">
              {section.title}
            </div>
            <ul className="text-sm text-[rgba(255,255,255,0.7)] font-normal gap-4 flex flex-col md:gap-8 mt-4 md:mt-8 ">
              {section.links.map((link) => (
                <li key={link.href} className="">
                  {link.href.startsWith("https://") ? (
                    <a
                      href={link.href}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="transition hover:text-[rgba(255,255,255,0.4)] "
                    >
                      {link.title}
                    </a>
                  ) : (
                    <Link
                      href={link.href}
                      className="transition hover:text-[rgba(255,255,255,0.4)] "
                    >
                      {link.title}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </nav>
  );
}

export function Footer() {
  return (
    <footer className="relative pt-32 border-t max-sm:pt-8 border-white/10 blog-footer-radial-gradient h-fit w-full">
      <div className="absolute inset-x-0 w-full h-full " />
      <div className="flex flex-col mx-auto lg:w-fit max-sm:w-full ">
        <div className="flex flex-row justify-center max-sm:flex-col sm:flex-col md:flex-row lg:gap-20 xl:gap-48">
          <div className="flex mb-8 lg:mx-auto max-sm:pl-12 max-sm:flex sm:flex-row sm:w-full sm:pl-28 md:pl-12 lg:pl-14 md:w-fit shrink-0 xl:pl-28">
            <CompanyInfo />
          </div>
          <div className="flex w-full max-sm:pl-12 max-sm:pt-6 max-sm:mt-22 sm:pl-28 md:pl-18 lg:pl-6 max-sm:mb-8">
            <Navigation />
          </div>
        </div>
      </div>
      <div className="flex justify-center w-full mt-24">
        <UnkeyFooterLogo />
      </div>
    </footer>
  );
}
