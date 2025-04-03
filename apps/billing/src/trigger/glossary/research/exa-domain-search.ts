import type { SelectEntry } from "@/lib/db-marketing/schemas";
import { task } from "@trigger.dev/sdk/v3";
import Exa, { type ContentsOptions, type RegularSearchOptions } from "exa-js";
import type { ExaCosts } from "./types";

export const domainCategories = [
  {
    name: "Official",
    domains: ["tools.ietf.org", "datatracker.ietf.org", "rfc-editor.org", "w3.org", "iso.org"],
    description: "Official standards and specifications sources",
  },
  {
    name: "Community",
    domains: [
      "stackoverflow.com",
      "github.com",
      "wikipedia.org",
      "news.ycombinator.com",
      "stackexchange.com",
    ],
    description: "Community-driven platforms and forums",
  },
  {
    name: "Neutral",
    domains: ["owasp.org", "developer.mozilla.org"],
    description: "Educational and vendor-neutral resources",
  },
  {
    name: "Google",
    domains: [], // Empty domains array to search without domain restrictions
    description: "General search results without domain restrictions",
  },
] as const;

// Define the main search task
export const exaDomainSearchTask = task({
  id: "exa_domain_search",
  run: async ({
    inputTerm,
    numResults = 10,
    domain,
  }: {
    inputTerm: SelectEntry["inputTerm"];
    numResults?: number;
    domain: (typeof domainCategories)[number]["name"];
  }) => {
    const apiKey = process.env.EXA_API_KEY;
    if (!apiKey) {
      throw new Error("EXA_API_KEY environment variable is not set");
    }
    const exa = new Exa(apiKey);
    const domainCategory = domainCategories.find((c) => c.name === domain);

    // Initial search with only summaries
    const searchOptions = {
      numResults,
      type: "keyword",
      // we only include summary (not text) so that we fetch the content for the results after the Gemini evaluation
      summary: {
        query: "Exhaustive summary what the web page is about",
      },
      // we unpack the array in a new array because out domainCategories returns `readonly`
      includeDomains: [...(domainCategories.find((c) => c.name === domain)?.domains || [])],
    } satisfies RegularSearchOptions & ContentsOptions;

    console.info("🔍 Starting Exa search with summaries only:", {
      query: inputTerm,
      category: domainCategory?.name,
    });
    const searchResult = await exa.searchAndContents(inputTerm, searchOptions);

    // add our domain category to the search result:

    // we cast the `ExaCosts` type as the `exa-js` types don't contain the costDollars
    const searchResultWithCategory = searchResult as unknown as typeof searchResult &
      ExaCosts & { category: typeof domainCategory };
    searchResultWithCategory.category = domainCategory;
    return searchResultWithCategory;
  },
});
