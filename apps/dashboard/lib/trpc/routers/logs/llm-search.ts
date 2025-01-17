import { METHODS } from "@/app/(app)/logs-v2/constants";
import { filterFieldConfig, filterOutputSchema } from "@/app/(app)/logs-v2/filters.schema";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod";
import { z } from "zod";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

async function getStructuredSearchFromLLM(userSearchMsg: string) {
  try {
    if (!openai) {
      return null; // Skip LLM processing in development environment when OpenAI API key is not configured
    }
    const completion = await openai.beta.chat.completions.parse({
      // Don't change the model only a few models allow structured outputs
      model: "gpt-4o-2024-08-06",
      temperature: 0.2, // Range 0-2, lower = more focused/deterministic
      top_p: 0.1, // Alternative to temperature, controls randomness
      frequency_penalty: 0.5, // Range -2 to 2, higher = less repetition
      presence_penalty: 0.5, // Range -2 to 2, higher = more topic diversity
      n: 1, // Number of completions to generate
      messages: [
        {
          role: "system",
          content: getSystemPrompt(),
        },
        {
          role: "user",
          content: userSearchMsg,
        },
      ],
      response_format: zodResponseFormat(filterOutputSchema, "searchQuery"),
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Try using phrases like:\n" +
          "• 'find all POST requests'\n" +
          "• 'show requests with status 404'\n" +
          "• 'find requests to api/v1'\n" +
          "• 'show requests from test.example.com'\n" +
          "• 'find all GET and POST requests'\n" +
          "For additional help, contact support@unkey.dev",
      });
    }

    return completion.choices[0].message.parsed;
  } catch (error) {
    console.error(
      `Something went wrong when querying OpenAI. Input: ${JSON.stringify(
        userSearchMsg,
      )}\n Output ${(error as Error).message}}`,
    );
    if (error instanceof TRPCError) {
      throw error;
    }

    if ((error as any).response?.status === 429) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Search rate limit exceeded. Please try again in a few minutes.",
      });
    }

    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "Failed to process your search query. Please try again or contact support@unkey.dev if the issue persists.",
    });
  }
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
            "Failed to verify workspace access. Please try again or contact support@unkey.dev if this persists.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    return await getStructuredSearchFromLLM(input);
  });

// HELPERS

const getSystemPrompt = () => {
  const operatorsByField = Object.entries(filterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      let constraints = "";
      if (field === "methods") {
        constraints = ` and must be one of: ${METHODS.join(", ")}`;
      } else if (field === "status") {
        constraints = " and must be between 200-599";
      }
      return `- ${field} accepts ${operators} operator${
        config.operators.length > 1 ? "s" : ""
      }${constraints}`;
    })
    .join("\n");
  return `You are an expert at converting natural language queries into filters. For queries with multiple conditions, output all relevant filters. We will process them in sequence to build the complete filter. For status codes, always return one for each variant like 200,400 or 500 instead of 200,201, etc... - the application will handle status code grouping internally.

Examples:
Query: "path should start with /api/oz and method should be POST"
Result: [
  { 
    field: "paths",
    filters: [{ operator: "startsWith", value: "/api/oz" }]
  },
  {
    field: "methods", 
    filters: [{ operator: "is", value: "POST" }]
  }
]

Query: "find POST and GET requests to api/v1"
Result: [
  {
    field: "paths",
    filters: [{ operator: "startsWith", value: "api/v1" }]
  },
  {
    field: "methods",
    filters: [
      { operator: "is", value: "POST" },
      { operator: "is", value: "GET" }
    ]
  }
]

Query: "show me all okay statuses"
Result: [
  {
    field: "status",
    filters: [{ operator: "is", value: 200 }]
  }
]

Query: "get me request with ID req_3HagbMuvTs6gtGbijeHoqbU9Cijg"
Result: [
  {
    field: "requestId",
    filters: [{ operator: "is", value: "req_3HagbMuvTs6gtGbijeHoqbU9Cijg" }]
  }
]

Query: "show 404 requests from test.example.com"
Result: [
  {
    field: "host",
    filters: [{ operator: "is", value: "test.example.com" }]
  },
  {
    field: "status",
    filters: [{ operator: "is", value: 404 }]
  }
]

Query: "find all POST requests"
Result: [
  {
    field: "methods",
    filters: [{ operator: "is", value: "POST" }]
  }
]

Remember:
${operatorsByField}`;
};
