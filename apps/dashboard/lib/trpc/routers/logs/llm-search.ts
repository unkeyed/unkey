import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod";
import { env } from "@/lib/env";

const METHODS = ["GET", "POST", "PUT", "DELETE", "PATCH"] as const;
const FilterOperatorEnum = z.enum(["is", "contains", "startsWith", "endsWith"]);
const FilterFieldSchema = z.enum([
  "host",
  "requestId",
  "methods",
  "paths",
  "status",
]);

export const FilterOutputSchema = z
  .object({
    field: FilterFieldSchema,
    filters: z.array(
      z.object({
        operator: FilterOperatorEnum,
        value: z.union([z.string(), z.number()]),
      })
    ),
  })
  .refine(
    (data) => {
      switch (data.field) {
        case "status":
          return data.filters.every(
            (f) =>
              f.operator === "is" &&
              typeof f.value === "number" &&
              f.value >= 100 &&
              f.value <= 599
          );

        case "methods":
          return data.filters.every(
            (f) =>
              f.operator === "is" &&
              typeof f.value === "string" &&
              METHODS.includes(f.value as any)
          );

        case "paths":
          return data.filters.every(
            (f) =>
              ["is", "contains", "startsWith", "endsWith"].includes(
                f.operator
              ) && typeof f.value === "string"
          );

        case "host":
          return data.filters.every(
            (f) =>
              ["is", "contains"].includes(f.operator) &&
              typeof f.value === "string"
          );

        case "requestId":
          return data.filters.every(
            (f) => f.operator === "is" && typeof f.value === "string"
          );
      }
    },
    {
      message: "Invalid field/operator/value combination",
    }
  );

const openai = new OpenAI({
  apiKey: env().OPENAI_API_KEY,
});

function transformToQuerySearchParams(
  result: z.infer<typeof FilterOutputSchema>
): any {
  const output: any = {
    host: null,
    requestId: null,
    methods: null,
    paths: null,
    status: null,
  };

  switch (result.field) {
    case "host":
    case "requestId":
      if (result.filters.length > 0) {
        output[result.field] = result.filters[0];
      }
      break;

    case "methods":
    case "paths":
    case "status":
      if (result.filters.length > 0) {
        output[result.field] = result.filters;
      }
      break;
  }

  return output;
}

async function getStructuredSearchFromLLM(userSearchMsg: string) {
  const completion = await openai.beta.chat.completions.parse({
    model: "gpt-4o-2024-08-06",
    messages: [
      {
        role: "system",
        content: `You are an expert at converting natural language queries into structured filters. Examples:
        
        Query: "find all POST requests"
        Result: { 
          field: "methods",
          filters: [{ operator: "is", value: "POST" }]
        }
        
        Query: "show me GET and POST requests"
        Result: {
          field: "methods",
          filters: [
            { operator: "is", value: "GET" },
            { operator: "is", value: "POST" }
          ]
        }
        
        Query: "find requests with status 404"
        Result: {
          field: "status",
          filters: [{ operator: "is", value: 404 }]
        }
        
        Query: "find requests to api/v1"
        Result: {
          field: "paths",
          filters: [{ operator: "startsWith", value: "api/v1" }]
        }
        
        Query: "show requests from test.example.com"
        Result: {
          field: "host",
          filters: [{ operator: "is", value: "test.example.com" }]
        }
        
        Remember:
        - Methods only accept "is" operator and must be one of: GET, POST, PUT, DELETE, PATCH
        - Status codes must be between 100-599 and only use "is" operator
        - Host accepts "is" and "contains" operators
        - RequestId only accepts "is" operator
        - Paths accept all operators (is, contains, startsWith, endsWith)`,
      },
      {
        role: "user",
        content: userSearchMsg,
      },
    ],
    response_format: zodResponseFormat(FilterOutputSchema, "searchQuery"),
  });

  return transformToQuerySearchParams(completion.choices[0].message.parsed);
}

export const llmSearch = rateLimitedProcedure(ratelimit.update)
  .input(z.string())
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const result = await getStructuredSearchFromLLM(input);
    return result;
  });
