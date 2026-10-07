import { describe, expect, it } from "vitest";
import {
  CONDITION_TYPES,
  conditionRowState,
  getDefaultCondition,
  withOperator,
} from "./condition-kinds";

const STRING_OPTIONS = [
  { value: "exact", label: "equals" },
  { value: "prefix", label: "starts with" },
  { value: "regex", label: "matches regex" },
];

describe("CONDITION_TYPES", () => {
  it("lists the condition fields in menu order", () => {
    expect(CONDITION_TYPES).toEqual([
      { value: "path", label: "Path" },
      { value: "method", label: "Method" },
      { value: "header", label: "Header" },
      { value: "queryParam", label: "Query param" },
      { value: "remoteIp", label: "Client IP" },
    ]);
  });
});

describe("getDefaultCondition", () => {
  it("starts every field on its first operator with an empty value", () => {
    expect(getDefaultCondition("path", "1")).toEqual({
      id: "1",
      type: "path",
      operator: "exact",
      value: "",
    });
    expect(getDefaultCondition("method", "1")).toEqual({
      id: "1",
      type: "method",
      operator: "anyOf",
      methods: [],
    });
    expect(getDefaultCondition("header", "1")).toEqual({
      id: "1",
      type: "header",
      name: "",
      operator: "exact",
      value: "",
    });
    expect(getDefaultCondition("remoteIp", "1")).toEqual({
      id: "1",
      type: "remoteIp",
      operator: "in",
      value: "",
    });
  });
});

describe("conditionRowState", () => {
  it("path with an exact match: no name, operator select, path placeholder, no regex", () => {
    expect(conditionRowState({ id: "1", type: "path", operator: "exact", value: "/v1" })).toEqual({
      name: { type: "none" },
      operator: { type: "select", value: "exact", options: STRING_OPTIONS },
      value: { type: "value", ariaLabel: "Path", placeholder: "/v1/checkout" },
      regex: { type: "none" },
    });
  });

  it("path with a broken regex carries the regex placeholder, prompt and syntax error", () => {
    const path = { id: "1", type: "path" as const, operator: "regex" as const, value: "(" };
    const state = conditionRowState(path);
    expect(state.value).toEqual({ type: "value", ariaLabel: "Path", placeholder: "^/v1/.*" });
    expect(state.regex).toEqual({
      type: "regex",
      condition: path,
      prompt: "Describe the paths, e.g. all API routes under /api/v2",
      syntaxError: "Invalid regular expression: /(/: Unterminated group",
    });
  });

  it("header with a valid regex has a name field and no syntax error", () => {
    const header = {
      id: "1",
      type: "header" as const,
      name: "Authorization",
      operator: "regex" as const,
      value: "^Bearer .+",
    };
    const state = conditionRowState(header);
    expect(state.name).toEqual({
      type: "name",
      label: "Header name",
      placeholder: "X-Tenant-Id",
      condition: header,
    });
    expect(state.value).toEqual({ type: "value", ariaLabel: "Value", placeholder: "^Bearer .+" });
    expect(state.regex).toEqual({
      type: "regex",
      condition: header,
      prompt: "Describe the values, e.g. bearer tokens",
      syntaxError: undefined,
    });
  });

  it("query param regex keeps the plain value placeholder", () => {
    const state = conditionRowState({
      id: "1",
      type: "queryParam",
      name: "id",
      operator: "regex",
      value: "^[0-9]+$",
    });
    expect(state.value).toEqual({ type: "value", ariaLabel: "Value", placeholder: "value" });
    expect(state.regex).toMatchObject({
      type: "regex",
      prompt: "Describe the values, e.g. numeric values only",
    });
  });

  it("query param presence check has a name, every named operator, no value and no regex", () => {
    const query = {
      id: "1",
      type: "queryParam" as const,
      name: "debug",
      operator: "present" as const,
      value: "",
    };
    expect(conditionRowState(query)).toEqual({
      name: { type: "name", label: "Parameter name", placeholder: "debug", condition: query },
      operator: {
        type: "select",
        value: "present",
        options: [...STRING_OPTIONS, { value: "present", label: "is present" }],
      },
      value: { type: "none" },
      regex: { type: "none" },
    });
  });

  it("method has a fixed operator and the methods picker copy", () => {
    expect(
      conditionRowState({ id: "1", type: "method", operator: "anyOf", methods: ["GET"] }),
    ).toEqual({
      name: { type: "none" },
      operator: { type: "fixed", label: "is any of" },
      value: { type: "value", ariaLabel: "Methods", placeholder: "Pick methods" },
      regex: { type: "none" },
    });
  });

  it("client IP offers in and not in", () => {
    expect(
      conditionRowState({ id: "1", type: "remoteIp", operator: "notIn", value: "10.0.0.0/8" }),
    ).toEqual({
      name: { type: "none" },
      operator: {
        type: "select",
        value: "notIn",
        options: [
          { value: "in", label: "is in" },
          { value: "notIn", label: "is not in" },
        ],
      },
      value: {
        type: "value",
        ariaLabel: "IP ranges",
        placeholder: "203.0.113.0/24, 198.51.100.7",
      },
      regex: { type: "none" },
    });
  });
});

describe("withOperator", () => {
  it("turns a header into a presence check and drops its value", () => {
    expect(
      withOperator(
        { id: "1", type: "header", name: "X-Tenant", operator: "regex", value: "a" },
        "present",
      ),
    ).toEqual({ id: "1", type: "header", name: "X-Tenant", operator: "present", value: "" });
  });

  it("turns a presence check back into a string match with an empty value", () => {
    expect(
      withOperator(
        { id: "1", type: "queryParam", name: "debug", operator: "present", value: "" },
        "prefix",
      ),
    ).toEqual({ id: "1", type: "queryParam", name: "debug", operator: "prefix", value: "" });
  });

  it("keeps the value when switching between string matches", () => {
    expect(
      withOperator(
        { id: "1", type: "header", name: "X-Tenant", operator: "exact", value: "acme" },
        "regex",
      ),
    ).toEqual({ id: "1", type: "header", name: "X-Tenant", operator: "regex", value: "acme" });
  });

  it("switches a path mode and a client IP operator", () => {
    expect(
      withOperator({ id: "1", type: "path", operator: "exact", value: "/v1" }, "prefix"),
    ).toEqual({ id: "1", type: "path", operator: "prefix", value: "/v1" });
    expect(
      withOperator({ id: "1", type: "remoteIp", operator: "in", value: "10.0.0.0/8" }, "notIn"),
    ).toEqual({ id: "1", type: "remoteIp", operator: "notIn", value: "10.0.0.0/8" });
  });

  it("ignores an operator the field does not allow", () => {
    const path = { id: "1", type: "path" as const, operator: "exact" as const, value: "/v1" };
    expect(withOperator(path, "present")).toEqual(path);
    const ip = { id: "1", type: "remoteIp" as const, operator: "in" as const, value: "10.0.0.0/8" };
    expect(withOperator(ip, "regex")).toEqual(ip);
  });
});
