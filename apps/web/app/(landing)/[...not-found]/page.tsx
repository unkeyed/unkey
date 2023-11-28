import { Button } from "@/components/landing/button";
export const runtime = "edge";
export default function NotFound() {
  return (
    <section className="flex flex-col items-center justify-center h-screen max-w-7xl width-full lg:mx-auto">
      <h1 className="-z-10 text-[200px] lg:text-[525px] text-gray-300 blur-md absolute">404</h1>
      <p className="mb-4 text-lg font-bold tracking-tight md:text-3xl lg:text-5xl dark:text-white">
        Looking for something?
      </p>
      <p className="px-10 mb-4 font-light md:text-2xl lg:text-3xl">
        We couldn't find the page that you're looking for!{" "}
      </p>
      <Button href="/" className="mt-8 text-sm font-semibold rounded-full ">
        Go Home
      </Button>
    </section>
  );
}
