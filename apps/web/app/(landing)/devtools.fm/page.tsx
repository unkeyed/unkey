import { CopyButton } from "@/components/dashboard/copy-button";
import Link from "next/link";
export default function Example() {
  return (
    <div className="">
      <div className="px-6 py-24 mx-auto max-w-7xl sm:py-32 lg:flex lg:items-center lg:gap-x-10 lg:px-8 lg:py-40">
        <div className="max-w-2xl mx-auto lg:mx-0 lg:flex-auto">
          <div className="flex">
            <div className="relative flex items-center px-4 py-1 text-xs md:text-sm leading-6 rounded-full text-primary gap-x-4 ring-1 ring-ring hover:ring-gray-900/20">
              <span className="font-semibold text-indigo-600 ">
                Get 3 months of Unkey Pro free with the code:
              </span>
              <span className="w-px h-4 bg-gray-900/10" aria-hidden="true" />
              <div className="flex items-center font-mono font-semibold gap-x-1">
                DEVTOOLSFM <CopyButton value="DEVTOOLSFM" />
              </div>
            </div>
          </div>
          <h1 className="max-w-lg mt-10 text-4xl font-bold tracking-tight text-gray-900 sm:text-6xl">
            API authentication made easy
          </h1>
          <p className="mt-6 text-lg leading-8 text-gray-600">
            James sat down with Andrew Lisowski and Justin Bennett to talk about Unkey. James
            discusses his experiences at Clerk, his career transition to co-founding Unkey, and his
            thoughts on the future direction of backend services. He shares the challenging reality
            of balancing two programming jobs at once and underscores the necessity for strong
            security practices in every tech startup.
          </p>
          <div className="flex items-center mt-10 gap-x-6">
            <Link
              href="/auth/sign-up"
              className="rounded-md border border-primary bg-primary px-3.5 py-1.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-secondary hover:text-secondary-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600"
            >
              Sign up
            </Link>
            <Link
              href="https://www.devtools.fm/episode/73"
              className="text-sm font-semibold leading-6 text-primary"
            >
              Listen to the episode <span aria-hidden="true">→</span>
            </Link>
          </div>
        </div>
        <div className="mt-16 sm:mt-24 lg:mt-0 lg:flex-shrink-0 lg:flex-grow">
          <img
            alt="devtools.fm logo"
            className="mx-auto w-[22.875rem] max-w-full drop-shadow-xl"
            src="https://www.devtools.fm/_next/image?url=%2Flogo.png&w=1920&q=75"
          />
        </div>
      </div>
    </div>
  );
}
