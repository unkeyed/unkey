import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, ReactElement, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { Policy } from "../lib/policy";
import { TEMPLATES } from "../lib/templates";
import { TemplateGallery } from "./template-gallery";

vi.mock("@unkey/icons", () => ({
  Eye: () => null,
  Gauge: () => null,
  PenWriting3: () => null,
  Plus: () => null,
  ShieldCheck: () => null,
  XMark: () => null,
}));

vi.mock("@unkey/ui", () => ({
  Button: ({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  Item: ({ render }: { render: ReactElement }) => render,
  ItemContent: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  ItemDescription: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  ItemMedia: ({ children }: { children: ReactNode }) => <span>{children}</span>,
  ItemTitle: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}));

afterEach(cleanup);

const renderGallery = (onCancel?: () => void) => {
  const onPick = vi.fn<[Policy[]], void>();
  render(<TemplateGallery onPick={onPick} onCancel={onCancel} />);
  return onPick;
};

describe("TemplateGallery", () => {
  it("shows a tile for every template", () => {
    renderGallery();
    for (const template of TEMPLATES) {
      expect(screen.getByText(template.title)).toBeTruthy();
      expect(screen.getByText(template.description)).toBeTruthy();
    }
  });

  it("hands the picked template's policies to the caller", () => {
    const onPick = renderGallery();
    fireEvent.click(screen.getByText("All read permissions"));
    expect(onPick).toHaveBeenCalledTimes(1);
    expect(onPick.mock.calls[0][0].map((policy) => policy.scope)).toEqual(["workspace"]);
  });

  it("starts a custom policy with no grants", () => {
    const onPick = renderGallery();
    fireEvent.click(screen.getByText("Start new"));
    expect(onPick.mock.calls[0][0]).toEqual([
      { scope: "workspace", instances: ["__all__"], selection: {} },
    ]);
  });

  it("offers a close control only when the gallery can be dismissed", () => {
    renderGallery();
    expect(screen.queryByLabelText("Close templates")).toBeNull();
    cleanup();

    const onCancel = vi.fn();
    renderGallery(onCancel);
    fireEvent.click(screen.getByLabelText("Close templates"));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
