import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InstancePicker } from "./instance-picker";
import { ALL_INSTANCES } from "./lib/catalogue.types";

afterEach(cleanup);

const INSTANCES = [
  { id: "ks_1", label: "payments", hint: "ks_1" },
  { id: "ks_2", label: "internal", hint: "ks_2" },
  { id: "ks_3", label: "billing", hint: "ks_3" },
];

const renderPicker = (value: string[]) => {
  const onChange = vi.fn();
  render(
    <InstancePicker
      noun="keyspaces"
      instances={INSTANCES}
      isLoading={false}
      value={value}
      onChange={onChange}
    />,
  );
  return onChange;
};

describe("InstancePicker", () => {
  it("labels the trigger with the all row", () => {
    renderPicker([ALL_INSTANCES]);
    expect(screen.getByRole("combobox").textContent).toContain("All keyspaces");
  });

  it("joins the first two names and counts the rest", () => {
    renderPicker(["ks_1", "ks_2", "ks_3"]);
    const trigger = screen.getByRole("combobox").textContent ?? "";
    expect(trigger).toContain("payments, internal");
    expect(trigger).toContain("+1");
  });

  it("asks for a pick when nothing is selected", () => {
    renderPicker([]);
    expect(screen.getByRole("combobox").textContent).toContain("Select keyspaces");
  });

  it("drops the all row when a named instance is picked", () => {
    const onChange = renderPicker([ALL_INSTANCES]);
    fireEvent.click(screen.getByRole("combobox"));
    fireEvent.click(screen.getByText("payments"));

    expect(onChange).toHaveBeenCalledWith(["ks_1"]);
  });

  it("drops named instances when the all row is picked", () => {
    const onChange = renderPicker(["ks_1"]);
    fireEvent.click(screen.getByRole("combobox"));
    fireEvent.click(screen.getByText("All keyspaces"));

    expect(onChange).toHaveBeenCalledWith([ALL_INSTANCES]);
  });
});
