import { RainbowDarkButton } from "@/components/button";
import { SectionTitle } from "@/components/section-title";

import { ChangelogLight } from "@/components/svg/changelog";
import { ArrowRight } from "lucide-react";

export default function Page() {
  return (
    <div className="min-h-screen mt-[200px] flex flex-col items-center">
      <ChangelogLight className="absolute" />
      <RainbowDarkButton label="We're hiring!" IconRight={ArrowRight} />
      <SectionTitle
        title="API auth for fast and scalable software"
        titleWidth={680}
        contentWidth={680}
        align="center"
        text="Unkey simplifies API authentication and authorization, making securin g and managing APIs effortless. The platform delivers a fast and seamless developer experience for creating and verifying API keys, ensuring smooth integration and robust security."
      />
      <div className="relative px-[144px] py-[120px] text-white flex flex-col items-center rounded-[48px] border-l border-r border-b border-white/10 max-w-[1000px]">
        <h2 className="text-[32px] font-medium leading-[48px] text-center">
          Founded to level up the API authentication landscape
        </h2>
        <p className="mt-[40px] text-white/50 leading-[32px] max-w-[720px] text-center">
          Unkey emerged in 2023 from the frustration of <span>James Perkins</span> and
          <span> Andreas Thomas</span> with the lack of a straightforward, fast, and scalable API
          authentication solution. This void prompted a mission to create a tool themselves. Thus,
          the platform was born, driven by their shared determination to simplify API authentication
          and democratize access for all developers. Today, the solution stands as a powerful tool,
          continuously evolving to meet the dynamic needs of a worldwide developer community.
        </p>
        <div className="absolute bottom-0">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="943"
            height="272"
            viewBox="0 0 943 272"
            fill="none"
          >
            <g style={{ mixBlendMode: "lighten" }} filter="url(#filter0_f_8003_5216)">
              <ellipse
                cx="321.5"
                cy="187.5"
                rx="321.5"
                ry="187.5"
                transform="matrix(1 1.37458e-08 -1.37458e-08 -1 150 525.5)"
                fill="url(#paint0_linear_8003_5216)"
                fill-opacity="0.5"
              />
            </g>
            <defs>
              <filter
                id="filter0_f_8003_5216"
                x="0"
                y="0.5"
                width="943"
                height="675"
                filterUnits="userSpaceOnUse"
                color-interpolation-filters="sRGB"
              >
                <feFlood flood-opacity="0" result="BackgroundImageFix" />
                <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
                <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_8003_5216" />
              </filter>
              <linearGradient
                id="paint0_linear_8003_5216"
                x1="321.5"
                y1="0"
                x2="321.5"
                y2="375"
                gradientUnits="userSpaceOnUse"
              >
                <stop stop-color="white" />
                <stop offset="1" stop-color="white" stop-opacity="0" />
              </linearGradient>
            </defs>
          </svg>
        </div>
      </div>
      <SectionTitle
        className="mt-20"
        align="center"
        title="And now, we got people to take care of"
        titleWidth={640}
        contentWidth={640}
        text="We grew in number, and we love that. Here are some of our precious moments. Although we collaborate as a fully remote team, occasionally we unite!"
      />
    </div>
  );
}
