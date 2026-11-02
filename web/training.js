'use strict';

// Short teaching copy and pure helpers shared by setup and the practice view.
window.godokuTraining = (() => {
  const levels = {
    easy: { title: 'Easy', summary: 'Singles', detail: 'Find a cell with one possible number, or the only place a number can go.' },
    medium: { title: 'Medium', summary: 'Pairs, triples & locked candidates', detail: 'Use small groups of cells and the overlap between a box and a line to remove candidates.' },
    hard: { title: 'Hard', summary: 'Fish, wings & coloring', detail: 'Connect candidates across the grid: X-Wings, Swordfish, wings, and simple coloring.' },
    expert: { title: 'Expert', summary: 'Chains, ALS & complex fish', detail: 'Follow longer connections with chains, almost locked sets, advanced fish, and forcing logic.' }
  };
  function describe(name) {
    if (name === 'Naked Single') return 'Look for a cell with only one candidate left.';
    if (name === 'Hidden Single') return 'Find a number that has only one possible place in a row, column, or box.';
    if (name === 'Full House') return 'Find a row, column, or box with one empty cell.';
    if (name.startsWith('Naked ')) return 'Look for a small group of cells that together contain only the same number of distinct candidates.';
    if (name.startsWith('Hidden ') && /Pair|Triple|Quadruple/.test(name)) return 'Look for a group of numbers confined to the same small group of cells in a row, column, or box.';
    if (name.includes('Pointing')) return 'Find a number whose candidates in a box all lie in one row or column.';
    if (name.includes('Claiming')) return 'Find a number whose candidates in a row or column all lie inside one box.';
    if (name === 'XY-Wing') return 'Look for a two-candidate pivot linked to two other two-candidate cells.';
    if (name === 'XYZ-Wing') return 'Look for a three-candidate pivot with two smaller wings inside its view.';
    if (name === 'W-Wing') return 'Look for two cells with the same pair, connected through a strong link.';
    if (/Wing|Swordfish|Jellyfish|Squirmbag|Whale|Leviathan/.test(name)) return 'Trace one number across the base and cover houses. Check any fins before removing a candidate.';
    if (/Rectangle|BUG/.test(name)) return 'Look for a two-number pattern that would allow a second solution unless an extra candidate is used.';
    if (/ALS|Blossom/.test(name)) return 'Look for groups with one more candidate than cells, connected by a restricted common number.';
    if (/Color|Medusa|Remote Pairs/.test(name)) return 'Follow strong links with alternating colors, then look for a candidate that conflicts with the possible assignments.';
    if (/Forcing/.test(name)) return 'Follow the possible choices and look for a contradiction or a conclusion they all share.';
    return 'Follow the strong and weak links between candidates and look for the deduction they force.';
  }
  function targetIndex(states, technique) {
    return states.findIndex((state, i) => i > 0 && state.step?.technique === technique);
  }
  return { levels, describe, targetIndex };
})();
