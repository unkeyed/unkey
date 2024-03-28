"use client";
import Link from "next/link";
import {
  UnkeyFooterLogo,
  UnkeyFooterLogoMobile,
  UnkeyLogoSmall,
  UnkeyLogoSmallMobile,
} from "./footer-svgs";
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
    <div className="flex flex-col sm:mx-auto">
      <UnkeyLogoSmall />
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.5)] mt-8">
        Seriously Fast API Authentication.
      </div>
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.3)]">
        Unkeyed, Inc. {new Date().getUTCFullYear()}
      </div>
    </div>
  );
}

function CompanyInfoMobile() {
  return (
    <div className="flex flex-col items-center">
      <UnkeyLogoSmallMobile />
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.5)] mt-10">
        Seriously Fast API Authentication.
      </div>
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.3)]">
        Unkeyed, Inc. 2023
      </div>
      <div className="mt-10">
        {navigation.map((section) => (
          <div key={section.title}>
            <h3 className="py-4 text-sm font-medium text-white">{section.title}</h3>
            <ul className="text-sm text-[rgba(255,255,255,0.7)] font-normal">
              {section.links.map((link) => (
                <li key={link.href.toString()} className="py-4">
                  {link.href.startsWith("https://") ? (
                    <a
                      href={link.href}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="transition hover:text-[rgba(255,255,255,0.4)]"
                    >
                      {link.title}
                    </a>
                  ) : (
                    <Link
                      href={link.href}
                      className="transition hover:text-[rgba(255,255,255,0.4)]"
                    >
                      {link.title}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="flex justify-center w-full lg:mt-24">
        <UnkeyFooterLogo />
      </div>
    </div>
  );
}

function Navigation() {
  return (
    <nav className=" sm:w-full">
      <ul className="flex flex-col flex-auto gap-16 text-left sm:flex-row sm:mx-auto justify-evenly">
        {navigation.map((section) => (
          <li key={section.title.toString()}>
            <div className="text-sm font-medium tracking-wider text-white font-display">
              {section.title}
            </div>
            <ul className="text-sm text-[rgba(255,255,255,0.7)] font-normal gap-4 flex flex-col md:gap-8 mt-4 md:mt-8 ">
              {section.links.map((link) => (
                <li key={link.href.toString()}>
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

function MobileNavigation() {
  return (
    <nav className="flex flex-col pt-4 sm:hidden mobile-blog-footer-radial">
      <div className="flex flex-col items-center justify-center w-full text-center">
        <CompanyInfoMobile />
      </div>
      <div className="flex justify-center w-full lg:mt-24">
        <UnkeyFooterLogoMobile />
      </div>
    </nav>
  );
}

export function Footer() {
  return (
    <>
      <footer className="relative hidden pt-32 overflow-hidden border-t sm:block xl:pt-10 max-sm:pt-8 border-white/10 blog-footer-radial-gradient">
        <div className="container flex flex-col mx-auto">
          <div className="flex flex-row justify-center max-sm:flex-col sm:flex-col md:flex-row xl:gap-20 xxl:gap-48">
            <div className="flex mb-8 lg:mx-auto max-sm:pl-12 max-sm:flex sm:flex-row sm:w-full xl:pl-14 md:w-fit shrink-0 xxl:pl-28">
              <CompanyInfo />
            </div>
            <div className="flex w-full max-sm:pl-12 max-sm:pt-6 max-sm:mt-22 md:pl-18 lg:pl-6 max-sm:mb-8">
              <Navigation />
            </div>
          </div>
        </div>
        <div className="flex justify-center w-full">
          <UnkeyFooterLogo className="mt-4" />
        </div>
      </footer>
      <MobileNavigation />
    </>
  );
}
