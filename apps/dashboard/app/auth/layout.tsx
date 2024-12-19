import { FadeIn } from "@/components/landing/fade-in";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { auth } from "@/lib/auth/server";
import { FileText } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import type React from "react";

export const dynamic = "force-dynamic";

const quotes: {
  text: React.ReactNode;
  source: string; // just for internal reference
  author: {
    name: string;
    title?: string;
    image: string;
    href: string;
  };
}[] = [
  {
    text: "Unkey's product helped launch our public API features in a matter of hours. Easy setup, exactly the features we needed, great DX ergonomics, and low latency - fantastic experience.",
    source: "slack dm",
    author: {
      name: "Rick Blalock",
      title: "Cofounder/CTO onestudy.ai",
      image: "/images/quoteImages/rick-blalock.jpg",
      href: "https://x.com/rblalock",
    },
  },
  {
    text: "The product is super well built, intuitive and powerful.",
    source: "slack",
    author: {
      name: "Dexter Storey",
      title: "Cofounder Rubric Labs",
      image: "/images/quoteImages/dexter-storey.jpg",
      href: "https://www.linkedin.com/in/dexterstorey/",
    },
  },
  {
    text: "Just used Unkey, by far the easiest and cheapest ( its free ) solution I have used so far for saas to manage their api keys. Its amazing how easy it is use.",
    source: "https://x.com/tkejr_/status/1731613302378164440",
    author: {
      name: "Tanmay",
      image: "/images/quoteImages/tanmay.jpg",
      href: "https://x.com/tkejr_",
    },
  },
  {
    text: "Diving into Unkey for a project, and I'm impressed! Love the straightforward setup for managing API keys.",
    source: "https://x.com/ojabowalola/status/1724134790670999919",
    author: {
      name: "Lola",
      image: "/images/quoteImages/lola.jpg",
      href: "https://x.com/ojabowalola",
      title: "Founder of lunchpaillabs.com",
    },
  },
  {
    text: "Unkey increases our velocity and helps us focus on what's relevant to the user, not the infrastructure behind it.",
    source: "https://www.openstatus.dev/blog/secure-api-with-unkey",
    author: {
      name: "Maximilian Kaske",
      image: "/images/quoteImages/maximilian-kaske.jpg",
      href: "https://x.com/mxkaske",
      title: "Founder of openstatus.dev",
    },
  },
];

export default async function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await auth.getCurrentUser();

  if (user) {
    return redirect("/apis");
  }
  const quote = quotes[Math.floor(Math.random() * quotes.length)];

  return (
    <div className="bg-black">
      <div className="absolute top-0 left-0 flex justify-start w-screen overflow-hidden pointer-events-none">
        <TopLeftShine />
      </div>
      <div className="absolute top-0 right-0 flex justify-end w-screen overflow-hidden pointer-events-none">
        <TopRightShine />
      </div>
      <nav className="container flex items-center justify-between h-16">
        <Link href="/">
          <Logo className="min-w-sm" />
        </Link>
        <Link
          className="flex items-center h-8 gap-2 px-4 text-sm text-white duration-500 border rounded-lg bg-white/5 hover:bg-white hover:text-black border-white/10"
          href="/docs"
          target="_blank"
        >
          <FileText className="w-4 h-4" strokeWidth={1} />
          Documentation
        </Link>
      </nav>
      <div className="flex min-h-screen pt-16 -mt-16">
        <div className="container relative flex flex-col items-center justify-center gap-8 lg:w-2/5">
          <div className="w-full max-w-sm">{children}</div>
          <div className="flex items-center justify-center ">
            <p className="p-4 text-xs text-center text-white/50 text-balance">
              By continuing, you agree to Unkey's{" "}
              <Link className="underline" href="/policies/terms">
                Terms of Service
              </Link>{" "}
              and{" "}
              <Link className="underline" href="/policies/privacy">
                Privacy Policy
              </Link>
              , and to receive periodic emails with updates.
            </p>
          </div>
        </div>
        <Separator orientation="vertical" className="hidden -mt-16 bg-white/20 lg:block" />
        <div className="items-center justify-center hidden w-3/5 h-[calc(100vh-4rem)] lg:flex">
          <FadeIn>
            <div className="relative max-w-lg pl-12">
              <div className="absolute top-0 left-0 w-px bg-white/30 h-1/2" />
              <div className="absolute bottom-0 left-0 w-px bg-white h-1/2" />

              <p className="text-3xl leading-10 text-transparent bg-clip-text bg-gradient-to-r from-white via-white to-white/30 text-pretty">
                {quote.text}
              </p>

              <div className="flex items-center mt-8">
                <Avatar className="w-8 h-8">
                  <AvatarImage src={quote.author.image} />
                  <AvatarFallback>{quote.author.name}</AvatarFallback>
                </Avatar>
                <Link
                  href={quote.author.href}
                  target="_blank"
                  className="ml-4 text-sm text-white hover:underline"
                >
                  {quote.author.name}
                </Link>{" "}
                <span className="ml-2 text-sm text-white/50">{quote.author.title}</span>
              </div>
            </div>
          </FadeIn>
        </div>
      </div>
    </div>
  );
}

