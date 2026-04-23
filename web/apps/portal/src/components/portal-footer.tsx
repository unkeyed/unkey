import { UnkeyLogo } from "~/components/ui/unkey-logo";

export function PortalFooter() {
  return (
    <footer className="w-full bg-gray-3/50">
      <div className="flex items-center justify-between px-8 py-5 text-xs text-gray-11">
        <span className="inline-flex items-center gap-1.5">
          Powered by
          <a
            href="https://unkey.com"
            className="inline-flex text-gray-12 transition-colors hover:text-gray-10"
          >
            <UnkeyLogo className="h-3.5 w-auto" />
          </a>
        </span>
        <nav className="flex items-center gap-5">
          <a href="https://unkey.com/portal" className="transition-colors hover:text-gray-12">
            Learn about Unkey Portal
          </a>
          <a href="https://unkey.com/terms" className="transition-colors hover:text-gray-12">
            Terms
          </a>
          <a href="https://unkey.com/privacy" className="transition-colors hover:text-gray-12">
            Privacy
          </a>
        </nav>
      </div>
    </footer>
  );
}
