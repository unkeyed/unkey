"use client";

import clsx from "clsx";
import { MotionConfig, motion, useReducedMotion } from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { createContext, useEffect, useId, useRef, useState } from "react";

import { Button } from "@/components/landing/button";
import { Container } from "@/components/landing/container";
import { Footer } from "@/components/landing/footer";
import { GridPattern } from "@/components/landing/grid-pattern";
import { allJobs } from "contentlayer/generated";

const RootLayoutContext = createContext({});

function XIcon(props: any) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" {...props}>
      <path d="m5.636 4.223 14.142 14.142-1.414 1.414L4.222 5.637z" />
      <path d="M4.222 18.363 18.364 4.22l1.414 1.p414L5.636 19.777z" />
    </svg>
  );
}

function MenuIcon(props: any) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" {...props}>
      <path d="M2 6h20v2H2zM2 16h20v2H2z" />
    </svg>
  );
}

function Header({
  panelId,
  invert = false,
  icon: Icon,
  expanded,
  onToggle,
  toggleRef,
}: {
  panelId: string;
  invert?: boolean;
  icon: any;
  expanded: boolean;
  onToggle: any;
  toggleRef: any;
}) {
  return (
    <Container>
      <div className="flex items-center justify-between">
        <Link href="/" aria-label="Home">
          <svg
            width="75"
            height="75"
            viewBox="0 0 276 276"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <g filter="url(#filter0_d_101_3_root)">
              <path
                d="M160.206 70H197V156.749C197 167.064 194.529 175.99 189.588 183.528C184.691 191.021 177.853 196.818 169.074 200.917C160.294 204.972 150.103 207 138.5 207C126.809 207 116.574 204.972 107.794 200.917C99.0147 196.818 92.1765 191.021 87.2794 183.528C82.4265 175.99 80 167.064 80 156.749V70H116.794V153.575C116.794 157.763 117.721 161.51 119.574 164.816C121.426 168.078 123.985 170.634 127.25 172.486C130.559 174.337 134.309 175.263 138.5 175.263C142.735 175.263 146.485 174.337 149.75 172.486C153.015 170.634 155.574 168.078 157.426 164.816C159.279 161.51 160.206 157.763 160.206 153.575V70Z"
                fill="url(#paint0_linear_101_3_root)"
                shapeRendering="crispEdges"
              />
              <path
                d="M160.206 69.5H159.706V70V153.575C159.706 157.686 158.797 161.346 156.991 164.57C155.183 167.753 152.689 170.244 149.503 172.051C146.323 173.854 142.66 174.763 138.5 174.763C134.386 174.763 130.722 173.855 127.496 172.05C124.311 170.244 121.817 167.753 120.009 164.57C118.203 161.346 117.294 157.686 117.294 153.575V70V69.5H116.794H80H79.5V70V156.749C79.5 167.145 81.9466 176.168 86.859 183.798L86.8609 183.801C91.813 191.379 98.726 197.235 107.583 201.37L107.584 201.371C116.442 205.462 126.751 207.5 138.5 207.5C150.161 207.5 160.426 205.462 169.283 201.371L169.285 201.37C178.141 197.235 185.054 191.379 190.006 183.802C195.008 176.171 197.5 167.147 197.5 156.749V70V69.5H197H160.206Z"
                stroke="url(#paint1_linear_101_3_root)"
                shapeRendering="crispEdges"
              />
            </g>
            <defs>
              <filter
                id="filter0_d_101_3_root"
                x="75"
                y="69"
                width="127"
                height="147"
                filterUnits="userSpaceOnUse"
                colorInterpolationFilters="sRGB"
              >
                <feFlood floodOpacity="0" result="BackgroundImageFix" />
                <feColorMatrix
                  in="SourceAlpha"
                  type="matrix"
                  values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                  result="hardAlpha"
                />
                <feOffset dy="4" />
                <feGaussianBlur stdDeviation="2" />
                <feComposite in2="hardAlpha" operator="out" />
                <feColorMatrix type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0" />
                <feBlend mode="normal" in2="BackgroundImageFix" result="effect1_dropShadow_101_3" />
                <feBlend
                  mode="normal"
                  in="SourceGraphic"
                  in2="effect1_dropShadow_101_3"
                  result="shape"
                />
              </filter>
              <linearGradient
                id="paint0_linear_101_3_root"
                x1="80"
                y1="70"
                x2="176.419"
                y2="207.057"
                gradientUnits="userSpaceOnUse"
              >
                <stop offset="0.161458" />
                <stop offset="1" stopColor="#B6B6B6" stopOpacity="0" />
              </linearGradient>
              <linearGradient
                id="paint1_linear_101_3_root"
                x1="47.5"
                y1="168.5"
                x2="212.999"
                y2="167.862"
                gradientUnits="userSpaceOnUse"
              >
                <stop offset="0.194498" />
                <stop offset="0.411458" stopColor="white" stopOpacity="0" />
              </linearGradient>
            </defs>
          </svg>
        </Link>
        <div className="flex items-center gap-x-2 md:gap-x-8">
          {allJobs.some((j) => j.visible) ? (
            <Button className="whitespace-nowrap" href="/careers" invert={invert}>
              We're hiring
            </Button>
          ) : null}
          <Button href="/app" invert={invert}>
            Dashboard
          </Button>
          {/* @ts-expect-error */}
          <button
            ref={toggleRef}
            type="button"
            onClick={onToggle}
            aria-expanded={expanded.toString()}
            aria-controls={panelId}
            className={clsx(
              "group -m-2.5 rounded-full p-2.5 transition max-sm:ml-3 max-md:ml-4",
              invert ? "hover:bg-white/10" : "hover:bg-gray-950/10",
            )}
            aria-label="Toggle navigation"
          >
            <Icon
              className={clsx(
                "h-6 w-6",
                invert
                  ? "fill-white group-hover:fill-gray-200"
                  : "fill-gray-950 group-hover:fill-gray-700",
              )}
            />
          </button>
        </div>
      </div>
    </Container>
  );
}