const TopRightShine: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    width="529"
    height="499"
    viewBox="0 0 529 499"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_8026_110)">
      <ellipse
        cx="524.924"
        cy="22.6709"
        rx="22.3794"
        ry="381.284"
        fill="url(#paint2_linear_8026_110)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_8026_110)">
      <ellipse
        cx="524.924"
        cy="-177.946"
        rx="22.3794"
        ry="180.667"
        fill="url(#paint3_linear_8026_110)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_8026_110)">
      <ellipse
        cx="409.694"
        cy="40.8839"
        rx="22.25"
        ry="381.5"
        transform="rotate(15 409.694 40.8839)"
        fill="url(#paint4_linear_8026_110)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_8026_110)">
      <ellipse
        cx="467.558"
        cy="-176.031"
        rx="321.5"
        ry="187.5"
        transform="rotate(15 467.558 -176.031)"
        fill="url(#paint5_linear_8026_110)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }}>
      <ellipse
        cx="491.369"
        cy="-264.896"
        rx="160.5"
        ry="95.5"
        transform="rotate(15 491.369 -264.896)"
        fill="url(#paint6_linear_8026_110)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }}>
      <ellipse
        cx="489.493"
        cy="-257.893"
        rx="135"
        ry="80.25"
        transform="rotate(15 489.493 -257.893)"
        fill="url(#paint7_linear_8026_110)"
        fill-opacity="0.5"
      />
    </g>
    <defs>
      <filter
        id="filter0_f_8026_110"
        x="97.2672"
        y="-423.485"
        width="477.307"
        height="686.892"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_110" />
      </filter>
      <filter
        id="filter1_f_8026_110"
        x="274.448"
        y="-445.321"
        width="338.241"
        height="744.685"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_110" />
      </filter>
      <filter
        id="filter2_f_8026_110"
        x="413.545"
        y="-447.613"
        width="222.759"
        height="940.567"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_110" />
      </filter>
      <filter
        id="filter3_f_8026_110"
        x="413.545"
        y="-447.613"
        width="222.759"
        height="539.335"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_110" />
      </filter>
      <filter
        id="filter4_f_8026_110"
        x="219.622"
        y="-416.663"
        width="380.145"
        height="915.093"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_110" />
      </filter>
      <filter
        id="filter5_f_8026_110"
        x="3.19824"
        y="-525.391"
        width="928.719"
        height="698.72"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_8026_110" />
      </filter>
      <linearGradient
        id="paint0_linear_8026_110"
        x1="335.921"
        y1="-373.384"
        x2="335.921"
        y2="213.307"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint1_linear_8026_110"
        x1="443.569"
        y1="-366.229"
        x2="443.569"
        y2="220.271"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint2_linear_8026_110"
        x1="524.924"
        y1="-358.613"
        x2="524.924"
        y2="403.954"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint3_linear_8026_110"
        x1="524.924"
        y1="-358.613"
        x2="524.924"
        y2="2.72144"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint4_linear_8026_110"
        x1="409.694"
        y1="-340.616"
        x2="409.694"
        y2="422.384"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint5_linear_8026_110"
        x1="467.558"
        y1="-363.531"
        x2="467.558"
        y2="11.4689"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint6_linear_8026_110"
        x1="491.369"
        y1="-360.396"
        x2="491.369"
        y2="-169.396"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint7_linear_8026_110"
        x1="489.493"
        y1="-338.143"
        x2="489.493"
        y2="-177.643"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
    </defs>
  </svg>
);

