import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import z from "zod";

const conditionTypeContext: Record<string, string> = {
  path: 'matching URL paths (e.g. /api/v1/users, /health, /graphql)',
  header: 'matching HTTP header values (e.g. "Bearer xxx", "application/json", "gzip")',
  queryParam: 'matching query parameter values (e.g. "true", "123", "admin")',
};

const regexOutputSchema = z.object({
  pattern: z.string(),
});

export async function generateRegexFromLLM(
  openai: OpenAI | null,
  query: string,
  conditionType: string,
) {
  if (!openai) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "OpenAI isn't configured correctly, please check your API key",
    });
  }

  try {
    const completion = await openai.chat.completions.parse({
      model: "gpt-4o-mini",
      temperature: 0.2, // Range 0-2, lower = more focused/deterministic
      top_p: 0.1, // Alternative to temperature, controls randomness
      frequency_penalty: 0.5, // Range -2 to 2, higher = less repetition
      presence_penalty: 0.5, // Range -2 to 2, higher = more topic diversity
      n: 1, // Number of completions to generate
      messages: [
        {
          role: "system",
          content: getSystemPrompt(conditionType),
        },
        {
          role: "user",
          content: query,
        },
      ],
      response_format: {
        type: "json_schema",
        json_schema: {
          name: "regex-pattern",
          strict: true,
          schema: z.toJSONSchema(regexOutputSchema, { target: "draft-7" }),
        },
      },
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Could not generate a regex pattern. Try phrases like:\n" +
          "• 'all API routes under /api/v2'\n" +
          "• 'paths ending in .json'\n" +
          "• 'any value starting with Bearer'",
      });
    }

    return completion.choices[0].message.parsed as z.infer<typeof regexOutputSchema>;
  } catch (error) {
    console.error(
      `Failed to generate regex. Input: ${JSON.stringify(query)}\n Error: ${(error as Error).message}`,
    );

    if (error instanceof TRPCError) {
      throw error;
    }

    if ((error as { response: { status: number } }).response?.status === 429) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Rate limit exceeded. Please try again in a few minutes.",
      });
    }

    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to generate regex pattern. Please try again.",
    });
  }
}

function getSystemPrompt(conditionType: string): string {
  const context = conditionTypeContext[conditionType] ?? "matching string values";

  return `You convert natural language descriptions into RE2-compatible regex patterns for ${context}.

Rules:
- Output ONLY valid RE2 syntax. RE2 does NOT support lookaheads (?=), lookbehinds (?<=), backreferences (\\1), or possessive quantifiers.
- Anchor patterns with ^ (start) and $ (end) when the user's intent implies full-string matching.
- Use ^ without $ when the user says "starts with" or "under".
- Use $ without ^ when the user says "ends with".
- Omit both anchors when the user says "contains".
- Keep patterns as simple as possible.
- Use character classes like [0-9] instead of \\d (RE2 supports \\d but character classes are clearer).

Examples:
- "all API routes" → ^/api/.*
- "versioned API endpoints" → ^/api/v[0-9]+/.*
- "paths ending in .json" → .*\\.json$
- "anything under /users" → ^/users/.*
- "exact path /health" → ^/health$
- "contains admin" → .*admin.*
- "bearer tokens" → ^Bearer .+
- "numeric values" → ^[0-9]+$
- "email addresses" → ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$
- "any value" → .*
- "json or xml content type" → ^application/(json|xml)`;
}
