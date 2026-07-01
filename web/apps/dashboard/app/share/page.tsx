import { FullScreenContent, FullScreenLayout, Logo } from "@unkey/ui";
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
    <FullScreenLayout className="px-4">
      <FullScreenContent>
        <main className="w-full max-w-sm">
          <ShareReveal />
        </main>
      </FullScreenContent>
      <div className="flex items-center gap-1.5 pb-6 text-gray-9">
        <span className="text-[13px] leading-5 opacity-50">Powered by</span>
        <a
          href="https://www.unkey.com"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Unkey"
          className="opacity-50 transition-opacity hover:opacity-100"
        >
          <Logo className="h-4 w-auto" />
        </a>
      </div>
    </FullScreenLayout>
  );
}
