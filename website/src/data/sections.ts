import type { StepCard, ComparisonItem, UseCase, ComparisonMatrix } from "./types";

export const steps: StepCard[] = [
	{
		step: "1",
		stepColor: "accent",
		title: "Drop Markdown",
		desc: "Place .md files in any directory structure. Add frontmatter, diagrams, code blocks.",
		code: "echo '# Hello' > docs/index.md",
	},
	{
		step: "2",
		stepColor: "accent",
		title: "Run the Binary",
		desc: "Point the server at your files or cloud bucket. No config file, no build step.",
		code: "dynamic-markdown-site -root ./docs",
	},
	{
		step: "3",
		stepColor: "amber",
		title: "Browse & Search",
		desc: "Directory navigation, full-text search, syntax highlighting, diagrams — all automatic.",
		code: "// open http://localhost:8080",
	},
	{
		step: "4",
		stepColor: "amber",
		title: "Edit & Reload",
		desc: "Dev mode watches files and live-reloads the browser via SSE. 500ms debounce.",
		code: "dynamic-markdown-site -dev -root ./docs",
	},
];

export const comparisonMatrix: ComparisonMatrix = {
	columns: ["Static Site Gen", "Wiki Engine", "Dynamic Markdown Site"],
	rows: [
		{ feature: "Zero build step", values: ["no", "yes", "yes"] },
		{ feature: "Live reload", values: ["partial", "yes", "yes"] },
		{ feature: "Cloud storage (S3/GCS)", values: ["no", "no", "yes"] },
		{ feature: "Full-text search", values: ["partial", "yes", "yes"] },
		{ feature: "Syntax highlighting", values: ["yes", "partial", "yes"] },
		{ feature: "D2 + Mermaid diagrams", values: ["partial", "no", "yes"] },
		{ feature: "Single binary", values: ["partial", "no", "yes"] },
		{ feature: "No database", values: ["yes", "no", "yes"] },
	],
};

export const comparisons: ComparisonItem[] = [
	{
		variant: "Static Site Gen",
		accent: false,
		pros: ["Fast static output", "Mature ecosystem"],
		cons: ["Build step required", "No live editing", "No cloud storage", "Config-heavy"],
	},
	{
		variant: "Dynamic Markdown Site",
		accent: true,
		pros: [
			"Zero config, zero build",
			"S3, GCS, Azure, or filesystem",
			"Live reload via SSE",
			"Full-text search built in",
			"D2 + Mermaid diagrams",
			"Single static binary",
		],
		cons: [],
	},
	{
		variant: "Wiki Engine",
		accent: false,
		pros: ["Collaboration features", "Version control"],
		cons: ["Database required", "Complex setup", "No cloud-native storage", "Heavy runtime"],
	},
];

export const useCases: UseCase[] = [
	{
		title: "Documentation",
		desc: "Serve API docs, architecture docs, or runbooks from a git repo",
		icon: "docs",
	},
	{
		title: "Internal Wiki",
		desc: "Team knowledge base with search, navigation, and live updates",
		icon: "wiki",
	},
	{
		title: "Static Content",
		desc: "Publish articles, tutorials, or guides without a build pipeline",
		icon: "book",
	},
];