const TopLeftShine: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    width="659"
    height="537"
    viewBox="0 0 659 537"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g style={{ mixBlendMode: "color-dodge" }} filter="url(#filter1_f_8026_43)">
      <ellipse
        cx="262.378"
        cy="-50.076"
        rx="26.5"
        ry="293.25"
        transform="rotate(5 262.378 -50.076)"
        fill="url(#paint1_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter2_f_8026_43)">
      <ellipse
        cx="359.106"
        cy="29.9931"
        rx="22.3794"
        ry="381.284"
        transform="rotate(-10 359.106 29.9931)"
        fill="url(#paint2_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter3_f_8026_43)">
      <ellipse
        cx="324.269"
        cy="-167.576"
        rx="22.3794"
        ry="180.667"
        transform="rotate(-10 324.269 -167.576)"
        fill="url(#paint3_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter4_f_8026_43)">
      <ellipse
        cx="168.788"
        cy="67.9388"
        rx="22.25"
        ry="381.5"
        transform="rotate(5 168.788 67.9388)"
        fill="url(#paint4_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }} filter="url(#filter5_f_8026_43)">
      <ellipse
        cx="188.107"
        cy="-155.729"
        rx="321.5"
        ry="187.5"
        transform="rotate(5 188.107 -155.729)"
        fill="url(#paint5_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }}>
      <ellipse
        cx="196.124"
        cy="-247.379"
        rx="160.5"
        ry="95.5"
        transform="rotate(5 196.124 -247.379)"
        fill="url(#paint6_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <g style={{ mixBlendMode: "lighten" }}>
      <ellipse
        cx="195.494"
        cy="-240.156"
        rx="135"
        ry="80.25"
        transform="rotate(5 195.494 -240.156)"
        fill="url(#paint7_linear_8026_43)"
        fill-opacity="0.5"
      />
    </g>
    <defs>
      <filter
        id="filter0_f_8026_43"
        x="-39.0848"
        y="-403.13"
        width="388.448"
        height="729.588"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_43" />
      </filter>
      <filter
        id="filter1_f_8026_43"
        x="136.633"
        y="-431.219"
        width="251.489"
        height="762.287"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_43" />
      </filter>
      <filter
        id="filter2_f_8026_43"
        x="200.306"
        y="-434.518"
        width="317.6"
        height="929.023"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_43" />
      </filter>
      <filter
        id="filter3_f_8026_43"
        x="196.926"
        y="-434.542"
        width="254.687"
        height="533.932"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_43" />
      </filter>
      <filter
        id="filter4_f_8026_43"
        x="39.8235"
        y="-401.115"
        width="257.93"
        height="938.107"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="44.5" result="effect1_foregroundBlur_8026_43" />
      </filter>
      <filter
        id="filter5_f_8026_43"
        x="-282.594"
        y="-494.631"
        width="941.402"
        height="677.805"
        filterUnits="userSpaceOnUse"
        color-interpolation-filters="sRGB"
      >
        <feFlood flood-opacity="0" result="BackgroundImageFix" />
        <feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape" />
        <feGaussianBlur stdDeviation="75" result="effect1_foregroundBlur_8026_43" />
      </filter>
      <linearGradient
        id="paint0_linear_8026_43"
        x1="155.139"
        y1="-331.681"
        x2="155.139"
        y2="255.01"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint1_linear_8026_43"
        x1="262.378"
        y1="-343.326"
        x2="262.378"
        y2="243.174"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint2_linear_8026_43"
        x1="359.106"
        y1="-351.29"
        x2="359.106"
        y2="411.277"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint3_linear_8026_43"
        x1="324.269"
        y1="-348.243"
        x2="324.269"
        y2="13.0914"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint4_linear_8026_43"
        x1="168.788"
        y1="-313.561"
        x2="168.788"
        y2="449.439"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint5_linear_8026_43"
        x1="188.107"
        y1="-343.229"
        x2="188.107"
        y2="31.7714"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint6_linear_8026_43"
        x1="196.124"
        y1="-342.879"
        x2="196.124"
        y2="-151.879"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
      <linearGradient
        id="paint7_linear_8026_43"
        x1="195.494"
        y1="-320.406"
        x2="195.494"
        y2="-159.906"
        gradientUnits="userSpaceOnUse"
      >
        <stop stop-color="white" />
        <stop offset="1" stop-color="white" stop-opacity="0" />
      </linearGradient>
    </defs>
  </svg>
);

