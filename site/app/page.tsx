import Link from 'next/link';

export default function HomePage() {
  return (
    <main className="eve-home">
      <div className="eve-home-shell">
        <nav className="eve-home-nav" aria-label="Primary">
          <div className="eve-wordmark">EVE</div>
          <div className="eve-nav-links">
            <Link href="/docs">Docs</Link>
            <a href="https://github.com/nhestrompia/eve">GitHub</a>
          </div>
        </nav>

        <section className="eve-hero">
          <div>
            <p className="eve-kicker">Product history for agentic development</p>
            <h1>Explain the change, not just the diff.</h1>
            <p>
              EVE records completed product snapshots beside Git commits. It keeps the user-visible
              change, decisions, risks, validation, artifacts, and implementation facts in the repository
              so humans and agents can understand what shipped.
            </p>
            <div className="eve-actions">
              <Link className="eve-button" data-tone="primary" href="/docs/guides/get-started">
                Get started
              </Link>
              <Link className="eve-button" href="/docs/agents/agent-guide">
                Agent guide
              </Link>
            </div>
          </div>

          <div className="eve-record" aria-label="Example EVE record">
            <div className="eve-record-header">
              <span>.eve/snapshots/snap_123.json</span>
              <span>passed</span>
            </div>
            <div className="eve-record-body">
              <div className="eve-field">
                <span>Title</span>
                <strong>Add GitHub OAuth</strong>
              </div>
              <div className="eve-field">
                <span>User-visible change</span>
                <strong>The login screen now includes a GitHub sign-in option.</strong>
              </div>
              <div className="eve-field">
                <span>Validation</span>
                <code>go test ./...</code>
              </div>
              <div className="eve-field">
                <span>Implementation</span>
                <code>branch main, gitState abc123</code>
              </div>
            </div>
          </div>
        </section>

        <section className="eve-bands" aria-label="EVE value">
          <article className="eve-band">
            <h2>Repository-native</h2>
            <p>Canonical records live in `.eve/`, travel with the code, and remain reviewable in GitHub.</p>
          </article>
          <article className="eve-band">
            <h2>Agent-friendly</h2>
            <p>Agents can complete snapshots over MCP or CLI with structured fields instead of prose-only logs.</p>
          </article>
          <article className="eve-band">
            <h2>Verification-first</h2>
            <p>Every product record can include the commands, outcomes, and artifacts used to prove the change.</p>
          </article>
        </section>
      </div>
    </main>
  );
}
