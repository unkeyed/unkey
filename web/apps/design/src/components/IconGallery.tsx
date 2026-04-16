import * as Icons from "@unkey/icons";
import { useState } from "react";

export function IconGallery() {
  const entries = (Object.entries(Icons) as [string, unknown][])
    .filter(
      ([name, value]) =>
        typeof value === "function" && /^[A-Z]/.test(name) && name !== "Icon",
    )
    .sort(([a], [b]) => a.localeCompare(b));

  return (
    <div className="mt-10 mb-16 grid grid-cols-3 sm:grid-cols-4 md:grid-cols-6 gap-4">
      {entries.map(([name, Icon]) => (
        <IconCell
          key={name}
          name={name}
          Icon={Icon as React.ComponentType<Record<string, never>>}
        />
      ))}
    </div>
  );
}

function IconCell({
  name,
  Icon,
}: {
  name: string;
  Icon: React.ComponentType<Record<string, never>>;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(`<${name} />`);
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    } catch {
      /* ignored */
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      aria-label={name}
      className="group relative flex items-center justify-center aspect-square rounded-md border border-grayA-3 bg-background hover:border-grayA-6 transition-colors text-gray-12 cursor-pointer"
    >
      <Icon />
      <span
        className={`absolute left-1/2 -translate-x-1/2 bottom-2 px-2 py-0.5 rounded-[4px] bg-gray-12 text-gray-1 font-mono text-[10px] whitespace-nowrap transition-opacity duration-150 pointer-events-none ${
          copied ? "opacity-0" : "opacity-0 group-hover:opacity-100"
        }`}
      >
        {name}
      </span>
      <span
        className={`absolute inset-0 flex items-center justify-center rounded-md bg-gray-12 text-gray-1 text-[10px] font-medium uppercase tracking-[0.24em] transition-opacity pointer-events-none ${
          copied ? "opacity-100" : "opacity-0"
        }`}
      >
        Copied
      </span>
    </button>
  );
}
