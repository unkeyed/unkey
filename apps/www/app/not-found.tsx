import { PrimaryButton } from "@/components/button";
import { cn } from "@/lib/utils";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import { Spotlight } from "./(not-found)/components/spotlight";

export default function NotFoundPage() {
  return (
    <div className="w-full mt-2 lg:mt-[4.03125rem] py-20 pb-10 lg:py-40 relative opacity-0 [animation-delay:.85s] animate-fade-in-down">
      <div className="isolate relative w-full py-0 lg:pt-8 lg:pb-24 [mask-image:linear-gradient(to_right,transparent,black_10%,black_90%,transparent)] lg:[mask-image:radial-gradient(circle_at_10%_50%,black,transparent)]">
        <div
          aria-hidden
          className="absolute inset-0 hidden lg:[display:block] mix-blend-soft-light"
        >
          <PerlinNoise />
        </div>

        <div className="container relative">
          <div className="relative w-full">
            <div className="relative flex flex-col gap-4 lg:gap-8 text-white/50">
              <GridBorder left right overflow />

              <div className="relative flex flex-col w-full text-[white]">
                <span className="relative !leading-[.76] text-8xl lg:text-[200px] font-bold -tracking-[.06em] -ml-[.7%]">
                  <div
                    aria-hidden
                    className="absolute inset-0 pointer-events-none select-none opacity-10 [mask-image:linear-gradient(to_bottom,black,transparent)]"
                  >
                    404
                  </div>
                  <div className="relative [-webkit-text-fill-color:transparent] [-webkit-text-stroke:rgba(255,255,255,.5)_1px] [mask-image:linear-gradient(to_bottom,black,transparent)]">
                    404
                  </div>
                </span>

                <GridBorder top bottom overflow />
              </div>

              <div className="relative flex flex-col w-full text-[white]">
                <GridBorder top bottom />

                <span className="!leading-[.73] text-6xl lg:text-[7.125rem] font-bold -tracking-[.06em] -ml-1 shadow-2xl">
                  Not found.
                </span>
              </div>

              <div className="relative flex flex-col w-full">
                <GridBorder top bottom />

                <Link
                  href="/"
                  className="w-max brightness-100 hover:brightness-50 [transition:filter_.5s_ease]"
                >
                  <PrimaryButton
                    shiny
                    label="Go to homepage"
                    IconRight={ChevronRight}
                    className="h-10 lg:h-14 lg:text-lg"
                  />
                </Link>
              </div>

              <div className="relative flex flex-col w-full mt-4 text-lg lg:mt-8 lg:text-4xl">
                {/* <div className="relative flex flex-col w-full mt-4 text-sm lg:mt-8 lg:text-base"> */}
                <GridBorder top />
                <GridBorder bottom overflow />

                <span>Build better APIs faster.</span>
              </div>
            </div>
          </div>
        </div>

        <Spotlight />
      </div>
    </div>
  );
}

function GridBorder({
  overflow = false,
  ...props
}: {
  top?: boolean;
  bottom?: boolean;
  left?: boolean;
  right?: boolean;
  overflow?: boolean;
}) {
  const className =
    "pointer-events-none absolute inset-0 [&_line]:stroke-white/25 [&_line]:[stroke-width:2px] [&_line]:[stroke-dasharray:3,5]";

  // TODO: deduplicate
  return (
    <>
      {(props.left || props.right) && (
        <div
          aria-hidden
          className={cn(
            className,
            overflow &&
              "-top-[20%] -bottom-[20%] [mask-image:linear-gradient(to_bottom,transparent_3%,white_10%,white_90%,transparent_97%)]",
          )}
        >
          <svg height="100%" width="100%" preserveAspectRatio="none">
            {props.left && <line x1="0" y1="0" x2="0" y2="100%" />}
            {props.right && <line x1="100%" y1="0" x2="100%" y2="100%" />}
          </svg>
        </div>
      )}
      {(props.bottom || props.top) && (
        <div
          aria-hidden
          className={cn(
            className,
            overflow &&
              "-left-[20%] -right-[20%] [mask-image:linear-gradient(to_right,transparent_3%,white_10%,white_90%,transparent_97%)]",
          )}
        >
          <svg height="100%" width="100%" preserveAspectRatio="none">
            {props.top && <line x1="0" y1="0" x2="100%" y2="0" />}
            {props.bottom && <line x1="0" y1="100%" x2="100%" y2="100%" />}
          </svg>
        </div>
      )}
    </>
  );
}

function PerlinNoise() {
  return (
    <svg
      aria-hidden
      className="pointer-events-none absolute inset-0 opacity-40 [mask-image:radial-gradient(30%_40%_at_center,black,transparent)]"
      xmlns="http://www.w3.org/2000/svg"
      version="1.1"
      xmlnsXlink="http://www.w3.org/1999/xlink"
      width="100%"
      height="100%"
      opacity="1"
    >
      <defs>
        <filter
          id="noise"
          width="100%"
          height="100%"
          filterUnits="objectBoundingBox"
          primitiveUnits="userSpaceOnUse"
          color-interpolation-filters="linearRGB"
        >
          <feTurbulence
            type="fractalNoise"
            result="turbulence"
            baseFrequency="0.8"
            numOctaves="4"
            seed="10"
            stitchTiles="stitch"
          >
            {/* <animate
              id="noiseAnimate"
              attributeName="baseFrequency"
              attributeType="XML"
              values="10;11;10"
              keyTimes="0;.5;1"
              begin="0s"
              dur="5s"
              repeatCount="indefinite"
            /> */}
          </feTurbulence>
        </filter>
      </defs>
      <rect width="100%" height="100%" filter="url(#noise)" />
    </svg>
  );
}
