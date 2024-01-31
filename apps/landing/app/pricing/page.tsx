import { PrimaryButton } from "@/components/button";
import { Particles } from "@/components/particles";
import { ShinyCard, ShinyCardGroup, WhiteShinyCard } from "@/components/shiny-card";
import { cn } from "@/lib/utils";
import { desc } from "drizzle-orm";
import { Check, LucideIcon, Stars } from "lucide-react";
import Link from "next/link";
import { PropsWithChildren } from "react";

import { HeroSvg } from "./hero-svgs";

export default function PricingPage() {
  return (
    <div>
      <HeroSvg className="absolute inset-x-0 top-0" />

      <div className="flex flex-col items-center justify-center mt-32 min-h-72">
        <h1 className="section-title-heading-gradient font-medium text-[4rem] leading-[4rem] max-w-xl text-center ">
          Pricing built for everyone.
        </h1>

        <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg text-center">
          We wanted pricing to be simple and affordable for anyone, so we've created flexible plans
          that don't need an accounting degree to figure out.
        </p>
      </div>

      <ShinyCardGroup className="grid h-full max-w-4xl grid-cols-2 gap-6 mx-auto group">
        <PricingCard color="#FFFFFF" className="col-span-2 lg:col-span-1">
          <FreeCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <PricingCardHeader
            title="Free Tier"
            description="Everything you need to start!"
            className="bg-gradient-to-tr from-black/50 to-[#ffffff]/10 "
            iconColor="#ffffff"
          />
          <Separator />
          <Particles
            className="absolute inset-0 transition-opacity duration-1000 ease-in-out -z-10 opacity-20 group-hover/item:opacity-100"
            quantity={50}
            color="#ffffff"
            vy={-0.2}
          />

          <PricingCardContent>
            <Cost dollar="$0" />
            <CTA label="Start for Free" />
            <Bullets>
              <Bullet
                Icon={Check}
                label="100 active keys / month"
                iconColors="text-white bg-white/10"
              />
              <Bullet
                Icon={Check}
                label="2.5k successful verifications / month"
                iconColors="text-white bg-white/10"
              />
              <Bullet
                Icon={Check}
                label="7-day analytics retention"
                iconColors="text-white bg-white/10"
              />
              <Bullet Icon={Check} label="Unlimited APIs" iconColors="text-white bg-white/10" />
            </Bullets>
          </PricingCardContent>
        </PricingCard>
        <PricingCard color="#FFD600" className="col-span-2 lg:col-span-1">
          <ProCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <PricingCardHeader
            title="Pro Tier"
            description="For growing teams with powerful demands"
            className="bg-gradient-to-tr from-black/50 to-[#FFD600]/10 "
            iconColor="#FFD600"
          />
          <Separator />

          <Particles
            className="absolute inset-0 transition-opacity duration-1000 ease-in-out -z-10 opacity-20 group-hover/item:opacity-100"
            quantity={50}
            color="#FFD600"
            vy={-0.2}
          />

          <PricingCardContent>
            <Cost dollar="$25" />
            <CTA label="Get Started with Pr" />
            <Bullets>
              <Bullet
                Icon={Check}
                label="250 active keys / month"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
              />
              <Bullet
                Icon={Check}
                label="150k successful verifications / month"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
              />
              <Bullet
                Icon={Check}
                label="90-day analytics retention"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
              />
              <Bullet
                Icon={Check}
                label="90-day audit log retention"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
              />
              <Bullet
                Icon={Check}
                label="Unlimited APIs"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
              />
              <Bullet
                Icon={Check}
                label="Workspaces with team members"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
              />
              <Bullet
                Icon={Stars}
                label="More coming soon"
                iconColors="text-[#FFD600] bg-[#FFD600]/10"
                textColor="text-white/40"
              />
            </Bullets>
          </PricingCardContent>
          <PricingCardFooter>
            <div className="flex flex-col gap-2">
              <Asterisk tag="$0.10" label="/ additional active key" />
              <Asterisk tag="$1" label="/ additional 10k successful verifications" />
            </div>
          </PricingCardFooter>
        </PricingCard>
        <PricingCard color="#9D72FF" className="col-span-2">
          <CustomCardHighlight className="absolute top-0 right-0 pointer-events-none" />

          <Particles
            className="absolute inset-0 transition-opacity duration-1000 ease-in-out -z-10 opacity-20 group-hover/item:opacity-100"
            quantity={100}
            color="#9D72FF"
            vy={-0.2}
          />

          <div className="flex h-full ">
            <div className="flex flex-col w-full gap-8">
              <PricingCardHeader
                title="Enterprise Tier"
                description={
                  <>
                    Need something custom?
                    <br /> We'll find a way.
                  </>
                }
                iconColor="#9D72FF"
                className="bg-gradient-to-tr from-transparent to-[#9D72FF]/10 "
              />
              <PricingCardContent>
                <Cost dollar="Custom $" />
                <Link href="mailto:support@unkey.dev?subject=Unkey Enterprise Quote">
                  <div className="w-full p-px rounded-lg h-10 bg-gradient-to-r from-[#02DEFC] via-[#0239FC] to-[#7002FC] overflow-hidden">
                    <div className="bg-black rounded-[7px] h-full bg-opacity-95 hover:bg-opacity-25 duration-500">
                      <div className="flex items-center justify-center w-full h-full bg-gradient-to-tr from-[#02DEFC]/20 via-[#0239FC]/20 to-[#7002FC]/20  rounded-[7px]">
                        <span className="text-sm font-semibold text-white">Contact Us</span>
                      </div>
                    </div>
                  </div>
                </Link>
              </PricingCardContent>
            </div>
            <Separator orientation="vertical" />
            <div className="w-full p-8">
              <Bullets>
                <Bullet
                  Icon={Check}
                  label="Custom Quotas"
                  iconColors="text-[#9D72FF] bg-[#9D72FF]/10"
                />
                <Bullet
                  Icon={Check}
                  label="IP Whitelisting"
                  iconColors="text-[#9D72FF] bg-[#9D72FF]/10"
                />
                <Bullet
                  Icon={Check}
                  label="Dedicated Support"
                  iconColors="text-[#9D72FF] bg-[#9D72FF]/10"
                />
                <Bullet
                  Icon={Stars}
                  label="More coming soon"
                  iconColors="text-[#9D72FF] bg-[#9D72FF]/10"
                  textColor="text-white/40"
                />
              </Bullets>
            </div>
          </div>
        </PricingCard>
      </ShinyCardGroup>
    </div>
  );
}

