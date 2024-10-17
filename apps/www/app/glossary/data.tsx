import { KeyIcon } from "@/components/svg/glossary-page";

// generate a list of api terms, where each term has a title, description, a category (e.g. "Security", "Performance", "Management", "Design", "Knowledge", etc.) and an icon that's just a hard coded SVG representation of the title right now:
export const terms = [
  {
    slug: "api-design",
    title: "API Design",
    description: "API design is the process of designing an API.",
    category: "Design",
    image: <KeyIcon />,
  },
  {
    slug: "api-keys",
    title: "API Keys",
    description: "API keys are used to authenticate and authorize access to APIs.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "rate-limiting",
    title: "Rate Limiting",
    description: "Rate limiting is a technique used to limit the number of requests a client can make to an API within a given time period.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "authentication",
    title: "Authentication",
    description: "Authentication is the process of verifying the identity of a user or client.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "key-encryption",
    title: "Key Encryption",
    description: "Key encryption is a technique used to encrypt data using a key.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "api-monitoring",
    title: "API Monitoring",
    description: "API monitoring is the process of monitoring the performance and availability of an API.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "api-testing",
    title: "API Testing",
    description: "API testing is the process of testing an API to ensure it is working correctly.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "api-documentation",
    title: "API Documentation",
    description: "API documentation is the process of documenting an API.",
    category: "Security",
    image: <KeyIcon />,
  },
  {
    slug: "api-versioning",
    title: "API Versioning",
    description: "API versioning is the process of versioning an API.",
    category: "Security",
    image: <KeyIcon />,
  }
  
];

export const categories = [
  {
    slug: "api-security",
    title: "API Security",
    description: "Security is the process of securing an API.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-design",
    title: "API Design",
    description: "Principles and practices for creating effective API structures.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-developer-experience",
    title: "API Developer Experience",
    description: "Focusing on making APIs easy and enjoyable for developers to use.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-gateway",
    title: "API Gateway",
    description: "A server that acts as an API front-end, receiving API requests and routing them.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-governance",
    title: "API Governance",
    description: "Policies and procedures for managing API development and usage within an organization.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-platform-management",
    title: "API Platform Management",
    description: "Overseeing the entire lifecycle of APIs within a platform ecosystem.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-product-management",
    title: "API Product Management",
    description: "Treating APIs as products and managing their development and evolution.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-specification",
    title: "API Specification",
    description: "Formal documentation describing the structure and behavior of an API.",
    icon: <KeyIcon />,
  },
  {
    slug: "api-strategy",
    title: "API Strategy",
    description: "Long-term planning for API development and integration within business goals.",
    icon: <KeyIcon />,
  },
  {
    slug: "opentelemetry",
    title: "OpenTelemetry",
    description: "An observability framework for cloud-native software, including APIs.",
    icon: <KeyIcon />,
  },
];
