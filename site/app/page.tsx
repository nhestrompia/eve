import Image from 'next/image';
import Link from 'next/link';

const comparisonRows = [
  ['What changed in files', 'Yes', 'Why it changed and what it means'],
  ['Commits and diffs', 'Yes', 'Plans, decisions, and snapshots'],
  ['Implementation history', 'Yes', 'Product history with verification'],
  ['Hard to answer why', 'Yes', 'Reproducible reasoning'],
  ['No verification context', 'Yes', 'Checks, evidence, and links'],
];

const steps = [
  {
    icon: 'file',
    title: 'Plan',
    text: 'Agent proposes a plan with scope, checks, and acceptance criteria.',
  },
  {
    icon: 'code',
    title: 'Implementation',
    text: 'Agent implements the plan in your repository.',
  },
  {
    icon: 'data',
    title: 'Snapshot',
    text: 'EVE captures the change, runs checks, and records evidence.',
  },
  {
    icon: 'review',
    title: 'Review',
    text: 'You review the snapshot, request changes, or approve.',
  },
];

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
            <Link className="eve-nav-cta" href="/docs/guides/get-started">
              Get started <span aria-hidden="true">-&gt;</span>
            </Link>
          </div>
        </nav>

        <section className="eve-hero">
          <div className="eve-hero-copy">
            <p className="eve-kicker">OPEN SOURCE PRODUCT HISTORY</p>
            <h1>
              Context for your code. History for your <span>product.</span>
            </h1>
            <p className="eve-hero-lede">
              EVE records what changed in your product, why it changed, how it was verified, and
              which Git state implemented it. Every snapshot is committed to your repository under{' '}
              <code>.eve/</code>.
            </p>
            <div className="eve-command" aria-label="Install command">
              <code>
                <span>$</span> npx eve init
              </code>
              <span aria-hidden="true">copy</span>
            </div>
            <div className="eve-actions">
              <a href="https://github.com/nhestrompia/eve">
                View on GitHub <span aria-hidden="true">-&gt;</span>
              </a>
              <Link href="/docs">
                Read the docs <span aria-hidden="true">-&gt;</span>
              </Link>
            </div>
          </div>

          <aside className="eve-hero-visual" aria-label="Animated EVE product-history flow">
            <p className="eve-flow-summary">
              EVE links snapshots, plans, repository state, and Git commits under .eve.
            </p>
            <div className="eve-visual-grid" aria-hidden="true">
              <div className="eve-stack">
                <div className="eve-stack-card eve-snapshot-card">
                  <div className="eve-card-title">
                    <span className="eve-card-icon" data-icon="snapshot" />
                    <div>
                      <span>Snapshot</span>
                      <code>abc123def</code>
                    </div>
                  </div>
                  <p>Users can sign in with GitHub</p>
                  <div className="eve-card-meta">
                    <span>codex</span>
                    <span>2h ago</span>
                  </div>
                </div>
                <div className="eve-stack-card eve-plan-card">
                  <span className="eve-card-icon" data-icon="plan" />
                  <span>Plans</span>
                </div>
                <div className="eve-stack-card eve-repo-card">
                  <span className="eve-card-icon" data-icon="repo" />
                  <span>Repository</span>
                </div>
                <div className="eve-stack-card eve-git-card">
                  <span className="eve-git-mark">git</span>
                  <span className="eve-git-line" />
                </div>
              </div>

              <div className="eve-live-label">
                <span>Lives in</span>
                <code>.eve/</code>
              </div>

              <div className="eve-visual-notes">
                <span>What changed and why</span>
                <span>Evidence and verification</span>
                <span>Linked to Git state</span>
              </div>

              <span className="eve-flow-token" data-token="plan">
                plan
              </span>
              <span className="eve-flow-token" data-token="snapshot">
                snapshot
              </span>
              <span className="eve-flow-token" data-token="evidence">
                checks
              </span>
            </div>
          </aside>
        </section>

        <section className="eve-compare" aria-label="Git and EVE comparison">
          <div className="eve-compare-copy">
            <h2>Git records implementation.</h2>
            <p>EVE records product intent and evidence.</p>
          </div>
          <div className="eve-compare-table" role="table">
            <div className="eve-compare-head" role="row">
              <span role="columnheader" />
              <span role="columnheader">Git</span>
              <span role="columnheader">EVE</span>
            </div>
            {comparisonRows.map(([label, git, eve]) => (
              <div className="eve-compare-row" role="row" key={label}>
                <span role="cell">{label}</span>
                <span role="cell">{git}</span>
                <strong role="cell">{eve}</strong>
              </div>
            ))}
          </div>
        </section>

        <section className="eve-steps" aria-label="From plan to verifiable change">
          <h2>From plan to verifiable change</h2>
          <div className="eve-step-list">
            {steps.map((step, index) => (
              <article className="eve-step" key={step.title}>
                <div className="eve-step-icon" data-icon={step.icon} aria-hidden="true" />
                <div>
                  <span>{index + 1}</span>
                  <h3>{step.title}</h3>
                  <p>{step.text}</p>
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className="eve-local" aria-label="Local-first Git-native storage">
          <div className="eve-local-copy">
            <h2>Local-first. Git-native.</h2>
            <p>EVE stores everything in your repository. No lock-in, no cloud, no surprises.</p>
          </div>
          <div className="eve-local-flow" aria-label="EVE repository storage flow">
            <pre><code>{`Agent / CLI

  eve plan
  eve run
  eve snapshot
  eve review`}</code></pre>
            <span aria-hidden="true">-&gt;</span>
            <pre><code>{`.eve/

  plans/
  snapshots/
  checks/
  evidence/`}</code></pre>
            <span aria-hidden="true">-&gt;</span>
            <pre><code>{`Your Git repository

  .git/
  src/
  docs/
  .eve/
  ...`}</code></pre>
            <div className="eve-commit-chip">Git commit abc123def</div>
          </div>
        </section>

        <section className="eve-final">
          <h2>
            Your product has a history.
            <span>EVE makes it verifiable.</span>
          </h2>
          <div className="eve-final-actions">
            <Link className="eve-final-primary" href="/docs/guides/get-started">
              Get started <span aria-hidden="true">-&gt;</span>
            </Link>
            <a href="https://github.com/nhestrompia/eve">
              View on GitHub <span aria-hidden="true">-&gt;</span>
            </a>
          </div>
          <p>MIT licensed · Built by engineers · Open to all</p>
        </section>
      </div>
    </main>
  );
}
