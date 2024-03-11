/**
 * Testing for insufficient roles is very similar for all endpoints.
 *
 * Here we create some utilities that can be imported in the respective `{path}.security.test.ts`
 * files.
 */

import { describe, expect, test } from "vitest";

import { randomUUID } from "crypto";
import type { ErrorResponse } from "@/pkg/errors";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { StepRequest } from "./request";

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
  prepareRequest: (h: RouteHarness) => MaybePromise<StepRequestWithoutAuthorizationHeader<TReq>>;
}) {
  describe("disabled workspace", () => {
    test("should reject the request", async () => {
      const h = await RouteHarness.init();
      await h.db
        .update(schema.workspaces)
        .set({ enabled: false })
        .where(eq(schema.workspaces.id, h.resources.userWorkspace.id));

      const req = await config.prepareRequest(h);
      const res = await h.do<TReq, ErrorResponse>(req);
      expect(res.status).toEqual(403);
      expect(res.body).toMatchObject({
        error: {
          code: "UNAUTHORIZED",
          docs: "https://unkey.dev/docs/api-reference/errors/code/UNAUTHORIZED",
          message: "workspace disabled",
        },
      });
    });
  });

  describe("shared role tests", () => {
    test("without a key", async () => {
      const h = await RouteHarness.init();
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

    test("with wrong key", async () => {
      const h = await RouteHarness.init();
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
          message: "unauthorized",
        },
      });
    });

    describe("without permission", () => {
      test.each([
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
        const h = await RouteHarness.init();
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
            message: "unauthorized",
          },
        });
      });
    });
  });
}
