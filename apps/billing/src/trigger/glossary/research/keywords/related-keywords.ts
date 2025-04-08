import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";

// Define schemas for task input and output
const RelatedKeywordSchema = z.object({
  keyword: z.string(),
  avg_monthly_searches: z.number(),
  competition: z.number(),
  competition_index: z.number(),
  high_top_of_page_bid_micros: z.number(),
  low_top_of_page_bid_micros: z.number(),
});

export type RelatedKeyword = z.infer<typeof RelatedKeywordSchema>;

export const RelatedKeywordsOutputSchema = z.object({
  inputTerm: z.string(),
  keywordIdeas: z.array(RelatedKeywordSchema),
});

export type RelatedKeywordsOutput = z.infer<typeof RelatedKeywordsOutputSchema>;

// Schema for the raw data from the website
const KeywordDataSchema = z.object({
  keywordIdeas: z.array(RelatedKeywordSchema),
});

/**
 * Task to scrape massiveonlinemarketing.nl for related keywords with volume, CPC, and competition data
 */
export const relatedKeywordsTask = task({
  id: "related_keywords",
  retry: {
    maxAttempts: 3,
  },
  run: async ({ inputTerm }: { inputTerm: string }): Promise<RelatedKeywordsOutput> => {
    // Validate input
    if (!inputTerm) {
      throw new AbortTaskRunError("Input term is required");
    }

    try {
      // 1. Construct URL
      const url = `https://www.massiveonlinemarketing.nl/en/tools/keyword-research/${encodeURIComponent(inputTerm)}`;

      // 2. Fetch page content
      const response = await fetch(url);

      if (!response.ok) {
        throw new AbortTaskRunError(
          `Failed to fetch data: ${response.status} ${response.statusText}`,
        );
      }

      const html = await response.text();

      // 3. Extract data from script tag
      const extraction = extractKeywordData(html);

      if (extraction.error) {
        throw new AbortTaskRunError(extraction.error.message);
      }

      const validationResult = KeywordDataSchema.safeParse(extraction.data);
      if (!validationResult.success) {
        throw new AbortTaskRunError("Invalid keyword data structure");
      }

      // 4. Return results in the original format
      return {
        inputTerm,
        keywordIdeas: validationResult.data.keywordIdeas,
      };
    } catch (error) {
      if (error instanceof AbortTaskRunError) {
        throw error;
      }
      throw new AbortTaskRunError(
        `Error scraping related keywords: ${error instanceof Error ? error.message : String(error)}`,
      );
    }
  },
});

/**
 * Extracts and parses keyword data from HTML content containing Next.js hydration scripts.
 *
 * This function specifically targets Next.js script tags containing serialized keyword data
 * from massiveonlinemarketing.nl. It performs the following steps:
 *
 * 1. Finds all script tags containing 'self.__next_f.push'
 * 2. Iterates through matches to find the one containing 'keywordData'
 * 3. Carefully extracts the JSON object using brace matching
 * 4. Unescapes and parses the JSON data
 *
 * The function includes extensive debug logging in non-production environments
 * to help diagnose extraction issues.
 *
 * @param html - Raw HTML string from the webpage
 *
 * @returns {Object} Result object
 * @returns {Object|null} result.data - Parsed keyword data if found
 * @returns {Array} [result.data.keywordIdeas] - Array of keyword ideas if present
 * @returns {Error|null} result.error - Error object if extraction fails
 *
 * @example
 * const { data, error } = extractKeywordData(htmlContent);
 * if (error) {
 *   console.error('Failed to extract keyword data:', error);
 *   return;
 * }
 * console.info('Found keyword ideas:', data.keywordIdeas);
 *
 * @remarks
 * - Uses regex to find Next.js hydration scripts
 * - Implements careful JSON extraction with brace counting
 * - Handles escaped quotes in the serialized data
 * - Includes debug logging in non-production environments
 */
