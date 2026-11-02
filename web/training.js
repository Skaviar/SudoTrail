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
  // Section anchors come from HoDoKu's technique index. More specific fish
  // families take precedence over their fins or basic fish size.
  function guideUrl(name) {
    if (typeof name !== 'string') return null;
    if (name === '3D Medusa') return 'https://www.sudokuwiki.org/3d_medusa';
    const base = 'https://hodoku.sourceforge.net/en/';
    const sections = {
      'Full House': 'tech_singles.php#fh',
      'Naked Single': 'tech_singles.php#n1',
      'Hidden Single': 'tech_singles.php#h1',
      'Peer Elimination': 'tech_intro.php#te',
      'Locked Candidates — Pointing': 'tech_intersections.php#lc1',
      'Locked Candidates — Claiming': 'tech_intersections.php#lc2',
      'Skyscraper': 'tech_sdp.php#sk',
      '2-String Kite': 'tech_sdp.php#t2sk',
      'Turbot Fish': 'tech_sdp.php#tf',
      'Empty Rectangle': 'tech_sdp.php#er',
      'XY-Wing': 'tech_wings.php#xy',
      'XYZ-Wing': 'tech_wings.php#xyz',
      'W-Wing': 'tech_wings.php#w',
      'Hidden Rectangle': 'tech_ur.php#hr',
      'Avoidable Rectangle Type 1': 'tech_ur.php#ar',
      'Avoidable Rectangle Type 2': 'tech_ur.php#ar',
      'BUG+1': 'tech_ur.php#bug1',
      'Simple Coloring — Color Wrap': 'tech_col.php#sc',
      'Simple Coloring — Color Trap': 'tech_col.php#sc',
      'Multi-Coloring': 'tech_col.php#mc',
      'Remote Pairs': 'tech_chains.php#rp',
      'X-Chain': 'tech_chains.php#x',
      'XY-Chain': 'tech_chains.php#xyc',
      'Alternating Inference Chain': 'tech_chains.php#nl',
      'ALS-XZ': 'tech_als.php#axz',
      'Doubly Linked ALS-XZ': 'tech_als.php#axz',
      'ALS-XY-Wing': 'tech_als.php#axy',
      'ALS Chain': 'tech_als.php#ach',
      'Death Blossom': 'tech_als.php#db',
      'Forcing Chain': 'tech_last.php#fc',
      'Forcing Net': 'tech_last.php#fn'
    };
    if (Object.hasOwn(sections, name)) return base + sections[name];
    const subset = name.match(/^(Naked|Hidden) (Pair|Triple|Quadruple)$/);
    if (subset) {
      const size = ['Pair', 'Triple', 'Quadruple'].indexOf(subset[2]) + 2;
      return `${base}tech_${subset[1].toLowerCase()}.php#${subset[1] === 'Naked' ? 'n' : 'h'}${size}`;
    }
    if (/^(Unique Rectangle Type [1-6]|Hidden Rectangle) \(Missing Candidates\)$/.test(name)) return base + 'tech_ur.php#umc';
    const rectangle = name.match(/^Unique Rectangle Type ([1-6])$/);
    if (rectangle) return base + `tech_ur.php#u${rectangle[1]}`;
    const fish = name.match(/^(?:(?:Siamese|Finned|Sashimi|Franken|Mutant) )*(X-Wing|Swordfish|Jellyfish|Squirmbag|Whale|Leviathan)$/);
    if (fish) {
      if (name.includes('Siamese ')) return base + 'tech_fishc.php#sf';
      if (name.includes('Franken ')) return base + 'tech_fishc.php#ff';
      if (name.includes('Mutant ')) return base + 'tech_fishc.php#mf';
      const size = ['X-Wing', 'Swordfish', 'Jellyfish', 'Squirmbag', 'Whale', 'Leviathan'].indexOf(fish[1]) + 2;
      if (/^(Finned|Sashimi) /.test(name)) return base + `tech_fishfs.php#fbf${size <= 4 ? size : '567'}`;
      return base + `tech_fishb.php#bf${Math.min(size, 5)}`;
    }
    return null;
  }
  return { levels, describe, targetIndex, guideUrl };
})();
