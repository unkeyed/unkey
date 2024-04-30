import { deployTask } from "@/trigger/deploy";

async function main() {
  const res = await deployTask.trigger({
    gateway: {
      id: "1",
    },
  });
  console.log({ res });
}

main();




