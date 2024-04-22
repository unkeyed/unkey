import { Particles } from "@/components/particles";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import type { PropsWithChildren } from "react";

export enum Color {
  White = "#FFFFFF",
  Yellow = "#FFD600",
  Purple = "#9D72FF",
}

export const PricingCardHeader: React.FC<{
  title: string;
  description: React.ReactNode;
  className?: string;
  color: Color;
  withIcon?: boolean;
}> = ({ title, description, className, color, withIcon = true }) => {
  return (
    <div
      className={cn(
        "p-10 flex items-start justify-between w-full gap-10 min-h-40 relative ",
        className,
      )}
    >
      <div>
        <span className="bg-gradient-to-br text-transparent bg-gradient-stop  bg-clip-text from-white via-white via-30% to-white/30 font-medium ">
          {title}
        </span>
        <p className="mt-4 text-sm text-white/60">{description}</p>
      </div>
      {withIcon ? (
        <div
          className={cn(
            "relative z-30 flex items-center justify-center ring-1 h-14 min-w-14 w-14 duration-150 rounded-xl backdrop-blur rounded-2 overflow-hidden drop-shadow-[0_20px_20px_rgba(256,0,0,1) ]",
            {
              " ring-white/10 hover:ring-white/25 ": color === Color.White,
              " ring-[#FFD600]/10 hover:ring-[#FFD600]/25": color === Color.Yellow,
              " ring-[#9D72FF]/10 hover:ring-[#9D72FF]/25": color === Color.Purple,
            },
          )}
        >
          <Particles
            className="absolute inset-0 duration-500 opacity-50 -z-10 group-hover:opacity-100"
            quantity={color === Color.White ? 10 : color === Color.Yellow ? 20 : 40}
            color={color}
            vy={color === Color.White ? -0.05 : color === Color.Yellow ? -0.1 : -0.15}
          />
          <div
            className={cn("absolute -top-1  bg-gradient-radial  blur h-6 w-8 ", {
              "from-white/50": color === Color.White,
              "from-[#FFD600]/50": color === Color.Yellow,
              "from-[#9D72FF]/50": color === Color.Purple,
            })}
          />
          <KeyIcon color={color} />
        </div>
      ) : null}
    </div>
  );
};

export const Cost: React.FC<{ dollar: string; className?: string }> = ({ dollar, className }) => {
  return (
    <div className={cn("flex items-center gap-4", className)}>
      <span className="text-4xl font-semibold text-transparent bg-gradient-to-br bg-clip-text from-white via-white to-white/30">
        {dollar}
      </span>
      <span className=" text-white/60">/ month</span>
    </div>
  );
};

