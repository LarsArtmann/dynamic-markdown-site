import { defineConfig, fontProviders } from "astro/config";
import starlight from "@astrojs/starlight";
import sitemap from "@astrojs/sitemap";

import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
	site: "https://dynamicmarkdown.lars.software",

	compressHTML: true,

	prefetch: {
		prefetchAll: false,
		defaultStrategy: "hover",
	},

	fonts: [
		{
			provider: fontProviders.google(),
			name: "Space Grotesk",
			cssVariable: "--font-space-grotesk",
			weights: [300, 400, 500, 600, 700],
			styles: ["normal"],
			subsets: ["latin"],
			fallbacks: ["sans-serif"],
		},
		{
			provider: fontProviders.fontsource(),
			name: "JetBrains Mono",
			cssVariable: "--font-jetbrains-mono",
			weights: [400, 500, 600, 700],
			styles: ["normal"],
			subsets: ["latin"],
			fallbacks: ["monospace"],
		},
	],

	integrations: [
		sitemap(),
		starlight({
			title: "Dynamic Markdown Site",
			favicon: "/favicon.svg",
			customCss: ["./src/styles/starlight.css"],
			expressiveCode: {
				themes: ["github-light", "github-dark"],
				frames: {
					showCopyToClipboardButton: true,
				},
			},
			sidebar: [
				{
					label: "Getting Started",
					items: [
						{ label: "Installation", slug: "getting-started/installation" },
						{ label: "Quick Start", slug: "getting-started/quick-start" },
					],
				},
				{
					label: "Guides",
					items: [
						{ label: "Configuration", slug: "guides/configuration" },
						{ label: "Docker", slug: "guides/docker" },
						{ label: "Cloud Storage", slug: "guides/cloud-storage" },
						{ label: "Markdown Features", slug: "guides/markdown-features" },
						{ label: "API Endpoints", slug: "guides/api-endpoints" },
					],
				},
				{
					label: "Community",
					items: [
						{ label: "Changelog", slug: "changelog" },
						{ label: "Contributing", slug: "contributing" },
						{ label: "Related Tools", slug: "related-tools" },
					],
				},
			],
			social: [
				{
					icon: "github",
					label: "GitHub",
					href: "https://github.com/LarsArtmann/dynamic-markdown-site",
				},
			],
			head: [
				{
					tag: "meta",
					attrs: {
						name: "description",
						content:
							"A Go server that turns any directory of markdown files into a beautiful, navigable website. Syntax highlighting, search, diagrams, live reload, cloud storage, and caching built in.",
					},
				},
			],
		}),
	],

	vite: {
		plugins: [tailwindcss()],
	},
});
