import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { z } from "zod";

// Define schemas for task input and output
const RelatedKeywordSchema = z.object({
  keyword: z.string(),
  avg_monthly_searches: z.number(),
  competition: z.number(),
  competition_index: z.number(),
  high_top_of_page_bid_micros: z.number(),
  low_top_of_page_bid_micros: z.number()
});

export type RelatedKeyword = z.infer<typeof RelatedKeywordSchema>;

export const RelatedKeywordsOutputSchema = z.object({
  inputTerm: z.string(),
  keywordIdeas: z.array(RelatedKeywordSchema)
});

export type RelatedKeywordsOutput = z.infer<typeof RelatedKeywordsOutputSchema>;

// Schema for the raw data from the website
const KeywordDataSchema = z.object({
  keywordIdeas: z.array(RelatedKeywordSchema)
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
        keywordIdeas: validationResult.data.keywordIdeas
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
  // First find all script tags containing self.__next_f.push
  const scriptTagRegex = /<script>self\.__next_f\.push\(\[(.*?)\]\)<\/script>/g;
  const matches = Array.from(html.matchAll(scriptTagRegex));
  
  console.log(`Found ${matches.length} __next_f.push scripts`);
  
  for (const [index, match] of matches.entries()) {
    const content = match[1];
    console.log(`\nChecking script ${index + 1}/${matches.length}`);
    console.log("Content:", content.slice(0, 200));
    
    if (!content) {
      console.log("Empty content, skipping");
      continue;
    }

    if (!content.includes('keywordData')) {
      console.log('No keywordData found, skipping');
      continue;
    }

    console.log("\nFound script containing keywordData!");
    
    // Find the keywordData object start - handle escaped quotes
    const keywordDataStr = '\\"keywordData\\"';
    const keywordDataStart = content.indexOf(keywordDataStr);
    console.log("keywordData position:", keywordDataStart);
    
    if (keywordDataStart === -1) {
      console.log("Could not find keywordData start position");
      continue;
    }

    // Find the opening brace after the escaped quotes
    const objectStart = content.indexOf(':{', keywordDataStart);
    if (objectStart === -1) {
      console.log("Could not find opening brace");
      continue;
    }

    // Skip the colon to get to the actual brace
    const actualObjectStart = objectStart + 1;
    console.log("Opening brace position:", actualObjectStart);

    // Find the closing brace by counting opening/closing braces
    let braceCount = 1;
    let objectEnd = -1;

    for (let i = actualObjectStart + 1; i < content.length; i++) {
      const char = content[i];
      if (char === '{') {
        braceCount++;
      }
      if (char === '}') {
        braceCount--;
      }
      if (braceCount === 0) {
        objectEnd = i;
        break;
      }
    }

    if (objectEnd === -1) {
      console.log("Could not find matching closing brace");
      continue;
    }
    console.log("Found closing brace at position:", objectEnd);

    // Extract and parse the JSON
    const jsonStr = content.substring(actualObjectStart, objectEnd + 1);
    console.log("Extracted JSON string:", jsonStr);

    let parsed: { keywordIdeas?: Array<unknown> };
    try {
      // First unescape the JSON string
      const unescapedJson = jsonStr.replace(/\\"/g, '"');
      parsed = JSON.parse(unescapedJson);
      console.log("Successfully parsed JSON. Found keys:", Object.keys(parsed));
    } catch (error) {
      console.log("JSON parse error:", error instanceof Error ? error.message : String(error));
      continue;
    }

    if (!parsed?.keywordIdeas?.length) {
      console.log("Missing keywordIdeas array or empty");
      continue;
    }

    console.log(`Successfully found ${parsed.keywordIdeas.length} keyword ideas`);
    return { data: parsed, error: null };
  }

  console.log("\nNo valid keyword data found in any matches");
  return { 
    data: null, 
    error: new Error("Could not find valid keywordData in the page") 
  };
}
