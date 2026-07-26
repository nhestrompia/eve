import Image from 'next/image';
import Link from 'next/link';
import { Fragment } from 'react';
import { CopyCommand } from './copy-command';

const installCommand = 'npx --yes @nhestrompia/eve@latest install';

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
              Get started <span className="eve-cta-arrow" aria-hidden="true" />
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
            <CopyCommand command={installCommand} />
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
              <div className="eve-flow-stage">
                <div className="eve-flow-path">
                  <span data-path="plan-snapshot" />
                  <span data-path="snapshot-commit" />
                  <b />
                </div>

                <article className="eve-flow-node eve-flow-node-plan">
                  <span className="eve-flow-icon" data-icon="plan" />
                  <div className="eve-flow-node-copy">
                    <span>Plan</span>
                    <strong>Release notes generated</strong>
                    <p>decision, scope, checks</p>
                  </div>
                  <div className="eve-flow-meta">
                    <span className="eve-codex-avatar" />
                    <span>Codex</span>
                    <span>Just now</span>
                  </div>
                </article>

                <article className="eve-flow-node eve-flow-node-snapshot">
                  <span className="eve-flow-icon" data-icon="snapshot" />
                  <div className="eve-flow-node-copy">
                    <span>Snapshot</span>
                    <strong>Release notes generated</strong>
                    <p>validation, evidence, links</p>
                    <code>snap_124.json</code>
                  </div>
                </article>

                <article className="eve-flow-node eve-flow-node-commit">
                  <span className="eve-flow-icon eve-git-logo">
                    <svg viewBox="0 0 92 92" focusable="false">
                      <rect x="13" y="13" width="66" height="66" rx="8" />
                      <path d="M35 28l29 29M42 35a7 7 0 11-14 0 7 7 0 0114 0zm22 22a7 7 0 11-14 0 7 7 0 0114 0zM36 41v21a7 7 0 107 0V46" />
                    </svg>
                  </span>
                  <div className="eve-flow-node-copy">
                    <span>Commit</span>
                    <strong>Committed to Git</strong>
                    <p>.eve/snapshots/snap_124.json</p>
                  </div>
                  <div className="eve-flow-meta">
                    <span>main</span>
                    <span>1m ago</span>
                  </div>
                </article>

                <div className="eve-flow-note" data-note="why">
                  <span>What changed and why</span>
                </div>
                <div className="eve-flow-note" data-note="evidence">
                  <span>Evidence and verification</span>
                </div>
                <div className="eve-flow-note" data-note="git">
                  <span>Linked to Git state</span>
                </div>
              </div>
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
              <Fragment key={step.title}>
                <article className="eve-step">
                  <div className="eve-step-icon" data-icon={step.icon} aria-hidden="true" />
                  <div>
                    <span>{index + 1}</span>
                    <h3>{step.title}</h3>
                    <p>{step.text}</p>
                  </div>
                </article>
                {index < steps.length - 1 ? <div className="eve-step-connector" aria-hidden="true" /> : null}
              </Fragment>
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
              Get started <span className="eve-cta-arrow" aria-hidden="true" />
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
