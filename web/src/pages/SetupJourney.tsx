import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Header } from "@/components/Header"
import { Footer } from "@/components/Footer"
import { CopyButton } from "@/components/CopyButton"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface SetupJourneyProps {
  org?: string
  repo?: string
}

export function SetupJourney({ org = "your-org", repo = "mcp-registry" }: SetupJourneyProps) {
  const baseUrl = window.location.origin
  const navigate = useNavigate()
  const [url, setUrl] = useState("")
  const [error, setError] = useState("")

  // Parse GitHub URL to extract org/repo
  // Supports: github.com/org/repo, https://github.com/org/repo, www.github.com/org/repo, org/repo
  function parseGitHubUrl(input: string): { org: string; repo: string } | null {
    const trimmed = input.trim()
    if (!trimmed) return null
    
    // Remove protocol and www if present
    let path = trimmed
      .replace(/^https?:\/\//, "")
      .replace(/^www\./, "")
    
    // Check if it starts with github.com
    if (path.startsWith("github.com/")) {
      path = path.replace("github.com/", "")
    } else if (path.includes("/") && !path.includes(".")) {
      // Looks like org/repo format (no dots, has slash)
      // Allow it
    } else if (!path.startsWith("github.com")) {
      // Must be github.com domain
      return null
    }
    
    // Remove trailing slashes and .git
    path = path.replace(/\/+$/, "").replace(/\.git$/, "")
    
    // Split and validate
    const parts = path.split("/").filter(Boolean)
    if (parts.length >= 2) {
      return { org: parts[0], repo: parts[1] }
    }
    
    return null
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")
    
    const parsed = parseGitHubUrl(url)
    if (!parsed) {
      setError("Please enter a valid GitHub URL (e.g., github.com/org/repo)")
      return
    }
    
    navigate(`/${parsed.org}/${parsed.repo}`)
  }

  function handleExampleClick() {
    navigate("/mcp-reg/mcp-registry-template")
  }

  return (
    <div className="dark min-h-screen bg-background-dark text-slate-100 font-display">
      <Header />
      <main className="max-w-[960px] mx-auto px-6 py-12">
        {/* Intro: What / Who / Why (compact) */}
        <section className="mb-8">
          <div className="py-8 px-6 rounded-2xl bg-primary/5 border border-primary/20">
            <div className="flex flex-wrap items-center gap-2 mb-5">
              <span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/10 text-primary text-sm font-medium">
                <span className="material-symbols-outlined text-base">auto_awesome</span>
                Free & Open Source
              </span>
            </div>
            <div className="grid gap-6 sm:grid-cols-3 text-left">
              <div>
                <h2 className="text-slate-300 font-bold text-xs uppercase tracking-wider mb-1.5">What is this?</h2>
                <p className="text-slate-400 text-sm leading-relaxed">
                  A self-hosted MCP registry that aggregates public registries and your private servers into one unified marketplace and API.
                </p>
              </div>
              <div>
                <h2 className="text-slate-300 font-bold text-xs uppercase tracking-wider mb-1.5">Who is it for?</h2>
                <p className="text-slate-400 text-sm leading-relaxed">
                  Teams and organizations that want a single, curated MCP catalog and control over what developers see in the IDE.
                </p>
              </div>
              <div>
                <h2 className="text-slate-300 font-bold text-xs uppercase tracking-wider mb-1.5">Why use it?</h2>
                <p className="text-slate-400 text-sm leading-relaxed">
                  No signup, no login, free. GitOps-driven: your repo is the source of truth. One place for public and private MCP servers.
                </p>
              </div>
            </div>
            <p className="text-slate-400 text-sm mt-4 pt-4 border-t border-slate-700/50">
              The service is open source — run it in your own environment:{" "}
              <a
                href="https://github.com/mcp-reg/mcp-sub-registry"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                mcp-reg/mcp-sub-registry
              </a>
            </p>
          </div>
        </section>

        {/* Get started: Step 1 + Step 2 (separate section) */}
        <section className="mb-16">
          <h2 className="text-lg font-bold text-slate-200 mb-4">Get started</h2>
          <div className="grid gap-6 sm:grid-cols-2">
            {/* Step 1 */}
            <div className="rounded-xl border border-slate-700 bg-slate-900/50 p-6 flex flex-col">
              <span className="text-primary font-bold text-xs uppercase tracking-wider">Step 1</span>
              <h3 className="text-lg font-bold text-slate-200 mt-1 mb-2">Copy the template</h3>
              <p className="text-slate-400 text-sm leading-relaxed flex-1">
                Create a repo from our template so you have a registry config and place for private server definitions.
              </p>
              <a
                href="https://github.com/mcp-reg/mcp-registry-template/generate"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 mt-4 px-4 py-2.5 rounded-lg bg-primary text-white font-bold text-sm hover:bg-primary/90 transition-colors w-fit"
              >
                Use this template
                <span className="material-symbols-outlined text-lg">open_in_new</span>
              </a>
            </div>

            {/* Step 2 */}
            <div className="rounded-xl border border-slate-700 bg-slate-900/50 p-6 flex flex-col">
              <span className="text-primary font-bold text-xs uppercase tracking-wider">Step 2</span>
              <h3 className="text-lg font-bold text-slate-200 mt-1 mb-2">View your registry</h3>
              <p className="text-slate-400 text-sm leading-relaxed mb-4">
                Paste your GitHub repo URL (or org/repo) to open the registry.
              </p>
              <form onSubmit={handleSubmit} className="flex flex-col gap-3">
                <div className="relative">
                  <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 text-lg">
                    link
                  </span>
                  <Input
                    type="text"
                    value={url}
                    onChange={(e) => {
                      setUrl(e.target.value)
                      setError("")
                    }}
                    placeholder="github.com/your-org/your-registry"
                    className="pl-10 h-11 text-sm bg-slate-800 border-slate-600 focus:border-primary"
                  />
                </div>
                <Button type="submit" className="h-11 font-bold">
                  <span className="flex items-center gap-2">
                    View Registry
                    <span className="material-symbols-outlined text-lg">arrow_forward</span>
                  </span>
                </Button>
                {error && (
                  <p className="text-red-400 text-sm">{error}</p>
                )}
              </form>
              <p className="text-slate-500 text-xs mt-2">
                Try an example:{" "}
                <button
                  type="button"
                  onClick={handleExampleClick}
                  className="text-primary hover:underline"
                >
                  mcp-reg/mcp-registry-template
                </button>
              </p>
            </div>
          </div>
        </section>

        <p className="text-slate-500 text-sm mb-6">
          After you&apos;ve created your repo and opened it here, follow the setup journey below.
        </p>

        {/* Steps Header */}
        <div className="flex items-center justify-between border-b border-[#283339] pb-6 mb-10">
          <div>
            <h2 className="text-2xl font-bold">Registry Setup Journey</h2>
            <p className="text-slate-500 mt-1">Complete these 5 steps to establish your private server ecosystem.</p>
          </div>
          <div className="hidden sm:flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-primary bg-primary/10 px-3 py-1 rounded-full">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
            </span>
            Step-by-Step
          </div>
        </div>

        {/* Step 1 */}
        <div className="space-y-24 pb-20">
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-primary font-bold text-sm uppercase tracking-wider">Step 01</span>
              <h3 className="text-2xl font-bold leading-tight">Create from Template</h3>
              <p className="text-slate-400 leading-relaxed">
                Create a repository for your organization. This GitHub repo will act as your registry's database and control plane.
              </p>
            </div>
            <p className="text-slate-400 leading-relaxed">
              <a
                href="https://github.com/mcp-reg/mcp-registry-template/generate"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline"
              >
                Use this template →
              </a>
            </p>
          </div>

          {/* Step 2 */}
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-primary font-bold text-sm uppercase tracking-wider">Step 02</span>
              <h3 className="text-2xl font-bold leading-tight">Define Your Internal Server</h3>
              <p className="text-slate-400 leading-relaxed">
                Create a <code className="bg-[#283339] px-1.5 py-0.5 rounded text-sm font-mono">mcps/my-server/server.json</code> file with your MCP server definition.
              </p>
            </div>
            <div className="bg-slate-900 rounded-xl p-5 border border-slate-800">
              <div className="flex justify-between items-center mb-4">
                <span className="text-xs font-mono text-slate-500 uppercase tracking-widest">mcps/my-server/server.json</span>
                <CopyButton text={`{
  "name": "${org}/my-server",
  "description": "My Internal MCP Server",
  "version": "1.0.0",
  "remotes": [
    {
      "type": "streamable-http",
      "url": "https://your-service.example.com/mcp"
    }
  ]
}`} />
              </div>
              <pre className="text-sm text-slate-300 leading-relaxed overflow-x-auto">
                <code>{`{
  "name": "${org}/my-server",
  "description": "My Internal MCP Server",
  "version": "1.0.0",
  "remotes": [
    {
      "type": "streamable-http",
      "url": "https://your-service.example.com/mcp"
    }
  ]
}`}</code>
              </pre>
            </div>
            <p className="text-xs text-slate-500">
              The <code className="text-primary">name</code> must follow pattern: <code className="text-slate-400">org/server-name</code>
            </p>
          </div>

          {/* Step 3 */}
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-primary font-bold text-sm uppercase tracking-wider">Step 03</span>
              <h3 className="text-2xl font-bold leading-tight">Configure Your Registry</h3>
              <p className="text-slate-400 leading-relaxed">
                Update <code className="bg-[#283339] px-1.5 py-0.5 rounded text-sm font-mono">registry.json</code> at root. Mix internal servers with public ones from GitHub's MCP registry.
              </p>
            </div>
            <div className="bg-slate-900 rounded-xl p-5 border border-slate-800">
              <div className="flex justify-between items-center mb-4">
                <span className="text-xs font-mono text-slate-500 uppercase tracking-widest">registry.json</span>
                <CopyButton text={`{
  "registries": [
    {
      "name": "private",
      "type": "private",
      "servers_relative_path": [
        "servers/my-server/server.json"
      ]
    },
    {
      "name": "GitHub Copilot MCP Registry",
      "url": "https://api.mcp.github.com",
      "servers": {
        "microsoft/markitdown": "latest"
      }
    }
  ]
}`} />
              </div>
              <pre className="text-sm text-slate-300 leading-relaxed overflow-x-auto">
                <code>{`{
  "registries": [
    {
      "name": "private",
      "type": "private",
      "servers_relative_path": [
        "servers/my-server/server.json"
      ]
    },
    {
      "name": "GitHub Copilot MCP Registry",
      "url": "https://api.mcp.github.com",
      "servers": {
        "microsoft/markitdown": "latest"
      }
    }
  ]
}`}</code>
              </pre>
            </div>
          </div>

          {/* Step 4 */}
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-primary font-bold text-sm uppercase tracking-wider">Step 04</span>
              <h3 className="text-2xl font-bold leading-tight">Configure GitHub Copilot</h3>
              <p className="text-slate-400 leading-relaxed">
                Enable discovery in the IDE. Your developers only see what you approve.
              </p>
            </div>
            <div className="bg-[#1a242a] rounded-xl border border-[#283339] p-8">
              <div className="max-w-[600px] mx-auto space-y-6">
                <div className="space-y-2">
                  <label className="text-xs font-bold text-slate-500 uppercase tracking-widest">
                    Settings &gt; Custom MCP Registries
                  </label>
                  <div className="flex flex-col sm:flex-row gap-3">
                    <div className="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-4 py-2.5 text-sm text-slate-300 font-mono flex items-center overflow-hidden shadow-sm">
                      <span className="truncate">{baseUrl}/{org}/{repo}</span>
                    </div>
                    <CopyButton
                      text={`${baseUrl}/${org}/${repo}`}
                      className="bg-primary hover:bg-primary/90 text-white px-4 py-2 rounded-lg text-sm font-bold whitespace-nowrap transition-all shadow-md shadow-primary/10"
                    />
                  </div>
                </div>
                <div className="flex items-start gap-4 p-4 bg-primary/5 rounded-lg border border-primary/10">
                  <span className="material-symbols-outlined text-primary mt-0.5">help_outline</span>
                  <p className="text-[12px] text-slate-400 leading-relaxed italic">
                    Use this URL in the VS Code 'Settings' panel under MCP Extension. This connects your private index to the developer environment.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Step 5 */}
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-primary font-bold text-sm uppercase tracking-wider">Step 05</span>
              <h3 className="text-2xl font-bold leading-tight">Team View</h3>
              <p className="text-slate-400">
                Your developers now have a centralized, searchable view of all approved MCP servers directly in their IDE.
              </p>
            </div>
            {/* VSCode Mockup */}
            <div className="bg-[#1e1e1e] rounded-xl border border-slate-700 overflow-hidden shadow-2xl flex flex-col h-[520px]">
              {/* Title Bar */}
              <div className="bg-[#323233] h-9 flex items-center justify-center relative select-none">
                <div className="absolute left-3 flex gap-2">
                  <div className="size-3 rounded-full bg-[#ff5f56]"></div>
                  <div className="size-3 rounded-full bg-[#ffbd2e]"></div>
                  <div className="size-3 rounded-full bg-[#27c93f]"></div>
                </div>
                <span className="text-xs text-slate-400">Visual Studio Code — registry.json — mcp-registry</span>
              </div>
              <div className="flex flex-1 overflow-hidden">
                {/* Activity Bar */}
                <div className="w-12 bg-[#333333] flex flex-col items-center py-4 gap-4 text-slate-400">
                  <span className="material-symbols-outlined text-2xl">file_copy</span>
                  <span className="material-symbols-outlined text-2xl">search</span>
                  <span className="material-symbols-outlined text-2xl">account_tree</span>
                  <span className="material-symbols-outlined text-2xl">bug_report</span>
                  <div className="relative">
                    <span className="material-symbols-outlined text-2xl text-white">extension</span>
                    <div className="absolute -top-1 -right-1 size-4 bg-primary rounded-full flex items-center justify-center text-[8px] font-bold text-white">3</div>
                  </div>
                </div>
                {/* Sidebar */}
                <div className="w-64 bg-[#252526] flex flex-col border-r border-[#1e1e1e]">
                  <div className="px-4 py-3 flex items-center justify-between text-[11px] font-bold uppercase tracking-wider text-slate-400">
                    <span>Extensions: Marketplace</span>
                    <span className="material-symbols-outlined text-sm">more_horiz</span>
                  </div>
                  <div className="px-3 mb-4">
                    <div className="bg-[#3c3c3c] rounded px-2 py-1.5 flex items-center gap-2 border border-primary/50">
                      <span className="text-xs text-primary font-bold">@mcp</span>
                      <div className="flex-1 h-3 border-l border-slate-500 ml-1"></div>
                    </div>
                  </div>
                  <div className="flex flex-col">
                    <div className="px-4 py-1.5 bg-[#37373d] text-[11px] font-bold flex items-center gap-1 select-none text-slate-300">
                      <span className="material-symbols-outlined text-sm">expand_more</span>
                      <span>{org.toUpperCase()} REGISTRY (3)</span>
                    </div>
                    {/* Selected Extension */}
                    <div className="bg-[#094771]/40 flex items-start gap-3 p-3 border-l-2 border-primary">
                      <div className="size-9 bg-primary/20 rounded flex items-center justify-center shrink-0">
                        <span className="material-symbols-outlined text-primary text-xl">database</span>
                      </div>
                      <div className="overflow-hidden">
                        <h4 className="text-[12px] font-bold text-white truncate leading-tight">Corporate KB</h4>
                        <p className="text-[11px] text-slate-400 truncate leading-tight">Access company docs via LLM</p>
                        <p className="text-[9px] text-primary/80 mt-1 font-bold">Verified Organization</p>
                      </div>
                    </div>
                    {/* Other Extensions */}
                    <div className="flex items-start gap-3 p-3 hover:bg-[#2a2d2e] cursor-pointer">
                      <div className="size-9 bg-emerald-600/20 rounded flex items-center justify-center shrink-0">
                        <span className="material-symbols-outlined text-emerald-400 text-xl">analytics</span>
                      </div>
                      <div className="overflow-hidden">
                        <h4 className="text-[12px] font-bold text-white truncate leading-tight">BI Data Tool</h4>
                        <p className="text-[11px] text-slate-400 truncate leading-tight">Natural language SQL queries</p>
                        <p className="text-[9px] text-slate-500 mt-1">{org} • v1.2.0</p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 p-3 hover:bg-[#2a2d2e] cursor-pointer">
                      <div className="size-9 bg-amber-600/20 rounded flex items-center justify-center shrink-0">
                        <span className="material-symbols-outlined text-amber-400 text-xl">webhook</span>
                      </div>
                      <div className="overflow-hidden">
                        <h4 className="text-[12px] font-bold text-white truncate leading-tight">CI/CD Manager</h4>
                        <p className="text-[11px] text-slate-400 truncate leading-tight">Trigger Jenkins/GitHub workflows</p>
                        <p className="text-[9px] text-slate-500 mt-1">{org} • v0.8.4</p>
                      </div>
                    </div>
                  </div>
                </div>
                {/* Main Editor Area */}
                <div className="flex-1 bg-[#1e1e1e] flex flex-col overflow-y-auto">
                  <div className="h-9 bg-[#252526] flex border-b border-[#1e1e1e]">
                    <div className="bg-[#1e1e1e] px-4 flex items-center gap-2 text-[12px] text-white border-t border-primary">
                      <span className="material-symbols-outlined text-sm text-primary">extension</span>
                      <span>Extension: Corporate KB</span>
                      <span className="material-symbols-outlined text-[14px] text-slate-400 hover:text-white cursor-pointer ml-2">close</span>
                    </div>
                  </div>
                  <div className="p-6 flex-1">
                    <div className="flex items-start gap-4 mb-6">
                      <div className="size-16 bg-primary/20 rounded-lg flex items-center justify-center">
                        <span className="material-symbols-outlined text-primary text-4xl">database</span>
                      </div>
                      <div>
                        <h3 className="text-xl font-bold text-white mb-1">Corporate KB</h3>
                        <p className="text-sm text-slate-400 mb-2">Access company docs via LLM</p>
                        <div className="flex items-center gap-2">
                          <span className="bg-primary text-white text-[10px] px-2 py-0.5 rounded font-bold">INSTALLED</span>
                          <span className="text-[11px] text-slate-500">v2.1.0</span>
                        </div>
                      </div>
                    </div>
                    <div className="border-t border-[#3c3c3c] pt-4">
                      <p className="text-sm text-slate-300 leading-relaxed">
                        Connect your LLM to the corporate knowledge base. Query internal documentation,
                        policies, and procedures using natural language.
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
      <Footer />
    </div>
  )
}
