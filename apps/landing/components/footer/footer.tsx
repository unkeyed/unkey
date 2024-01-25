"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { UnkeyFooterLogo, UnkeyLogoSmall } from "./footer-svgs";
import { socialMediaProfiles } from "./social-media";
const navigation = [
  {
    title: "Company",
    links: [
      { title: "About", href: "/about" },
      { title: "Blog", href: "/blog" },
      { title: "Changelog", href: "/changelog" },
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
    links: [...socialMediaProfiles],
  },
  {
    title: "Legal",
    links: [
      { title: "Terms of Service", href: "/policies/terms" },
      { title: "Privacy Policy", href: "/policies/privacy" },
    ],
  },
];
function getWindowDimensions() {
  const { innerWidth: width, innerHeight: height } =
    typeof window !== "undefined" ? window : ({} as Window);

  return {
    width,
    height,
  };
}
export default function useWindowDimensions() {
  const [windowDimensions, setWindowDimensions] = useState(getWindowDimensions());

  useEffect(() => {
    function handleResize() {
      setWindowDimensions(getWindowDimensions());
    }

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return windowDimensions;
}
function CompanyInfo() {
  return (
    <div className="flex flex-col">
      <UnkeyLogoSmall />
      <div className="font-normal text-sm leading-6 text-[rgba(255,255,255,0.5)] mt-8 ">
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
      <ul className="flex max-sm:space-x-10 sm:space-x-24 lg:space-x-40">
        {navigation.map((section) => (
          <li key={section.title}>
            <div className="font-display text-sm text-white font-medium tracking-wider">
              {section.title}
            </div>
            <ul className="text-sm text-[rgba(255,255,255,0.7)] font-normal">
              {section.links.map((link) => (
                <li key={link.href} className="mt-12">
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
  const width = 1376;
  const height = 248;
  const svgRatio = width / height;
  const sideOffset = 0.9;

  return (
    <div className="bg-[radial-gradient(69.41%_100%_at_50%_0%,rgba(255,255,255,0.056)_0%,rgba(0,0,0,0.2)_50%,rgba(0,0,0,1)_100%)] pt-32">
      <div className="flex flex-col lg:w-fit mx-auto max-sm:w-full ">
        <div className="flex flex-row max-sm:flex-col justify-center sm:flex-col md:flex-row lg:gap-48">
          <div className="flex lg:mx-auto max-sm:pl-12 max-sm:flex sm:flex-row sm:w-full mb-8 sm:pl-28 lg:pl-0 md:w-fit shrink-0">
            <CompanyInfo />
          </div>
          <div className="flex w-full max-sm:pl-12 max-sm:pt-6 max-sm:mt-22 sm:pl-28 md:pl-18 lg:pl-6">
            <Navigation />
          </div>
        </div>
      </div>
      <div className="flex justify-center mt-24 w-full">
        <UnkeyFooterLogo
          width={
            getWindowDimensions().width > width ? 1376 : getWindowDimensions().width * sideOffset
          }
          height={
            (getWindowDimensions().width > width
              ? getWindowDimensions().width / svgRatio
              : height) * sideOffset
          }
        />
      </div>
    </div>
  );
}
