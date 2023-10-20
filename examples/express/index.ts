import { verifyKey } from "@unkey/api";
import dotenv from "dotenv";
import express, { Application, Request, Response } from "express";
//For env File
dotenv.config();

const app: Application = express();
const port = process.env.PORT || 8000;

app.get("/", (_req: Request, res: Response) => {
  res.send("Welcome to Express & TypeScript Server");
});

// This endpoint is protected by Unkey
app.get("/secret", async (req: Request, res: Response) => {
  const authHeader = req.headers.authorization;
  const key = authHeader?.toString().replace("Bearer ", "");
  if (!key) {
    return res.status(401).send("Unauthorized");
  }

  const { result, error } = await verifyKey(key);
  if (error) {
    // This may happen on network errors
    // We already retry the request 5 times, but if it still fails, we return an error
    console.error(error);
    res.status(500);
    return res.status(500).send("Internal Server Error");
  }

  if (!result.valid) {
    res.status(401);
    return res.status(401).send("Unauthorized");
  }

  return res.status(200).send(JSON.stringify(result));
});
app.listen(port, () => {
  console.log(`Server is listening at http://localhost:${port}`);
});
