type FlagCode = "us" | "de" | "sg" | "au" | "jp" | "in" | "br" | "local";

export type RegionInfo = {
  name: string;
  city: string;
  flag: FlagCode | null;
};

const REGIONS = new Map<string, Omit<RegionInfo, "name">>([
  ["us-west-2", { city: "Oregon", flag: "us" }],
  ["us-east-1", { city: "N. Virginia", flag: "us" }],
  ["eu-central-1", { city: "Frankfurt", flag: "de" }],
  ["ap-southeast-1", { city: "Singapore", flag: "sg" }],
  ["ap-southeast-2", { city: "Sydney", flag: "au" }],
  ["ap-northeast-1", { city: "Tokyo", flag: "jp" }],
  ["ap-south-1", { city: "Mumbai", flag: "in" }],
  ["sa-east-1", { city: "São Paulo", flag: "br" }],
  ["local", { city: "Local", flag: "local" }],
]);

const FLAG_BY_PREFIX: ReadonlyArray<readonly [string, FlagCode]> = [
  ["us-", "us"],
  ["eu-", "de"],
  ["ap-northeast-", "jp"],
  ["ap-south-", "in"],
  ["sa-", "br"],
];

export function regionInfo(name: string): RegionInfo {
  const known = REGIONS.get(name);
  if (known) {
    return { name, ...known };
  }
  const flag = FLAG_BY_PREFIX.find(([prefix]) => name.startsWith(prefix))?.[1] ?? null;
  return { name, city: name, flag };
}
