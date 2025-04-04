import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";

// Define schemas for task input and output
const RelatedKeywordSchema = z.object({
  keyword: z.string(),
  volume: z.number(),
  cpc: z.string(),
  competition: z.number(),
});

export type RelatedKeyword = z.infer<typeof RelatedKeywordSchema>;

export const RelatedKeywordsOutputSchema = z.object({
  inputTerm: z.string(),
  keywords: z.array(RelatedKeywordSchema),
});

export type RelatedKeywordsOutput = z.infer<typeof RelatedKeywordsOutputSchema>;

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

      // 3. Extract data from script tag as `any`
      const extraction = extractKeywordData(html);

      if (extraction.error) {
        throw new AbortTaskRunError(
          `Failed to extract keyword data: ${extraction.error.message}`,
        );
      }

      const keywordData = extraction.data;

      if (keywordData.length === 0) {
        throw new AbortTaskRunError(`No keyword data found for term: ${inputTerm}`);
      }

      // After extracting data
      const validationResult = z.array(RelatedKeywordSchema).safeParse(keywordData);
      if (!validationResult.success) {
        throw new AbortTaskRunError(`Invalid keyword data format: ${validationResult.error.message}`);
      }

      // 5. Return results
      return {
        inputTerm,
        keywords: keywordData,
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
 * Extract keyword data from HTML content
 * This finds the script tag with keywordData and parses the JSON
 */
function extractKeywordData(html: string) {
  try {
    // Look for script tag with self.__next_f.push and keywordData
    const scriptRegex = /self\.__next_f\.push\(\[([^\[\]]+keywordData[^\[\]]+)\]\)/g;
    const matches = html.matchAll(scriptRegex);

    let keywordDataMatch = null;
    for (const match of matches) {
      if (match[1]?.includes("keywordData")) {
        keywordDataMatch = match[1];
        break;
      }
    }

    if (!keywordDataMatch) {
      return { data: null, error: new Error("Could not find keywordData in the page") };
    }

    // Extract and parse the JSON data
    // This is a simplified version - actual implementation will need to properly extract the JSON
    // from the JavaScript code in the script tag
    const jsonStartIndex = keywordDataMatch.indexOf("{");
    const jsonEndIndex = keywordDataMatch.lastIndexOf("}") + 1;

    if (jsonStartIndex === -1 || jsonEndIndex === -1) {
      return { data: null, error: new Error("Could not locate JSON data in the script") };
    }

    const jsonStr = keywordDataMatch.substring(jsonStartIndex, jsonEndIndex);

    // Parse the JSON
    // Note: In the actual implementation, we may need to unescape characters
    // and handle other transformations to get valid JSON
    const data = JSON.parse(jsonStr);

    if (!data.keywords) {
      return { data: null, error: new Error("Could not find keywords in the data") };
    }

    // TODO: Should this be a string? RIght now it's any
    const keywords = data.keywords; 

    return { data: keywords, error: null };
  } catch (error) {
    return { data: null, error: new Error(
        `Failed to extract keyword data: ${error instanceof Error ? error.message : String(error)}`,
      ),
    };
  }
}
