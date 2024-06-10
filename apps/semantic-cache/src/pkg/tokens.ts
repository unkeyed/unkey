import model from "tiktoken/encoders/cl100k_base.json";
import { Tiktoken, init } from "tiktoken/lite/init";
import wasm from "../../node_modules/tiktoken/lite/tiktoken_bg.wasm";

export class Tokenizer {
  private static initialized: boolean;

  private constructor() {}

  static async init(): Promise<Tokenizer> {
    if (!Tokenizer.initialized) {
      await init((imports) => WebAssembly.instantiate(wasm, imports));
      Tokenizer.initialized = true;
    }
    return new Tokenizer();
  }

  public count(s: string): number {
    const encoder = new Tiktoken(model.bpe_ranks, model.special_tokens, model.pat_str);
    const tokens = encoder.encode(s);

    encoder.free();
    return tokens.length;
  }
}
