import { Button } from "@/components/landing/button";
export default function NotFound() {
  return (
    <section className="width-full flex h-screen max-w-7xl flex-col items-center justify-center lg:mx-auto">
      <h1 className="absolute -z-10 text-[200px] text-gray-300 blur-md lg:text-[525px]">404</h1>
      <p className="mb-4 text-lg font-bold tracking-tight dark:text-white md:text-3xl lg:text-5xl">
        Looking for something?
      </p>
      <p className="mb-4 px-10 font-light md:text-2xl lg:text-3xl">
        We couldn't find the page that you're looking for!{" "}
      </p>
      <Button href="/" className="mt-8 rounded-full text-sm font-semibold ">
        Go Home
      </Button>
    </section>
  );
}
