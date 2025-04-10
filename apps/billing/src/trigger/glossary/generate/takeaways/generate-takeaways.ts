import { takeawaysSchema } from "@/lib/db-marketing/schemas/takeaways-schema";
import { google } from "@/lib/google";
import { tryCatch } from "@/lib/utils/try-catch";
import { AbortTaskRunError, metadata, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { z } from "zod";

// Field Selection Schema - Similar to Prisma's select API
export const fieldSelectionSchema = z.object({
  tldr: z.boolean().optional(),
  definitionAndStructure: z.union([z.boolean(), z.array(z.number())]).optional(),
  historicalContext: z.union([z.boolean(), z.array(z.number())]).optional(),
  usageInAPIs: z
    .union([
      z.boolean(),
      z.object({
        tags: z.boolean().optional(),
        description: z.boolean().optional(),
      }),
    ])
    .optional(),
  bestPractices: z.union([z.boolean(), z.array(z.number())]).optional(),
  recommendedReading: z.union([z.boolean(), z.array(z.number())]).optional(),
  didYouKnow: z.boolean().optional(),
});

export type FieldSelection = z.infer<typeof fieldSelectionSchema>;

// Helper function to build dynamic schema based on field selection
function buildDynamicSchema(fields?: FieldSelection) {
  if (!fields) {
    return takeawaysSchema;
  }

  const schemaShape: Record<string, z.ZodTypeAny> = {};

  if (fields.tldr) {
    schemaShape.tldr = takeawaysSchema.shape.tldr;
  }

  if (fields.definitionAndStructure) {
    if (Array.isArray(fields.definitionAndStructure)) {
      // If array of indices is provided, keep the array type but we'll filter later
      schemaShape.definitionAndStructure = takeawaysSchema.shape.definitionAndStructure;
    } else if (fields.definitionAndStructure === true) {
      schemaShape.definitionAndStructure = takeawaysSchema.shape.definitionAndStructure;
    }
  }

  if (fields.historicalContext) {
    if (Array.isArray(fields.historicalContext)) {
      schemaShape.historicalContext = takeawaysSchema.shape.historicalContext;
    } else if (fields.historicalContext === true) {
      schemaShape.historicalContext = takeawaysSchema.shape.historicalContext;
    }
  }

  if (fields.usageInAPIs) {
    if (typeof fields.usageInAPIs === "object") {
      const usageShape: Record<string, z.ZodTypeAny> = {};
      if (fields.usageInAPIs.description) {
        usageShape.description = takeawaysSchema.shape.usageInAPIs.shape.description;
      }
      if (fields.usageInAPIs.tags) {
        usageShape.tags = takeawaysSchema.shape.usageInAPIs.shape.tags;
      }
      schemaShape.usageInAPIs = z.object(usageShape);
    } else if (fields.usageInAPIs === true) {
      schemaShape.usageInAPIs = takeawaysSchema.shape.usageInAPIs;
    }
  }

  if (fields.bestPractices) {
    if (Array.isArray(fields.bestPractices)) {
      schemaShape.bestPractices = takeawaysSchema.shape.bestPractices;
    } else if (fields.bestPractices === true) {
      schemaShape.bestPractices = takeawaysSchema.shape.bestPractices;
    }
  }

  if (fields.recommendedReading) {
    if (Array.isArray(fields.recommendedReading)) {
      schemaShape.recommendedReading = takeawaysSchema.shape.recommendedReading;
    } else if (fields.recommendedReading === true) {
      schemaShape.recommendedReading = takeawaysSchema.shape.recommendedReading;
    }
  }

  if (fields.didYouKnow) {
    schemaShape.didYouKnow = takeawaysSchema.shape.didYouKnow;
  }

  return z.object(schemaShape);
}

/**
 * Task that generates takeaways for a glossary term
 */
export const generateTakeawaysTask = task({
  id: "generate_takeaways",
  retry: {
    maxAttempts: 3,
  },
  onStart: async ({ term, fields }: { term: string; fields?: FieldSelection }) => {
    metadata.replace({
      term,
      status: "running",
      startedAt: new Date().toISOString(),
      fields: fields || "all",
      progress: 0,
    });
  },
  onSuccess: async () => {
    metadata.set("status", "completed");
    metadata.set("completedAt", new Date().toISOString());
    metadata.set("progress", 1);
  },
  run: async ({ term, fields }: { term: string; fields?: FieldSelection }) => {
    if (!term) {
      throw new AbortTaskRunError("Term is required");
    }

    metadata.set("progress", 0.2);

    const system = `You are an expert technical writer specializing in API documentation.
Your task is to generate structured takeaways about "${term}" for API developers.
Focus on providing valuable, accurate, and practical information.
Only generate the specifically requested sections.`;

    // Modify the prompt to be more explicit about field selection
    const prompt = `Generate structured takeaways about "${term}" for API developers.

${fields ? "Generate ONLY the following sections:" : "Generate all sections:"}
${fields ? JSON.stringify(fields, null, 2) : "All sections"}

The response should follow this structure (but only include requested fields):
${Object.keys(fields || {})
  .map((field) => `- ${field}`)
  .join("\n")}

Guidelines:
1. Be technically accurate and precise
2. Focus on practical application in APIs
3. Include specific examples where relevant
4. Keep the TLDR concise but informative
5. Ensure best practices are actionable
6. Include high-quality recommended reading sources
7. ONLY generate the requested sections, omit all others`;

    const dynamicSchema = buildDynamicSchema(fields);

    const { data, error } = await tryCatch(
      generateObject({
        model: google("gemini-2.0-flash-lite-preview-02-05") as any,
        schema: dynamicSchema,
        prompt,
        system,
        experimental_telemetry: {
          isEnabled: true,
          functionId: "generate_takeaways",
        },
      }),
    );

    if (error) {
      metadata.set("status", "failed");
      metadata.set("error", typeof error === "object" ? JSON.stringify(error) : String(error));
      metadata.set("completedAt", new Date().toISOString());
      throw new AbortTaskRunError(`Failed to generate takeaways for term: ${term}`);
    }

    if (data.usage) {
      metadata.set("tokenUsage", {
        total: data.usage.totalTokens,
        prompt: data.usage.promptTokens,
        completion: data.usage.completionTokens,
      });
    }

    // Post-process arrays if specific indices were requested
    const processedObject = data.object;
    if (fields) {
      Object.entries(fields).forEach(([key, value]) => {
        if (Array.isArray(value) && Array.isArray(processedObject[key])) {
          processedObject[key] = value.map((index) => processedObject[key][index]).filter(Boolean);
        }
      });
    }

    return {
      term,
      takeaways: processedObject,
      fields,
    };
  },
});
