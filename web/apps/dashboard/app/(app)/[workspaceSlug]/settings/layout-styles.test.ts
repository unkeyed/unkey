import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("Settings widget styles", () => {
  it("loads WorkOS styles only from managed widget routes", () => {
    const layout = readFileSync(
      resolve(process.cwd(), "app/(app)/[workspaceSlug]/settings/layout.tsx"),
      "utf8",
    );
    const managedAccount = readFileSync(
      resolve(process.cwd(), "app/(app)/[workspaceSlug]/account/managed-account.tsx"),
      "utf8",
    );
    const managedTeam = readFileSync(
      resolve(process.cwd(), "app/(app)/[workspaceSlug]/settings/team/managed-team.tsx"),
      "utf8",
    );

    expect(layout).not.toContain("@unkey/workos-widgets/styles.css");
    expect(managedAccount).toContain('import "@unkey/workos-widgets/styles.css";');
    expect(managedTeam).toContain('import "@unkey/workos-widgets/styles.css";');
  });

  it("styles the complete WorkOS element and widget surface with Unkey tokens", () => {
    const styles = readFileSync(
      resolve(process.cwd(), "../../internal/workos-widgets/styles.css"),
      "utf8",
    );

    expect(styles).toContain('@import "@radix-ui/themes/tokens/base.css";');
    expect(styles).toContain('@import "@radix-ui/themes/layout.css";');
    expect(styles).toContain('@import "@radix-ui/themes/components.css";');
    expect(styles).toContain('@import "@radix-ui/themes/utilities.css";');
    expect(styles).not.toMatch(
      /^\s*@import\s+["']@radix-ui\/themes\/(?:styles|tokens)(?:\.css)?["'];/m,
    );
    expect(styles).not.toMatch(
      /\.woswidgets-menu-item,\s*\.woswidgets-select-item\s*\{[^}]*padding-inline/,
    );
    expect(styles).toMatch(
      /\.woswidgets-dialog-overlay,\s*\.rt-DialogOverlay:has\(\.woswidgets-dialog\)\s*\{[^}]*z-index:\s*50;/s,
    );
    expect(styles).toMatch(
      /\.woswidgets-dropdown,\s*\.woswidgets-select-dropdown\s*\{[^}]*z-index:\s*200;/s,
    );

    for (const selector of [
      ".woswidgets-avatar",
      ".woswidgets-badge",
      ".woswidgets-button",
      ".woswidgets-button--destructive",
      ".woswidgets-button--primary",
      ".woswidgets-button--secondary",
      ".woswidgets-dialog",
      ".woswidgets-dialog-overlay",
      ".woswidgets-dropdown",
      ".woswidgets-icon-button",
      ".woswidgets-label",
      ".woswidgets-menu-item",
      ".woswidgets-menu-item--destructive",
      ".woswidgets-root",
      ".woswidgets-select",
      ".woswidgets-select-dropdown",
      ".woswidgets-select-item",
      ".woswidgets-skeleton",
      ".woswidgets-text-field",
      ".rt-Card",
      ".rt-TableRoot",
    ]) {
      expect(styles).toContain(selector);
    }

    expect(styles).toContain('[data-woswidgets-element="primary-button"]');
    expect(styles).toContain('[data-woswidgets-element="secondary-button"]');
    expect(styles).toContain('[data-woswidgets-element="destructive-button"]');
    expect(styles).toContain('[data-woswidgets-element="destructive-menu-item"]');
    expect(styles).toContain("hsla(var(--grayA-");
    expect(styles).toContain("hsl(var(--error-");
    expect(styles).toContain(":focus-visible");
    expect(styles).toContain(":disabled");
  });
});
