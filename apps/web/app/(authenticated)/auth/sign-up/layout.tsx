import { FadeIn } from "@/components/landing/fade-in";
import { auth } from "@clerk/nextjs";
import { redirect } from "next/navigation";

const features: {
  title: string;
  description: string;
}[] = [
  {
    title: "Save development time",
    description: "Issue, manage, and revoke keys for your APIs in seconds with built in analytics.",
  },
  {
    title: "Globally distributed",
    description: "Unkey Globally distrubtes keys in 35+ locations, making it fast for every user.",
  },
  {
    title: "Features for any use case",
    description:
      "Each key has unique settings such as rate limiting, expiration, and limited uses.",
  },
];

export const runtime = "edge";

export default function AuthLayout(props: { children: React.ReactNode }) {
  const { userId } = auth();

  if (userId) {
    return redirect("/app/apis");
  }
  return (
    <FadeIn>
      <div className="relative grid min-h-screen grid-cols-1 overflow-hidden md:grid-cols-3 lg:grid-cols-2">
        <div className="from-background to-background/60 absolute inset-0 bg-gradient-to-t md:hidden" />
        <div className="container absolute top-1/2 col-span-1 flex -translate-y-1/2 items-center md:static md:top-0 md:col-span-2 md:flex md:translate-y-0 lg:col-span-1">
          {props.children}
        </div>
        <div className="relative hidden items-center justify-center bg-white md:flex md:bg-black ">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="57.5"
            height="57.5"
            viewBox="0 0 113 113"
            fill="none"
            style={{
              position: "absolute",
              top: 70,
              left: 80,
              backgroundColor: "black",
            }}
            className="hidden md:block"
          >
            <rect
              x="113"
              width="113"
              height="113"
              rx="8"
              transform="rotate(90 113 0)"
              fill="url(#paint0_linear_107_29)"
            />
            <rect
              x="112.5"
              y="0.5"
              width="112"
              height="112"
              rx="7.5"
              transform="rotate(90 112.5 0.5)"
              stroke="url(#paint1_linear_107_29)"
              stroke-opacity="0.8"
            />
            <g filter="url(#filter0_d_107_29)">
              <path
                d="M66.3326 25H83V64.2966C83 68.969 81.8808 73.0125 79.6425 76.427C77.4242 79.8215 74.3265 82.4473 70.3495 84.3043C66.3725 86.1413 61.756 87.0598 56.5 87.0598C51.204 87.0598 46.5675 86.1413 42.5905 84.3043C38.6135 82.4473 35.5158 79.8215 33.2975 76.427C31.0992 73.0125 30 68.969 30 64.2966V25H46.6674V62.8589C46.6674 64.7558 47.0871 66.4531 47.9265 67.9507C48.7658 69.4283 49.925 70.5864 51.4038 71.4251C52.9027 72.2637 54.6014 72.683 56.5 72.683C58.4185 72.683 60.1173 72.2637 61.5962 71.4251C63.075 70.5864 64.2342 69.4283 65.0735 67.9507C65.9129 66.4531 66.3326 64.7558 66.3326 62.8589V25Z"
                fill="url(#paint2_linear_107_29)"
                shape-rendering="crispEdges"
              />
              <path
                d="M66.3326 24.7735H66.1061V25V62.8589C66.1061 64.7211 65.6945 66.3794 64.8761 67.8396C64.057 69.2812 62.9275 70.4097 61.4844 71.228C60.0438 72.045 58.3845 72.4565 56.5 72.4565C54.6363 72.4565 52.9766 72.0453 51.5152 71.2278C50.0722 70.4094 48.9428 69.281 48.1237 67.8393C47.3055 66.3792 46.8939 64.721 46.8939 62.8589V25V24.7735H46.6674H30H29.7735V25V64.2966C29.7735 69.0058 30.8818 73.0933 33.1071 76.5496L33.1079 76.5509C35.3512 79.9836 38.4827 82.6362 42.4947 84.5095L42.4955 84.5099C46.5078 86.3633 51.1779 87.2863 56.5 87.2863C61.7824 87.2863 66.4324 86.3632 70.4445 84.5099L70.4454 84.5095C74.4573 82.6362 77.5888 79.9836 79.8321 76.5509C82.0979 73.0945 83.2265 69.0065 83.2265 64.2966V25V24.7735H83H66.3326Z"
                stroke="url(#paint3_linear_107_29)"
                stroke-width="0.452991"
                shape-rendering="crispEdges"
              />
            </g>
            <defs>
              <filter
                id="filter0_d_107_29"
                x="27.7352"
                y="24.5471"
                width="57.5297"
                height="66.5898"
                filterUnits="userSpaceOnUse"
                color-interpolation-filters="sRGB"
              >
                <feFlood flood-opacity="0" result="BackgroundImageFix" />
                <feColorMatrix
                  in="SourceAlpha"
                  type="matrix"
                  values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
                  result="hardAlpha"
                />
                <feOffset dy="1.81197" />
                <feGaussianBlur stdDeviation="0.905983" />
                <feComposite in2="hardAlpha" operator="out" />
                <feColorMatrix type="matrix" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0" />
                <feBlend
                  mode="normal"
                  in2="BackgroundImageFix"
                  result="effect1_dropShadow_107_29"
                />
                <feBlend
                  mode="normal"
                  in="SourceGraphic"
                  in2="effect1_dropShadow_107_29"
                  result="shape"
                />
              </filter>
              <linearGradient
                id="paint0_linear_107_29"
                x1="34"
                y1="309"
                x2="261.749"
                y2="157.11"
                gradientUnits="userSpaceOnUse"
              >
                <stop stop-color="white" />
                <stop offset="1" stop-color="white" stop-opacity="0" />
              </linearGradient>
              <linearGradient
                id="paint1_linear_107_29"
                x1="79"
                y1="146"
                x2="241.326"
                y2="80.518"
                gradientUnits="userSpaceOnUse"
              >
                <stop offset="0.194498" stop-color="white" />
                <stop offset="0.411458" stop-color="white" stop-opacity="0" />
              </linearGradient>
              <linearGradient
                id="paint2_linear_107_29"
                x1="30"
                y1="25"
                x2="73.6772"
                y2="87.0855"
                gradientUnits="userSpaceOnUse"
              >
                <stop stop-color="white" />
                <stop offset="1" stop-color="white" stop-opacity="0" />
              </linearGradient>
              <linearGradient
                id="paint3_linear_107_29"
                x1="15.2778"
                y1="69.6197"
                x2="90.2476"
                y2="69.3306"
                gradientUnits="userSpaceOnUse"
              >
                <stop offset="0.194498" stop-color="white" />
                <stop offset="0.411458" stop-color="white" stop-opacity="0" />
              </linearGradient>
            </defs>
          </svg>
          <div className="hidden md:block lg:pr-4 lg:pt-4">
            {features.map((feature) => (
              <div key={feature.title} className="mb-8 lg:max-w-lg">
                <h3 className="my-2 text-3xl font-bold tracking-tight text-gray-100 sm:text-4xl">
                  {feature.title}
                </h3>
                <p className="text-lg leading-8 text-gray-400">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </FadeIn>
  );
}
