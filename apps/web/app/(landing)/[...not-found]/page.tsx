import { Button } from "@/components/landing/button";

export default function NotFound() {
  return (
    <section className=" flex flex-col h-screen justify-center items-center max-w-7xl width-full lg:mx-auto">
      <h1 className="-z-10 text-[200px] lg:text-[525px] text-gray-300 blur-md absolute">404</h1>
      <p className="mb-4 text-lg md:text-3xl lg:text-5xl tracking-tight font-bold dark:text-white">
        Looking for something?{" "}
        <span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="32"
            height="32"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            className="inline lucide lucide-search-x"
          >
            <path d="m13.5 8.5-5 5" />
            <path d="m8.5 8.5 5 5" />
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
        </span>
      </p>
      <p className=" mb-4 px-10 md:text-2xl lg:text-3xl font-light">
        We couldn't find the page that you're looking for!{" "}
      </p>
      <Button href="/" className=" text-sm font-semibold rounded-full mt-8">
        Go Home
      </Button>
    </section>
  );
}
