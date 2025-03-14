import { google } from "@/lib/google";
import { AbortTaskRunError, metadata, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { z } from "zod";
import type { CacheStrategy } from "../../_generate-glossary-entry";
import { technicalResearchTask } from "../../research/_technical-research";

// Define Zod schemas for the content generation and review
const contentGenerationSchema = z.object({
  markdown: z.string().min(1),
  reasoning: z.string().optional(),
  sources_used: z.array(z.string()).optional(),
});

const contentReviewSchema = z.object({
  markdown: z.string().min(1),
  rating: z.number().min(0).max(10),
  improvements: z.array(z.string()).optional(),
  reasoning: z.string().optional(),
});

/**
 * Task that generates markdown content for a glossary entry based on technical research
 */
export const generateContentTask = task({
  id: "generate_content",
  retry: {
    maxAttempts: 3,
  },
  onStart: async ({ inputTerm }: { inputTerm: string; onCacheHit?: CacheStrategy }) => {
    // Initialize metadata
    metadata.replace({
      term: inputTerm,
      status: "running",
      startedAt: new Date().toISOString(),
      steps: {
        research: { status: "pending" },
        generation: { status: "pending" },
        review: { status: "pending" },
      },
      progress: 0,
    });
  },
  onSuccess: async () => {
    // Update metadata for successful completion
    metadata.set("status", "completed");
    metadata.set("completedAt", new Date().toISOString());
    metadata.set("progress", 1);
  },
  run: async ({
    inputTerm,
  }: {
    inputTerm: string;
    onCacheHit?: CacheStrategy;
  }) => {
    // Update metadata to show we're starting the research step
    metadata.set("currentStep", "research");
    metadata.set("progress", 0.1);
    metadata.set("steps", {
      research: { status: "running" },
      generation: { status: "pending" },
      review: { status: "pending" },
    });

    // Get technical research results
    const technicalResearch = await technicalResearchTask.triggerAndWait({ inputTerm });
    if (!technicalResearch.ok) {
      // Update metadata for failure
      metadata.set("status", "failed");
      metadata.set("error", `Technical research failed for term: ${inputTerm}`);
      metadata.set("completedAt", new Date().toISOString());
      metadata.set("steps", {
        research: { status: "failed" },
        generation: { status: "pending" },
        review: { status: "pending" },
      });

      throw new AbortTaskRunError(`Technical research failed for term: ${inputTerm}`);
    }

    // Log the structure of the research results
    console.info(`Research results structure: ${typeof technicalResearch.output}`);
    console.info(`Research results keys: ${Object.keys(technicalResearch.output).join(", ")}`);

    // Update metadata for successful research
    metadata.set("steps", {
      research: { status: "completed" },
      generation: { status: "pending" },
      review: { status: "pending" },
    });
    metadata.set("progress", 0.3);

    // Update metadata to show we're starting the generation step
    metadata.set("currentStep", "generation");
    metadata.set("progress", 0.4);
    metadata.set("steps", {
      research: { status: "completed" },
      generation: { status: "running" },
      review: { status: "pending" },
    });

    // Generate content using AI
    const contentResult = await generateContentFromResearchTask.triggerAndWait({
      term: inputTerm,
      researchResults: technicalResearch.output,
    });

    if (!contentResult.ok) {
      // Update metadata for failure
      metadata.set("status", "failed");
      metadata.set("error", `Content generation failed for term: ${inputTerm}`);
      metadata.set("completedAt", new Date().toISOString());
      metadata.set("steps", {
        research: { status: "completed" },
        generation: { status: "failed" },
        review: { status: "pending" },
      });

      throw new AbortTaskRunError(`Content generation failed for term: ${inputTerm}`);
    }

    console.info(`Generated content for ${inputTerm}`);

    // Update metadata for successful generation
    metadata.set("steps", {
      research: { status: "completed" },
      generation: { status: "completed" },
      review: { status: "pending" },
    });
    metadata.set("progress", 0.7);

    // Update metadata to show we're starting the review step
    metadata.set("currentStep", "review");
    metadata.set("progress", 0.8);
    metadata.set("steps", {
      research: { status: "completed" },
      generation: { status: "completed" },
      review: { status: "running" },
    });

    // Review and improve the content
    const reviewResult = await reviewContentTask.triggerAndWait({
      term: inputTerm,
      content: contentResult.output.markdown,
    });

    if (!reviewResult.ok) {
      // Update metadata for failure
      metadata.set("status", "failed");
      metadata.set("error", `Content review failed for term: ${inputTerm}`);
      metadata.set("completedAt", new Date().toISOString());
      metadata.set("steps", {
        research: { status: "completed" },
        generation: { status: "completed" },
        review: { status: "failed" },
      });

      throw new AbortTaskRunError(`Content review failed for term: ${inputTerm}`);
    }

    console.info(`Reviewed content for ${inputTerm} with rating: ${reviewResult.output.rating}/10`);

    // Update steps and rating before onSuccess is called
    metadata.set("steps", {
      research: { status: "completed" },
      generation: { status: "completed" },
      review: { status: "completed" },
    });
    metadata.set("rating", reviewResult.output.rating);

    return {
      term: inputTerm,
      content: reviewResult.output.markdown,
      rating: reviewResult.output.rating,
      improvements: reviewResult.output.improvements || [],
      reasoning: reviewResult.output.reasoning,
    };
  },
});

/**
 * Task that generates content from research results using AI
 */
export const generateContentFromResearchTask = task({
  id: "generate_content_from_research",
  onStart: async ({ term }: { term: string; researchResults: any }) => {
    // Initialize metadata
    metadata.replace({
      term,
      status: "running",
      startedAt: new Date().toISOString(),
      sourcesCount: 0,
      progress: 0,
    });
  },
  onSuccess: async () => {
    // Update metadata for successful completion
    metadata.set("status", "completed");
    metadata.set("completedAt", new Date().toISOString());
    metadata.set("progress", 1);
  },
  run: async ({
    term,
    researchResults,
  }: {
    term: string;
    researchResults: any; // Using any for now, should be properly typed
  }) => {
    // Extract relevant information from research results
    // Log the structure for debugging
    console.info(`Research results type: ${typeof researchResults}`);
    if (typeof researchResults === "object" && researchResults !== null) {
      console.info(`Research results keys: ${Object.keys(researchResults).join(", ")}`);
    }

    // Try different possible structures to find the sources
    let sources: any[] = [];

    if (Array.isArray(researchResults)) {
      // If researchResults is directly an array
      sources = researchResults;
    } else if (researchResults && typeof researchResults === "object") {
      // Check for common properties that might contain the sources
      if (Array.isArray(researchResults.included)) {
        sources = researchResults.included;
      } else if (researchResults.included && Array.isArray(researchResults.included.results)) {
        // Handle the case where included is an object with a results array
        sources = researchResults.included.results;
      } else if (Array.isArray(researchResults.summary?.included)) {
        sources = researchResults.summary.included;
      } else if (Array.isArray(researchResults.results)) {
        sources = researchResults.results;
      } else {
        // Look for any array property that might contain sources
        const findSourcesArray = (obj: any, depth = 0, maxDepth = 3): any[] | null => {
          if (depth > maxDepth || !obj || typeof obj !== "object") {
            return null;
          }

          // Check if this object has properties that look like sources
          for (const key in obj) {
            if (Array.isArray(obj[key])) {
              // Check if this array contains objects with url or text properties
              const arr = obj[key];
              if (arr.length > 0 && typeof arr[0] === "object" && arr[0] !== null) {
                const firstItem = arr[0];
                if (firstItem.url || firstItem.text || firstItem.summary) {
                  console.info(`Found sources array in property: ${key}`);
                  return arr;
                }
              }
            } else if (typeof obj[key] === "object" && obj[key] !== null) {
              // Recursively search nested objects
              const result = findSourcesArray(obj[key], depth + 1, maxDepth);
              if (result !== null) {
                return result;
              }
            }
          }

          return null;
        };

        const foundSources = findSourcesArray(researchResults);
        if (foundSources) {
          sources = foundSources;
        }
      }
    }

    // Make sure we're getting an array of sources
    sources = Array.isArray(sources) ? sources : [];

    // Log the structure for debugging
    console.info(
      `Sources structure: ${typeof sources}, isArray: ${Array.isArray(sources)}, length: ${
        sources.length
      }`,
    );

    // If no sources are found, throw an error instead of using a fallback
    if (sources.length === 0) {
      console.error(`No sources found in research results for term: ${term}`);
      console.info(
        "Research results structure:",
        `${JSON.stringify(researchResults, null, 2).substring(0, 500)}...`,
      );

      // Update metadata for failure
      metadata.set("status", "failed");
      metadata.set("error", `No sources found for term: ${term}`);
      metadata.set("completedAt", new Date().toISOString());

      throw new AbortTaskRunError(
        `No sources found for term: ${term}. Cannot generate content without research data.`,
      );
    }

    // Update metadata with source information
    metadata.set("sourcesCount", sources.length);
    metadata.set("progress", 0.3);

    const system = `You are an expert technical writer specializing in API development documentation. 
    Your task is to create comprehensive, accurate, and well-structured content for a glossary entry on "${term}".
    Focus on providing valuable information for API developers (primarily backend developers).`;

    // Format sources for the prompt
    const sourcesText = sources
      .map((source: any, index: number) => {
        if (!source) {
          return `Source ${index + 1}: N/A`;
        }
        return `
Source ${index + 1}:
URL: ${source.url || "N/A"}
Content: ${source.text || source.summary || "N/A"}
`;
      })
      .join("\n");

    const prompt = `
    Create a comprehensive markdown article about "${term}" for API developers.
    
    Here are the research sources to inform your writing:
    ${sourcesText}
    
    Guidelines:
    1. Write in markdown format.
    2. Start with a clear, concise introduction that explains what "${term}" is and why it's important for API developers.
    3. Structure the content logically with appropriate headings (use ## for main sections).
    4. Include practical examples, code snippets, or diagrams where appropriate.
    5. When including code snippets, use TypeScript syntax and ESM (import/export, not require).
    6. Focus on technical accuracy and practical application.
    7. Include best practices and common pitfalls.
    8. Make the content valuable for backend developers working with APIs.
    9. Avoid fluff - every sentence should provide concrete information.
    10. Don't include a title at the top - that will be added separately.
    
    The content should be comprehensive but focused, aiming to be the definitive resource on "${term}" for API developers.
    
    In your response, include:
    - markdown: The complete markdown content
    - reasoning: Brief explanation of your approach and key decisions
    - sources_used: List of source URLs you found most valuable
    `;

    // Update metadata to show we're generating content
    metadata.set("progress", 0.5);
    metadata.set("status", "generating");

    try {
      const result = await generateObject({
        model: google("gemini-2.0-flash-lite-preview-02-05") as any,
        schema: contentGenerationSchema,
        prompt,
        system,
        experimental_telemetry: {
          isEnabled: true,
          functionId: "generate_content_from_research",
        },
      });

      // Update metadata with token usage if available
      if (result.usage) {
        metadata.set("tokenUsage", {
          total: result.usage.totalTokens,
          prompt: result.usage.promptTokens,
          completion: result.usage.completionTokens,
        });
      }

      // Update content-specific metadata before onSuccess is called
      metadata.set("contentLength", result.object.markdown.length);
      metadata.set("sourcesUsed", result.object.sources_used?.length || 0);

      return result.object;
    } catch (error) {
      console.error("Error generating content:", error);

      // Update metadata for failure
      metadata.set("status", "failed");
      metadata.set("error", typeof error === "object" ? JSON.stringify(error) : String(error));
      metadata.set("completedAt", new Date().toISOString());

      // Fallback with minimal response if generation fails
      return {
        markdown: `## ${term}\n\nContent generation failed. Please try again later.`,
        reasoning: "Generation failed due to an error",
        sources_used: [],
      };
    }
  },
});

/**
 * Task that reviews and improves the generated content
 */
export const reviewContentTask = task({
  id: "review_content",
  onStart: async ({ term, content }: { term: string; content: string }) => {
    // Initialize metadata
    metadata.replace({
      term,
      status: "running",
      startedAt: new Date().toISOString(),
      contentLength: content.length,
      progress: 0,
    });
  },
  onSuccess: async () => {
    // Update metadata for successful completion
    metadata.set("status", "completed");
    metadata.set("completedAt", new Date().toISOString());
    metadata.set("progress", 1);
  },
  run: async ({
    term,
    content,
  }: {
    term: string;
    content: string;
  }) => {
    const system = `You are a senior technical editor with expertise in API development documentation.
    Your task is to review, rate, and improve the content for a glossary entry on "${term}".`;

    const prompt = `
    Review and improve the following content for a glossary entry on "${term}":
    
    ${content}
    
    Guidelines for your review:
    1. Ensure technical accuracy and clarity.
    2. Improve the structure and flow if needed.
    3. Check that code examples follow best practices and use TypeScript syntax with ESM.
    4. Ensure the content is valuable for API developers (primarily backend developers).
    5. Remove any fluff or redundant information.
    6. Fix any grammatical or stylistic issues.
    7. Ensure the content is comprehensive but focused.
    
    In your response, include:
    - markdown: The improved content in markdown format
    - rating: A rating from 0-10 on the quality of the original content (10 being excellent)
    - improvements: A list of specific improvements you made
    - reasoning: Brief explanation of your rating and key improvements
    `;

    // Update metadata to show we're reviewing content
    metadata.set("progress", 0.5);
    metadata.set("status", "reviewing");

    try {
      const result = await generateObject({
        model: google("gemini-2.0-flash-lite-preview-02-05") as any,
        schema: contentReviewSchema,
        prompt,
        system,
        experimental_telemetry: {
          isEnabled: true,
          functionId: "review_content",
        },
      });

      // Update metadata with token usage if available
      if (result.usage) {
        metadata.set("tokenUsage", {
          total: result.usage.totalTokens,
          prompt: result.usage.promptTokens,
          completion: result.usage.completionTokens,
        });
      }

      // Update review-specific metadata before onSuccess is called
      metadata.set("rating", result.object.rating);
      metadata.set("improvementsCount", result.object.improvements?.length || 0);
      metadata.set("revisedContentLength", result.object.markdown.length);

      return result.object;
    } catch (error) {
      console.error("Error reviewing content:", error);

      // Update metadata for failure
      metadata.set("status", "failed");
      metadata.set("error", typeof error === "object" ? JSON.stringify(error) : String(error));
      metadata.set("completedAt", new Date().toISOString());

      // Fallback with original content if review fails
      return {
        markdown: content,
        rating: 5,
        improvements: ["Review failed due to an error"],
        reasoning: "Review process failed due to an error",
      };
    }
  },
});
