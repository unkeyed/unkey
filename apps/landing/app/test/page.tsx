import blogImg from "@/images/bloglarge.jpg";
import Image from "next/image";

export default function TestPage() {
  return (
    <div className="relative h-[200px] w-[400px] overflow-hidden rounded-md">
      <Image src={blogImg} className="w-full h-full" alt="Blog placeholder image" />
      <div className="absolute top-0 right-[-60px] backdrop-blur-xl placeholder-parallelogram" />
    </div>
  );
}
