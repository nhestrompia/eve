import Link from 'next/link';
import Image from 'next/image';

export default function HomePage() {
  return (
    <main className="eve-home">
      <div className="eve-home-shell">
        <nav className="eve-home-nav" aria-label="Primary">
          <Link className="eve-home-brand" href="/" aria-label="eve home">
            <Image src="/eve.svg" alt="" width={96} height={40} unoptimized priority />
          </Link>
          <div className="eve-nav-links">
            <Link href="/docs">Docs</Link>
            <a href="https://github.com/nhestrompia/eve">GitHub</a>
          </div>
        </nav>

        <section className="eve-hero">
          <div className="eve-hero-copy">
            <p className="eve-kicker">git tracks code, eve tracks product</p>
            <h1>Product memory that lives with the repo.</h1>
            <p>
              eve records the completed product change beside Git: what changed for users, why it
              changed, how it was verified, and which artifacts prove the result.
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

          <div className="eve-product-surface" aria-label="Example eve record">
            <div className="eve-surface-bar">
              <span>snapshot</span>
              <code>.eve/snapshots/snap_123.json</code>
            </div>
            <div className="eve-surface-grid">
              <div className="eve-surface-cell eve-surface-cell-large">
                <span>product change</span>
                <strong>Users can sign in with GitHub.</strong>
              </div>
              <div className="eve-surface-cell">
                <span>validation</span>
                <strong>passed</strong>
                <code>go test ./...</code>
              </div>
              <div className="eve-surface-cell">
                <span>decision</span>
                <strong>Keep Git as implementation truth.</strong>
              </div>
              <div className="eve-surface-cell">
                <span>implementation</span>
                <strong>branch main</strong>
                <code>gitState abc123</code>
              </div>
            </div>
          </div>
        </section>

        <section className="eve-bands" aria-label="eve value">
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
