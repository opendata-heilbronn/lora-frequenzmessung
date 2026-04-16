// Vitest global setup — runs before every test file.
// Installs a proper localStorage mock because the Node.js environment on this
// host has a system-injected localStorage stub that has no methods (clear, getItem, etc.)

const store = new Map<string, string>()

Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  enumerable: true,
  value: {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => store.set(key, value),
    removeItem: (key: string) => store.delete(key),
    clear: () => store.clear(),
    get length() { return store.size },
    key: (i: number) => [...store.keys()][i] ?? null,
  },
})

// Provide window + location for node environment
if (typeof window === 'undefined') {
  Object.defineProperty(globalThis, 'window', { configurable: true, enumerable: true, value: globalThis })
}
Object.defineProperty(globalThis, 'location', { configurable: true, enumerable: true, writable: true, value: { href: '' } })
