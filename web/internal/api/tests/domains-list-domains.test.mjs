import assert from "node:assert/strict";
import { test } from "node:test";
import { HTTPClient } from "../esm/lib/http.js";
import { NotFoundErrorResponse } from "../esm/models/errors/notfounderrorresponse.js";
import { Unkey } from "../esm/sdk/sdk.js";

test("listDomains preserves typed authentication 404 errors", async () => {
  const body = {
    meta: { requestId: "req_test" },
    error: {
      title: "Not Found",
      detail: "The workspace does not exist.",
      status: 404,
      type: "https://unkey.com/docs/errors/unkey/data/workspace_not_found",
    },
  };
  const client = new Unkey({
    rootKey: "test_key",
    httpClient: new HTTPClient({
      fetcher: async () => Response.json(body, { status: 404 }),
    }),
  });

  await assert.rejects(client.domains.listDomains({}), (error) => {
    assert.ok(error instanceof NotFoundErrorResponse);
    assert.deepEqual(error.error, body.error);
    assert.equal(error.meta.requestId, body.meta.requestId);
    return true;
  });
});

test("listDomains accepts an empty selection for a missing resource filter", async () => {
  const body = {
    meta: { requestId: "req_test" },
    data: [],
    pagination: { hasMore: false },
  };
  const client = new Unkey({
    rootKey: "test_key",
    httpClient: new HTTPClient({
      fetcher: async () => Response.json(body),
    }),
  });

  assert.deepEqual(await client.domains.listDomains({ project: "missing" }), body);
});
