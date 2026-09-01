import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import * as React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
vi.mock("@unkey/icons", () => ({
  Check: () => <span data-testid="check" />,
  ChevronExpandY: () => <span data-testid="chevron" />,
  Magnifier: () => <span data-testid="magnifier" />,
}));

import { Combobox, type ComboboxOption } from "./combobox";

const options: ComboboxOption[] = [
  { label: "Alpha", value: "alpha", searchValue: "alpha" },
  { label: "Beta", value: "beta", searchValue: "beta" },
  { label: "Locked", value: "locked", searchValue: "locked", disabled: true },
  { label: <span>Auto</span>, selectedLabel: "Auto", value: "", searchValue: "automatic" },
];

function Harness(props: { creatable?: boolean; onSelect: (value: string) => void }) {
  const [value, setValue] = React.useState("alpha");
  return (
    <Combobox
      options={options}
      value={value}
      creatable={props.creatable}
      onSelect={(next) => {
        props.onSelect(next);
        setValue(next);
      }}
      placeholder="Pick one"
      emptyMessage="Nothing here"
    />
  );
}

const getTrigger = () => screen.getAllByRole("combobox")[0];

describe("Combobox", () => {
  afterEach(cleanup);

  it("renders the selected label and opens the popup", () => {
    const onSelect = vi.fn();
    render(<Harness onSelect={onSelect} />);

    const trigger = getTrigger();
    expect(trigger.textContent).toContain("Alpha");

    fireEvent.click(trigger);
    expect(screen.getByRole("listbox")).toBeTruthy();
    expect(screen.getAllByRole("option").length).toBe(4);
  });

  it("filters by searchValue and selects on click", () => {
    const onSelect = vi.fn();
    render(<Harness onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    fireEvent.change(screen.getByPlaceholderText("Search..."), { target: { value: "bet" } });

    const shown = screen.getAllByRole("option");
    expect(shown.length).toBe(1);
    fireEvent.click(shown[0]);
    expect(onSelect).toHaveBeenCalledWith("beta");
  });

  it("matches the empty-string option through searchValue", () => {
    const onSelect = vi.fn();
    render(<Harness onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    fireEvent.change(screen.getByPlaceholderText("Search..."), { target: { value: "automa" } });
    expect(screen.getAllByRole("option").length).toBe(1);
  });

  it("shows the empty message when nothing matches", () => {
    const onSelect = vi.fn();
    render(<Harness onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    fireEvent.change(screen.getByPlaceholderText("Search..."), { target: { value: "zzz" } });
    expect(screen.getByText("Nothing here")).toBeTruthy();
  });

  it("does not select disabled options", () => {
    const onSelect = vi.fn();
    render(<Harness onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    fireEvent.change(screen.getByPlaceholderText("Search..."), { target: { value: "locked" } });
    fireEvent.click(screen.getAllByRole("option")[0]);
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("offers the typed value when creatable", () => {
    const onSelect = vi.fn();
    render(<Harness creatable onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    const input = screen.getByPlaceholderText("Search...");
    fireEvent.change(input, { target: { value: "src/**" } });

    const shown = screen.getAllByRole("option");
    expect(shown[0].textContent).toContain('Use "src/**"');

    fireEvent.click(shown[0]);
    expect(onSelect).toHaveBeenCalledWith("src/**");
  });

  it("selects the typed value on Enter when creatable", () => {
    const onSelect = vi.fn();
    render(<Harness creatable onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    const input = screen.getByPlaceholderText("Search...");
    fireEvent.change(input, { target: { value: "services/api" } });
    fireEvent.keyDown(input, { key: "Enter" });

    expect(onSelect).toHaveBeenCalledWith("services/api");
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("does not offer to create a value that an option already matches through searchValue", () => {
    const onSelect = vi.fn();
    render(<Harness creatable onSelect={onSelect} />);

    fireEvent.click(getTrigger());
    const input = screen.getByPlaceholderText("Search...");
    fireEvent.change(input, { target: { value: "Automatic" } });

    const shown = screen.getAllByRole("option");
    expect(shown).toHaveLength(1);
    expect(shown[0].textContent).not.toContain("Use");
  });

  it("keeps the open popup's search input enabled when disabled flips mid-search", () => {
    function DisableHarness(props: { disabled: boolean }) {
      return (
        <Combobox
          options={options}
          value="alpha"
          onSelect={() => {}}
          placeholder="Pick one"
          disabled={props.disabled}
        />
      );
    }
    const { rerender } = render(<DisableHarness disabled={false} />);

    fireEvent.click(getTrigger());
    const input = screen.getByPlaceholderText("Search...");
    fireEvent.change(input, { target: { value: "al" } });

    rerender(<DisableHarness disabled={true} />);

    expect((getTrigger() as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByPlaceholderText("Search...") as HTMLInputElement).disabled).toBe(false);
  });
});
