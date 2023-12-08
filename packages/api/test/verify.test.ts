import { verifyKey } from "../src";
import { resetFetchMock, testOneFetchCall } from "./mock-fetch.test";
import { beforeEach, describe, test } from "bun:test";

describe("verifyKey() function", () => {
  beforeEach(() => resetFetchMock());

  test("accepts plain string", async () =>
    testOneFetchCall({
      url: "https://api.unkey.dev/v1/keys/verify",
      method: "POST",
      headers: {
        Authorization: "Bearer public",
      },
      jsonBody: {
        key: "api key",
      },
      execute: () => verifyKey("api key"),
    }));

  test("accepts { key, apiId }", async () =>
    testOneFetchCall({
      url: "https://api.unkey.dev/v1/keys/verify",
      method: "POST",
      headers: {
        Authorization: "Bearer public",
      },
      jsonBody: {
        key: "api key",
        apiId: "api id",
      },
      execute: (req) => verifyKey(req),
    }));
});
