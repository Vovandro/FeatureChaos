export interface IStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export class MemoryStorage implements IStorage {
  private data: Record<string, string> = {};
  getItem(key: string): string | null {
    return Object.prototype.hasOwnProperty.call(this.data, key)
      ? this.data[key]
      : null;
  }
  setItem(key: string, value: string): void {
    this.data[key] = value;
  }
  removeItem(key: string): void {
    delete this.data[key];
  }
}

export class BrowserLocalStorage implements IStorage {
  private readonly prefix: string;
  constructor(prefix = "") {
    this.prefix = prefix ? prefix + ":" : "";
  }
  private k(key: string): string {
    return this.prefix + key;
  }
  getItem(key: string): string | null {
    try {
      if (typeof window !== "undefined" && window.localStorage) {
        return window.localStorage.getItem(this.k(key));
      }
    } catch {}
    return null;
  }
  setItem(key: string, value: string): void {
    try {
      if (typeof window !== "undefined" && window.localStorage) {
        window.localStorage.setItem(this.k(key), value);
      }
    } catch {}
  }
  removeItem(key: string): void {
    try {
      if (typeof window !== "undefined" && window.localStorage) {
        window.localStorage.removeItem(this.k(key));
      }
    } catch {}
  }
}

export function detectDefaultStorage(prefix = ""): IStorage {
  try {
    if (typeof window !== "undefined" && window.localStorage) {
      return new BrowserLocalStorage(prefix);
    }
  } catch {}
  return new MemoryStorage();
}
