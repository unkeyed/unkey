import type { Metadata } from "next";
import { ShareReveal } from "./share-reveal";

// noindex so the link can't be discovered; no-referrer so the fragment isn't
// leaked onward.
export const metadata: Metadata = {
  title: "Shared key",
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

export default function SharePage() {
  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-4 py-10">
      <svg
        aria-hidden="true"
        viewBox="0 0 512 512"
        fill="currentColor"
        className="pointer-events-none absolute left-1/2 top-1/2 size-[75vmin] -translate-x-1/2 -translate-y-1/2 text-grayA-2"
      >
        <title>Unkey</title>
        <path d="M170.8 115V340.6H341.2L284.4 397H170.8C139.418 397 114 371.761 114 340.6V115H170.8Z" />
        <path d="M398 284.2L341.2 340.6V115H398V284.2Z" />
      </svg>
      <main className="relative flex items-center justify-center">
        <ShareReveal />
      </main>
    </div>
  );
}