const PricingCardHeader: React.FC<{
  title: string;
  description: React.ReactNode;
  className?: string;
  iconColor: `#${string}`;
}> = ({ title, description, className, iconColor }) => {
  return (
    <div className={cn("p-10 flex items-start justify-between w-full gap-10", className)}>
      <div>
        <span className="bg-gradient-to-br text-transparent bg-gradient-stop  bg-clip-text from-white via-white via-30% to-white/30 font-medium ">
          {title}
        </span>
        <p className="mt-4 text-sm text-white/60">{description}</p>
      </div>

      <div className="z-30 flex items-center justify-center ring-1 bg-gradient-to-t from-black via-black via-[30%] to-white/10 h-14 w-14 rounded-xl ring-white/25 rounded-2">
        <KeyIcon color={iconColor} />
      </div>
    </div>
  );
};

const Cost: React.FC<{ dollar: string }> = ({ dollar }) => {
  return (
    <div className="flex items-center gap-4">
      <span className="text-4xl font-semibold text-transparent bg-gradient-to-br bg-clip-text from-white via-white to-white/30">
        {dollar}
      </span>
      <span className=" text-white/60">/ month</span>
    </div>
  );
};

const CTA: React.FC<{ label: string }> = ({ label }) => {
  return (
    <div>
      <Link href="/app">
        <button
          type="button"
          className="block w-full h-10 text-sm font-semibold text-center text-black duration-500 bg-white border border-white rounded-lg hover:bg-transparent hover:text-white"
        >
          {label}
        </button>
      </Link>
    </div>
  );
};

const Bullets: React.FC<PropsWithChildren> = ({ children }) => {
  return (
    <div>
      <p className="text-white/40">What's included:</p>
      <ul className="flex flex-col gap-4 mt-6">{children}</ul>
    </div>
  );
};

