import { useState, type ReactNode } from "react";

interface Props {
  children: ReactNode;
  /**
   * The source snippet shown in the "Show code" drawer. Normally derived
   * automatically at build time by the `previewCodePlugin` Vite transform
   * from the element's JSX children — authors don't pass this themselves
   * except when they want to override the default extraction (e.g.
   * `ColorSwatch` injects a list of Tailwind class names instead of its own
   * compiled JSX, which would be noise).
   *
   * Defaults to `""` so the copy button is a no-op when no code is wired
   * up; preferable to crashing on an undefined write.
   */
  code?: string;
}

/**
 * Preview renders a component demo inside a framed "stage" that matches the
 * dashboard's real rendering surface (`bg-background` with a thin
 * `border-grayA-3`), plus an optional code drawer revealed by a Show/Hide
 * toggle.
 *
 * Why it exists: component documentation needs to show the rendered result
 * and the source side by side without drifting. The source is derived from
 * the JSX children at build time via [previewCodePlugin], so authors write
 * the demo once — whatever they type inside `<Preview>` is what lands both
 * on the stage and in the code drawer. See
 * `src/lib/vite-plugin-preview-code.ts` for the extraction rules.
 *
 * Authoring note: JSX-valued props (`prop={<X />}`) and compound React
 * trees that rely on Context cannot be inlined directly inside `<Preview>`
 * in an MDX file — MDX compiles those expressions with Astro's JSX runtime
 * rather than React's. For those cases, wrap the demo in a sibling
 * `_examples.tsx` React component and hydrate that as the island instead.
 */
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
