import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import { mergeDeploymentTargets } from "./connection-rules";

describe("connection target deployment pagination", () => {
  it("keeps a live pin resolved independently when it is beyond the choices page", () => {
    const choices = Array.from({ length: 100 }, () => ({
      id: newId("test"),
    }));
    const livePin = { id: newId("test"), status: "ready" };
    const another = { id: newId("test"), status: "ready" };

    expect(mergeDeploymentTargets([livePin], choices)).toEqual([livePin, ...choices]);
    expect(
      mergeDeploymentTargets([livePin], [{ id: livePin.id, status: "stopped" }, another]),
    ).toEqual([livePin, another]);
  });
});
