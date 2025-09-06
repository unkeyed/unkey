import { describe, expect, it } from "vitest";
import { getKeysOverviewLogs, keysOverviewLogsParams } from "./keys";

// Mock Querier type for testing
type MockQuerier = {
  query: (params: any) => (queryParams: any) => Promise<{ val: any[]; err?: any }>;
};

describe("getKeysOverviewLogs", () => {
  it("should include creditSpendMode filter when enabled", async () => {
    let capturedQuery = "";

    // Mock ClickHouse querier that captures the generated query
    const mockQuerier: MockQuerier = {
      query: (params) => {
        capturedQuery = params.query;
        return async () => ({
          val: [
            {
              key_id: "test_key_1",
              time: 1234567890,
              request_id: "req_1",
              tags: [],
              valid_count: 5,
              error_count: 0,
              spent_credits: 10,
              outcome_counts_array: [["VALID", 5]],
            },
          ],
        });
      },
    };

    const logsFunction = getKeysOverviewLogs(mockQuerier);

    // Test with creditSpendMode enabled
    const paramsWithCreditMode = {
      workspaceId: "ws_test",
      keyspaceId: "ks_test",
      limit: 50,
      startTime: 1234567000,
      endTime: 1234568000,
      creditSpendMode: true,
      outcomes: null,
      names: null,
      identities: null,
      keyIds: null,
      tags: null,
      cursorTime: null,
      sorts: null,
    };

    await logsFunction(paramsWithCreditMode);

    // Verify that the query includes the credit spend filter
    expect(capturedQuery).toContain("spent_credits > 0");
    expect(capturedQuery).toContain("AND (spent_credits > 0)");
  });

  it("should not include creditSpendMode filter when disabled", async () => {
    let capturedQuery = "";

    // Mock ClickHouse querier that captures the generated query
    const mockQuerier: MockQuerier = {
      query: (params) => {
        capturedQuery = params.query;
        return async () => ({
          val: [
            {
              key_id: "test_key_1",
              time: 1234567890,
              request_id: "req_1",
              tags: [],
              valid_count: 5,
              error_count: 0,
              spent_credits: 0,
              outcome_counts_array: [["VALID", 5]],
            },
          ],
        });
      },
    };

    const logsFunction = getKeysOverviewLogs(mockQuerier);

    // Test with creditSpendMode disabled
    const paramsWithoutCreditMode = {
      workspaceId: "ws_test",
      keyspaceId: "ks_test",
      limit: 50,
      startTime: 1234567000,
      endTime: 1234568000,
      creditSpendMode: false,
      outcomes: null,
      names: null,
      identities: null,
      keyIds: null,
      tags: null,
      cursorTime: null,
      sorts: null,
    };

    await logsFunction(paramsWithoutCreditMode);

    // Verify that the query uses TRUE instead of credit filter
    expect(capturedQuery).toContain("AND (TRUE)");
    expect(capturedQuery).not.toContain("spent_credits > 0");
  });

  it("should validate creditSpendMode parameter in schema", () => {
    // Test that the schema accepts creditSpendMode parameter
    const validParams = {
      workspaceId: "ws_test",
      keyspaceId: "ks_test",
      limit: 50,
      startTime: 1234567000,
      endTime: 1234568000,
      creditSpendMode: true,
      outcomes: null,
      names: null,
      identities: null,
      keyIds: null,
      tags: null,
      cursorTime: null,
      sorts: null,
    };

    const result = keysOverviewLogsParams.safeParse(validParams);
    expect(result.success).toBe(true);

    if (result.success) {
      expect(result.data.creditSpendMode).toBe(true);
    }
  });

  it("should default creditSpendMode to false when not provided", () => {
    const paramsWithoutCreditMode = {
      workspaceId: "ws_test",
      keyspaceId: "ks_test",
      limit: 50,
      startTime: 1234567000,
      endTime: 1234568000,
      outcomes: null,
      names: null,
      identities: null,
      keyIds: null,
      tags: null,
      cursorTime: null,
      sorts: null,
    };

    const result = keysOverviewLogsParams.safeParse(paramsWithoutCreditMode);
    expect(result.success).toBe(true);

    if (result.success) {
      expect(result.data.creditSpendMode).toBe(false);
    }
  });

  it("should combine credit spend filter with other filters correctly", async () => {
    let capturedQuery = "";

    const mockQuerier: MockQuerier = {
      query: (params) => {
        capturedQuery = params.query;
        return async () => ({
          val: [],
        });
      },
    };

    const logsFunction = getKeysOverviewLogs(mockQuerier);

    // Test with creditSpendMode and other filters
    const paramsWithMultipleFilters = {
      workspaceId: "ws_test",
      keyspaceId: "ks_test",
      limit: 50,
      startTime: 1234567000,
      endTime: 1234568000,
      creditSpendMode: true,
      outcomes: [{ value: "VALID" as const, operator: "is" as const }],
      names: null,
      identities: null,
      keyIds: [{ value: "test_key", operator: "contains" as const }],
      tags: null,
      cursorTime: null,
      sorts: null,
    };

    await logsFunction(paramsWithMultipleFilters);

    // Verify that both credit spend filter and other filters are applied
    expect(capturedQuery).toContain("spent_credits > 0");
    expect(capturedQuery).toContain("outcome = {outcomeValue_0: String}");
    expect(capturedQuery).toContain("like(key_id, CONCAT('%', {keyIdValue_0: String}, '%'))");
  });
});
