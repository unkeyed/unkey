import { render, screen } from "@testing-library/react";
import * as React from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { SecondaryNav } from "./secondary-nav";

describe("SecondaryNav", () => {
  beforeAll(() => {
    vi.stubGlobal("React", React);
  });
  afterAll(() => {
    vi.unstubAllGlobals();
  });

  it("provides a full-height rail with sticky navigation", () => {
    render(
      React.createElement(SecondaryNav, {
        "aria-label": "Settings",
        className: "md:w-56",
      }),
    );

    const nav = screen.getByRole("navigation", { name: "Settings" });
    const rail = nav.parentElement;

    expect(nav.className).toContain("md:sticky");
    expect(nav.className).toContain("md:top-0");
    expect(nav.className).toContain("md:w-56");
    expect(nav.className).not.toContain("md:w-60");
    expect(rail?.className).toContain("md:self-stretch");
    expect(rail?.className).toContain("md:border-r");
    expect(rail?.className).toContain("md:border-grayA-4");
    expect(rail?.className).not.toContain("md:w-56");
  });
});
