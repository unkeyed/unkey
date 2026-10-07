export type FlagCode = "us" | "de" | "sg" | "au" | "jp" | "in" | "br" | "local";

type RegionPin = { lon: number; lat: number; label: "left" | "right" };

export type RegionInfo = {
  name: string;
  city: string;
  flag: FlagCode | null;
  pin: RegionPin | null;
};

const REGIONS = new Map<string, Omit<RegionInfo, "name">>([
  ["us-west-2", { city: "Oregon", flag: "us", pin: { lon: -120.5, lat: 45.8, label: "left" } }],
  [
    "us-east-1",
    { city: "N. Virginia", flag: "us", pin: { lon: -77.5, lat: 38.9, label: "right" } },
  ],
  ["eu-central-1", { city: "Frankfurt", flag: "de", pin: { lon: 8.7, lat: 50.1, label: "right" } }],
  [
    "ap-southeast-1",
    { city: "Singapore", flag: "sg", pin: { lon: 103.8, lat: 1.35, label: "left" } },
  ],
  [
    "ap-southeast-2",
    { city: "Sydney", flag: "au", pin: { lon: 151.2, lat: -33.9, label: "left" } },
  ],
  ["ap-northeast-1", { city: "Tokyo", flag: "jp", pin: { lon: 139.7, lat: 35.7, label: "left" } }],
  ["ap-south-1", { city: "Mumbai", flag: "in", pin: { lon: 72.9, lat: 19.1, label: "left" } }],
  ["sa-east-1", { city: "São Paulo", flag: "br", pin: { lon: -46.6, lat: -23.5, label: "right" } }],
  ["local", { city: "Local", flag: "local", pin: null }],
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
  return { name, city: name, flag, pin: null };
}
