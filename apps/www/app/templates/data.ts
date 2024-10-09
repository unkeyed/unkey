type StrArrayToUnion<TArr extends readonly string[]> = TArr[number];

// sort these alphabetically
export const frameworks = [
  "Django",
  "Next.js",
  "Svelte",
  "Express",
  "Bun",
  "Echo",
  "Flask",
  "Django",
  "Axum",
] as const;
export type Framework = StrArrayToUnion<typeof frameworks>;
// id -> label
export const languages = ["Typescript", "Python", "Golang", "Rust", "Elixir"] as const;
export type Language = StrArrayToUnion<typeof languages>;

export type Template = {
  title: string;
  description: string;

  /**
   * URL to the product or website
   */
  url?: string;
  /**
   * GitHub username or similar
   */
  authors: string[];

  /**
   * Url to the repository
   */
  repository: string;

  /**
   * Url to the image
   */
  image: string;

  /**
   * Url to the raw readme
   */
  readmeUrl: string;

  language: Language;
  framework?: Framework;
};

export const templates: Record<string, Template> = {
  "python-django": {
    title: "Django endpoint protection with Unkey",
    description: "Django application implementing API key-verification with Unkey RBAC.",
    authors: ["Ionfinisher"],
    repository: "https://github.com/Ionfinisher/unkey-django-template",
    image: "/images/templates/python-django.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Ionfinisher/unkey-django-template/refs/heads/main/README.md",
    language: "Python",
    framework: "Django",
  },
  "rust-axum": {
    title: "Secure your Rust Axum API",
    description: "A Rust API service using Unkey for API key validation with the Axum framwork.",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-rust-axum",
    image: "/images/templates/rust-axum.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-rust-axum/refs/heads/main/README.md",
    language: "Rust",
    framework: "Axum",
  },
  "cost-ratelimit": {
    title: "Cost based Ratelimiting",
    description: "Ratelimit your AI application based on estimated cost",
    authors: ["hashsbhat"],
    repository: "https://github.com/harshsbhat/ordox",
    image: "/images/templates/cost-ratelimit.png",
    readmeUrl: "https://raw.githubusercontent.com/harshsbhat/ordox/refs/heads/main/README.md",
    url: "https://ordox.vercel.app",
    language: "Typescript",
    framework: "Next.js",
  },
  "pdf-view": {
    title: "Protecting Digital Content Access",
    description:
      "Leverage Unkey’s short-lived keys to grant temporary access to digital content (e.g., e-books, videos, or streams), expiring after a set duration.",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-pdf-view",
    image: "/images/templates/pdf-view.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-pdf-view/refs/heads/main/README.md",
    language: "Typescript",
    framework: "Next.js",
  },
  "flask-rbac": {
    title: "Flask middleware with RBAC",
    description: "Protect your Flask API with Unkey",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-flask",
    image: "/images/templates/flask-rbac.png",
    readmeUrl: "https://raw.githubusercontent.com/harshsbhat/unkey-flask/refs/heads/main/README.md",
    language: "Python",
    framework: "Flask",
  },
  "unkey-trpc-ratelimit": {
    title: "Unkey ratelimiting with TRPC + Drizzle",
    description: "Quickstart using tRPC, Drizzle and Unkey Ratelimiting",
    authors: ["Michael"],
    repository: "https://github.com/unkeyed/examples/tree/main/unkey-ratelimit-trpc",
    image: "/images/templates/unkey-ratelimit-trpc.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unkeyed/examples/main/unkey-ratelimit-trpc/README.md",
    language: "Typescript",
    framework: "Next.js",
  },
  "echo-middleware": {
    title: "Middleware for golang's Echo framework",
    description: "Add API key authentication to your Echo API routes",
    authors: ["rithulkamesh"],
    repository: "https://github.com/rithulkamesh/unkey-echo",
    image: "/images/templates/go-echo.png",
    readmeUrl: "https://raw.githubusercontent.com/rithulkamesh/unkey-echo/main/README.md",
    language: "Golang",
    framework: "Echo",
  },
  "docs-with-keys": {
    title: "Documentation with embedded api keys",
    description: "Embed api keys in your documentation for easy copy-pasting",
    authors: ["chronark"],
    repository: "https://github.com/unkeyed/examples/tree/main/docs-with-real-keys",
    image: "/images/templates/docs-with-real-keys.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unkeyed/examples/main/docs-with-real-keys/README.md",
    language: "Typescript",
    framework: "Next.js",
    url: "https://docs-with-keys.vercel.app",
  },
  "license-keys-nextjs": {
    title: "License keys for selfhosting",
    description: "Protect your Next.js routes with license keys at runtime",
    authors: ["chronark"],
    repository: "https://github.com/unkeyed/examples/tree/main/license-keys/with-nextjs",
    image: "/images/templates/license-keys-nextjs.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unkeyed/examples/main/license-keys/with-nextjs/README.md",
    language: "Typescript",
    framework: "Next.js",
  },
  "express-with-middleware-permissions": {
    title: "Protecting express routes with permissions",
    description: "Prevent unauthorized access to routes using RBAC",
    authors: ["chronark"],
    repository: "https://github.com/unkeyed/examples/tree/main/express-with-middleware-permissions",
    image: "/images/templates/express-middleware.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unkeyed/examples/main/express-with-middleware-permissions/README.md",
    language: "Typescript",
    framework: "Express",
  },
  "ratelimit-nextjs": {
    title: "Ratelimit your Next.js routes",
    description: "Using @unkey/ratelimit to protect your API routes from abuse",
    authors: ["chronark"],
    repository: "https://github.com/unkeyed/examples/tree/main/ratelimit",
    image: "/images/templates/ratelimit.png",
    readmeUrl: "https://raw.githubusercontent.com/unkeyed/examples/main/ratelimit/README.md",
    language: "Typescript",
    framework: "Next.js",
    url: "https://github.com/unkeyed/examples/tree/main/ratelimit",
  },
  "bun-koyeb": {
    title: "Global API authentication with Unkey and Koyeb",
    description: "Deploy and secure your API globally with Unkey and Koyeb.",
    authors: ["chronark"],
    repository: "https://github.com/unkeyed/examples/tree/main/bun-koyeb",
    image: "/images/templates/bun_koyeb.png",
    readmeUrl: "https://raw.githubusercontent.com/unkeyed/examples/main/bun-koyeb/README.md",
    language: "Typescript",
    framework: "Bun",
    url: "https://www.koyeb.com/deploy/bunkey?ref=unkey",
  },
  "nextjs-boilderplate": {
    title: "Next.js Boilerplate",
    description: "A simple Next.js app using Unkey to generate and verify API keys.",
    authors: ["chronark"],
    image: "/images/templates/nextjs.png",
    repository: "https://github.com/unkeyed/examples/tree/main/nextjs",
    readmeUrl: "https://raw.githubusercontent.com/unkeyed/examples/main/nextjs/README.md",
    language: "Typescript",
    framework: "Next.js",
  },
  "nextjs-expiration": {
    title: "Next.js Example with Temporary API Keys",
    description:
      "A simple Next.js app using Unkey to generate and verify API keys that expire after 60 seconds.",
    authors: ["ethan-stone"],
    repository: "https://github.com/unkeyed/examples/tree/main/nextjs-expiration",
    image: "/images/templates/expire-keys.png",

    readmeUrl:
      "https://raw.githubusercontent.com/unkeyed/examples/main/nextjs-expiration/README.md",
    language: "Typescript",
    framework: "Next.js",
  },
  "elixir-mix-supervision": {
    title: "Unkey + Elixir Mix Supervision",
    description: "A simple example of using the Unkey Elixir SDK.",
    authors: ["glamboyosa"],
    repository: "https://github.com/unkeyed/unkey/examples/main/elixir_mix_supervision_example",
    readmeUrl:
      "https://raw.githubusercontent.com/unkeyed/examples/main/elixir_mix_supervision_example/README.md",
    language: "Elixir",
    image: "/images/templates/elixir.png",
  },
  "ai-billing": {
    title: "Next.js AI application with Unkey for billing credits",
    description:
      "Simple AI image generation application. Contains example code of generating and refilling Unkey API keys in response to a Stripe payment link, and using the `remaining` field for measuring usage.",
    authors: ["domeccleston"],
    repository: "https://github.com/unkeyed/examples/tree/main/ai-billing",
    readmeUrl: "https://raw.githubusercontent.com/unkeyed/examples/main/ai-billing/README.md",
    image: "/images/templates/ai-billing.png",
    language: "Typescript",
    framework: "Next.js",
  },
  "cli-auth": {
    title: "CLI Auth example with Unkey",
    description: "CLI application that generates a new Unkey API key and writes it to a file.",
    authors: ["domeccleston"],
    repository: "https://github.com/unkeyed/examples/tree/main/unkey-cli",
    language: "Typescript",
    framework: "Next.js",
    readmeUrl: "https://raw.githubusercontent.com/unkeyed/examples/main/unkey-cli/README.md",
    image: "/images/templates/unkey-cli.png",
  },
  openstatus: {
    title: "OpenStatus.dev",
    description:
      "OpenStatus is an open source alternative to your current monitoring service with a beautiful status page.",
    authors: ["mxkaske", "thibaultleouay"],
    repository: "https://github.com/openstatusHQ/openstatus",
    image: "/images/templates/openstatus.png",
    readmeUrl: "https://raw.githubusercontent.com/openstatusHQ/openstatus/main/README.md",
    language: "Typescript",
    framework: "Next.js",
    url: "https://openstatus.dev?ref=unkey.com",
  },
  "atridadl-sprintpadawan": {
    title: "sprintpadawan",
    description: "A scrum poker tool that helps agile teams plan their sprints in real-time.",
    authors: ["atridadl"],
    image: "/images/templates/sprintpadawan.png",
    repository: "https://github.com/atridadl/sprintpadawan",
    readmeUrl: "https://raw.githubusercontent.com/atridadl/sprintpadawan/main/README.md",
    url: "https://sprintpadawan.dev",
    language: "Typescript",
    framework: "Next.js",
  },
  "unkey-clerk": {
    title: "Unkey and Clerk",
    description: "A simple template that shows how to use Unkey with an authentication provider",
    authors: ["perkinsjr"],
    repository: "https://github.com/perkinsjr/unkey-clerk",
    readmeUrl: "https://raw.githubusercontent.com/perkinsjr/unkey-clerk/main/README.md",
    url: "https://github.com/perkinsjr/unkey-clerk",
    image: "/images/templates/clerk.png",
    language: "Typescript",
    framework: "Next.js",
  },
  ocr: {
    title: "OCR as a Service",
    description: "OCR API as a Service using Unkey",
    authors: ["WilfredAlmeida"],
    repository: "https://github.com/WilfredAlmeida/unkey-ocr",
    readmeUrl: "https://raw.githubusercontent.com/WilfredAlmeida/unkey-ocr/main/README.md",
    language: "Typescript",
    url: "https://unkey.com/blog/ocr-service",
    image: "/images/templates/ocr.png",

    framework: "Express",
  },
  yoga: {
    title: "Protect GraphQL APIs with Unkey",
    description: "GraphQL Yoga Plugin system to protect your API",
    authors: ["notrab"],
    repository: "https://github.com/graphqlwtf/91-protect-graphql-apis-with-unkey",
    image: "/images/templates/graphql-yoga.png",
    readmeUrl:
      "https://raw.githubusercontent.com/graphqlwtf/91-protect-graphql-apis-with-unkey/main/README.md",
    language: "Typescript",
    url: "https://graphql.wtf/episodes/91-protect-graphql-apis-with-unkey",
  },
};
