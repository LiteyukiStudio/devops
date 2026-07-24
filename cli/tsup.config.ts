import { defineConfig } from 'tsup'

export default defineConfig({
  entry: ['src/entry.ts'],
  format: ['esm'],
  dts: false,
  clean: true,
  sourcemap: false,
  noExternal: [
    '@luna-devops/api-client',
    '@luna-devops/api-contract',
  ],
})
