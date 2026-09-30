export type RegionFlag = "us" | "de" | "sg";

export type RegionInfo = {
  name: string;
  city: string;
  flag: RegionFlag | null;
  pin: { lon: number; lat: number; label: "left" | "right" } | null;
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
]);

export function regionInfo(name: string): RegionInfo {
  return { name, ...(REGIONS.get(name) ?? { city: name, flag: null, pin: null }) };
}
