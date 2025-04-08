import { batch, task } from "@trigger.dev/sdk/v3";
import Exa from "exa-js";
import { evaluateSearchResults } from "./evaluate-search-results";
import { domainCategories, exaDomainSearchTask } from "./exa-domain-search";
import type { ExaCosts } from "./types";

export const technicalResearchTask = task({
  id: "technical_research",
  run: async ({
    inputTerm,
  }: {
    inputTerm: string;
  }) => {
    console.info("Starting domain research:", {
      query: inputTerm,
    });

    // we perform a search for each search category in parallel:
    const { runs } = await batch.triggerByTaskAndWait(
      domainCategories.map((domainCategory) => ({
        task: exaDomainSearchTask,
        payload: {
          inputTerm,
          numResults: 10,
          domain: domainCategory.name,
        },
      })),
    );
    const failedResults = runs.filter((result) => !result.ok).map((result) => result.error);
    if (failedResults.length > 0) {
      console.warn("⚠️ Failed to run some search categories:", failedResults);
    }
    // Filter out failed searches and combine results
    const searchResults = runs.filter((result) => result.ok).flatMap((result) => result.output);

    // log the costs for the exa responses:
    const searchCosts = searchResults.flatMap((result) => ({
      ...result.costDollars,
      category: result.category,
    }));
    console.info(`💰 Exa API costs for initial search:
      Total: $${searchCosts.reduce((acc, cost) => acc + cost.total, 0)}
      Search: $${searchCosts.reduce(
        (acc, cost) => acc + (cost.search?.neural || cost.search?.keyword || 0),
        0,
      )} | ${searchCosts.length} requests made @ $0.0025/request | should result in $${
        searchCosts.length * 0.0025
      }
      Summaries: $${searchCosts.reduce(
        (acc, cost) => acc + (cost.contents?.summary || 0),
        0,
      )} | ${searchResults.reduce(
        (acc, result) => acc + result.results.length,
        0,
      )} summaries @ $0.001/summary | should result in $${
        searchResults.reduce((acc, result) => acc + result.results.length, 0) * 0.001
      }
    `);

    // process our results for the evaluation step (flatten & dedupe)
    const results = searchResults.flatMap((searchResult) =>
      searchResult.results.map((result) => ({
        ...result,
      })),
    );
    // dedupe the results based on `url`:
    const dedupedResults = results.filter(
      (result, index, self) => index === self.findIndex((t) => t.url === result.url),
    );

    // Step 2: Evaluate the search results
    const evaluationRun = await evaluateSearchResults.triggerAndWait({
      searchResults: dedupedResults,
      inputTerm,
    });

    if (!evaluationRun.ok) {
      throw new Error("Failed to evaluate search results");
    }

    const evaluationResults = evaluationRun.output;
    console.info(`💰 Evaluation costs:
      Total: $${evaluationResults.costs.total}
      Input: $${evaluationResults.costs.input}
      Output: $${evaluationResults.costs.output}
    `);

    // Step 3: Scrape the content of the results
    const exa = new Exa(process.env.EXA_API_KEY || "");
    const contentResults = await exa.getContents(
      evaluationResults.included.flatMap((result) => result.url),
    );

    // log the costs for the exa responses:
    const scrapingCosts = (contentResults as unknown as typeof contentResults & ExaCosts)
      .costDollars;
    console.info(`💰 Exa API costs for Content Scraping:
      Total: $${scrapingCosts.total}
      Summaries: $${scrapingCosts.contents?.text} texts @ $0.001/text
    `);

    return {
      summary: evaluationResults.evaluationSummary,
      included: contentResults,
      costs: {
        total:
          scrapingCosts.total +
          evaluationResults.costs.total +
          searchCosts.reduce((acc, cost) => acc + cost.total, 0),
        search: {
          search: searchCosts.reduce(
            (acc, cost) => acc + (cost.search?.neural || cost.search?.keyword || 0),
            0,
          ),
          summary: searchCosts.reduce((acc, cost) => acc + (cost.contents?.summary || 0), 0),
        },
        evaluation: {
          input: evaluationResults.costs.input,
          output: evaluationResults.costs.output,
        },
        contents: {
          text: scrapingCosts.contents?.text,
        },
      },
    };
  },
});
