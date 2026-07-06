import { describe, expect, it } from 'vitest';
import { relationshipEntries, relationshipLabel, relationshipSummary, titleCase } from './evolution-display';

describe('evolution display helpers', () => {
  it('labels canonical relationship types in product language', () => {
    expect(relationshipLabel('corrects')).toBe('Corrects');
    expect(relationshipLabel('supersedes')).toBe('Supersedes');
    expect(relationshipLabel('reverts')).toBe('Reverts');
    expect(relationshipLabel('dependsOn')).toBe('Depends on');
    expect(relationshipLabel('related')).toBe('Related to');
  });

  it('keeps future relationship keys readable', () => {
    expect(titleCase('blockedBy')).toBe('Blocked By');
    expect(relationshipLabel('blockedBy')).toBe('Blocked By');
  });

  it('flattens relationships without empty targets', () => {
    expect(
      relationshipEntries({
        corrects: ['EV-001', ''],
        dependsOn: ['EV-002'],
        related: undefined
      })
    ).toEqual([
      { kind: 'corrects', label: 'Corrects', value: 'EV-001' },
      { kind: 'dependsOn', label: 'Depends on', value: 'EV-002' }
    ]);
  });

  it('summarizes visible relationships and hidden counts', () => {
    expect(
      relationshipSummary(
        {
          corrects: ['EV-001'],
          dependsOn: ['EV-002'],
          related: ['EV-003']
        },
        2
      )
    ).toBe('Corrects EV-001 · Depends on EV-002 · +1 more');
  });
});
