export function RequiredSuiteSelector({ value, suites, onChange }: {
  value?: string;
  suites: string[];
  onChange: (suite: string | undefined) => void;
}) {
  return (
    <label className="phone-field">
      <span>Required suite</span>
      <select aria-label="Required suite" value={value ?? ''} onChange={(event) => onChange(event.target.value || undefined)}>
        <option value="">Use branch default</option>
        {suites.map((suite) => <option key={suite} value={suite}>{suite}</option>)}
      </select>
    </label>
  );
}
