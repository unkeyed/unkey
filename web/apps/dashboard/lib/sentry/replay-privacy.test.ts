import { describe, expect, it } from "vitest";
import {
  replayMaskedLogSectionTitles,
  replayPrivacyOptions,
  replaySampleRates,
} from "./replay-privacy";

describe("replayPrivacyOptions", () => {
  it("leaves ordinary UI text, inputs, and media visible", () => {
    expect(replayPrivacyOptions.maskAllText).toBe(false);
    expect(replayPrivacyOptions.maskAllInputs).toBe(false);
    expect(replayPrivacyOptions.blockAllMedia).toBe(false);
  });

  it("keeps masking API keys, PII, and passwords, including fields nested under those markers", () => {
    const mask = replayPrivacyOptions.mask ?? [];
    for (const selector of [
      "[type='email']",
      "[type='password']",
      ".secret",
      ".secret input",
      ".secret textarea",
      ".api-key",
      ".unkey-root-key",
      ".external-id",
      ".email",
      ".token",
      "[data-api-key]",
      "[data-api-key] input",
      "[data-secret]",
      "[data-secret] textarea",
      "[data-external-id]",
      "[data-email]",
      "[data-token]",
      "[data-unkey-root-key]",
    ]) {
      expect(mask).toContain(selector);
    }
  });

  it("does not capture network bodies or headers", () => {
    expect(replayPrivacyOptions.networkCaptureBodies).toBe(false);
    expect(replayPrivacyOptions.networkDetailAllowUrls).toEqual([]);
    expect(replayPrivacyOptions.networkRequestHeaders).toEqual([]);
    expect(replayPrivacyOptions.networkResponseHeaders).toEqual([]);
  });

  it("still blocks media marked as sensitive", () => {
    expect(replayPrivacyOptions.block).toEqual(
      expect.arrayContaining(["[data-sensitive-media]", ".sensitive-media"]),
    );
  });
});

describe("replaySampleRates", () => {
  it("uploads a replay only when a session has an error", () => {
    expect(replaySampleRates.replaysSessionSampleRate).toBe(0);
    expect(replaySampleRates.replaysOnErrorSampleRate).toBe(1);
  });
});

describe("replayMaskedLogSectionTitles", () => {
  it("masks on-screen request and response payloads without masking the rest of a log", () => {
    expect([...replayMaskedLogSectionTitles].sort()).toEqual([
      "Request Body",
      "Request Header",
      "Response Body",
      "Response Header",
    ]);
  });
});
