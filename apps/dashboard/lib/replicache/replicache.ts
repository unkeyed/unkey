import { Replicache, TEST_LICENSE_KEY, type WriteTransaction } from "replicache";
import { type Api, db, schema } from "../db";

export const mutators = {
  createApi: async (tx: WriteTransaction, { id, name }: Pick<Api, "id" | "name">) => {
    const apis = (await tx.scan({ prefix: "api/" }).values().toArray()) as Pick<
      Api,
      "id" | "name"
    >[];
    console.log({ apis });
    apis.push({ id, name });
    await tx.put(`api/${id}`, { id, name });
  },

  deleteApi: async (tx: WriteTransaction, { id }: Pick<Api, "id">) => {
    await tx.del(`api/${id}`);
  },
};