export const Button: React.FC<{ label: string }> = ({ label }) => {
  return (
    <div>
      <Link href="https://app.unkey.dev">
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

export const Bullets: React.FC<PropsWithChildren> = ({ children }) => {
  return (
    <div>
      <p className="text-white/50">What's included:</p>
      <ul className="flex flex-col gap-4 mt-6">{children}</ul>
    </div>
  );
};

export const Bullet: React.FC<{
  Icon: LucideIcon;
  label: string;
  color: Color;
  textColor?: string;
  className?: string;
}> = ({ Icon, label, color, textColor, className }) => {
  return (
    <div className={cn("flex items-center gap-4", className)}>
      <div
        className={cn("h-6 min-w-6 w-6 flex items-center justify-center rounded-md", {
          "text-[#FFFFFF] bg-[#FFFFFF]/10": color === Color.White,
          "text-[#FFD600] bg-[#FFD600]/10": color === Color.Yellow,
          "text-[#9D72FF] bg-[#9D72FF]/10": color === Color.Purple,
        })}
      >
        <Icon className="w-3 h-3" />
      </div>
      <span className={cn("text-sm text-white md:whitespace-nowrap sm:text-xs", textColor)}>
        {label}
      </span>
    </div>
  );
};

export const PricingCardContent: React.FC<
  PropsWithChildren<{ layout?: "horizontal" | "vertical" }>
> = ({ children, layout }) => {
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

export const PricingCardFooter: React.FC<PropsWithChildren> = ({ children }) => {
  return <div className="p-8 border-t border-white/10">{children}</div>;
};

export const Asterisk: React.FC<{ tag: string; label?: string }> = ({ tag, label }) => {
  return (
    <div className="flex items-center gap-2">
      <span className="flex items-center justify-start w-20 h-6 px-2 text-sm font-semibold text-white rounded bg-white/10">
        {tag}
      </span>
      <span className="flex-grow w-full col-span-1 text-sm text-white/60">{label}</span>
    </div>
  );
};

export const PricingCard: React.FC<PropsWithChildren<{ color: Color; className?: string }>> = ({
  children,
  color,
  className,
}) => {
  return (
    <div className={cn("relative h-full overflow-hidden  group/item", className)}>
      <div
        className={cn(
          "h-full relative bg-neutral-800 rounded-4xl p-px after:absolute after:inset-0 after:rounded-[inherit] after:opacity-0 after:transition-opacity after:duration-500  after:group-hover:opacity-100 after:z-10 overflow-hidden",
          // This is pretty annoying, but the only way I found to prevent tailwind from purging the class
          {
            "after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),#FFD600,transparent)]":
              color === Color.Yellow,
            "after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),#FFFFFF,transparent)]":
              color === Color.White,
            "after:[background:_radial-gradient(250px_circle_at_var(--mouse-x)_var(--mouse-y),#9D72FF,transparent)]":
              color === Color.Purple,
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

export const KeyIcon: React.FC<{ className?: string; color: Color }> = ({ className, color }) => {
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
        strokeWidth="0.75"
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
          <stop stopColor={color} stopOpacity="0" />
          <stop offset="0.5" stopColor={color} stopOpacity="0" />
          <stop offset="1" stopColor={color} stopOpacity="0.1" />
        </linearGradient>
        <linearGradient
          id={`paint1_linear_2076_26_${color}`}
          x1="12.9998"
          y1="0.999878"
          x2="12.9998"
          y2="24.9999"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor={color} />
          <stop offset="0.5" stopColor={color} stopOpacity="0.3" />
          <stop offset="1" stopColor={color} stopOpacity="0.1" />
        </linearGradient>
      </defs>
    </svg>
  );
};
export const FreeCardHighlight: React.FC<{ className: string }> = ({ className }) => {
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_3302"
          x1="13.25"
          y1="0"
          x2="13.25"
          y2="293.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_3302"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="381.284"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_3302"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="180.667"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_3302"
          x1="11.125"
          y1="0"
          x2="11.125"
          y2="381.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_3302"
          x1="160.75"
          y1="0"
          x2="160.75"
          y2="187.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_3302"
          x1="80.25"
          y1="0"
          x2="80.25"
          y2="95.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_3302"
          x1="67.5"
          y1="0"
          x2="67.5"
          y2="80.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
};

export const ProCardHighlight: React.FC<{ className: string }> = ({ className }) => {
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_3350"
          x1="13.25"
          y1="0"
          x2="13.25"
          y2="293.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_3350"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="381.284"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_3350"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="180.667"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_3350"
          x1="11.125"
          y1="0"
          x2="11.125"
          y2="381.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_3350"
          x1="160.75"
          y1="0"
          x2="160.75"
          y2="187.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_3350"
          x1="80.25"
          y1="0"
          x2="80.25"
          y2="95.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_3350"
          x1="67.5"
          y1="0"
          x2="67.5"
          y2="80.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFD600" />
          <stop offset="1" stopColor="#FFD600" stopOpacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
};

export const EnterpriseCardHighlight: React.FC<{ className: string }> = ({ className }) => {
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
            fillOpacity="0.5"
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
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
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_3437"
          x1="13.25"
          y1="0"
          x2="13.25"
          y2="293.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_3437"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="381.284"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_3437"
          x1="11.1897"
          y1="0"
          x2="11.1897"
          y2="180.667"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_3437"
          x1="11.125"
          y1="0"
          x2="11.125"
          y2="381.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_3437"
          x1="160.75"
          y1="0"
          x2="160.75"
          y2="187.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_3437"
          x1="80.25"
          y1="0"
          x2="80.25"
          y2="95.5"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_3437"
          x1="67.5"
          y1="0"
          x2="67.5"
          y2="80.25"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#6E56CF" />
          <stop offset="1" stopColor="#6E56CF" stopOpacity="0" />
        </linearGradient>
      </defs>
    </svg>
  );
};

export const Separator: React.FC<{ orientation?: "horizontal" | "vertical"; className?: string }> =
  ({ orientation = "horizontal", className }) => (
    <div
      className={cn(
        "shrink-0 bg-white/10",
        orientation === "horizontal" ? "h-[1px] w-full" : "h-full w-[1px] ",
        className,
      )}
    />
  );

export const BelowEnterpriseSvg: React.FC<{ className?: string }> = ({ className }) => {
  return (
    <svg
      className={className}
      width="472"
      height="638"
      viewBox="0 0 472 638"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <g opacity="0.4">
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_2076_2453)">
          <ellipse
            cx="184.597"
            cy="353.647"
            rx="16.3892"
            ry="146.673"
            transform="rotate(15.0538 184.597 353.647)"
            fill="url(#paint0_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "color-dodge" }} filter="url(#filter1_f_2076_2453)">
          <ellipse
            cx="237.5"
            cy="343.125"
            rx="13.25"
            ry="146.625"
            fill="url(#paint1_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_2076_2453)">
          <ellipse
            cx="289.17"
            cy="378.792"
            rx="11.1897"
            ry="190.642"
            transform="rotate(-15 289.17 378.792)"
            fill="url(#paint2_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_2076_2453)">
          <ellipse
            cx="263.208"
            cy="281.902"
            rx="11.1897"
            ry="90.3336"
            transform="rotate(-15 263.208 281.902)"
            fill="url(#paint3_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_2076_2453)">
          <ellipse
            cx="235.875"
            cy="402.5"
            rx="11.125"
            ry="190.75"
            fill="url(#paint4_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_2076_2453)">
          <ellipse
            cx="235.75"
            cy="290.25"
            rx="160.75"
            ry="93.75"
            fill="url(#paint5_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter6_f_2076_2453)">
          <ellipse
            cx="235.75"
            cy="244.25"
            rx="80.25"
            ry="47.75"
            fill="url(#paint6_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
        <g style={{ mixBlendMode: "lighten" }} filter="url(#filter7_f_2076_2453)">
          <ellipse
            cx="235.75"
            cy="247.875"
            rx="67.5"
            ry="40.125"
            fill="url(#paint7_linear_2076_2453)"
            fillOpacity="0.5"
          />
        </g>
      </g>
      <mask id="path-9-inside-1_2076_2453" fill="white">
        <path d="M204 161H212V593H204V161Z" />
      </mask>
      <path
        d="M211.5 161V593H212.5V161H211.5ZM204.5 593V161H203.5V593H204.5Z"
        fill="url(#paint8_angular_2076_2453)"
        fillOpacity="0.5"
        mask="url(#path-9-inside-1_2076_2453)"
      />
      <mask id="path-11-inside-2_2076_2453" fill="white">
        <path d="M180 51H188V483H180V51Z" />
      </mask>
      <path
        d="M187.5 51V483H188.5V51H187.5ZM180.5 483V51H179.5V483H180.5Z"
        fill="url(#paint9_angular_2076_2453)"
        fillOpacity="0.2"
        mask="url(#path-11-inside-2_2076_2453)"
      />
      <mask id="path-13-inside-3_2076_2453" fill="white">
        <path d="M228 101H236V533H228V101Z" />
      </mask>
      <path
        d="M235.5 101V533H236.5V101H235.5ZM228.5 533V101H227.5V533H228.5Z"
        fill="url(#paint10_angular_2076_2453)"
        fillOpacity="0.3"
        mask="url(#path-13-inside-3_2076_2453)"
      />
      <mask id="path-15-inside-4_2076_2453" fill="white">
        <path d="M252 191H260V623H252V191Z" />
      </mask>
      <path
        d="M259.5 191V623H260.5V191H259.5ZM252.5 623V191H251.5V623H252.5Z"
        fill="url(#paint11_angular_2076_2453)"
        fillOpacity="0.8"
        mask="url(#path-15-inside-4_2076_2453)"
      />
      <mask id="path-17-inside-5_2076_2453" fill="white">
        <path d="M276 1H284V433H276V1Z" />
      </mask>
      <path
        d="M283.5 1V433H284.5V1H283.5ZM276.5 433V1H275.5V433H276.5Z"
        fill="url(#paint12_angular_2076_2453)"
        fillOpacity="0.1"
        mask="url(#path-17-inside-5_2076_2453)"
      />
      <defs>
        <filter
          id="filter0_f_2076_2453"
          x="98.8346"
          y="167.442"
          width="171.525"
          height="372.409"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter1_f_2076_2453"
          x="179.75"
          y="152"
          width="115.5"
          height="382.25"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter2_f_2076_2453"
          x="194.148"
          y="150.123"
          width="190.045"
          height="457.338"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter3_f_2076_2453"
          x="192.945"
          y="150.097"
          width="140.527"
          height="263.609"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter4_f_2076_2453"
          x="180.25"
          y="167.25"
          width="111.25"
          height="470.5"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="22.25" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter5_f_2076_2453"
          x="-5.34058e-05"
          y="121.5"
          width="471.5"
          height="337.5"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter6_f_2076_2453"
          x="80.4999"
          y="121.5"
          width="310.5"
          height="245.5"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <filter
          id="filter7_f_2076_2453"
          x="93.2499"
          y="132.75"
          width="285"
          height="230.25"
          filterUnits="userSpaceOnUse"
          colorInterpolationFilters="sRGB"
        >
          <feFlood floodOpacity="0" result="BackgroundImageFix" />
          <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
          <feGaussianBlur stdDeviation="37.5" result="effect1_foregroundBlur_2076_2453" />
        </filter>
        <linearGradient
          id="paint0_linear_2076_2453"
          x1="184.597"
          y1="206.974"
          x2="184.597"
          y2="500.319"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint1_linear_2076_2453"
          x1="237.5"
          y1="196.5"
          x2="237.5"
          y2="489.75"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_2076_2453"
          x1="289.17"
          y1="188.151"
          x2="289.17"
          y2="569.434"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_2076_2453"
          x1="263.208"
          y1="191.568"
          x2="263.208"
          y2="372.236"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint4_linear_2076_2453"
          x1="235.875"
          y1="211.75"
          x2="235.875"
          y2="593.251"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint5_linear_2076_2453"
          x1="235.75"
          y1="196.5"
          x2="235.75"
          y2="384.001"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_2076_2453"
          x1="235.75"
          y1="196.5"
          x2="235.75"
          y2="292"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <linearGradient
          id="paint7_linear_2076_2453"
          x1="235.75"
          y1="207.75"
          x2="235.75"
          y2="288"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </linearGradient>
        <radialGradient
          id="paint8_angular_2076_2453"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(208 481) scale(32 185)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.784842" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <radialGradient
          id="paint9_angular_2076_2453"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(184 371) scale(32 185)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.784842" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <radialGradient
          id="paint10_angular_2076_2453"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(232 421) scale(32 185)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.784842" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <radialGradient
          id="paint11_angular_2076_2453"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(256 511) scale(32 185)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.784842" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <radialGradient
          id="paint12_angular_2076_2453"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(280 321) scale(32 185)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.784842" stopColor="white" stopOpacity="0" />
        </radialGradient>
      </defs>
    </svg>
  );
};
