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
  "Actix",
  "Rocket",
  "Oak",
  "FastAPI",
  "NestJS",
  "Koa",
  "Hono",
  "AdonisJS",
  "fastify",
  "feathers",
  "hapi",
  "Deno", // should we add runtimes as a category?
  "Rails",
  "Nuxt",
  "Flutter",
  "React Native",
  "Symfony",
  "Astro",
  "Shelf",
  "Laravel",
  "russh",
] as const;
export type Framework = StrArrayToUnion<typeof frameworks>;
// id -> label
export const languages = [
  "Typescript",
  "Python",
  "Golang",
  "Rust",
  "Elixir",
  "Ruby",
  "Dart",
  "PHP",
] as const;
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
  "rust-ssh": {
    title: "Secure Rust 🦀 SSH Server with Unkey",
    description: "Fine-grained control through time- and quota-limited API keys",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-rust-ssh",

    image: "/images/templates/rust-ssh.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-rust-ssh/refs/heads/main/README.md",

    language: "Rust",
    framework: "russh",
  },

  laravel: {
    title: "Laravel middleware using Unkey RBAC",
    description: "Protect your Laravel API",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-laravel-example",

    image: "/images/templates/laravel.png",
    readmeUrl:
      "https://raw.githubusercontent.com/harshsbhat/unkey-laravel-example/refs/heads/main/README.md",

    language: "PHP",
    framework: "Laravel",
  },

  "hono-ratelimit": {
    title: "Nuxt.js API Rate Limiter with Unkey",
    description: "Safeguard your API endpoints while maintaining optimal performance.",
    authors: ["Devansh-Baghel"],
    repository: "https://github.com/Devansh-Baghel/hono-unkey-ratelimit-starter",

    image: "/images/templates/hono-ratelimit.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Devansh-Baghel/hono-unkey-ratelimit-starter/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Hono",
  },

  "nuxt-ratelimit": {
    title: "Nuxt.js API Rate Limiter with Unkey",
    description: "Safeguard your API endpoints while maintaining optimal performance.",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-nuxt-ratelimit",

    image: "/images/templates/nuxt-ratelimit.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-nuxt-ratelimit/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Nuxt",
  },

  svelte: {
    title: "Svelte Route Protection",
    description: "API key authentication in Svelte",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-svelte",

    image: "/images/templates/svelte.png",
    readmeUrl:
      "https://raw.githubusercontent.com/harshsbhat/unkey-svelte/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Svelte",
  },

  "dart-shelf": {
    title: "Protect your Shelf Web Server",
    description: "Rate Limiting and Access Control Middleware",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-dart",

    image: "/images/templates/dart-shelf.png",
    readmeUrl: "https://raw.githubusercontent.com/unrenamed/unkey-dart/refs/heads/main/README.md",

    language: "Dart",
    framework: "Shelf",
  },

  "astro-ratelimit": {
    title: "Ratelimiting Astro",
    description: "Authentication and authorization",
    authors: ["harshsbhat"],
    repository: "https://github.com/template",

    image: "/images/templates/astro-ratelimit.png",
    readmeUrl: "https://raw.githubusercontent.com/harshsbhat/unkey-astro/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Astro",
  },

  "adonis-rbac": {
    title: "API keys in AdonisJS apps",
    description: "Authentication and authorization",
    authors: ["Ionfinisher"],
    repository: "https://github.com/Ionfinisher/unkey-adonisjs-template",

    image: "/images/templates/adonis-rbac.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Ionfinisher/unkey-adonisjs-template/refs/heads/main/README.md",

    language: "Typescript",
    framework: "AdonisJS",
  },

  "firecrawl-streamlit": {
    title: "Ratelimiting firecrawl",
    description: "Ensure fair use and protect your wallet",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-streamlit-firecrawl",
    image: "/images/templates/firecrawl-streamlit.png",
    readmeUrl:
      "https://raw.githubusercontent.com/harshsbhat/unkey-streamlit-firecrawl/refs/heads/main/README.md",
    language: "Python",
    framework: undefined,
  },
  symfony: {
    title: "Protecting your Symfony routes",
    description: "Quickstart for Symfony",
    authors: ["utkarshml"],
    repository: "https://github.com/utkarshml/unkey_symfony",
    image: "/images/templates/symfony.png",
    readmeUrl:
      "https://raw.githubusercontent.com/utkarshml/unkey_symfony/refs/heads/main/README.md",
    language: "PHP",
    framework: "Symfony",
  },
  "react-native": {
    title: "Ratelimiting in React Native",
    description: "Ratelimiting email sending in mobile apps",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-react-native",
    image: "/images/templates/react-native.png",
    readmeUrl:
      "https://raw.githubusercontent.com/harshsbhat/unkey-react-native/refs/heads/main/README.md",
    language: "Typescript",
    framework: "React Native",
  },
  "url-shortener": {
    title: "Next.js URL shortener",
    description: "Ratelimit based on billing tiers",
    authors: ["Khaan25"],
    repository: "https://github.com/unrenamed/Khaan25/url-shortner-time-based",
    image: "/images/templates/url-shortener.png",
    url: "https://url-shortner-time-based-zia-unkey.vercel.app",
    readmeUrl:
      "https://raw.githubusercontent.com/Khaan25/url-shortner-time-based/refs/heads/main/README.md",
    language: "Typescript",
    framework: "Next.js",
  },
  "flutter-coupons": {
    title: "Generate and validate coupon codes",
    description:
      "Invite others to view and contribute to your collection while ensuring that only authorized users have access to your valuable digital content.",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-coffee-shop",
    image: "/images/templates/flutter-coupons.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-coffee-shop/refs/heads/main/README.md",
    language: "Dart",
    framework: "Flutter",
  },
  "nuxt-image-gallery": {
    title: "Secure Your Media Library with Unkey",
    description:
      "Invite others to view and contribute to your collection while ensuring that only authorized users have access to your valuable digital content.",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-nuxt-image-gallery",
    image: "/images/templates/nuxt-image-gallery.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-nuxt-image-gallery/refs/heads/main/README.md",
    language: "Typescript",
    framework: "Nuxt",
  },
  "ruby-on-rails": {
    title: "Ruby on Rails",
    description: "Starter Kit with Unkey API Authentication",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-rails",
    image: "/images/templates/ruby-on-rails.png",
    readmeUrl: "https://raw.githubusercontent.com/unrenamed/unkey-rails/refs/heads/main/README.md",
    language: "Ruby",
    framework: "Rails",
  },
  hapi: {
    title: "Hono API with Unkey Middleware in Deno",
    description: "How to create a minimal API with Hapi.js, including public and protected routes",
    authors: ["Yash-1511"],
    repository: "https://github.com/Yash-1511/hapi-unkey-template",
    image: "/images/templates/hapi.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Yash-1511/hapi-unkey-template/refs/heads/master/README.md",
    language: "Typescript",
    framework: "hapi",
  },
  "deno-hono": {
    title: "Hono API with Unkey Middleware in Deno",
    description: "Basic API using the Hono framework with Deno",
    authors: ["Yash-1511"],
    repository: "https://github.com/Yash-1511/hono-unkey-deno",
    image: "/images/templates/deno-hono.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Yash-1511/hono-unkey-deno/refs/heads/master/README.md",
    language: "Typescript",
    framework: "Deno",
  },
  feathers: {
    title: "Protect your feathers backend",
    description: "Custom authentication strategy",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-feathers",
    image: "/images/templates/feathers.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-feathers/refs/heads/main/README.md",
    language: "Typescript",
    framework: "feathers",
  },
  "go-nethttp": {
    title: "Go standard lib",
    description: "Unkey with net/http",
    authors: ["diwasrimal"],
    repository: "https://github.com/diwasrimal/unkey-go-stdlib-auth",
    image: "/images/templates/go-nethttp.png",
    readmeUrl:
      "https://raw.githubusercontent.com/diwasrimal/unkey-go-stdlib-auth/refs/heads/main/README.md",
    language: "Golang",
    framework: undefined,
  },
  "hono-cloudflare": {
    title: "Hono Ratelimit Starter for Cloudflare Workers",
    description: "Simple hono and cloudflare workers api with rate limiting by unkey",
    authors: ["Devansh-Baghel"],
    repository: "https://github.com/Devansh-Baghel/hono-unkey-ratelimit-starter",

    image: "/images/templates/hono-cloudflare.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Devansh-Baghel/hono-unkey-ratelimit-starter/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Hono",
  },
  fastify: {
    title: "Protecting your fastify API",
    description: "API keys and ratelimiting for fastify",
    authors: ["Vardhaman619"],
    repository: "https://github.com/Vardhaman619/fastify-unkey",

    image: "/images/templates/fastify.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Vardhaman619/fastify-unkey/refs/heads/main/README.md",

    language: "Typescript",
    framework: "fastify",
  },
  "adonis-ratelimit": {
    title: "Ratelimiting in AdonisJS apps",
    description: "Dynamic IP based ratelimiting.",
    authors: ["Ionfinisher"],
    repository: "https://github.com/Ionfinisher/unkey-adonisjs-ratelimit",

    image: "/images/templates/adonis-ratelimit.png",
    readmeUrl:
      "https://raw.githubusercontent.com/Ionfinisher/unkey-adonisjs-ratelimit/refs/heads/main/README.md",

    language: "Typescript",
    framework: "AdonisJS",
  },
  sealshare: {
    title: "End-to-end encrypted secret sharing",
    description: "Share secrets securely, directly in your browser.",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/sealshare",

    image: "/images/templates/sealshare.png",
    readmeUrl: "https://raw.githubusercontent.com/unrenamed/sealshare/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Next.js",
  },
  koa: {
    title: "Koa.js middleware with Unkey RBAC",
    description: "Implement API key verification in your Koa apps",
    authors: ["harshsbhat"],
    repository: "https://github.com/harshsbhat/unkey-koa",

    image: "/images/templates/koa.png",
    readmeUrl: "https://raw.githubusercontent.com/harshsbhat/unkey-koa/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Koa",
  },
  "nextjs-supabase-payasyougo": {
    title: "Next.js Pay-as-you-Go starter kit",
    description: "Building Pay-As-You-Go apps with Next.js, Unkey and Supabase",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-nextjs-pay-as-you-go",

    image: "/images/templates/nextjs-supabase-payasyougo.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-nextjs-pay-as-you-go/refs/heads/main/README.md",

    language: "Typescript",
    framework: "Next.js",
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
  },
  "rust-rocket": {
    title: "Secure your Rust Rocket API with Unkey",
    description: "Generative AI REST API built with Rust and Rocket web framework with call quotas",
    authors: ["unrenamed"],
    repository: "https://github.com/unrenamed/unkey-rust-rocket",
    image: "/images/templates/rust-rocket.png",
    readmeUrl:
      "https://raw.githubusercontent.com/unrenamed/unkey-rust-rocket/refs/heads/main/README.md",
    language: "Rust",
    framework: "Rocket",
  },
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