const Logo: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    xmlns="http://www.w3.org/2000/svg"
    width="93"
    height="40"
    viewBox="0 0 93 40"
  >
    <path
      d="M10.8 30.3C4.8 30.3 1.38 27.12 1.38 21.66V9.9H4.59V21.45C4.59 25.5 6.39 27.18 10.8 27.18C15.21 27.18 17.01 25.5 17.01 21.45V9.9H20.25V21.66C20.25 27.12 16.83 30.3 10.8 30.3ZM26.3611 30H23.1211V15.09H26.0911V19.71H26.3011C26.7511 17.19 28.7311 14.79 32.5111 14.79C36.6511 14.79 38.6911 17.58 38.6911 21.03V30H35.4511V21.9C35.4511 19.11 34.1911 17.7 31.1011 17.7C27.8311 17.7 26.3611 19.38 26.3611 22.62V30ZM44.8181 30H41.5781V9.9H44.8181V21H49.0781L53.5481 15.09H57.3281L51.7181 22.26L57.2981 30H53.4881L49.0781 23.91H44.8181V30ZM66.4219 30.3C61.5319 30.3 58.3219 27.54 58.3219 22.56C58.3219 17.91 61.5019 14.79 66.3619 14.79C70.9819 14.79 74.1319 17.34 74.1319 21.87C74.1319 22.41 74.1019 22.83 74.0119 23.28H61.3519C61.4719 26.16 62.8819 27.69 66.3319 27.69C69.4519 27.69 70.7419 26.67 70.7419 24.9V24.66H73.9819V24.93C73.9819 28.11 70.8619 30.3 66.4219 30.3ZM66.3019 17.34C63.0019 17.34 61.5619 18.81 61.3819 21.48H71.0719V21.42C71.0719 18.66 69.4819 17.34 66.3019 17.34ZM78.9586 35.1H76.8286V32.16H79.7386C81.0586 32.16 81.5986 31.8 82.0486 30.78L82.4086 30L75.0586 15.09H78.6886L82.4986 23.01L83.9686 26.58H84.2086L85.6186 22.98L89.1286 15.09H92.6986L84.9286 31.62C83.6986 34.29 82.0186 35.1 78.9586 35.1Z"
      fill="url(#paint0_radial_301_76)"
    />
    <defs>
      <radialGradient
        id="paint0_radial_301_76"
        cx="0"
        cy="0"
        r="1"
        gradientUnits="userSpaceOnUse"
        gradientTransform="rotate(23.2729) scale(101.237 101.088)"
      >
        <stop offset="0.26875" stopColor="white" />
        <stop offset="0.904454" stopColor="white" stop-opacity="0.5" />
      </radialGradient>
    </defs>
  </svg>
);
