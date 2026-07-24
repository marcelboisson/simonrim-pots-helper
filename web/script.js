// Cookie helpers — same wire format as server/cookie.go (base64url-encoded JSON)

function b64uEncode(s) { return btoa(s).replace(/\+/g, '-').replace(/\//g, '_'); }
function b64uDecode(s) { return atob(s.replace(/-/g, '+').replace(/_/g, '/')); }

function readCookie() {
  const m = document.cookie.split('; ').find(r => r.startsWith('alchemy='));
  if (!m) return {};
  try { return JSON.parse(b64uDecode(m.slice(8))); } catch { return {}; }
}

function writeCookie(state) {
  document.cookie = 'alchemy=' + b64uEncode(JSON.stringify(state))
    + '; Path=/; Max-Age=31536000; SameSite=Lax';
}

// Effect index set helpers (arrays of ints, max 4 elements each)

function intersect(a, b) { return a.filter(x => b.includes(x)); }

function setUnion(a, b, c) { return [...new Set([...a, ...b, ...c])]; }

function containsAll(set, targets) { return targets.every(t => set.includes(t)); }

function isNegative(effect) {
  if (effect.startsWith('Damage ') ||
      effect.startsWith('Lingering Damage ') ||
      effect.startsWith('Weakness ')) return true;
  return ['Burden', 'Calm', 'Command', 'Fear', 'Frenzy', 'Paralysis', 'Silence'].includes(effect);
}

function makePotion(ings, effectIndices, allEffects) {
  return {
    ingredients: [...ings].sort((a, b) => a.name.localeCompare(b.name)),
    effects: [...new Set(effectIndices)].sort((a, b) => a - b).map(i => allEffects[i]),
  };
}

function brew(ingredients, ownedIndices, allEffects) {
  const ings = ownedIndices.map(i => ingredients[i]);
  const n = ings.length;
  const potions = [];

  for (let a = 0; a < n; a++) {
    for (let b = a + 1; b < n; b++) {
      const ab = intersect(ings[a].effects, ings[b].effects);
      if (ab.length) potions.push(makePotion([ings[a], ings[b]], ab, allEffects));
    }
  }

  for (let a = 0; a < n; a++) {
    for (let b = a + 1; b < n; b++) {
      const ab = intersect(ings[a].effects, ings[b].effects);
      for (let c = b + 1; c < n; c++) {
        const ac = intersect(ings[a].effects, ings[c].effects);
        const bc = intersect(ings[b].effects, ings[c].effects);
        if ((!ab.length && !ac.length) || (!ab.length && !bc.length) || (!ac.length && !bc.length)) continue;
        potions.push(makePotion([ings[a], ings[b], ings[c]], setUnion(ab, ac, bc), allEffects));
      }
    }
  }

  potions.sort((p, q) => {
    if (p.effects.length !== q.effects.length) return q.effects.length - p.effects.length;
    return p.ingredients[0].name.localeCompare(q.ingredients[0].name);
  });
  return potions;
}

function filterPure(potions) {
  return potions.filter(p => {
    const pos = p.effects.some(e => !isNegative(e));
    const neg = p.effects.some(e => isNegative(e));
    return !pos || !neg;
  });
}

function findCombos(ingredients, allEffects, targets) {
  if (!targets.length) return [];
  const cands = ingredients.filter(ing => ing.effects.some(e => targets.includes(e)));
  const n = cands.length;
  const potions = [];

  for (let a = 0; a < n; a++) {
    for (let b = a + 1; b < n; b++) {
      const ab = intersect(cands[a].effects, cands[b].effects);
      if (containsAll(ab, targets)) {
        potions.push(makePotion([cands[a], cands[b]], ab, allEffects));
        continue;
      }
      for (let c = b + 1; c < n; c++) {
        const ac = intersect(cands[a].effects, cands[c].effects);
        const bc = intersect(cands[b].effects, cands[c].effects);
        const u = setUnion(ab, ac, bc);
        if (containsAll(u, targets)) {
          potions.push(makePotion([cands[a], cands[b], cands[c]], u, allEffects));
        }
      }
    }
  }

  potions.sort((p, q) => {
    const cp = p.ingredients.reduce((s, i) => s + i.value, 0);
    const cq = q.ingredients.reduce((s, i) => s + i.value, 0);
    if (cp !== cq) return cp - cq;
    if (p.ingredients.length !== q.ingredients.length) return p.ingredients.length - q.ingredients.length;
    return p.ingredients[0].name.localeCompare(q.ingredients[0].name);
  });
  return potions;
}

function app() {
  return {
    effects: [],
    ingredients: [],
    tab: 'brew',
    owned: [],
    allowMixed: false,
    activeEffect: null,
    selectedEffects: [],
    brewView: 'pick',
    findView: 'pick',

    async init() {
      const d = await fetch('./data.json').then(r => r.json());
      this.effects = d.effects;
      this.ingredients = d.ingredients;
      const s = readCookie();
      this.owned = s.o || [];
      this.allowMixed = !!s.m;
    },

    isOwned(i) { return this.owned.includes(i); },

    toggle(i) {
      const pos = this.owned.indexOf(i);
      if (pos === -1) this.owned.push(i);
      else this.owned.splice(pos, 1);
      writeCookie({ o: this.owned, m: this.allowMixed });
    },

    reset() {
      this.owned = [];
      writeCookie({ o: [], m: this.allowMixed });
    },

    toggleMixed() {
      this.allowMixed = !this.allowMixed;
      writeCookie({ o: this.owned, m: this.allowMixed });
    },

    get potions() {
      const p = brew(this.ingredients, this.owned, this.effects);
      return this.allowMixed ? p : filterPure(p);
    },

    selectEffect(i) { this.activeEffect = this.activeEffect === i ? null : i; },

    get activeIngredients() {
      if (this.activeEffect === null) return [];
      return this.ingredients
        .filter(ing => ing.effects.includes(this.activeEffect))
        .sort((a, b) => a.name.localeCompare(b.name));
    },

    toggleSelectedEffect(i) {
      const pos = this.selectedEffects.indexOf(i);
      if (pos === -1) this.selectedEffects.push(i);
      else this.selectedEffects.splice(pos, 1);
    },

    isSelectedEffect(i) { return this.selectedEffects.includes(i); },

    get findResults() {
      const r = findCombos(this.ingredients, this.effects, this.selectedEffects);
      return this.allowMixed ? r : filterPure(r);
    },
  };
}
