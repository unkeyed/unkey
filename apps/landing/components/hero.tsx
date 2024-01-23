import { BookOpen, ChevronRight, LogIn } from "lucide-react";
import Link from "next/link";

export const Hero: React.FC = () => {
  return (
    <div className="flex min-h-[100vh] items-center justify-between">
      <div>
        <div>We're Hiring</div>

        <h1 className="bg-gradient-to-br text-transparent bg-gradient-stop  bg-clip-text from-white via-white via-30% to-white/30 text-hero font-medium">
          Build your API,
          <br />
          not Auth
        </h1>

        <p className="mt-8 bg-gradient-to-br text-transparent bg-gradient-stop bg-clip-text from-white via-white via-40% to-white/30 max-w-lg ">
          Unkey is an open source API authentication and authorization platform for scaling user
          facing APIs. Create, verify and manage low latency API keys in seconds.
        </p>

        <div className="flex items-center gap-6 mt-12">
          <Link
            href="/app"
            className="bg-white h-10 flex items-center border border-white px-4  rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
          >
            <LogIn className="w-4 h-4" /> Get Started <ChevronRight className="w-4 h-4" />
          </Link>

          <Link
            href="/docs"
            className="h-10 flex items-center px-4 gap-2 text-white/50 hover:text-white duration-500"
          >
            <BookOpen className="w-4 h-4" />
            Documentation
            <ChevronRight className="w-4 h-4" />
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