const Bullet: React.FC<{
  Icon: LucideIcon;
  label: string;
  iconColors: string;
  textColor?: string;
}> = ({ Icon, label, iconColors, textColor }) => {
  return (
    <li className="flex items-center gap-4">
      <div className={cn("h-6 w-6 flex items-center justify-center rounded-md", iconColors)}>
        <Icon className="w-3 h-3" />
      </div>
      <span className={cn("text-sm text-white whitespace-nowrap", textColor)}>{label}</span>
    </li>
  );
};

const PricingCardContent: React.FC<PropsWithChildren<{ layout?: "horizontal" | "vertical" }>> = ({
  children,
  layout,
}) => {
  return (
    <div
      className={cn("flex gap-8 p-8", {
        "flex-col": layout !== "horizontal",
      })}
    >
      {children}
    </div>
  );
};

const PricingCardFooter: React.FC<PropsWithChildren> = ({ children }) => {
  return <div className="p-8 border-t border-white/10">{children}</div>;
};

const Asterisk: React.FC<{ tag: string; label: string }> = ({ tag, label }) => {
  return (
    <div className="flex items-center gap-2">
      <span className="flex items-center justify-start w-20 h-6 px-2 text-sm font-semibold text-white rounded bg-white/10">
        {tag}
      </span>
      <span className="flex-grow w-full col-span-1 text-sm text-white/60">{label}</span>
    </div>
  );
};

const PricingCard: React.FC<
  PropsWithChildren<{ color: "#FFD600" | "#FFFFFF" | "#9D72FF"; className?: string }>
> = ({ children, color, className }) => {
  return (
    <div className={cn("relative h-full overflow-hidden  group/item", className)}>
      <div
        className={cn(
          "h-full relative bg-neutral-800 rounded-4xl p-px after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:transition-opacity after:duration-500  after:group-hover:opacity-100 after:z-10 overflow-hidden",
          // This is pretty annoying, but the only way I found to prevent tailwind from purging the class
          {
            "after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),#FFD600,transparent)]":
              color === "#FFD600",
            "after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),#FFFFFF,transparent)]":
              color === "#FFFFFF",
            "after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),#9D72FF,transparent)]":
              color === "#9D72FF",
          },
        )}
      >
        <div className="relative h-full bg-black rounded-[inherit] z-20 overflow-hidden ">
          {children}
        </div>
      </div>
    </div>
  );
};
type PricingCardProps = {
  tier: string;
  description: string;
  Highlight?: React.FC<{ className: string }>;
  color: `#${string}`;
  monthlyPrice: string;
  buttonLabel: string;
  particleIntensity: number;
};