function NavigationRow({ children }: { children: React.ReactNode }) {
  return (
    <div className="even:mt-px sm:bg-gray-950">
      <Container>
        <div className="grid grid-cols-1 sm:grid-cols-2">{children}</div>
      </Container>
    </div>
  );
}

function NavigationItem({
  href,
  children,
}: {
  href: string;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className="group relative isolate -mx-6 bg-gray-950 px-6 py-10 even:mt-px sm:mx-0 sm:px-0 sm:py-16 sm:odd:pr-16 sm:even:mt-0 sm:even:border-l sm:even:border-gray-800 sm:even:pl-16"
    >
      {children}
      <span className="absolute inset-y-0 -z-10 w-screen bg-gray-900 opacity-0 transition group-odd:right-0 group-even:left-0 group-hover:opacity-100" />
    </Link>
  );
}

function Navigation() {
  return (
    <nav className="font-display mt-px text-5xl font-medium tracking-tight text-white">
      <NavigationRow>
        <NavigationItem href="/pricing">Pricing</NavigationItem>
        <a
          className="group relative isolate -mx-6 bg-gray-950 px-6 py-10 even:mt-px sm:mx-0 sm:px-0 sm:py-16 sm:odd:pr-16 sm:even:mt-0 sm:even:border-l sm:even:border-gray-800 sm:even:pl-16"
          href="https://unkey.dev/docs"
        >
          {" "}
          Docs
          <span className="absolute inset-y-0 -z-10 w-screen bg-gray-900 opacity-0 transition group-odd:right-0 group-even:left-0 group-hover:opacity-100" />
        </a>
      </NavigationRow>
      <NavigationRow>
        <NavigationItem href="/blog">Blog</NavigationItem>
        <NavigationItem href="/changelog">Changelog</NavigationItem>
      </NavigationRow>
      <NavigationRow>
        <NavigationItem href="/about">About</NavigationItem>
        <NavigationItem href="/templates">Templates</NavigationItem>
      </NavigationRow>
      <NavigationRow>
        <NavigationItem href="https://unkey.dev/discord">Discord</NavigationItem>
        <NavigationItem href="mailto:support@unkey.dev">Support</NavigationItem>
      </NavigationRow>
    </nav>
  );
}

function RootLayoutInner({ children }: { children: React.ReactNode }) {
  const panelId = useId();
  const [expanded, setExpanded] = useState(false);
  const openRef = useRef();

  const closeRef = useRef();
  const navRef = useRef();
  const shouldReduceMotion = useReducedMotion();

  useEffect(() => {
    function onClick(event: MouseEvent) {
      const target = event.target as HTMLElement;
      if (target.closest("a")?.href === window.location.href) {
        setExpanded(false);
      }
    }

    window.addEventListener("click", onClick);

    return () => {
      window.removeEventListener("click", onClick);
    };
  }, []);

  return (
    <MotionConfig transition={shouldReduceMotion ? { duration: 0 } : undefined}>
      <header>
        <div
          className="absolute left-0 right-0 top-2 z-40 pt-14"
          aria-hidden={expanded ? "true" : undefined}
          data-inert={expanded ? "" : undefined}
        >
          <Header
            panelId={panelId}
            icon={MenuIcon}
            toggleRef={openRef}
            expanded={expanded}
            onToggle={() => {
              setExpanded((expanded) => !expanded);
              window.setTimeout(() =>
                // @ts-expect-error
                closeRef.current?.focus({ preventScroll: true }),
              );
            }}
          />
        </div>

        <motion.div
          layout
          id={panelId}
          style={{ height: expanded ? "auto" : "0.5rem" }}
          className={
            expanded
              ? "relative z-50 overflow-hidden bg-gray-950 pt-2"
              : "relative z-50 overflow-hidden pt-2"
          }
          aria-hidden={expanded ? undefined : "true"}
          data-inert={expanded ? undefined : ""}
        >
          <motion.div layout className="bg-gray-800">
            {/* @ts-expect-error */}
            <div ref={navRef} className="bg-gray-950 pb-16 pt-14">
              <Header
                invert
                panelId={panelId}
                icon={XIcon}
                toggleRef={closeRef}
                expanded={expanded}
                onToggle={() => {
                  setExpanded((expanded) => !expanded);
                  window.setTimeout(() =>
                    // @ts-expect-error
                    openRef.current?.focus({ preventScroll: true }),
                  );
                }}
              />
            </div>
            <Navigation />
          </motion.div>
        </motion.div>
      </header>

      <motion.div
        layout
        style={{ borderTopLeftRadius: 40, borderTopRightRadius: 40 }}
        className="relative flex flex-auto bg-white pt-14"
      >
        <motion.div layout className="relative isolate flex w-full flex-col pt-9">
          <GridPattern
            className="absolute inset-x-0 -top-14 -z-10 h-[1000px] w-full fill-gray-50 stroke-gray-950/5 [mask-image:linear-gradient(to_bottom_left,white_40%,transparent_50%)]"
            yOffset={-96}
            interactive
          />

          <main className="w-full flex-auto">{children}</main>

          <Footer />
        </motion.div>
      </motion.div>
    </MotionConfig>
  );
}

export function RootLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [logoHovered, setLogoHovered] = useState(false);

  return (
    <RootLayoutContext.Provider value={{ logoHovered, setLogoHovered }}>
      <RootLayoutInner key={pathname}>{children}</RootLayoutInner>
    </RootLayoutContext.Provider>
  );
}
