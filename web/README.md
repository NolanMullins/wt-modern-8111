# WT Modern 8111 web frontend

React and TypeScript frontend embedded in the Go companion.

```powershell
npm install
npm run dev
```

The Vite development server proxies `/api` to `http://127.0.0.1:17711`, so run
the Go companion alongside it.

`npm run build` validates TypeScript and writes the production bundle to
`../internal/webui/dist` for embedding in the executable.