const KeyIcon: React.FC<{ className?: string; color: `#${string}` }> = ({ className, color }) => {
  return (
    <svg
      className={className}
      width="26"
      height="26"
      viewBox="0 0 26 26"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M14.9998 14.9999L17.9998 17.9999L24.9998 10.9999L14.9998 0.999878L7.99976 7.99988L10.9998 10.9999L0.999756 20.9999V24.9999H6.99976V20.9999H10.9998V18.9999L14.9998 14.9999Z"
        fill={`url(#paint0_linear_2076_26_${color})`}
      />
      <path
        d="M14.9998 7.99988L17.9998 10.9999M0.999756 24.9999H6.99976V20.9999H10.9998V18.9999L14.9998 14.9999L10.9998 10.9999L0.999756 20.9999V24.9999ZM7.99976 7.99988L17.9998 17.9999L24.9998 10.9999L14.9998 0.999878L7.99976 7.99988Z"
        stroke={`url(#paint1_linear_2076_26_${color})`}
        stroke-width="0.75"
      />
      <defs>
        <linearGradient
          id={`paint0_linear_2076_26_${color}`}
          x1="12.9998"
          y1="0.999878"
          x2="12.9998"
          y2="24.9999"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color={color} stop-opacity="0" />
          <stop offset="0.5" stop-color={color} stop-opacity="0" />
          <stop offset="1" stop-color={color} stop-opacity="0.1" />
        </linearGradient>
        <linearGradient
          id={`paint1_linear_2076_26_${color}`}
          x1="12.9998"
          y1="0.999878"
          x2="12.9998"
          y2="24.9999"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color={color} />
          <stop offset="0.5" stop-color={color} stop-opacity="0.3" />
          <stop offset="1" stop-color={color} stop-opacity="0.1" />
        </linearGradient>
      </defs>
    </svg>
  );
};
const FreeCardHighlight: React.FC<{ className: string }> = ({ className }) => {
  return (
    <svg
      className={className}
      width="263"
      height="356"
      viewBox="0 0 263 356"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g opacity="0.4">
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_2076_3302)">
          <ellipse
            cx="16.3892"
            cy="146.673"
            rx="16.3892"
            ry="146.673"
            transform="matrix(-0.966169 -0.257911 -0.257911 0.966169 357.14 -44.3057)"
            fill="url(#paint0_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "color-dodge" }} filter="url(#filter1_f_2076_3302)">
          <ellipse
            cx="13.25"
            cy="146.625"
            rx="13.25"
            ry="146.625"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 347.709 -62.7411)"
            fill="url(#paint1_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_2076_3302)">
          <ellipse
            cx="11.1897"
            cy="190.642"
            rx="11.1897"
            ry="190.642"
            transform="matrix(-0.707107 -0.707107 -0.707107 0.707107 343.057 -64.2233)"
            fill="url(#paint2_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_2076_3302)">
          <ellipse
            cx="11.1897"
            cy="90.3336"
            rx="11.1897"
            ry="90.3336"
            transform="matrix(-0.707107 -0.707107 -0.707107 0.707107 343.057 -64.2235)"
            fill="url(#paint3_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_2076_3302)">
          <ellipse
            cx="11.125"
            cy="190.75"
            rx="11.125"
            ry="190.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 339.651 -49.7842)"
            fill="url(#paint4_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_2076_3302)">
          <ellipse
            cx="160.75"
            cy="93.75"
            rx="160.75"
            ry="93.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 476.963 11.8842)"
            fill="url(#paint5_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter6_f_2076_3302)">
          <ellipse
            cx="80.25"
            cy="47.75"
            rx="80.25"
            ry="47.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 407.248 -28.366)"
            fill="url(#paint6_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter7_f_2076_3302)">
          <ellipse
            cx="67.5"
            cy="40.125"
            rx="67.5"
            ry="40.125"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 390.581 -24.9982)"
            fill="url(#paint7_linear_2076_3302)"
            fill-opacity="0.5"
          />
        </g>
      </g>
      <defs>
        <filter
          id="filter0_f_2076_3302"
          x="217.957"
          y="-93.0969"
          width="171.039"
          height="372.55"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter1_f_2076_3302"
          x="144.206"
          y="-114.042"
          width="237.432"
          height="343.314"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter2_f_2076_3302"
          x="20.8005"
          y="-116.872"
          width="359.08"
          height="359.08"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter3_f_2076_3302"
          x="162.399"
          y="-117.131"
          width="217.741"
          height="217.741"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter4_f_2076_3302"
          x="94.2737"
          y="-99.9421"
          width="280.735"
          height="419.58"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter5_f_2076_3302"
          x="68.9412"
          y="-176.546"
          width="443.867"
          height="378.49"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter6_f_2076_3302"
          x="165.37"
          y="-159.758"
          width="297.01"
          height="265.24"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <filter
          id="filter7_f_2076_3302"
          x="175.242"
          y="-147.44"
          width="273.641"
          height="246.883"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3302" />
        </filter>
        <linearGradient
          id="paint0_linear_2076_3302"
          x1="16.3892"
          y1="0"
          x2="16.3892"
          y2="293.346"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_3302"
          x1="13.25"
          y1="0"
          x2="13.25"
          y2="293.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_3302"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="381.284"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_3302"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="180.667"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_3302"
          x1="11.125"
          y1="0"
          x2="11.125"
          y2="381.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_3302"
          x1="160.75"
          y1="0"
          x2="160.75"
          y2="187.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_3302"
          x1="80.25"
          y1="0"
          x2="80.25"
          y2="95.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_3302"
          x1="67.5"
          y1="0"
          x2="67.5"
          y2="80.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="white" />
          <stop offset="1" stop-color="white" stop-opacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
};

