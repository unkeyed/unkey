import type { ReactNode } from "react";

/** A titled settings card with no Save, for settings that act immediately. */
export function SettingsSection({ title, children }: { title?: ReactNode; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      {title ? <h2 className="text-base font-medium text-gray-12">{title}</h2> : null}
      {children}
    </section>
  );
}
