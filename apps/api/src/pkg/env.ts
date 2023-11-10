export type Env = {
  Bindings: {
    DATABASE_HOST: string;
    DATABASE_USERNAME: string;
    DATABASE_PASSWORD: string;
    AXIOM_TOKEN: string;
    CLOUDFLARE_API_KEY: string;
    CLOUDFLARE_ZONE_ID: string;
    ENVIRONMENT: "development" | "preview" | "production";
  };
};
