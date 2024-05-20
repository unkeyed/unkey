export class ManagedStream {
  stream;
  reader;
  isDone;
  data;
  isComplete;
  constructor(stream) {
    this.stream = stream;
    this.reader = this.stream.getReader();
    this.isDone = false;
    this.data = "";
    this.isComplete = false;
  }
  async readToEnd() {
    try {
      while (true) {
        const { done, value } = await this.reader.read();
        if (done) {
          this.isDone = true;
          break;
        }
        this.data += new TextDecoder().decode(value);
      }
    } catch (error) {
      console.error("Stream error:", error);
      this.isDone = false;
    } finally {
      this.reader.releaseLock();
    }
    return this.isDone;
  }
  checkComplete() {
    if (this.data.includes("[DONE]")) {
      this.isComplete = true;
    }
  }
  getReader() {
    return this.reader;
  }
  getData() {
    return this.data;
  }
}
