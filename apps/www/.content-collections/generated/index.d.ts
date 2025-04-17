import configuration from "../../content-collections.ts";
import { GetTypeByName } from "@content-collections/core";

export type Post = GetTypeByName<typeof configuration, "posts">;
export declare const allPosts: Array<Post>;

export type Changelog = GetTypeByName<typeof configuration, "changelog">;
export declare const allChangelogs: Array<Changelog>;

export type Policy = GetTypeByName<typeof configuration, "policy">;
export declare const allPolicies: Array<Policy>;

export type Career = GetTypeByName<typeof configuration, "careers">;
export declare const allCareers: Array<Career>;

export type Glossary = GetTypeByName<typeof configuration, "glossary">;
export declare const allGlossaries: Array<Glossary>;

export {};
