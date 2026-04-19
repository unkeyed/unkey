import type { Plugin } from "vite";

/**
 * Vite transform that auto-injects a `code` prop onto `<Preview>` JSX
 * elements, derived from the element's children source text. Works on both
 * `.tsx` and `.mdx` files.
 *
 * Authoring:
 *   <Preview>
 *     <Button variant="primary">Primary</Button>
 *   </Preview>
 *
 * After transform:
 *   <Preview code={`<Button variant="primary">Primary</Button>`}>
 *     <Button variant="primary">Primary</Button>
 *   </Preview>
 *
 * An explicit `code={...}` prop is preserved as-is. Astro `client:*`
 * directives are stripped from the extracted code so snippets match what
 * a consumer would write.
 */
export function previewCodePlugin(): Plugin {
  return {
    name: "preview-code-injector",
    enforce: "pre",
    transform(source, id) {
      if (!/\.(tsx|mdx)$/.test(id)) return null;
      if (!source.includes("<Preview")) return null;
      const transformed = inject(source);
      if (transformed === source) return null;
      return { code: transformed, map: null };
    },
  };
}

const OPEN_TAG = "<Preview";

function inject(source: string): string {
  const parts: string[] = [];
  let cursor = 0;

  while (cursor < source.length) {
    const tagStart = source.indexOf(OPEN_TAG, cursor);
    if (tagStart === -1) {
      parts.push(source.slice(cursor));
      break;
    }
    const afterName = source[tagStart + OPEN_TAG.length];
    // Ignore `<PreviewSomething` — must be whitespace, `>`, or `/`.
    if (/[A-Za-z0-9_]/.test(afterName)) {
      parts.push(source.slice(cursor, tagStart + 1));
      cursor = tagStart + 1;
      continue;
    }

    const openingEnd = findOpeningTagEnd(source, tagStart);
    if (openingEnd === -1) {
      parts.push(source.slice(cursor));
      break;
    }

    const openingAttrs = source.slice(tagStart + OPEN_TAG.length, openingEnd);
    const selfClosing = source[openingEnd - 1] === "/";

    // Skip if already has a `code=` prop.
    if (/\bcode\s*=/.test(openingAttrs)) {
      parts.push(source.slice(cursor, openingEnd + 1));
      cursor = openingEnd + 1;
      continue;
    }
    if (selfClosing) {
      parts.push(source.slice(cursor, openingEnd + 1));
      cursor = openingEnd + 1;
      continue;
    }

    const closingStart = source.indexOf("</Preview>", openingEnd + 1);
    if (closingStart === -1) {
      parts.push(source.slice(cursor));
      break;
    }
    const closingEnd = closingStart + "</Preview>".length;

    const childrenRaw = source.slice(openingEnd + 1, closingStart);
    const cleaned = stripClientDirectives(childrenRaw);
    const dedented = dedent(cleaned).trim();
    // Emit a single-line double-quoted string with \n escape sequences.
    // A multi-line template literal would survive JS parsing, but something
    // downstream (MDX/JSX whitespace normalization) re-dedents multi-line
    // string literals in JSX attribute expressions. Escape sequences in a
    // single-line string keep the payload out of reach of that pass.
    const escaped = JSON.stringify(dedented);

    // Preserve everything up to the injection point, then the original
    // opening tag with the injected `code` prop, then the original children
    // and closing tag.
    parts.push(source.slice(cursor, openingEnd));
    parts.push(` code={${escaped}}`);
    parts.push(source.slice(openingEnd, closingEnd));
    cursor = closingEnd;
  }

  return parts.join("");
}

/**
 * Scan forward from the `<` position to find the matching `>` of the opening
 * tag, tracking JSX expression braces and string literals so `>` inside
 * `{...}` or `"..."` doesn't confuse us.
 */
function findOpeningTagEnd(source: string, start: number): number {
  let i = start + 1;
  let inString: '"' | "'" | "`" | null = null;
  let braceDepth = 0;
  while (i < source.length) {
    const ch = source[i];
    if (inString) {
      if (ch === "\\") {
        i += 2;
        continue;
      }
      if (ch === inString) inString = null;
    } else if (braceDepth > 0) {
      if (ch === "{") braceDepth++;
      else if (ch === "}") braceDepth--;
      else if (ch === '"' || ch === "'" || ch === "`") inString = ch;
    } else {
      if (ch === "{") braceDepth++;
      else if (ch === '"' || ch === "'") inString = ch;
      else if (ch === ">") return i;
    }
    i++;
  }
  return -1;
}

/**
 * Remove Astro `client:*` directives from JSX source text. These are
 * implementation noise; users copying the snippet shouldn't see them.
 */
function stripClientDirectives(input: string): string {
  return input.replace(
    /\s+client:[a-zA-Z-]+(\s*=\s*(\{[^}]*\}|"[^"]*"|'[^']*'))?/g,
    "",
  );
}

function dedent(input: string): string {
  const lines = input.split("\n");
  const widths = lines
    .filter((l) => l.trim().length > 0)
    .map((l) => l.match(/^[ \t]*/)?.[0].length ?? 0);
  if (widths.length === 0) return input;
  const min = Math.min(...widths);
  return lines.map((l) => l.slice(min)).join("\n");
}
