import { describe, expect, test } from "vitest";
import { imageDisplay, imageTag } from "./image-reference";

describe("imageTag", () => {
  test("reads the tag from the last path segment", () => {
    expect(imageTag("ghcr.io/unkeyed/vault:v1.0.1")).toBe("v1.0.1");
  });

  test("does not mistake a registry port for a tag", () => {
    expect(imageTag("localhost:5000/vault")).toBe("latest");
  });

  test("defaults to latest", () => {
    expect(imageTag("unkeyed/vault")).toBe("latest");
  });
});

describe("imageDisplay", () => {
  test("drops a registry host", () => {
    expect(imageDisplay("ghcr.io/unkeyed/vault:v1.0.1")).toBe("unkeyed/vault:v1.0.1");
    expect(imageDisplay("localhost:5000/vault:dev")).toBe("vault:dev");
  });

  test("keeps a Docker Hub namespace", () => {
    expect(imageDisplay("unkeyed/vault:v1.0.1")).toBe("unkeyed/vault:v1.0.1");
    expect(imageDisplay("redis")).toBe("redis");
  });
});
