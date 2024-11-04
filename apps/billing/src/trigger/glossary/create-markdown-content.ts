import { db } from "@/lib/db-marketing/client";
import { entries, sections } from "@/lib/db-marketing/schemas";
import { task } from "@trigger.dev/sdk/v3";
import { and, desc, eq, isNotNull } from "drizzle-orm";

export const createMarkdownContent = task({
  id: "create_markdown_content",
  retry: {
    maxAttempts: 0,
  },
  run: async ({ term }: { term: string }) => {
    // Fetch the entry
    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, term),
    });

    if (!entry) {
      throw new Error(`Entry not found for term: ${term}`);
    }

    // Fetch the latest sections for each order value
    const latestSections = await db
      .select()
      .from(sections)
      .where(and(eq(sections.entryId, entry.id), isNotNull(sections.markdown)))
      .orderBy(desc(sections.order))
      .limit(10);

    console.info(`Found ${latestSections.length} sections for ${term}`);

    // Compile the markdown content
    let markdownContent = `# ${term}\n\n`;

    for (const section of latestSections) {
      markdownContent += `${section.markdown || ""}\n\n`;
    }

    // store the markdown content in the database
    await db.update(entries).set({
      markdown: markdownContent,
    }).where(eq(entries.id, entry.id));

    return db.query.entries.findFirst({
      where: eq(entries.id, entry.id),
    });
  },
});
