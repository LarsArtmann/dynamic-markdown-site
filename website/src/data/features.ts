import type { Feature } from "./types";

export const features: Feature[] = [
	{
		icon: "cloud",
		title: "Cloud-Native Storage",
		desc: "Filesystem, S3, GCS, or Azure Blob. Same binary, any backend. Powered by gocloud.dev.",
	},
	{
		icon: "search",
		title: "Full-Text Search",
		desc: "Relevance-scored search with highlighted snippets. Title match weighted over body match.",
	},
	{
		icon: "lightning",
		title: "Live Reload",
		desc: "Edit markdown, see changes instantly in the browser. SSE-based, 500ms debounce, auto-reconnect.",
	},
	{
		icon: "code",
		title: "Syntax Highlighting",
		desc: "Chroma with Monokai theme for 200+ languages. Server-side rendered, zero client JS.",
	},
	{
		icon: "diagram",
		title: "Diagrams",
		desc: "D2 server-side SVG rendering and Mermaid client-side. Both via fenced code blocks.",
	},
	{
		icon: "refresh",
		title: "Auto-Tuning Cache",
		desc: "Otter cache with access-based TTL. Popular pages stay hot, cold pages evict automatically.",
	},
	{
		icon: "shield",
		title: "Security-First",
		desc: "Path traversal prevention at the type level. Rate limiting, security headers, distroless runtime.",
	},
	{
		icon: "folder",
		title: "Zero Build Step",
		desc: "No static-site generator, no database, no frontend framework. Drop files, run binary, get a site.",
	},
];
