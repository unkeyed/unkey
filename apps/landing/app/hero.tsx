import { BookOpen, ChevronRight, LogIn } from "lucide-react";
import Link from "next/link";
import { PrimaryButton, SecondaryButton } from "../components/button";

export const Hero: React.FC = () => {
  return (
    <div className="flex flex-col lg:flex-row justify-between items-center lg:items-start mt-[200px] max-w-[1440px]">
      <div className="relative text-center lg:text-left flex flex-col items-center lg:items-start">
        <div className="absolute top-[-50px] hero-hiring-gradient text-white text-sm flex space-x-2 py-1.5 px-2 items-center">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
          >
            <g clip-path="url(#clip0_840_5284)">
              <path
                d="M13 0.75C13 1.89705 11.8971 3 10.75 3C11.8971 3 13 4.10295 13 5.25C13 4.10295 14.1029 3 15.25 3C14.1029 3 13 1.89705 13 0.75Z"
                stroke="white"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
              <path
                d="M13 10.75C13 11.8971 11.8971 13 10.75 13C11.8971 13 13 14.1029 13 15.25C13 14.1029 14.1029 13 15.25 13C14.1029 13 13 11.8971 13 10.75Z"
                stroke="white"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
              <path
                d="M5 3.75C5 5.91666 2.91666 8 0.75 8C2.91666 8 5 10.0833 5 12.25C5 10.0833 7.0833 8 9.25 8C7.0833 8 5 5.91666 5 3.75Z"
                stroke="white"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </g>
            <defs>
              <clipPath id="clip0_840_5284">
                <rect width="16" height="16" fill="white" />
              </clipPath>
            </defs>
          </svg>
          <Link href="/careers">We are hiring!</Link>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
          >
            <path d="M12 8.5L8 4.5M12 8.5L8 12.5M12 8.5H4" stroke="white" />
          </svg>
        </div>

        <h1 className="bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white max-w-[546px] via-30% to-white/30 font-medium text-[4rem] leading-[5rem]  ">
          Build your API, not Auth
        </h1>

        <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg leading-[28px]">
          Unkey is an open source API authentication and authorization platform for scaling user
          facing APIs. Create, verify and manage low latency API keys in seconds.
        </p>

        <div className="flex items-center gap-6 mt-12">
          <Link href="/app" className="group">
            <PrimaryButton IconLeft={LogIn} label="Get Started" className="h-10" />
          </Link>

          <Link href="/docs">
            <SecondaryButton IconLeft={BookOpen} label="Documentation" IconRight={ChevronRight} />
          </Link>
        </div>
      </div>

      <div className="rounded-[38px] bg-white/5 border border-gray-800 z-10 mt-14 lg:mt-0 ">
        <div className="m-[10px] rounded-[28px] border border-gray-800">
          <img src="/images/hero.png" alt="Youtube" />
        </div>
      </div>
    </div>
  );
};
