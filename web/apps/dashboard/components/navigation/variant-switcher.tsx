"use client";

import { type NavbarVariant, useNavbarVariant } from "@/hooks/use-navbar-variant";
import {
  type V2bDeploymentsVariant,
  useV2bDeploymentsVariant,
} from "@/hooks/use-v2b-deployments-variant";
import { useRouter } from "next/navigation";

const VARIANTS: ReadonlyArray<{ id: NavbarVariant; label: string }> = [
  { id: "current", label: "current" },
  { id: "v1a", label: "v1a · inline" },
  { id: "v1b", label: "v1b · linked" },
  { id: "v2", label: "v2 · static" },
  { id: "v3", label: "v3 · header" },
];

const V2B_SUB_VARIANTS: ReadonlyArray<{ id: V2bDeploymentsVariant; label: string }> = [
  { id: "a", label: "v2b · a · rail+crumb" },
  { id: "b", label: "v2b · b · railless" },
  { id: "c", label: "v2b · c · merged-crumb" },
  { id: "d", label: "v2b · d · eyebrow" },
  { id: "e", label: "v2b · e · drawer" },
  { id: "f", label: "v2b · f · split" },
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
      {VARIANTS.slice(0, 4).map((v) => (
        <VariantPill key={v.id} v={v} active={v.id === variant} onChoose={choose} />
      ))}
      <V2bSelect />
      {VARIANTS.slice(4).map((v) => (
        <VariantPill key={v.id} v={v} active={v.id === variant} onChoose={choose} />
      ))}
    </fieldset>
  );
}

function VariantPill({
  v,
  active,
  onChoose,
}: {
  v: { id: NavbarVariant; label: string };
  active: boolean;
  onChoose: (id: NavbarVariant) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onChoose(v.id)}
      className={
        active
          ? "cursor-pointer rounded-full bg-white px-2.5 py-1 text-[12px] font-medium text-gray-12"
          : "cursor-pointer rounded-full px-2.5 py-1 text-[12px] font-medium text-gray-5 hover:text-white"
      }
    >
      {v.label}
    </button>
  );
}

/**
 * v2b's slot in the switcher: a native <select> so picking a sub-variant
 * also activates v2b. Rendered active-styled when v2b is the top-level
 * variant; inactive-styled otherwise (matches the pill states).
 */
function V2bSelect() {
  const router = useRouter();
  const { variant, setVariant } = useNavbarVariant();
  const { variant: subVariant, setVariant: setSubVariant } = useV2bDeploymentsVariant();
  const active = variant === "v2b";

  const onChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const next = e.target.value as V2bDeploymentsVariant;
    setSubVariant(next);
    if (variant !== "v2b") {
      setVariant("v2b");
    }
    router.refresh();
  };

  return (
    <select
      value={subVariant}
      onChange={onChange}
      className={
        active
          ? "cursor-pointer appearance-none rounded-full bg-white px-2.5 py-1 text-[12px] font-medium text-gray-12 outline-hidden"
          : "cursor-pointer appearance-none rounded-full bg-transparent px-2.5 py-1 text-[12px] font-medium text-gray-5 outline-hidden hover:text-white"
      }
    >
      {V2B_SUB_VARIANTS.map((sv) => (
        <option key={sv.id} value={sv.id} className="bg-gray-12 text-white">
          {sv.label}
        </option>
      ))}
    </select>
  );
}
