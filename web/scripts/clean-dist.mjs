import { readdir, rm } from 'node:fs/promises'
import { resolve } from 'node:path'

const dist = resolve(import.meta.dirname, '../../internal/webui/dist')
const entries = await readdir(dist)

await Promise.all(
  entries
    .filter((entry) => entry !== 'placeholder.txt')
    .map((entry) => rm(resolve(dist, entry), { recursive: true, force: true })),
)
