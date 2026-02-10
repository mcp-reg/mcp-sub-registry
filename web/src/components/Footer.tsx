export function Footer() {
  return (
    <footer className="border-t border-[#283339] py-8 mt-12">
      <div className="max-w-[1200px] mx-auto px-6 flex justify-between items-center text-sm text-slate-500">
        <span>MCP Registry</span>
        <a
          className="hover:text-primary transition-colors"
          href="https://github.com/mcp-reg/mcp-sub-registry"
          target="_blank"
          rel="noopener noreferrer"
        >
          GitHub
        </a>
      </div>
    </footer>
  )
}
