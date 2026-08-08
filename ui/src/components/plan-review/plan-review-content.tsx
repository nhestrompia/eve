import type { ReactNode } from 'react';

export function PlanReviewSection({ title, children, headingID }: { title: string; children: ReactNode; headingID?: string }) {
  return <section className="phone-review-section"><h2 id={headingID}>{title}</h2><div>{children}</div></section>;
}

export function PlanReviewText({ title, value, editing, rows, monospaced, validationMessage, onChange }: {
  title: string;
  value: string;
  editing: boolean;
  rows: number;
  monospaced?: boolean;
  validationMessage?: string;
  onChange: (value: string) => void;
}) {
  const headingID = `phone-field-${title.toLowerCase().replace(/[^a-z]+/g, '-')}`;
  return (
    <PlanReviewSection title={title} headingID={headingID}>
      {editing ? (
        <textarea
          className={monospaced ? 'font-mono' : ''}
          value={value}
          rows={rows}
          aria-label={title}
          aria-describedby={validationMessage ? 'proposal-validation' : undefined}
          aria-invalid={Boolean(validationMessage)}
          onChange={(event) => onChange(event.target.value)}
        />
      ) : <p className={monospaced ? 'whitespace-pre-wrap font-mono' : 'whitespace-pre-wrap'}>{value || 'Not specified'}</p>}
    </PlanReviewSection>
  );
}
