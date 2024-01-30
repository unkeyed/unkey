import { BookOpen, ChevronRight, LogIn } from "lucide-react";
import Link from "next/link";
import { PrimaryButton, SecondaryButton } from "../components/button";

export const Hero: React.FC = () => {
  return (
    <div className="flex min-h-[100vh] items-center justify-between">
      <div>
        <div>We're Hiring</div>

        <h1 className="bg-gradient-to-br text-transparent bg-gradient-stop  bg-clip-text from-white via-white via-30% to-white/30 font-medium text-[4rem] leading-[4rem]  ">
          Build your API,
          <br />
          not Auth
        </h1>

        <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg ">
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

      <div className="rounded-[38px] bg-white/5 border border-gray-800 z-10">
        <div className="m-[10px] rounded-[28px] border border-gray-800">
          <img src="/images/hero.png" alt="Youtube" />
        </div>
      </div>
    </div>
  );
};
