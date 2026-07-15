import { defineConfig } from "blume";

export default defineConfig({
  title: "Unkey Design",
  description: "Design system guidance for building consistent Unkey interfaces.",
  content: {
    root: "docs",
  },
  examples: "examples",
  logo: {
    image: "/unkey-logo.svg",
    text: "Design",
  },
  // Blume 0.6.7 can serialize empty UI defaults when i18n is omitted, which
  // leaves sidebar action labels blank. Keep the English strings explicit.
  i18n: {
    defaultLocale: "en",
    hideDefaultLocalePrefix: true,
    locales: [{ code: "en", label: "English" }],
    ui: {
      en: {
        actions: {
          addToCursor: "Add to Cursor",
          addToVscode: "Add to VS Code",
          askAI: "Ask AI about this page",
          connectMcp: "Connect to MCP",
          copied: "Copied!",
          copyClaudeCode: "Copy Claude Code command",
          copyCodex: "Copy Codex command",
          copyMarkdown: "Copy as Markdown",
          copyServerUrl: "Copy server URL",
          edit: "Edit on GitHub",
          openInChat: "Open in chat",
          scrollToTop: "Scroll to top",
        },
        ask: {
          clear: "Clear conversation",
          close: "Close",
          copy: "Copy conversation",
          empty: "Ask a question about the docs.",
          error: "Sorry, something went wrong.",
          label: "Ask a question",
          placeholder: "Ask a question...",
          send: "Send",
          tip: "Tip: You can open and close chat with",
          title: "Ask AI",
        },
        feedback: {
          no: "No",
          question: "Was this page helpful?",
          thanks: "Thanks for your feedback!",
          yes: "Yes",
        },
        languageSwitcher: {
          label: "Language",
          untranslated: "Not translated",
        },
        notFound: {
          description: "We couldn't find the page you're looking for.",
          home: "Back to home",
          title: "Page not found",
        },
        page: {
          lastUpdated: "Last updated on",
          next: "Next",
          previous: "Previous",
          skipToContent: "Skip to content",
        },
        search: {
          allLanguages: "All languages",
          button: "Search",
          devOnly: "Search is available in the production build.",
          label: "Search docs",
          noResults: "No results found.",
          placeholder: "Search documentation...",
        },
        toc: {
          title: "On this page",
        },
      },
    },
  },
  theme: {
    accent: "blue",
    radius: "md",
    mode: "system",
    fonts: {
      display: "geist",
      body: "geist",
      mono: "geist-mono",
    },
  },
  navigation: {
    sidebar: {
      display: "flat",
    },
  },
  search: {
    provider: "orama",
  },
  markdown: {
    code: {
      icons: true,
      wrap: false,
    },
  },
  ai: {
    llmsTxt: true,
  },
  seo: {
    og: {
      enabled: false,
    },
  },
});
