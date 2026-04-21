"use client";

import { type NavbarVariant, useNavbarVariant } from "@/hooks/use-navbar-variant";
import { useRouter } from "next/navigation";

const VARIANTS: ReadonlyArray<{ id: NavbarVariant; label: string }> = [
  { id: "current", label: "current" },
  { id: "v1a", label: "v1a · inline" },
  { id: "v1b", label: "v1b · linked" },
  { id: "v2", label: "v2 · static" },
  { id: "v3", label: "v3 · header" },
];

/**
 * Dev-only prototype switcher. Renders nothing in production.
 * Floating pill, bottom-right. Triggers router.refresh() on change
 * so the chrome fully remounts and sidebar data queries re-fire.
 */
export function VariantSwitcher() {
  if (process.env.NODE_ENV === "production") {
    return null;
  }
  return <VariantSwitcherInner />;
}

function VariantSwitcherInner() {
  const router = useRouter();
  const { variant, setVariant } = useNavbarVariant();

  const choose = (next: NavbarVariant) => {
    if (next === variant) {
      return;
    }
    setVariant(next);
    router.refresh();
  };

  return (
    <fieldset className="fixed bottom-3 right-3 z-50 m-0 flex items-center gap-0.5 rounded-full border-0 bg-gray-12 p-[3px] text-white shadow-2xl">
      <legend className="sr-only">Navbar variant</legend>
      <span className="px-2.5 text-[11px] uppercase tracking-wide text-gray-8">Variant</span>
      {VARIANTS.map((v) => {
        const active = v.id === variant;
        return (
          <button
            key={v.id}
            type="button"
            onClick={() => choose(v.id)}
            className={
              active
                ? "cursor-pointer rounded-full bg-white px-2.5 py-1 text-[12px] font-medium text-gray-12"
                : "cursor-pointer rounded-full px-2.5 py-1 text-[12px] font-medium text-gray-5 hover:text-white"
            }
          >
            {v.label}
          </button>
        );
      })}
    </fieldset>
  );
}
