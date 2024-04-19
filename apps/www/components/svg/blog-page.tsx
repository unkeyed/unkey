import { cn } from "@/lib/utils";

export const BlogBackgroundLines = ({
  className,
}: {
  className?: string;
}) => (
  <svg
    className={cn("absolute top-0 left-0 right-0 pointer-events-none", className)}
    viewBox="0 0 717 393"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path d="M319.5 -159.658L1 392" stroke="url(#paint0_linear_4046_6105)" strokeWidth="0.75" />
    <path
      d="M319.5 -159.658L1 392"
      stroke="url(#paint1_angular_4046_6105)"
      strokeOpacity="0.1"
      strokeWidth="0.75"
    />
    <path d="M518 -159.658L199.5 392" stroke="url(#paint2_linear_4046_6105)" strokeWidth="0.75" />
    <path
      d="M518 -159.658L199.5 392"
      stroke="url(#paint3_angular_4046_6105)"
      strokeOpacity="0.4"
      strokeWidth="0.75"
    />
    <path d="M716.5 -159.658L398 392" stroke="url(#paint4_linear_4046_6105)" strokeWidth="0.75" />
    <path
      d="M716.5 -159.658L398 392"
      stroke="url(#paint5_angular_4046_6105)"
      strokeOpacity="0.5"
      strokeWidth="0.75"
    />
    <path
      d="M716.5 -159.658L398 392"
      stroke="url(#paint5_linear_4046_6105)"
      strokeOpacity="0.5"
      strokeWidth="0.75"
    />
    <defs>
      <linearGradient
        id="paint0_linear_4046_6105"
        x1="319.746"
        y1="-160.003"
        x2="0.748781"
        y2="391.999"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="white" stopOpacity="0.15" />
        <stop offset="1" stopColor="white" stopOpacity="0" />
      </linearGradient>
      <radialGradient
        id="paint1_angular_4046_6105"
        cx="0"
        cy="0"
        r="1"
        gradientUnits="userSpaceOnUse"
        gradientTransform="translate(463.5 27) rotate(-158.499) scale(106.405 106.405)"
      >
        <stop offset=".0" stopColor="white" stopOpacity="0" />
        <stop offset="1.5" stopColor="white" stopOpacity="1" />
        <stop offset="0" stopColor="white" stopOpacity="0" />
      </radialGradient>
      <linearGradient
        id="paint2_linear_4046_6105"
        x1="518.246"
        y1="-160.003"
        x2="199.249"
        y2="391.999"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="white" stopOpacity="0.15" />
        <stop offset="1" stopColor="white" stopOpacity="0" />
      </linearGradient>
      <radialGradient
        id="paint3_angular_4046_6105"
        cx="0"
        cy="0"
        r="1"
        gradientUnits="userSpaceOnUse"
        gradientTransform="translate(463.5 27) rotate(-158.499) scale(106.405 106.405)"
      >
        <stop offset=".0" stopColor="white" stopOpacity="0" />
        <stop offset=".5" stopColor="white" stopOpacity="1" />
        <stop offset="0" stopColor="white" stopOpacity="0" />
      </radialGradient>
      <linearGradient
        id="paint4_linear_4046_6105"
        x1="716.746"
        y1="-160.003"
        x2="397.749"
        y2="391.999"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="white" stopOpacity="0.15" />
        <stop offset="1" stopColor="white" stopOpacity="0" />
      </linearGradient>
      <linearGradient
        id="paint5_linear_4046_6105"
        x1="319.746"
        y1="-160.003"
        x2="0.748781"
        y2="391.999"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset=".17" stopColor="white" stopOpacity="0" />
        <stop offset=".3" stopColor="white" stopOpacity="1" />
        <stop offset="0" stopColor="white" stopOpacity="0" />
      </linearGradient>
    </defs>
  </svg>
);