const ProCardHighlight: React.FC<{ className: string }> = ({ className }) => {
  return (
    <svg
      className={className}
      width="263"
      height="356"
      viewBox="0 0 263 356"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g opacity="0.4">
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_2076_3350)">
          <ellipse
            cx="16.3892"
            cy="146.673"
            rx="16.3892"
            ry="146.673"
            transform="matrix(-0.966169 -0.257911 -0.257911 0.966169 357.14 -44.3057)"
            fill="url(#paint0_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "color-dodge" }} filter="url(#filter1_f_2076_3350)">
          <ellipse
            cx="13.25"
            cy="146.625"
            rx="13.25"
            ry="146.625"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 347.709 -62.7411)"
            fill="url(#paint1_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_2076_3350)">
          <ellipse
            cx="11.1897"
            cy="190.642"
            rx="11.1897"
            ry="190.642"
            transform="matrix(-0.707107 -0.707107 -0.707107 0.707107 343.057 -64.2233)"
            fill="url(#paint2_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_2076_3350)">
          <ellipse
            cx="11.1897"
            cy="90.3336"
            rx="11.1897"
            ry="90.3336"
            transform="matrix(-0.707107 -0.707107 -0.707107 0.707107 343.057 -64.2235)"
            fill="url(#paint3_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_2076_3350)">
          <ellipse
            cx="11.125"
            cy="190.75"
            rx="11.125"
            ry="190.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 339.651 -49.7842)"
            fill="url(#paint4_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_2076_3350)">
          <ellipse
            cx="160.75"
            cy="93.75"
            rx="160.75"
            ry="93.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 476.963 11.8842)"
            fill="url(#paint5_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter6_f_2076_3350)">
          <ellipse
            cx="80.25"
            cy="47.75"
            rx="80.25"
            ry="47.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 407.248 -28.366)"
            fill="url(#paint6_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter7_f_2076_3350)">
          <ellipse
            cx="67.5"
            cy="40.125"
            rx="67.5"
            ry="40.125"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 390.581 -24.9982)"
            fill="url(#paint7_linear_2076_3350)"
            fill-opacity="0.5"
          />
        </g>
      </g>
      <defs>
        <filter
          id="filter0_f_2076_3350"
          x="217.957"
          y="-93.0969"
          width="171.039"
          height="372.55"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter1_f_2076_3350"
          x="144.206"
          y="-114.042"
          width="237.432"
          height="343.314"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter2_f_2076_3350"
          x="20.8005"
          y="-116.872"
          width="359.08"
          height="359.08"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter3_f_2076_3350"
          x="162.399"
          y="-117.131"
          width="217.741"
          height="217.741"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter4_f_2076_3350"
          x="94.2738"
          y="-99.9421"
          width="280.735"
          height="419.58"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter5_f_2076_3350"
          x="68.9412"
          y="-176.546"
          width="443.867"
          height="378.49"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter6_f_2076_3350"
          x="165.37"
          y="-159.758"
          width="297.01"
          height="265.24"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <filter
          id="filter7_f_2076_3350"
          x="175.242"
          y="-147.44"
          width="273.641"
          height="246.883"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3350" />
        </filter>
        <linearGradient
          id="paint0_linear_2076_3350"
          x1="16.3892"
          y1="0"
          x2="16.3892"
          y2="293.346"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_3350"
          x1="13.25"
          y1="0"
          x2="13.25"
          y2="293.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_3350"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="381.284"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_3350"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="180.667"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_3350"
          x1="11.125"
          y1="0"
          x2="11.125"
          y2="381.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_3350"
          x1="160.75"
          y1="0"
          x2="160.75"
          y2="187.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_3350"
          x1="80.25"
          y1="0"
          x2="80.25"
          y2="95.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_3350"
          x1="67.5"
          y1="0"
          x2="67.5"
          y2="80.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#FFD600" />
          <stop offset="1" stop-color="#FFD600" stop-opacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
};