function extractKeywordData(html: string) {
  // First find all script tags containing self.__next_f.push
  const scriptTagRegex = /<script>self\.__next_f\.push\(\[(.*?)\]\)<\/script>/g;
  const matches = Array.from(html.matchAll(scriptTagRegex));

  if (process.env.NODE_ENV !== "production") {
    console.info(`Found ${matches.length} __next_f.push scripts`);
  }

  for (const [index, match] of matches.entries()) {
    const content = match[1];
    if (process.env.NODE_ENV !== "production") {
      console.info(`\nChecking script ${index + 1}/${matches.length}`);
      console.info("Content:", content.slice(0, 200));
    }

    if (!content) {
      if (process.env.NODE_ENV !== "production") {
        console.info("Empty content, skipping");
      }
      continue;
    }

    if (!content.includes("keywordData")) {
      if (process.env.NODE_ENV !== "production") {
        console.info("No keywordData found, skipping");
      }
      continue;
    }

    if (process.env.NODE_ENV !== "production") {
      console.info("\nFound script containing keywordData!");
    }

    // Find the keywordData object start - handle escaped quotes
    const keywordDataStr = '\\"keywordData\\"';
    const keywordDataStart = content.indexOf(keywordDataStr);
    if (process.env.NODE_ENV !== "production") {
      console.info("keywordData position:", keywordDataStart);
    }

    if (keywordDataStart === -1) {
      if (process.env.NODE_ENV !== "production") {
        console.info("Could not find keywordData start position");
      }
      continue;
    }

    // Find the opening brace after the escaped quotes
    const objectStart = content.indexOf(":{", keywordDataStart);
    if (objectStart === -1) {
      if (process.env.NODE_ENV !== "production") {
        console.info("Could not find opening brace");
      }
      continue;
    }

    // Skip the colon to get to the actual brace
    const actualObjectStart = objectStart + 1;
    if (process.env.NODE_ENV !== "production") {
      console.info("Opening brace position:", actualObjectStart);
    }

    // Find the closing brace by counting opening/closing braces
    let braceCount = 1;
    let objectEnd = -1;

    for (let i = actualObjectStart + 1; i < content.length; i++) {
      const char = content[i];
      if (char === "{") {
        braceCount++;
      }
      if (char === "}") {
        braceCount--;
      }
      if (braceCount === 0) {
        objectEnd = i;
        break;
      }
    }

    if (objectEnd === -1) {
      if (process.env.NODE_ENV !== "production") {
        console.info("Could not find matching closing brace");
      }
      continue;
    }
    if (process.env.NODE_ENV !== "production") {
      console.info("Found closing brace at position:", objectEnd);
    }

    // Extract and parse the JSON
    const jsonStr = content.substring(actualObjectStart, objectEnd + 1);
    if (process.env.NODE_ENV !== "production") {
      console.info("Extracted JSON string:", jsonStr);
    }

    let parsed: { keywordIdeas?: Array<unknown> };
    try {
      // First unescape the JSON string
      const unescapedJson = jsonStr.replace(/\\"/g, '"');
      parsed = JSON.parse(unescapedJson);
      if (process.env.NODE_ENV !== "production") {
        console.info("Successfully parsed JSON. Found keys:", Object.keys(parsed));
      }
    } catch (error) {
      if (process.env.NODE_ENV !== "production") {
        console.info("JSON parse error:", error instanceof Error ? error.message : String(error));
      }
      continue;
    }

    if (!parsed?.keywordIdeas?.length) {
      if (process.env.NODE_ENV !== "production") {
        console.info("Missing keywordIdeas array or empty");
      }
      continue;
    }

    if (process.env.NODE_ENV !== "production") {
      console.info(`Successfully found ${parsed.keywordIdeas.length} keyword ideas`);
    }
    return { data: parsed, error: null };
  }

  if (process.env.NODE_ENV !== "production") {
    console.info("\nNo valid keyword data found in any matches");
  }
  return {
    data: null,
    error: new Error("Could not find valid keywordData in the page"),
  };
}
