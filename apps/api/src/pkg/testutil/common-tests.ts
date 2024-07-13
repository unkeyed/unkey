/**
 * Testing for insufficient roles is very similar for all endpoints.
 *
 * Here we create some utilities that can be imported in the respective `{path}.security.test.ts`
 * files.
 */

import { describe, expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type { ErrorResponse } from "../errors";
import type { StepRequest } from "./request";

type MaybePromise<T> = T | Promise<T>;

/**
 * The prepareRequest function must not return a request with Authorization header, because we take
 * care of that here.
 */
type StepRequestWithoutAuthorizationHeader<TReq> = Omit<StepRequest<TReq>, "headers"> & {
  headers?: {
    [key: string]: string;
  } & {
    Authorization?: never;
  };
};

export function runCommonRouteTests<TReq>(config: {
  prepareRequest: (
    h: IntegrationHarness,
  ) => MaybePromise<StepRequestWithoutAuthorizationHeader<TReq>>;
}) {
  describe("disabled workspace", () => {
    test("should reject the request", async (t) => {
      const h = await IntegrationHarness.init(t);
      await h.db.primary
        .update(schema.workspaces)
        .set({ enabled: false })
        .where(eq(schema.workspaces.id, h.resources.userWorkspace.id));

      const req = await config.prepareRequest(h);

      req.headers = {
        ...req.headers,
        // @ts-expect-error
        Authorization: `Bearer ${(await h.createRootKey(["*"])).key}`,
      };
      const res = await h.do<TReq, ErrorResponse>(req);
      expect(res.status, `expected: 403, received: ${JSON.stringify(res, null, 2)}`).toEqual(403);
      expect(res.body).toMatchObject({
        error: {
          code: "FORBIDDEN",
          docs: "https://unkey.dev/docs/api-reference/errors/code/FORBIDDEN",
          message: "workspace is disabled",
        },
      });
    });
  });

  describe("shared permissions tests", () => {
    test("without a key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const req = await config.prepareRequest(h);
      const res = await h.do<TReq, ErrorResponse>(req);
      expect(res.status).toEqual(403);
      expect(res.body).toMatchObject({
        error: {
          code: "UNAUTHORIZED",
          docs: "https://unkey.dev/docs/api-reference/errors/code/UNAUTHORIZED",
          message: "key required",
        },
      });
    });

    test("with wrong key", async (t) => {
      const h = await IntegrationHarness.init(t);
      const req = await config.prepareRequest(h);
      const res = await h.do<TReq, ErrorResponse>({
        ...req,
        headers: {
          ...req.headers,
          Authorization: "Bearer INVALID_KEY",
        },
      });

      expect(res.status).toEqual(403);
      expect(res.body).toMatchObject({
        error: {
          code: "UNAUTHORIZED",
          docs: "https://unkey.dev/docs/api-reference/errors/code/UNAUTHORIZED",
        },
      });
    });

    describe("without permission", () => {
      describe.each([
        { name: "no roles", roles: [] },
        { name: "wrong roles", roles: [randomUUID(), randomUUID()] },
        {
          name: "full access to wrong api",
          roles: [
            `api.${newId("api")}.read_api`,
            `api.${newId("api")}.update_api`,
            `api.${newId("api")}.delete_api`,
            `api.${newId("api")}.read_key`,
            `api.${newId("api")}.update_key`,
            `api.${newId("api")}.delete_key`,
            `api.${newId("api")}.create_key`,
          ],
        },
      ])("$name", async ({ roles }) => {
        test("should reject", async (t) => {
          const h = await IntegrationHarness.init(t);
          const { key: rootKey } = await h.createRootKey(roles);

          const req = await config.prepareRequest(h);
          const res = await h.do<TReq, ErrorResponse>({
            ...req,
            headers: {
              ...req.headers,
              Authorization: `Bearer ${rootKey}`,
            },
          });

          expect(res.status).toEqual(403);
          expect(res.body).toMatchObject({
            error: {
              code: "INSUFFICIENT_PERMISSIONS",
              docs: "https://unkey.dev/docs/api-reference/errors/code/INSUFFICIENT_PERMISSIONS",
            },
          });
        });
      });
    });
  });
}
