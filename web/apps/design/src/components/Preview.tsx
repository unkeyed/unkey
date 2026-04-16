import { useState, type ReactNode } from "react";

interface Props {
  children: ReactNode;
  /** Derived from the JSX children at build time by the preview-code Vite plugin. */
  code?: string;
}

export function Preview({ children, code = "" }: Props) {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  async function copy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <div className="my-12">
      {/* Preview stage — the dashboard's default rendering surface */}
      <div className="rounded-lg bg-background border border-grayA-3 px-8 py-20 min-h-[220px] flex flex-wrap items-center justify-center gap-4">
        {children}
      </div>

      {/* Toggle — just type, breathing room above and below */}
      <div className="flex items-center justify-center py-4">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="text-[10px] text-gray-9 hover:text-gray-12 font-medium uppercase tracking-[0.2em] transition-colors duration-200"
        >
          {open ? "Hide" : "Show code"}
        </button>
      </div>

      {open && (
        <div className="relative rounded-lg bg-grayA-2 overflow-hidden">
          <button
            type="button"
            onClick={copy}
            className="absolute top-3 right-3 text-[10px] text-gray-9 hover:text-gray-12 font-medium uppercase tracking-[0.2em] transition-colors duration-200"
          >
            {copied ? "Copied" : "Copy"}
          </button>
          <pre className="px-6 py-5 text-[12.5px] font-mono text-gray-12 leading-[1.7] overflow-x-auto">
            <code>{code}</code>
          </pre>
        </div>
      )}
    </div>
  );
}
