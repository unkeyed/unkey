import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import * as React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HiddenValueCell } from "./hidden-value-cell";

beforeEach(() => vi.stubGlobal("React", React));
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("HiddenValueCell", () => {
  it("shows the separate prefix with an underscore before the start", () => {
    render(<HiddenValueCell prefix="unkey" start="abcd" end="9xY2" title="Key" selected={false} />);

    expect(screen.getByText("unkey_abcd••••9xY2")).toBeDefined();
  });

  it.each([undefined, ""])("omits the underscore when prefix is %s", (prefix) => {
    render(
      <HiddenValueCell prefix={prefix} start="abcd" end="9xY2" title="Key" selected={false} />,
    );

    expect(screen.getByText("abcd••••9xY2")).toBeDefined();
  });

  it.each([undefined, ""])("keeps the prefix-only display when end is %s", (end) => {
    render(<HiddenValueCell start="unkey_abcd" end={end} title="Key" selected={false} />);

    expect(screen.getByText("unkey_abcd••••")).toBeDefined();
  });

  it("leaves legacy starts that already contain a prefix intact", () => {
    render(<HiddenValueCell start="production_abcd" end="7zQ3" title="Key" selected={false} />);

    expect(screen.getByText("production_abcd••••7zQ3")).toBeDefined();
  });

  it("copies the complete displayed value, including the mask and suffix", () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    render(<HiddenValueCell prefix="unkey" start="abcd" end="9xY2" title="Key" selected={false} />);

    fireEvent.click(screen.getByText("unkey_abcd••••9xY2"));

    expect(writeText).toHaveBeenCalledWith("unkey_abcd••••9xY2");
  });
});
