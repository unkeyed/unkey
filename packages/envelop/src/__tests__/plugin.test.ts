import { assertSingleExecutionValue, createTestkit } from "@envelop/testing";
import { makeExecutableSchema } from "@graphql-tools/schema";
import "dotenv/config";

import { NetworkError, RateLimitError } from "../errors";
import { useUnkey } from "../plugin";

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

  describe("onContextBuilding", () => {
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

    it("Should error with invalid Unkey token", async () => {
      const testkit = createTestkit([useUnkey({ token: "123" })], schema);
      try {
        await testkit.execute("query { foo }");
      } catch (e) {
        expect(e).toBeInstanceOf(RateLimitError);
      }
    });

    it("Should error with invalid Unkey token", async () => {
      const testkit = createTestkit([useUnkey({ token: "" })], schema);
      try {
        await testkit.execute("query { foo }");
      } catch (e) {
        expect(e).toBeInstanceOf(NetworkError);
      }
    });

    it("Should rate limit", async () => {
      const token = process.env.RATE_LIMIT_5_PER_1_SECOND || "";
      const testkit = createTestkit([useUnkey({ token })], schema);
      try {
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
        // Assert that the result is correct
        expect(result.data).toBeUndefined();
      } catch (e) {
        expect(e).toBeInstanceOf(RateLimitError);
      }
    });
  });

  describe("errors", () => {
    describe("network errors", () => {
      it("A network error should have a default message", () => {
        const error = new NetworkError();
        expect(error.message).toMatchSnapshot();
      });
      it("A network error should have a custom message", () => {
        const error = new NetworkError("Custom message");
        expect(error.message).toMatchSnapshot();
      });
    });

    describe("rate limit errors", () => {
      it("A rate limit error should have a default message", () => {
        const error = new RateLimitError();
        expect(error.message).toMatchSnapshot();
      });
      it("A rate limit error should have a custom message", () => {
        const error = new RateLimitError("Custom message");
        expect(error.message).toMatchSnapshot();
      });
    });
  });
});
