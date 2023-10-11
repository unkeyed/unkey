import { assertSingleExecutionValue, createTestkit } from "@envelop/testing";
import { makeExecutableSchema } from "@graphql-tools/schema";
/* eslint-disable @typescript-eslint/no-explicit-any */
import "dotenv/config";

import { useUnkey } from "../plugin";
import { extractGraphQLHttpError } from "./errors.test";

describe("useUnkey", () => {
  const schema = makeExecutableSchema({
    typeDefs: /* GraphQL */ `
      type Query {
        foo: String!
      }
    `,
    resolvers: {
      Query: {
        foo: () => "hi",
      },
    },
  });

  describe("Test Rate Limits", () => {
    it("Should query when not using the Unkey plugin", async () => {
      // Create a testkit for the plugin, using the plugin and a dummy schema
      const testkit = createTestkit([], schema);
      // Execute the envelop using a simple query
      const result = await testkit.execute("query { foo }");
      // During tests, it's simpler to assume you are dealing with a non-stream responses
      assertSingleExecutionValue(result);
      // Assert that the result is correct
      expect(result.errors).toBeUndefined();
      expect(result.data?.foo).toBe("hi");
    });

    it("Should error with missing Unkey token", async () => {
      const testkit = createTestkit([useUnkey({ token: "" })], schema);
      const result = await testkit.execute("query { foo }");

      const error = extractGraphQLHttpError(result);
      expect(error.extensions.http.status).toBe(400);
    });

    it("Should rate limit with invalid Unkey token", async () => {
      const testkit = createTestkit([useUnkey({ token: "123" })], schema);
      const result = await testkit.execute("query { foo }");

      const error = extractGraphQLHttpError(result);
      expect(error.extensions.http.status).toBe(429);
      expect(error.extensions.http).toHaveProperty("headers");
      expect(error.extensions.http.headers).toHaveProperty("Retry-After");
    });

    describe("Should rate limit", () => {
      it("with a token that allows 5 requests per 1 second", async () => {
        const token = process.env.RATE_LIMIT_5_PER_1_SECOND || "";
        const testkit = createTestkit([useUnkey({ token })], schema);
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        await testkit.execute("query { foo }");
        const result = await testkit.execute("query { foo }");
        assertSingleExecutionValue(result);

        expect(result.data).toBeNull();

        const error = extractGraphQLHttpError(result);
        expect(error.extensions.http.status).toBe(429);
        expect(error.extensions.http).toHaveProperty("headers");
        expect(error.extensions.http.headers).toHaveProperty("Retry-After");
      });
    });
  });
});
