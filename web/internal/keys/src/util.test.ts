import { expect, test } from "vitest";
import { newKey } from "./util";

test("returns key display metadata", async () => {
  const result = await newKey({ prefix: "prod_sk", byteLength: 16 });

  expect(result.prefix).toBe("prod_sk");
  expect(result.start).toBe(result.key.slice(8, 12));
  expect(result.end).toBe(result.key.slice(-4));
});
