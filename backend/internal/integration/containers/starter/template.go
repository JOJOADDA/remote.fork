package starter

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
)

// Seed writes the opinionated web starter into an empty project workspace. It
// deliberately runs only from project creation, so existing workspaces are
// never overwritten.
func Seed(root, projectName string) error {
	if root == "" {
		return fmt.Errorf("starter workspace path is empty")
	}
	if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	name := projectName
	if name == "" {
		name = "Remote Web App"
	}
	pkg := map[string]any{
		"name":            "remote-web-starter",
		"private":         true,
		"version":         "0.1.0",
		"type":            "module",
		"scripts":         map[string]string{"dev": "vite", "build": "tsc -b && vite build", "preview": "vite preview"},
		"dependencies":    map[string]string{"@tailwindcss/vite": "latest", "tailwindcss": "latest", "vite": "latest", "typescript": "latest", "react": "latest", "react-dom": "latest", "lucide-react": "latest"},
		"devDependencies": map[string]string{"@vitejs/plugin-react": "latest", "@types/react": "latest", "@types/react-dom": "latest"},
	}
	if err := writeJSON(filepath.Join(root, "package.json"), pkg); err != nil {
		return err
	}

	files := map[string]string{
		"index.html": `<!doctype html>
<html lang="ar" dir="rtl">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="theme-color" content="#0b1020" />
    <title>` + escapeHTML(name) + `</title>
  </head>
  <body><div id="root"></div><script type="module" src="/src/main.tsx"></script></body>
</html>
`,
		"tsconfig.json": `{
  "files": [],
  "references": [{ "path": "./tsconfig.app.json" }, { "path": "./tsconfig.node.json" }]
}
`,
		"tsconfig.app.json": `{
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.app.tsbuildinfo",
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "allowJs": false,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx"
  },
  "include": ["src"]
}
`,
		"tsconfig.node.json": `{
  "compilerOptions": { "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.node.tsbuildinfo", "target": "ES2023", "lib": ["ES2023"], "module": "ESNext", "skipLibCheck": true, "moduleResolution": "Bundler", "allowImportingTsExtensions": true, "verbatimModuleSyntax": true, "moduleDetection": "force", "noEmit": true, "strict": true },
  "include": ["vite.config.ts"]
}
`,
		"vite.config.ts": `import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: { host: "0.0.0.0", port: 4173, strictPort: false },
  preview: { host: "0.0.0.0", port: 4173, strictPort: false },
});
`,
		"src/vite-env.d.ts": `/// <reference types="vite/client" />
`,
		"src/main.tsx": `import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./index.css";

createRoot(document.getElementById("root")!).render(<StrictMode><App /></StrictMode>);
`,
		"src/App.tsx": `import { ArrowLeft, CheckCircle2, Sparkles } from "lucide-react";

const features = [
  { title: "تجربة واضحة", body: "ابدأ من قالب منظم وقابل للتوسعة." },
  { title: "تصميم متجاوب", body: "يعمل بسلاسة على الهاتف وسطح المكتب." },
  { title: "جاهز للتطوير", body: "مكونات صغيرة ونظام ألوان متسق." },
];

export default function App() {
  return (
    <main className="min-h-screen bg-slate-950 text-slate-100">
      <section className="mx-auto flex min-h-screen max-w-6xl flex-col justify-center px-5 py-16 sm:px-8">
        <div className="mb-10 inline-flex w-fit items-center gap-2 rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1.5 text-sm text-cyan-200">
          <Sparkles className="h-4 w-4" /> قالب Remote Web Starter
        </div>
        <div className="grid gap-12 lg:grid-cols-[1.15fr_0.85fr] lg:items-end">
          <div>
            <h1 className="max-w-3xl text-4xl font-bold tracking-tight text-white sm:text-6xl">ابنِ واجهة جميلة تبدأ من أساس صحيح.</h1>
            <p className="mt-6 max-w-2xl text-lg leading-9 text-slate-300">قالب React وTypeScript وTailwind مع دعم RTL ونظام ألوان متناسق ومكونات قابلة لإعادة الاستخدام.</p>
            <button className="mt-8 inline-flex items-center gap-2 rounded-xl bg-cyan-400 px-5 py-3 font-semibold text-slate-950 transition hover:bg-cyan-300 focus:outline-none focus:ring-2 focus:ring-cyan-300 focus:ring-offset-2 focus:ring-offset-slate-950">ابدأ الآن <ArrowLeft className="h-4 w-4" /></button>
          </div>
          <div className="rounded-3xl border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-cyan-950/30">
            <div className="mb-6 flex items-center justify-between"><span className="text-sm text-slate-400">مؤشرات القالب</span><span className="rounded-full bg-emerald-400/10 px-2.5 py-1 text-xs text-emerald-300">جاهز</span></div>
            <div className="space-y-4">{features.map((feature) => <div className="flex gap-3" key={feature.title}><CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-cyan-300" /><div><h2 className="font-semibold text-white">{feature.title}</h2><p className="mt-1 text-sm leading-6 text-slate-400">{feature.body}</p></div></div>)}</div>
          </div>
        </div>
      </section>
    </main>
  );
}
`,
		"src/index.css": `@import "tailwindcss";

:root { font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color-scheme: dark; background: #020617; font-synthesis: none; text-rendering: optimizeLegibility; }
* { box-sizing: border-box; }
html { direction: rtl; }
body { margin: 0; min-width: 320px; min-height: 100vh; }
button { font: inherit; }
`,
		"AGENTS.md": `# Web product quality contract

You are a senior product designer and staff frontend engineer. Build polished, production-quality web interfaces rather than bare functional pages.

Use the existing React + TypeScript + Vite + Tailwind stack. Keep the interface responsive on mobile and desktop, use the existing theme tokens, semantic HTML, accessible focus states, hover/active states, loading/empty/error states, reusable components, realistic product copy, and the Lucide icon set. This project is RTL by default: preserve the document direction and verify Arabic alignment, spacing, truncation, and mixed Arabic/English content.

Before declaring a web task complete, run the build, start the preview on 0.0.0.0, inspect the result on mobile and desktop, and fix visual or runtime issues. Never expose internal ports, debug URLs, stack traces, or implementation details in the product UI. Do not report completion merely because files were written.
`,
		"README.md": `# Remote Web Starter

This project is a React + TypeScript + Vite + Tailwind starter with RTL enabled by default.

## Development

npm install

npm run dev -- --host 0.0.0.0 --port 4173

Use reusable components, semantic HTML, responsive layouts, accessible focus states, and verify the preview on desktop and mobile before considering a task complete.
`,
	}
	for name, contents := range files {
		if err := writeFile(filepath.Join(root, name), []byte(contents)); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(path, data)
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func escapeHTML(value string) string {
	return html.EscapeString(fmt.Sprintf("%s", value))
}
