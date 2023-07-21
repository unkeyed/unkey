import Link from "next/link";

import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";

export default function NotFound() {
  return (
    <Container className="flex items-center h-full pt-24 sm:pt-32 lg:pt-40">
      <FadeIn className="flex flex-col items-center">
        <p className="text-4xl font-semibold font-display text-neutral-950 sm:text-5xl">404</p>
        <h1 className="mt-4 text-2xl font-semibold font-display text-neutral-950">
          Page not found
        </h1>
        <p className="mt-2 text-sm text-neutral-600">
          Sorry, we couldn’t find the page you’re looking for.
        </p>
        <Link
          href="/"
          className="mt-4 text-sm font-semibold transition text-neutral-950 hover:text-neutral-700"
        >
          Go to the home page
        </Link>
      </FadeIn>
    </Container>
  );
}
