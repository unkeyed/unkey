import { FeatureGrid } from "@/components/feature/feature-grid";

export default async function Blog() {
  return (
    <>
      <div className="bg-black">
        <div className="max-w-[1200px] mx-auto">
          <p className=" text-gray-100 text-center p-6">Demo Page</p>
          <p>Feature Example</p>
          <div className="mx-auto mt-10">
            <FeatureGrid />
          </div>
        </div>
      </div>
    </>
  );
}
