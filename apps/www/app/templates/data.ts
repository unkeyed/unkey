type StrArrayToUnion<TArr extends readonly string[]> = TArr[number];

export const frameworks = [
  "Actix",
  "Axum",
  "Bun",
  "Django",
  "Echo",
  "Express",
  "FastAPI",
  "Flask",
  "NestJS",
  "Next.js",
  "Oak",
  "Rocket",
  "Svelte",
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
  "rust-actix": {
    title: "Secure your Rust Actix API with Unkey",
    description: "A Rust API service using Unkey for API key validation with the Actix framework.",
    authors: ["djnovin"],
    repository: "https://github.com/djnovin/unkey-rust-actix",
    image: "/images/templates/rust-actix.png",
    readmeUrl:
      "https://raw.githubusercontent.com/djnovin/unkey-rust-actix/refs/heads/main/README.md",
    language: "Rust",
    framework: "Actix",
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
  "python-fastapi": {
    title: "Secure your Fast API",
    description: "Learn how to setup Fast API with Unkey and protect your routes",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-fastapi-boilerplate",
    image: "/images/templates/python-fastapi.png",
    readmeUrl:
      "https://raw.githubusercontent.com/harshsbhat/unkey-fastapi-boilerplate/refs/heads/main/README.md",
    language: "Python",
    framework: "FastAPI",
  },
  "typescript-nestjs": {
    title: "Protect your NestJS API with Unkey",
    description: "Starter kit for NestJS protected by Unkey",
    authors: ["djnovin"],
    repository: "https://github.com/djnovin/unkey-ts-nestjs",
    image: "/images/templates/typescript-nestjs.png",
    readmeUrl:
      "https://raw.githubusercontent.com/djnovin/unkey-ts-nestjs/refs/heads/main/README.md",
    language: "Typescript",
    framework: "NestJS",
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
  "reno-oak-ratelimit": {
    title: "Ratelimiting your Oak API",
    description: "Simple deno API with Oak and ratelimiting with Unkey",
    authors: ["Devansh-Baghel"],
    repository: "https://github.com/Devansh-Baghel/deno-unkey-ratelimit-starter",
    image: "/images/templates/deno-oak-ratelimit.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Devansh-Baghel/deno-unkey-ratelimit-starter/refs/heads/main/README.md",
    language: "Typescript",
    framework: "Oak",
  }
};
