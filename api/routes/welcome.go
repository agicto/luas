package routes

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zgiai/zgo/internal/infra/config"
)

const welcomeHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>{{APP_NAME}}</title>
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
	<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=Space+Grotesk:wght@500;700&display=swap" rel="stylesheet">
	<style>
		:root {
			--bg: #f5f1e8;
			--bg-strong: #efe7d8;
			--panel: rgba(255, 255, 255, 0.82);
			--panel-strong: #ffffff;
			--ink: #132238;
			--muted: #5f6d7b;
			--line: rgba(19, 34, 56, 0.1);
			--line-strong: rgba(19, 34, 56, 0.16);
			--accent: #1668e3;
			--accent-strong: #0f4fb3;
			--accent-soft: rgba(22, 104, 227, 0.12);
			--teal: #0f8f7d;
			--amber: #c7641a;
			--shadow: 0 22px 70px rgba(16, 29, 46, 0.14);
		}
		* { box-sizing: border-box; }
		html { line-height: 1.5; -webkit-text-size-adjust: 100%; }
		body {
			margin: 0;
			min-height: 100vh;
			font-family: "IBM Plex Sans", system-ui, sans-serif;
			color: var(--ink);
			background:
				radial-gradient(circle at top left, rgba(22, 104, 227, 0.18), transparent 28%),
				radial-gradient(circle at top right, rgba(15, 143, 125, 0.14), transparent 24%),
				linear-gradient(180deg, var(--bg) 0%, #fbfaf7 40%, #f5f1e8 100%);
		}
		body::before {
			content: "";
			position: fixed;
			inset: 0;
			pointer-events: none;
			background-image:
				linear-gradient(rgba(19, 34, 56, 0.035) 1px, transparent 1px),
				linear-gradient(90deg, rgba(19, 34, 56, 0.035) 1px, transparent 1px);
			background-size: 32px 32px;
			mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.14), transparent 82%);
		}
		a { color: inherit; text-decoration: none; }
		code, pre {
			font-family: "SFMono-Regular", ui-monospace, "Cascadia Code", "Source Code Pro", monospace;
		}
		.page {
			position: relative;
			max-width: 1180px;
			margin: 0 auto;
			padding: 28px 20px 64px;
		}
		.topbar {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 16px;
			margin-bottom: 26px;
		}
		.brand {
			display: flex;
			align-items: center;
			gap: 14px;
		}
		.brand-mark {
			width: 54px;
			height: 54px;
			border-radius: 18px;
			background: linear-gradient(145deg, #0e58cf, #2c8bfd 70%, #75d3c4 100%);
			box-shadow: 0 18px 40px rgba(22, 104, 227, 0.28);
			display: grid;
			place-items: center;
			color: white;
			font-family: "Space Grotesk", sans-serif;
			font-size: 24px;
			font-weight: 700;
			letter-spacing: -0.04em;
		}
		.brand-copy strong {
			display: block;
			font-family: "Space Grotesk", sans-serif;
			font-size: 19px;
			letter-spacing: -0.03em;
		}
		.brand-copy span {
			display: block;
			margin-top: 2px;
			color: var(--muted);
			font-size: 14px;
		}
		.topbar-actions {
			display: flex;
			flex-wrap: wrap;
			align-items: center;
			gap: 10px;
		}
		.pill {
			display: inline-flex;
			align-items: center;
			gap: 8px;
			padding: 10px 14px;
			border-radius: 999px;
			background: rgba(255, 255, 255, 0.65);
			border: 1px solid rgba(255, 255, 255, 0.72);
			box-shadow: 0 10px 28px rgba(19, 34, 56, 0.08);
			font-size: 13px;
			color: var(--muted);
			backdrop-filter: blur(14px);
		}
		.pill strong {
			color: var(--ink);
			font-weight: 600;
		}
		.pill-dot {
			width: 8px;
			height: 8px;
			border-radius: 999px;
			background: var(--teal);
			box-shadow: 0 0 0 6px rgba(15, 143, 125, 0.12);
		}
		.hero {
			display: grid;
			grid-template-columns: minmax(0, 1.15fr) minmax(320px, 0.85fr);
			gap: 28px;
			align-items: stretch;
		}
		.hero-panel,
		.status-panel,
		.surface,
		.route-card,
		.arch-card {
			background: var(--panel);
			border: 1px solid rgba(255, 255, 255, 0.68);
			backdrop-filter: blur(16px);
			box-shadow: var(--shadow);
		}
		.hero-panel {
			position: relative;
			overflow: hidden;
			border-radius: 34px;
			padding: 34px 34px 30px;
		}
		.hero-panel::after {
			content: "";
			position: absolute;
			inset: auto -120px -140px auto;
			width: 280px;
			height: 280px;
			border-radius: 999px;
			background: radial-gradient(circle, rgba(22, 104, 227, 0.22), transparent 68%);
			pointer-events: none;
		}
		.eyebrow {
			display: inline-flex;
			align-items: center;
			gap: 10px;
			padding: 7px 12px;
			border-radius: 999px;
			background: var(--accent-soft);
			color: var(--accent-strong);
			font-size: 12px;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}
		.hero-panel h1 {
			margin: 18px 0 14px;
			max-width: 11.5ch;
			font-family: "Space Grotesk", sans-serif;
			font-size: clamp(42px, 7vw, 72px);
			line-height: 0.94;
			letter-spacing: -0.06em;
		}
		.hero-panel p {
			max-width: 640px;
			margin: 0;
			font-size: 18px;
			line-height: 1.7;
			color: var(--muted);
		}
		.cta-row {
			display: flex;
			flex-wrap: wrap;
			gap: 12px;
			margin-top: 26px;
		}
		.button {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			gap: 8px;
			padding: 14px 18px;
			border-radius: 16px;
			font-weight: 600;
			font-size: 14px;
			transition: transform 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
		}
		.button:hover {
			transform: translateY(-1px);
		}
		.button-primary {
			background: linear-gradient(140deg, var(--accent), #4a9cff);
			color: white;
			box-shadow: 0 18px 34px rgba(22, 104, 227, 0.28);
		}
		.button-secondary {
			background: rgba(255, 255, 255, 0.82);
			border: 1px solid rgba(19, 34, 56, 0.1);
			color: var(--ink);
		}
		.signal-row {
			display: grid;
			grid-template-columns: repeat(3, minmax(0, 1fr));
			gap: 12px;
			margin-top: 28px;
		}
		.signal {
			padding: 14px 16px;
			border-radius: 18px;
			background: rgba(255, 255, 255, 0.76);
			border: 1px solid rgba(19, 34, 56, 0.08);
		}
		.signal span {
			display: block;
			font-size: 12px;
			font-weight: 600;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: rgba(19, 34, 56, 0.55);
		}
		.signal strong {
			display: block;
			margin-top: 6px;
			font-size: 15px;
			line-height: 1.45;
		}
		.status-panel {
			border-radius: 30px;
			padding: 24px;
			display: flex;
			flex-direction: column;
			gap: 18px;
		}
		.panel-title {
			margin: 0;
			font-family: "Space Grotesk", sans-serif;
			font-size: 24px;
			letter-spacing: -0.04em;
		}
		.panel-subtitle {
			margin: 6px 0 0;
			font-size: 14px;
			color: var(--muted);
			line-height: 1.6;
		}
		.status-grid {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 12px;
		}
		.status-card {
			padding: 14px;
			border-radius: 18px;
			background: rgba(255, 255, 255, 0.88);
			border: 1px solid rgba(19, 34, 56, 0.08);
		}
		.status-card span {
			display: block;
			font-size: 12px;
			font-weight: 600;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: rgba(19, 34, 56, 0.55);
		}
		.status-card strong {
			display: block;
			margin-top: 8px;
			font-size: 16px;
			line-height: 1.35;
		}
		.status-card small {
			display: block;
			margin-top: 6px;
			color: var(--muted);
			font-size: 13px;
			line-height: 1.5;
		}
		.terminal {
			padding: 18px;
			border-radius: 22px;
			background: linear-gradient(180deg, #15253d, #0e1728);
			color: #dce7f7;
			box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
		}
		.terminal-head {
			display: flex;
			align-items: center;
			gap: 8px;
			margin-bottom: 14px;
		}
		.terminal-head span {
			width: 10px;
			height: 10px;
			border-radius: 999px;
			background: rgba(255, 255, 255, 0.28);
		}
		.terminal-head span:nth-child(1) { background: #ff7c66; }
		.terminal-head span:nth-child(2) { background: #f7c24b; }
		.terminal-head span:nth-child(3) { background: #31d387; }
		.terminal-head strong {
			margin-left: 8px;
			font-size: 13px;
			font-weight: 600;
			color: rgba(220, 231, 247, 0.76);
		}
		.terminal pre {
			margin: 0;
			white-space: pre-wrap;
			word-break: break-word;
			font-size: 14px;
			line-height: 1.8;
		}
		.terminal .dim { color: #7f93b5; }
		.section {
			margin-top: 30px;
		}
		.section-header {
			display: flex;
			align-items: flex-end;
			justify-content: space-between;
			gap: 16px;
			margin-bottom: 16px;
		}
		.section-header h2 {
			margin: 0;
			font-family: "Space Grotesk", sans-serif;
			font-size: 28px;
			letter-spacing: -0.04em;
		}
		.section-header p {
			margin: 0;
			max-width: 560px;
			color: var(--muted);
			font-size: 15px;
			line-height: 1.7;
		}
		.grid-three {
			display: grid;
			grid-template-columns: repeat(3, minmax(0, 1fr));
			gap: 16px;
		}
		.surface {
			border-radius: 26px;
			padding: 22px;
		}
		.step-kicker,
		.card-kicker,
		.arch-kicker {
			font-size: 11px;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: rgba(19, 34, 56, 0.5);
		}
		.surface h3,
		.route-card h3,
		.arch-card h3 {
			margin: 12px 0 8px;
			font-size: 20px;
			font-family: "Space Grotesk", sans-serif;
			letter-spacing: -0.03em;
		}
		.surface p,
		.route-card p,
		.arch-card p {
			margin: 0;
			color: var(--muted);
			font-size: 14px;
			line-height: 1.7;
		}
		.command-snippet {
			margin-top: 16px;
			padding: 14px 16px;
			border-radius: 18px;
			background: rgba(19, 34, 56, 0.94);
			color: #ebf2ff;
			font-size: 13px;
			line-height: 1.75;
			overflow-x: auto;
		}
		.starter-grid,
		.route-grid,
		.arch-grid {
			display: grid;
			gap: 16px;
		}
		.starter-grid {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.route-grid {
			grid-template-columns: minmax(0, 1.1fr) minmax(0, 0.9fr);
		}
		.arch-grid {
			grid-template-columns: repeat(4, minmax(0, 1fr));
		}
		.route-card,
		.arch-card {
			border-radius: 24px;
			padding: 22px;
		}
		.endpoint-list,
		.bullet-list {
			margin: 16px 0 0;
			padding: 0;
			list-style: none;
			display: grid;
			gap: 10px;
		}
		.endpoint-list li,
		.bullet-list li {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 12px;
			padding: 12px 14px;
			border-radius: 16px;
			background: rgba(255, 255, 255, 0.84);
			border: 1px solid rgba(19, 34, 56, 0.07);
			font-size: 13px;
		}
		.endpoint-list code,
		.mini-note code {
			padding: 4px 8px;
			border-radius: 10px;
			background: rgba(19, 34, 56, 0.06);
			color: var(--ink);
		}
		.method {
			min-width: 56px;
			padding: 5px 10px;
			border-radius: 999px;
			text-align: center;
			font-size: 11px;
			font-weight: 700;
			letter-spacing: 0.08em;
			text-transform: uppercase;
			background: var(--accent-soft);
			color: var(--accent-strong);
		}
		.arch-card {
			position: relative;
			overflow: hidden;
		}
		.arch-card::after {
			content: "";
			position: absolute;
			inset: auto -18px -18px auto;
			width: 92px;
			height: 92px;
			border-radius: 999px;
			background: radial-gradient(circle, rgba(22, 104, 227, 0.13), transparent 70%);
		}
		.footer {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 18px;
			margin-top: 34px;
			padding-top: 18px;
			border-top: 1px solid var(--line);
			color: var(--muted);
			font-size: 13px;
		}
		.footer a {
			color: var(--ink);
			font-weight: 600;
		}
		.footer-links {
			display: flex;
			flex-wrap: wrap;
			gap: 14px;
		}
		@keyframes rise {
			from { opacity: 0; transform: translateY(14px); }
			to { opacity: 1; transform: translateY(0); }
		}
		.reveal {
			animation: rise 0.55s ease both;
		}
		.reveal-delay-1 { animation-delay: 0.08s; }
		.reveal-delay-2 { animation-delay: 0.16s; }
		.reveal-delay-3 { animation-delay: 0.24s; }
		@media (max-width: 1080px) {
			.hero,
			.grid-three,
			.starter-grid,
			.route-grid,
			.arch-grid {
				grid-template-columns: 1fr;
			}
			.section-header,
			.footer,
			.topbar {
				align-items: flex-start;
				flex-direction: column;
			}
		}
		@media (max-width: 720px) {
			.page { padding: 18px 14px 42px; }
			.hero-panel,
			.status-panel,
			.surface,
			.route-card,
			.arch-card { border-radius: 24px; }
			.hero-panel { padding: 24px; }
			.signal-row,
			.status-grid { grid-template-columns: 1fr; }
			.cta-row { flex-direction: column; }
			.button { width: 100%; }
			.hero-panel h1 { max-width: none; }
		}
	</style>
</head>
<body>
	<div class="page">
		<header class="topbar reveal">
			<div class="brand">
				<div class="brand-mark">ZG</div>
				<div class="brand-copy">
					<strong>{{APP_NAME}}</strong>
					<span>Modular Go API scaffold for real product work</span>
				</div>
			</div>
			<div class="topbar-actions">
				<div class="pill">
					<span class="pill-dot"></span>
					<strong>Running</strong>
					<span>{{APP_ENV_UPPER}}</span>
				</div>
				<a class="pill" href="{{APP_URL}}" target="_blank" rel="noreferrer">
					<strong>Base URL</strong>
					<span>{{APP_URL}}</span>
				</a>
			</div>
		</header>

		<section class="hero">
			<div class="hero-panel reveal reveal-delay-1">
				<div class="eyebrow">Pure Go API scaffold</div>
				<h1>Start shipping backend work without rebuilding auth, API keys, or CLI plumbing.</h1>
				<p>
					This homepage is here to answer the first three developer questions fast:
					did the service boot correctly, what ships by default, and what should I do next.
				</p>
				<div class="cta-row">
					<a href="/swagger/index.html" class="button button-primary">Open Swagger</a>
					<a href="https://github.com/zgiai/zgo" target="_blank" rel="noreferrer" class="button button-secondary">Read GitHub Docs</a>
					<a href="/v1/health" class="button button-secondary">Check Health</a>
				</div>
				<div class="signal-row">
					<div class="signal">
						<span>Default starters</span>
						<strong>User auth, API keys, and audit logging are already wired.</strong>
					</div>
					<div class="signal">
						<span>CLI seam</span>
						<strong>Generation, migration, and AI commands share one manifest surface.</strong>
					</div>
					<div class="signal">
						<span>Extension path</span>
						<strong>Add modules behind starter manifests instead of editing five boot files.</strong>
					</div>
				</div>
			</div>

			<aside class="status-panel reveal reveal-delay-2">
				<div>
					<h2 class="panel-title">Current runtime</h2>
					<p class="panel-subtitle">Useful context for the first 30 seconds after boot. This should feel like a developer control card, not marketing copy.</p>
				</div>
				<div class="status-grid">
					<div class="status-card">
						<span>Environment</span>
						<strong>{{APP_ENV}}</strong>
						<small>Matches the currently loaded app config.</small>
					</div>
					<div class="status-card">
						<span>Go runtime</span>
						<strong>{{GO_VERSION}}</strong>
						<small>Reported by the running process.</small>
					</div>
					<div class="status-card">
						<span>AI capability</span>
						<strong>{{AI_PROVIDER}}</strong>
						<small>Default model: {{AI_MODEL}}</small>
					</div>
					<div class="status-card">
						<span>Default scaffold</span>
						<strong>3 starters</strong>
						<small>User auth, API key access, audit history, and AI-ready core infra.</small>
					</div>
				</div>
				<div class="terminal">
					<div class="terminal-head">
						<span></span><span></span><span></span>
						<strong>First useful commands</strong>
					</div>
					<pre><span class="dim"># project bootstrap</span>
cp .env.example .env
make wire
go run ./cmd/server

<span class="dim"># explore the scaffold</span>
go run ./cmd/zgo route:list
go run ./cmd/zgo migrate
go run ./cmd/zgo ai:chat "Summarize this scaffold"</pre>
				</div>
			</aside>
		</section>

		<section class="section reveal reveal-delay-1">
			<div class="section-header">
				<h2>Get productive in three moves</h2>
				<p>首页不该把所有能力平铺出来。对新启动的项目，最关键的是确认服务、探索接口、然后开始改代码。</p>
			</div>
			<div class="grid-three">
				<article class="surface">
					<div class="step-kicker">Step 01</div>
					<h3>Verify the service</h3>
					<p>先确认服务真的在跑，环境值加载正常，数据库和中间件链没有阻断启动。</p>
					<div class="command-snippet">GET {{APP_URL}}/v1/health</div>
				</article>
				<article class="surface">
					<div class="step-kicker">Step 02</div>
					<h3>Explore the default API</h3>
					<p>默认 scaffold 已经带了认证、API key 管理和全局审计。优先通过 Swagger 看清现成 surface，再决定加什么 starter。</p>
					<div class="command-snippet">open {{APP_URL}}/swagger/index.html</div>
				</article>
				<article class="surface">
					<div class="step-kicker">Step 03</div>
					<h3>Generate the next module</h3>
					<p>从完整模块脚手架起步，而不是零散补文件。单文件生成命令只用来修复现有模块，不用来起新模块。</p>
					<div class="command-snippet">go run ./cmd/zgo make:module Invoice</div>
				</article>
			</div>
		</section>

		<section class="section reveal reveal-delay-2">
			<div class="section-header">
				<h2>What ships in the default scaffold</h2>
				<p>这里展示的是现在真正可用的 product surface，不再混入已经被移除的部署平台或控制面业务。</p>
			</div>
			<div class="starter-grid">
				<article class="surface">
					<div class="card-kicker">Starter</div>
					<h3>User auth</h3>
					<p>注册、登录、JWT 鉴权、当前用户资料和密码相关流程。适合作为几乎所有后台项目的最小业务起点。</p>
				</article>
				<article class="surface">
					<div class="card-kicker">Starter</div>
					<h3>API key access</h3>
					<p>机器访问场景的 key 创建、列表、撤销和中间件组。默认 scaffold 已经能服务自动化调用和服务到服务请求。</p>
				</article>
				<article class="surface">
					<div class="card-kicker">Starter</div>
					<h3>Audit logging</h3>
					<p>默认对写请求做全局审计，并给当前用户提供历史查询接口。这样安全回溯不再是业务项目后补的一层。</p>
				</article>
			</div>
		</section>

		<section class="section reveal reveal-delay-3">
			<div class="section-header">
				<h2>Know the shape of the app</h2>
				<p>这是给 PM、Tech Lead 和新开发者的架构地图。首页应该帮助人形成 mental model，而不是只堆 feature badge。</p>
			</div>
			<div class="arch-grid">
				<article class="arch-card">
					<div class="arch-kicker">Core</div>
					<h3>Runtime and infra</h3>
					<p>Bootstrap、router、response、migration、testing 和 shared packages. 这些是每个 ZGO 应用都会用到的底层能力。</p>
				</article>
				<article class="arch-card">
					<div class="arch-kicker">Starter</div>
					<h3>Business-ready defaults</h3>
					<p><code>user</code>、<code>apikey</code> 和 <code>audit</code> 默认装配到新应用。它们通过 starter manifest 贡献模块、迁移和 seed。</p>
				</article>
				<article class="arch-card">
					<div class="arch-kicker">Capability</div>
					<h3>Reusable technical seams</h3>
					<p>像 AI 这种技术能力留在 capability 层，避免把外部集成和业务对象混成一个宽模块。</p>
				</article>
				<article class="arch-card">
					<div class="arch-kicker">Optional starter</div>
					<h3>Keep expansion explicit</h3>
					<p>RBAC 一类增强模块可以保留，但不会默认侵入 scaffold 主链。这样主仓库保持纯净，扩展也更诚实。</p>
				</article>
			</div>
		</section>

		<section class="section reveal reveal-delay-3">
			<div class="route-grid">
				<article class="route-card">
					<div class="card-kicker">Default endpoints</div>
					<h3>Ready-to-call routes</h3>
					<p>这些是启动后最值得先验证的接口，能快速确认默认 starter 的行为面是否健康。</p>
					<ul class="endpoint-list">
						<li><span class="method">POST</span><code>/v1/register</code></li>
						<li><span class="method">POST</span><code>/v1/login</code></li>
						<li><span class="method">GET</span><code>/v1/users/profile</code></li>
						<li><span class="method">GET</span><code>/v1/api-keys</code></li>
						<li><span class="method">GET</span><code>/v1/audit-logs</code></li>
						<li><span class="method">GET</span><code>/v1/health</code></li>
					</ul>
				</article>
				<article class="route-card">
					<div class="card-kicker">Direct links</div>
					<h3>Fast paths for the next click</h3>
					<p>把真正有决策价值的入口放在这里。不是所有链接都要上首页，只有最可能成为下一步动作的才应该出现。</p>
					<ul class="bullet-list">
						<li><a href="/swagger/index.html">Swagger explorer</a><code>/swagger/index.html</code></li>
						<li><a href="/v1/health">Health endpoint</a><code>/v1/health</code></li>
						<li><a href="https://github.com/zgiai/zgo" target="_blank" rel="noreferrer">GitHub repository</a><code>README + source</code></li>
						<li><span>CLI route listing</span><code>go run ./cmd/zgo route:list</code></li>
					</ul>
				</article>
			</div>
		</section>

		<footer class="footer reveal reveal-delay-3">
			<div>{{APP_NAME}} is a modular Go API scaffold focused on fast product starts and clean architecture seams.</div>
			<div class="footer-links">
				<a href="https://github.com/zgiai/zgo" target="_blank" rel="noreferrer">GitHub</a>
				<a href="/swagger/index.html">Swagger</a>
				<a href="/v1/health">Health</a>
			</div>
		</footer>
	</div>
</body>
</html>`

// RegisterWelcome registers the welcome page route.
func RegisterWelcome(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		appName := "ZGO"
		appEnv := "development"
		appURL := "http://localhost:8025"
		aiProvider := "openai"
		aiModel := "gpt-5.4"

		if config.GlobalConfig != nil {
			appName = config.GlobalConfig.App.Name
			appEnv = config.GlobalConfig.App.Env
			if config.GlobalConfig.App.URL != "" {
				appURL = config.GlobalConfig.App.URL
			} else if config.GlobalConfig.Server.Port > 0 {
				appURL = "http://localhost:" + strconv.Itoa(config.GlobalConfig.Server.Port)
			}

			if config.GlobalConfig.AI.Enabled {
				if config.GlobalConfig.AI.DefaultProvider != "" {
					aiProvider = config.GlobalConfig.AI.DefaultProvider
				}
				if config.GlobalConfig.AI.DefaultModel != "" {
					aiModel = config.GlobalConfig.AI.DefaultModel
				}
			} else {
				aiProvider = "disabled"
				aiModel = "not configured"
			}
		}

		html := strings.NewReplacer(
			"{{APP_NAME}}", appName,
			"{{APP_ENV}}", appEnv,
			"{{APP_ENV_UPPER}}", strings.ToUpper(appEnv),
			"{{APP_URL}}", appURL,
			"{{GO_VERSION}}", runtime.Version(),
			"{{AI_PROVIDER}}", aiProvider,
			"{{AI_MODEL}}", aiModel,
		).Replace(welcomeHTML)

		c.Data(200, "text/html; charset=utf-8", []byte(html))
	})
}