const CustomCardHighlight: React.FC<{ className: string }> = ({ className }) => {
  return (
    <svg
      className={className}
      width="379"
      height="336"
      viewBox="0 0 379 336"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g opacity="0.4">
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_2076_3437)">
          <ellipse
            cx="16.3892"
            cy="146.673"
            rx="16.3892"
            ry="146.673"
            transform="matrix(-0.966169 -0.257911 -0.257911 0.966169 357.14 -44.3058)"
            fill="url(#paint0_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "color-dodge" }} filter="url(#filter1_f_2076_3437)">
          <ellipse
            cx="13.25"
            cy="146.625"
            rx="13.25"
            ry="146.625"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 347.709 -62.7412)"
            fill="url(#paint1_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_2076_3437)">
          <ellipse
            cx="11.1897"
            cy="190.642"
            rx="11.1897"
            ry="190.642"
            transform="matrix(-0.707107 -0.707107 -0.707107 0.707107 343.057 -64.2233)"
            fill="url(#paint2_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_2076_3437)">
          <ellipse
            cx="11.1897"
            cy="90.3336"
            rx="11.1897"
            ry="90.3336"
            transform="matrix(-0.707107 -0.707107 -0.707107 0.707107 343.057 -64.2235)"
            fill="url(#paint3_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_2076_3437)">
          <ellipse
            cx="11.125"
            cy="190.75"
            rx="11.125"
            ry="190.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 339.651 -49.7842)"
            fill="url(#paint4_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_2076_3437)">
          <ellipse
            cx="160.75"
            cy="93.75"
            rx="160.75"
            ry="93.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 476.963 11.8842)"
            fill="url(#paint5_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter6_f_2076_3437)">
          <ellipse
            cx="80.25"
            cy="47.75"
            rx="80.25"
            ry="47.75"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 407.248 -28.366)"
            fill="url(#paint6_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter7_f_2076_3437)">
          <ellipse
            cx="67.5"
            cy="40.125"
            rx="67.5"
            ry="40.125"
            transform="matrix(-0.866025 -0.5 -0.5 0.866025 390.581 -24.9983)"
            fill="url(#paint7_linear_2076_3437)"
            fill-opacity="0.5"
          />
        </g>
      </g>
      <defs>
        <filter
          id="filter0_f_2076_3437"
          x="217.957"
          y="-93.097"
          width="171.039"
          height="372.55"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter1_f_2076_3437"
          x="144.206"
          y="-114.042"
          width="237.432"
          height="343.314"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter2_f_2076_3437"
          x="20.8005"
          y="-116.872"
          width="359.08"
          height="359.08"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter3_f_2076_3437"
          x="162.399"
          y="-117.131"
          width="217.741"
          height="217.741"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter4_f_2076_3437"
          x="94.2737"
          y="-99.9421"
          width="280.735"
          height="419.58"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter5_f_2076_3437"
          x="68.9412"
          y="-176.546"
          width="443.867"
          height="378.49"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter6_f_2076_3437"
          x="165.37"
          y="-159.758"
          width="297.01"
          height="265.24"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <filter
          id="filter7_f_2076_3437"
          x="175.242"
          y="-147.441"
          width="273.641"
          height="246.883"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_3437" />
        </filter>
        <linearGradient
          id="paint0_linear_2076_3437"
          x1="16.3892"
          y1="0"
          x2="16.3892"
          y2="293.346"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_3437"
          x1="13.25"
          y1="0"
          x2="13.25"
          y2="293.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_3437"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="381.284"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_3437"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="180.667"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_3437"
          x1="11.125"
          y1="0"
          x2="11.125"
          y2="381.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_3437"
          x1="160.75"
          y1="0"
          x2="160.75"
          y2="187.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_3437"
          x1="80.25"
          y1="0"
          x2="80.25"
          y2="95.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_3437"
          x1="67.5"
          y1="0"
          x2="67.5"
          y2="80.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stop-color="#6E56CF" />
          <stop offset="1" stop-color="#6E56CF" stop-opacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
};

const Separator: React.FC<{ orientation?: "horizontal" | "vertical" }> = ({
  orientation = "horizontal",
}) => (
  <div
    className={cn(
      "shrink-0 bg-white/10",
      orientation === "horizontal" ? "h-[1px] w-full" : "h-full w-[1px] ",
    )}
  />
);
