
import type { Logger, Fields } from './interface'

export class ConsoleLogger implements Logger {




  public debug(message: string, fields?: Fields): void {
    console.debug(message, fields ? JSON.stringify(fields) : undefined)
  }
  public info(message: string, fields?: Fields): void {
    console.info(message, fields ? JSON.stringify(fields) : undefined)
  }
  public warn(message: string, fields?: Fields): void {
    console.warn(message, fields ? JSON.stringify(fields) : undefined)
  }
  public error(message: string, fields?: Fields): void {
    console.error(message, fields ? JSON.stringify(fields) : undefined)
  }
  public flush() {
    return Promise.resolve()
  }
}
