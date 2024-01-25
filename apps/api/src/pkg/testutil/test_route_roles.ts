/**
 * Testing for insufficient roles is very similar for all endpoints.
 *
 * Here we create some utilities that can be imported in the respective `{path}.security.test.ts`
 * files.
 */

import { describe, expect, test } from "vitest";

import { randomUUID } from "crypto";
import type { ErrorResponse } from "@/pkg/errors";
import { Harness } from "@/pkg/testutil/harness";
import { newId } from "@unkey/id";
import { App } from "../hono/app";
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

export function runSharedRoleTests<TReq>(config: {
  prepareRequest: (h: Harness) => MaybePromise<StepRequestWithoutAuthorizationHeader<TReq>>;
  registerHandler: (app: App) => any;
}) {
  describe("shared role tests", () => {
    test("without a key", async () => {
      const h = await Harness.init();
      h.useRoutes(config.registerHandler);

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
      const h = await Harness.init();
      h.useRoutes(config.registerHandler);

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
          message: "you're not allowed to do this",
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
        const h = await Harness.init();
        h.useRoutes(config.registerHandler);

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
            message: "you're not allowed to do this",
          },
        });
      });
    });
  });
}
