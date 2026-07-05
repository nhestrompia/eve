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
            <p className="eve-kicker">open source product history</p>
            <h1>git tracks code, eve tracks product.</h1>
            <p className="eve-hero-lede">
              eve adds a small, reviewable record for every completed change: what shipped, why it
              mattered, how it was verified, and which Git state implemented it.
            </p>
            <p className="eve-hero-note">
              The record lives in your repository under <code>.eve/</code>. It moves through forks,
              reviews, releases, and agent sessions with the code.
            </p>
            <div className="eve-actions">
              <Link className="eve-button" data-tone="primary" href="/docs/guides/get-started">
                Read the docs
              </Link>
              <Link className="eve-button" href="/docs/agents/agent-guide">
                Agent guide
              </Link>
              <a className="eve-button" href="https://github.com/nhestrompia/eve">
                GitHub
              </a>
            </div>
          </div>

          <aside className="eve-specimen" aria-label="Example eve snapshot">
            <div className="eve-specimen-head">
              <span>snapshot</span>
              <code>.eve/snapshots/snap_123.json</code>
            </div>
            <dl className="eve-specimen-list">
              <div>
                <dt>product change</dt>
                <dd>Users can sign in with GitHub.</dd>
              </div>
              <div>
                <dt>user-visible change</dt>
                <dd>The login screen includes a GitHub sign-in option.</dd>
              </div>
              <div>
                <dt>validation</dt>
                <dd>
                  <code>go test ./...</code>
                  <span>passed</span>
                </dd>
              </div>
              <div>
                <dt>implementation</dt>
                <dd>
                  <code>branch main</code>
                  <code>gitState abc123</code>
                </dd>
              </div>
            </dl>
          </aside>
        </section>

        <section className="eve-proof" aria-label="Why eve exists">
          <div className="eve-section-heading">
            <span>why it exists</span>
            <h2>Implementation moves fast. Product context should not disappear.</h2>
          </div>
          <div className="eve-proof-rows">
            <article>
              <span>01</span>
              <h3>Git answers what changed in files.</h3>
              <p>eve answers what changed in the product, which decision mattered, and what evidence proved the result.</p>
            </article>
            <article>
              <span>02</span>
              <h3>Agents need durable context.</h3>
              <p>Snapshots give agents structured local memory before they edit a feature and a standard record after they finish.</p>
            </article>
            <article>
              <span>03</span>
              <h3>Reviews need evidence.</h3>
              <p>Validation commands, screenshots, logs, URLs, and related snapshots stay attached to the completed product change.</p>
            </article>
          </div>
        </section>

        <section className="eve-start" aria-label="Start using eve">
          <div>
            <span>from a fork</span>
            <h2>Run the product, not the docs site.</h2>
            <p>
              Normal contributors only need the CLI and local product UI. The Fumadocs site is for documentation work and Vercel deployment.
            </p>
          </div>
          <pre aria-label="Local eve commands"><code>{`npm --prefix ui ci
npm --prefix ui run build
go run ./cmd/eve init
go run ./cmd/eve dev`}</code></pre>
        </section>
      </div>
    </main>
  );
}
