"use client";

import { BrandLogo } from "@/lib/extensions/brand-logos";
import { CATEGORY_LABELS, EXTENSION_TYPE_LABELS, type Extension } from "@/lib/extensions/registry";
import { CircleCheck } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import Link from "next/link";
import { PreviewPill } from "./preview-pill";

type ExtensionCardProps = {
  extension: Extension;
  href: string;
  installed?: boolean;
};

export function ExtensionCard({ extension, href, installed }: ExtensionCardProps) {
  return (
    <Link
      href={href}
      className="relative flex h-full flex-col gap-4 rounded-2xl border border-grayA-4 bg-background p-5 transition-all duration-300 hover:border-grayA-7 hover:shadow-sm hover:shadow-grayA-8/10"
    >
      <div className="flex items-start gap-3">
        <ExtensionLogo extension={extension} />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-sm font-medium text-accent-12 leading-[18px]">
              {extension.name}
            </span>
            {extension.verified ? (
              <InfoTooltip content="Verified by Unkey" asChild>
                <CircleCheck className="size-3.5 text-accent-11 shrink-0" />
              </InfoTooltip>
            ) : null}
          </div>
          <span className="text-[11px] uppercase tracking-wide text-gray-10 font-mono">
            {EXTENSION_TYPE_LABELS[extension.type]}
          </span>
        </div>
        <div className="flex flex-col items-end gap-1 shrink-0">
          {installed ? (
            <span className="rounded-md bg-successA-3 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-successA-11">
              Installed
            </span>
          ) : null}
          <PreviewPill extension={extension} />
        </div>
      </div>

      <p className="text-[13px] leading-5 text-gray-11 line-clamp-2 min-h-[40px]">
        {extension.tagline}
      </p>

      <div className="mt-auto flex items-center justify-between text-[11px] text-gray-10 pt-1 border-t border-grayA-3 -mx-5 px-5 -mb-5 pb-3.5">
        <div className="flex items-center gap-1.5 truncate">
          {extension.categories.slice(0, 2).map((category) => (
            <span key={category} className="font-mono uppercase tracking-wide">
              {CATEGORY_LABELS[category]}
            </span>
          ))}
        </div>
        <span className="tabular-nums shrink-0">{formatInstalls(extension.installs)} installs</span>
      </div>
    </Link>
  );
}

function ExtensionLogo({ extension }: { extension: Extension }) {
  return (
    <div className="size-10 bg-white rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20 ring-1 ring-grayA-3 overflow-hidden">
      <BrandLogo
        slug={extension.slug}
        iconUrl={extension.iconUrl}
        name={extension.name}
        className="size-6"
      />
    </div>
  );
}

function formatInstalls(n: number): string {
  if (n >= 1000) {
    return `${(n / 1000).toFixed(1).replace(/\.0$/, "")}k`;
  }
  return String(n);
}
