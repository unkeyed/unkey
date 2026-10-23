import { expect, test } from "vitest";
import { cva } from "./cva";

const button = cva("base", {
  variants: {
    size: { sm: "s", lg: "l" },
    tone: { x: "tx", y: "ty" },
    on: { true: "on", false: "off" },
  },
  defaultVariants: { size: "sm", tone: "x", on: false },
  compoundVariants: [
    { size: ["sm", "lg"], tone: "y", class: "sy" },
    { on: true, className: "lit" },
  ],
});

test("matches class-variance-authority output", () => {
  expect(button()).toBe("base s tx off");
  expect(button({ size: "lg" })).toBe("base l tx off");
  expect(button({ tone: "y", on: true, className: "extra" })).toBe("base s ty on sy lit extra");
  expect(button({ size: null, tone: undefined })).toBe("base tx off");
});
