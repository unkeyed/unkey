import { readFile } from "node:fs/promises";
import { codeToHtml } from "shiki";
import type { Plugin } from "vite";

/**
 * Two related build-time transforms for documenting components:
 *
 * 1. **Auto-injects a `code` prop onto `<Preview>` JSX elements** from the
 *    element's children source text, Shiki-highlighted. Used for inline
 *    snippets:
 *      <Preview>
 *        <Skeleton className="h-4 w-32" />
 *      </Preview>
 *    becomes a Preview that renders the Skeleton AND shows the JSX source
 *    underneath. Astro `client:*` directives are stripped from the
 *    extracted source. An explicit `code={...}` prop is preserved.
 *
 * 2. **`?highlight` import suffix** — loads any file as a Shiki-highlighted
 *    HTML string, for showing the whole component source under a Preview:
 *      import source from "./_card-skeleton.tsx?highlight"
 *      <Preview code={source}><SkeletonCard /></Preview>
 */
const HIGHLIGHT_SUFFIX = "?highlight";
const VIRTUAL_PREFIX = "\0highlight:";
const SHIKI_OPTS = {
  lang: "tsx",
  themes: { light: "github-light", dark: "github-dark" },
  defaultColor: false,
} as const;

export function previewCodePlugin(): Plugin {
  return {
    name: "preview-code-injector",
    enforce: "pre",
    async resolveId(id, importer) {
      if (!id.endsWith(HIGHLIGHT_SUFFIX)) {
        return null;
      }
      const cleanId = id.slice(0, -HIGHLIGHT_SUFFIX.length);
      const resolved = await this.resolve(cleanId, importer, { skipSelf: true });
      if (!resolved) {
        return null;
      }
      // \0 prefix marks this as a virtual module so other plugins
      // (Astro's JSX/TSX handlers) don't try to claim it.
      return VIRTUAL_PREFIX + resolved.id;
    },
    async load(id) {
      if (!id.startsWith(VIRTUAL_PREFIX)) {
        return null;
      }
      const filePath = id.slice(VIRTUAL_PREFIX.length);
      const source = await readFile(filePath, "utf-8");
      const html = await codeToHtml(source.trim(), SHIKI_OPTS);
      return `export default ${JSON.stringify(html)};`;
    },
    async transform(source, id) {
      if (!/\.(tsx|mdx)$/.test(id)) {
        return null;
      }
      if (!source.includes("<Preview")) {
        return null;
      }
      const transformed = await inject(source);
      if (transformed === source) {
        return null;
      }
      return { code: transformed, map: null };
    },
  };
}

const OPEN_TAG = "<Preview";

async function inject(source: string): Promise<string> {
  const parts: string[] = [];
  let cursor = 0;

  while (cursor < source.length) {
    const tagStart = source.indexOf(OPEN_TAG, cursor);
    if (tagStart === -1) {
      parts.push(source.slice(cursor));
      break;
    }
    const afterName = source[tagStart + OPEN_TAG.length];
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
    const highlighted = await highlight(dedented);
    const escaped = JSON.stringify(highlighted);

    parts.push(source.slice(cursor, openingEnd));
    parts.push(` code={${escaped}}`);
    parts.push(source.slice(openingEnd, closingEnd));
    cursor = closingEnd;
  }

  return parts.join("");
}

async function highlight(code: string): Promise<string> {
  return codeToHtml(code, SHIKI_OPTS);
}

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
      if (ch === inString) {
        inString = null;
      }
    } else if (braceDepth > 0) {
      if (ch === "{") {
        braceDepth++;
      } else if (ch === "}") {
        braceDepth--;
      } else if (ch === '"' || ch === "'" || ch === "`") {
        inString = ch;
      }
    } else {
      if (ch === "{") {
        braceDepth++;
      } else if (ch === '"' || ch === "'") {
        inString = ch;
      } else if (ch === ">") {
        return i;
      }
    }
    i++;
  }
  return -1;
}

function stripClientDirectives(input: string): string {
  return input.replace(/\s+client:[a-zA-Z-]+(\s*=\s*(\{[^}]*\}|"[^"]*"|'[^']*'))?/g, "");
}

function dedent(input: string): string {
  const lines = input.split("\n");
  const widths = lines
    .filter((l) => l.trim().length > 0)
    .map((l) => l.match(/^[ \t]*/)?.[0].length ?? 0);
  if (widths.length === 0) {
    return input;
  }
  const min = Math.min(...widths);
  return lines.map((l) => l.slice(min)).join("\n");
}
