import { type ReactNode, useEffect, useRef, useState } from "react";
import { attachCopyButton } from "../lib/copy-button.ts";

interface Props {
  children: ReactNode;
  /**
   * Shiki-rendered HTML for the snippet shown below the preview. Injected at
   * build time by the `previewCodePlugin` Vite transform from the JSX
   * children.
   */
  code?: string;
  /**
   * Let the preview fill the frame instead of centering a component in
   * padding. Use for full-page layout examples that own their own width and
   * spacing.
   */
  bleed?: boolean;
}

const EXPAND_THRESHOLD = 8;

export function Preview({ children, code = "", bleed = false }: Props) {
  const [expanded, setExpanded] = useState(false);
  const codePanelRef = useRef<HTMLDivElement>(null);
  const needsExpand = (code.match(/class="line"/g)?.length ?? 0) > EXPAND_THRESHOLD;

  useEffect(() => {
    if (!codePanelRef.current) {
      return;
    }
    // Decode shiki's HTML entities by letting the DOM parse + textContent extract.
    // A regex tag-strip leaves `&#x3C;`, `&quot;`, etc. literal in the clipboard.
    const decoder = document.createElement("div");
    decoder.innerHTML = code;
    return attachCopyButton(codePanelRef.current, decoder.textContent ?? "", {
      topClass: "top-11",
    });
  }, [code]);

  return (
    <div className="not-prose isolate my-6">
      <div
        className={`relative z-10 flex min-h-72 overflow-hidden rounded-xl border bg-background shadow-xs ${
          bleed ? "flex-col" : "items-center justify-center p-10"
        }`}
      >
        {bleed ? (
          children
        ) : (
          <div className="flex w-full flex-wrap items-center justify-center gap-4">{children}</div>
        )}
      </div>
      <div
        ref={codePanelRef}
        className="-mt-8 relative z-0 overflow-hidden rounded-b-xl border bg-muted/40 pt-8"
      >
        <div className={`overflow-hidden ${!expanded && needsExpand ? "max-h-64" : ""}`}>
          <div
            className="preview-code"
            // biome-ignore lint/security/noDangerouslySetInnerHtml: shiki output is generated at build time
            dangerouslySetInnerHTML={{ __html: code }}
          />
        </div>
        {!expanded && needsExpand && (
          <>
            <div className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-background to-transparent" />
            <div className="-mt-10 relative flex items-center justify-center pb-3">
              <button
                type="button"
                onClick={() => setExpanded(true)}
                className="rounded-md border bg-background px-3 py-1 font-medium text-foreground text-xs transition-colors hover:bg-accent"
              >
                Expand
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
