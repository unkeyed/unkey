var Xx = Object.defineProperty;
var Yx = (t, e, r) =>
  e in t ? Xx(t, e, { enumerable: !0, configurable: !0, writable: !0, value: r }) : (t[e] = r);
var ci = (t, e, r) => (Yx(t, typeof e != "symbol" ? e + "" : e, r), r);
(function () {
  const e = document.createElement("link").relList;
  if (e && e.supports && e.supports("modulepreload")) return;
  for (const s of document.querySelectorAll('link[rel="modulepreload"]')) o(s);
  new MutationObserver((s) => {
    for (const u of s)
      if (u.type === "childList")
        for (const f of u.addedNodes) f.tagName === "LINK" && f.rel === "modulepreload" && o(f);
  }).observe(document, { childList: !0, subtree: !0 });
  function r(s) {
    const u = {};
    return (
      s.integrity && (u.integrity = s.integrity),
      s.referrerPolicy && (u.referrerPolicy = s.referrerPolicy),
      s.crossOrigin === "use-credentials"
        ? (u.credentials = "include")
        : s.crossOrigin === "anonymous"
        ? (u.credentials = "omit")
        : (u.credentials = "same-origin"),
      u
    );
  }
  function o(s) {
    if (s.ep) return;
    s.ep = !0;
    const u = r(s);
    fetch(s.href, u);
  }
})();
function ah(t, e) {
  const r = Object.create(null),
    o = t.split(",");
  for (let s = 0; s < o.length; s++) r[o[s]] = !0;
  return e ? (s) => !!r[s.toLowerCase()] : (s) => !!r[s];
}
const we = {},
  Go = [],
  wr = () => {},
  Zx = () => !1,
  Jx = /^on[^a-z]/,
  _c = (t) => Jx.test(t),
  ch = (t) => t.startsWith("onUpdate:"),
  ze = Object.assign,
  uh = (t, e) => {
    const r = t.indexOf(e);
    r > -1 && t.splice(r, 1);
  },
  Qx = Object.prototype.hasOwnProperty,
  le = (t, e) => Qx.call(t, e),
  It = Array.isArray,
  Ko = (t) => kc(t) === "[object Map]",
  Cm = (t) => kc(t) === "[object Set]",
  jt = (t) => typeof t == "function",
  Ie = (t) => typeof t == "string",
  Sc = (t) => typeof t == "symbol",
  ye = (t) => t !== null && typeof t == "object",
  Tm = (t) => (ye(t) || jt(t)) && jt(t.then) && jt(t.catch),
  Em = Object.prototype.toString,
  kc = (t) => Em.call(t),
  t1 = (t) => kc(t).slice(8, -1),
  Lm = (t) => kc(t) === "[object Object]",
  fh = (t) => Ie(t) && t !== "NaN" && t[0] !== "-" && "" + parseInt(t, 10) === t,
  Ia = ah(
    ",key,ref,ref_for,ref_key,onVnodeBeforeMount,onVnodeMounted,onVnodeBeforeUpdate,onVnodeUpdated,onVnodeBeforeUnmount,onVnodeUnmounted",
  ),
  Cc = (t) => {
    const e = Object.create(null);
    return (r) => e[r] || (e[r] = t(r));
  },
  e1 = /-(\w)/g,
  _r = Cc((t) => t.replace(e1, (e, r) => (r ? r.toUpperCase() : ""))),
  n1 = /\B([A-Z])/g,
  co = Cc((t) => t.replace(n1, "-$1").toLowerCase()),
  Tc = Cc((t) => t.charAt(0).toUpperCase() + t.slice(1)),
  ju = Cc((t) => (t ? `on${Tc(t)}` : "")),
  ro = (t, e) => !Object.is(t, e),
  Fa = (t, e) => {
    for (let r = 0; r < t.length; r++) t[r](e);
  },
  Za = (t, e, r) => {
    Object.defineProperty(t, e, { configurable: !0, enumerable: !1, value: r });
  },
  vf = (t) => {
    const e = parseFloat(t);
    return isNaN(e) ? t : e;
  },
  Am = (t) => {
    const e = Ie(t) ? Number(t) : NaN;
    return isNaN(e) ? t : e;
  };
let ug;
const mf = () =>
  ug ||
  (ug =
    typeof globalThis < "u"
      ? globalThis
      : typeof self < "u"
      ? self
      : typeof window < "u"
      ? window
      : typeof global < "u"
      ? global
      : {});
function An(t) {
  if (It(t)) {
    const e = {};
    for (let r = 0; r < t.length; r++) {
      const o = t[r],
        s = Ie(o) ? s1(o) : An(o);
      if (s) for (const u in s) e[u] = s[u];
    }
    return e;
  } else if (Ie(t) || ye(t)) return t;
}
const r1 = /;(?![^(]*\))/g,
  i1 = /:([^]+)/,
  o1 = /\/\*[^]*?\*\//g;
function s1(t) {
  const e = {};
  return (
    t
      .replace(o1, "")
      .split(r1)
      .forEach((r) => {
        if (r) {
          const o = r.split(i1);
          o.length > 1 && (e[o[0].trim()] = o[1].trim());
        }
      }),
    e
  );
}
function ve(t) {
  let e = "";
  if (Ie(t)) e = t;
  else if (It(t))
    for (let r = 0; r < t.length; r++) {
      const o = ve(t[r]);
      o && (e += o + " ");
    }
  else if (ye(t)) for (const r in t) t[r] && (e += r + " ");
  return e.trim();
}
const l1 = "itemscope,allowfullscreen,formnovalidate,ismap,nomodule,novalidate,readonly",
  a1 = ah(l1);
function Mm(t) {
  return !!t || t === "";
}
const Ut = (t) =>
    Ie(t)
      ? t
      : t == null
      ? ""
      : It(t) || (ye(t) && (t.toString === Em || !jt(t.toString)))
      ? JSON.stringify(t, Nm, 2)
      : String(t),
  Nm = (t, e) =>
    e && e.__v_isRef
      ? Nm(t, e.value)
      : Ko(e)
      ? { [`Map(${e.size})`]: [...e.entries()].reduce((r, [o, s]) => ((r[`${o} =>`] = s), r), {}) }
      : Cm(e)
      ? { [`Set(${e.size})`]: [...e.values()] }
      : ye(e) && !It(e) && !Lm(e)
      ? String(e)
      : e;
let $n;
class c1 {
  constructor(e = !1) {
    (this.detached = e),
      (this._active = !0),
      (this.effects = []),
      (this.cleanups = []),
      (this.parent = $n),
      !e && $n && (this.index = ($n.scopes || ($n.scopes = [])).push(this) - 1);
  }
  get active() {
    return this._active;
  }
  run(e) {
    if (this._active) {
      const r = $n;
      try {
        return ($n = this), e();
      } finally {
        $n = r;
      }
    }
  }
  on() {
    $n = this;
  }
  off() {
    $n = this.parent;
  }
  stop(e) {
    if (this._active) {
      let r, o;
      for (r = 0, o = this.effects.length; r < o; r++) this.effects[r].stop();
      for (r = 0, o = this.cleanups.length; r < o; r++) this.cleanups[r]();
      if (this.scopes) for (r = 0, o = this.scopes.length; r < o; r++) this.scopes[r].stop(!0);
      if (!this.detached && this.parent && !e) {
        const s = this.parent.scopes.pop();
        s && s !== this && ((this.parent.scopes[this.index] = s), (s.index = this.index));
      }
      (this.parent = void 0), (this._active = !1);
    }
  }
}
function u1(t, e = $n) {
  e && e.active && e.effects.push(t);
}
function Pm() {
  return $n;
}
function f1(t) {
  $n && $n.cleanups.push(t);
}
const hh = (t) => {
    const e = new Set(t);
    return (e.w = 0), (e.n = 0), e;
  },
  Om = (t) => (t.w & Ti) > 0,
  Dm = (t) => (t.n & Ti) > 0,
  h1 = ({ deps: t }) => {
    if (t.length) for (let e = 0; e < t.length; e++) t[e].w |= Ti;
  },
  d1 = (t) => {
    const { deps: e } = t;
    if (e.length) {
      let r = 0;
      for (let o = 0; o < e.length; o++) {
        const s = e[o];
        Om(s) && !Dm(s) ? s.delete(t) : (e[r++] = s), (s.w &= ~Ti), (s.n &= ~Ti);
      }
      e.length = r;
    }
  },
  Ja = new WeakMap();
let rl = 0,
  Ti = 1;
const yf = 30;
let ir;
const Qi = Symbol(""),
  bf = Symbol("");
class dh {
  constructor(e, r = null, o) {
    (this.fn = e),
      (this.scheduler = r),
      (this.active = !0),
      (this.deps = []),
      (this.parent = void 0),
      u1(this, o);
  }
  run() {
    if (!this.active) return this.fn();
    let e = ir,
      r = Si;
    for (; e; ) {
      if (e === this) return;
      e = e.parent;
    }
    try {
      return (
        (this.parent = ir),
        (ir = this),
        (Si = !0),
        (Ti = 1 << ++rl),
        rl <= yf ? h1(this) : fg(this),
        this.fn()
      );
    } finally {
      rl <= yf && d1(this),
        (Ti = 1 << --rl),
        (ir = this.parent),
        (Si = r),
        (this.parent = void 0),
        this.deferStop && this.stop();
    }
  }
  stop() {
    ir === this
      ? (this.deferStop = !0)
      : this.active && (fg(this), this.onStop && this.onStop(), (this.active = !1));
  }
}
function fg(t) {
  const { deps: e } = t;
  if (e.length) {
    for (let r = 0; r < e.length; r++) e[r].delete(t);
    e.length = 0;
  }
}
let Si = !0;
const $m = [];
function ps() {
  $m.push(Si), (Si = !1);
}
function gs() {
  const t = $m.pop();
  Si = t === void 0 ? !0 : t;
}
function Nn(t, e, r) {
  if (Si && ir) {
    let o = Ja.get(t);
    o || Ja.set(t, (o = new Map()));
    let s = o.get(r);
    s || o.set(r, (s = hh())), Rm(s);
  }
}
function Rm(t, e) {
  let r = !1;
  rl <= yf ? Dm(t) || ((t.n |= Ti), (r = !Om(t))) : (r = !t.has(ir)),
    r && (t.add(ir), ir.deps.push(t));
}
function Fr(t, e, r, o, s, u) {
  const f = Ja.get(t);
  if (!f) return;
  let h = [];
  if (e === "clear") h = [...f.values()];
  else if (r === "length" && It(t)) {
    const d = Number(o);
    f.forEach((g, v) => {
      (v === "length" || (!Sc(v) && v >= d)) && h.push(g);
    });
  } else
    switch ((r !== void 0 && h.push(f.get(r)), e)) {
      case "add":
        It(t) ? fh(r) && h.push(f.get("length")) : (h.push(f.get(Qi)), Ko(t) && h.push(f.get(bf)));
        break;
      case "delete":
        It(t) || (h.push(f.get(Qi)), Ko(t) && h.push(f.get(bf)));
        break;
      case "set":
        Ko(t) && h.push(f.get(Qi));
        break;
    }
  if (h.length === 1) h[0] && wf(h[0]);
  else {
    const d = [];
    for (const g of h) g && d.push(...g);
    wf(hh(d));
  }
}
function wf(t, e) {
  const r = It(t) ? t : [...t];
  for (const o of r) o.computed && hg(o);
  for (const o of r) o.computed || hg(o);
}
function hg(t, e) {
  (t !== ir || t.allowRecurse) && (t.scheduler ? t.scheduler() : t.run());
}
function p1(t, e) {
  var r;
  return (r = Ja.get(t)) == null ? void 0 : r.get(e);
}
const g1 = ah("__proto__,__v_isRef,__isVue"),
  zm = new Set(
    Object.getOwnPropertyNames(Symbol)
      .filter((t) => t !== "arguments" && t !== "caller")
      .map((t) => Symbol[t])
      .filter(Sc),
  ),
  dg = v1();
function v1() {
  const t = {};
  return (
    ["includes", "indexOf", "lastIndexOf"].forEach((e) => {
      t[e] = function (...r) {
        const o = ae(this);
        for (let u = 0, f = this.length; u < f; u++) Nn(o, "get", u + "");
        const s = o[e](...r);
        return s === -1 || s === !1 ? o[e](...r.map(ae)) : s;
      };
    }),
    ["push", "pop", "shift", "unshift", "splice"].forEach((e) => {
      t[e] = function (...r) {
        ps();
        const o = ae(this)[e].apply(this, r);
        return gs(), o;
      };
    }),
    t
  );
}
function m1(t) {
  const e = ae(this);
  return Nn(e, "has", t), e.hasOwnProperty(t);
}
class Im {
  constructor(e = !1, r = !1) {
    (this._isReadonly = e), (this._shallow = r);
  }
  get(e, r, o) {
    const s = this._isReadonly,
      u = this._shallow;
    if (r === "__v_isReactive") return !s;
    if (r === "__v_isReadonly") return s;
    if (r === "__v_isShallow") return u;
    if (r === "__v_raw" && o === (s ? (u ? M1 : Bm) : u ? Hm : qm).get(e)) return e;
    const f = It(e);
    if (!s) {
      if (f && le(dg, r)) return Reflect.get(dg, r, o);
      if (r === "hasOwnProperty") return m1;
    }
    const h = Reflect.get(e, r, o);
    return (Sc(r) ? zm.has(r) : g1(r)) || (s || Nn(e, "get", r), u)
      ? h
      : Le(h)
      ? f && fh(r)
        ? h
        : h.value
      : ye(h)
      ? s
        ? Lc(h)
        : Un(h)
      : h;
  }
}
class Fm extends Im {
  constructor(e = !1) {
    super(!1, e);
  }
  set(e, r, o, s) {
    let u = e[r];
    if (rs(u) && Le(u) && !Le(o)) return !1;
    if (
      !this._shallow &&
      (!Qa(o) && !rs(o) && ((u = ae(u)), (o = ae(o))), !It(e) && Le(u) && !Le(o))
    )
      return (u.value = o), !0;
    const f = It(e) && fh(r) ? Number(r) < e.length : le(e, r),
      h = Reflect.set(e, r, o, s);
    return e === ae(s) && (f ? ro(o, u) && Fr(e, "set", r, o) : Fr(e, "add", r, o)), h;
  }
  deleteProperty(e, r) {
    const o = le(e, r);
    e[r];
    const s = Reflect.deleteProperty(e, r);
    return s && o && Fr(e, "delete", r, void 0), s;
  }
  has(e, r) {
    const o = Reflect.has(e, r);
    return (!Sc(r) || !zm.has(r)) && Nn(e, "has", r), o;
  }
  ownKeys(e) {
    return Nn(e, "iterate", It(e) ? "length" : Qi), Reflect.ownKeys(e);
  }
}
class y1 extends Im {
  constructor(e = !1) {
    super(!0, e);
  }
  set(e, r) {
    return !0;
  }
  deleteProperty(e, r) {
    return !0;
  }
}
const b1 = new Fm(),
  w1 = new y1(),
  x1 = new Fm(!0),
  ph = (t) => t,
  Ec = (t) => Reflect.getPrototypeOf(t);
function wa(t, e, r = !1, o = !1) {
  t = t.__v_raw;
  const s = ae(t),
    u = ae(e);
  r || (ro(e, u) && Nn(s, "get", e), Nn(s, "get", u));
  const { has: f } = Ec(s),
    h = o ? ph : r ? yh : pl;
  if (f.call(s, e)) return h(t.get(e));
  if (f.call(s, u)) return h(t.get(u));
  t !== s && t.get(e);
}
function xa(t, e = !1) {
  const r = this.__v_raw,
    o = ae(r),
    s = ae(t);
  return (
    e || (ro(t, s) && Nn(o, "has", t), Nn(o, "has", s)), t === s ? r.has(t) : r.has(t) || r.has(s)
  );
}
function _a(t, e = !1) {
  return (t = t.__v_raw), !e && Nn(ae(t), "iterate", Qi), Reflect.get(t, "size", t);
}
function pg(t) {
  t = ae(t);
  const e = ae(this);
  return Ec(e).has.call(e, t) || (e.add(t), Fr(e, "add", t, t)), this;
}
function gg(t, e) {
  e = ae(e);
  const r = ae(this),
    { has: o, get: s } = Ec(r);
  let u = o.call(r, t);
  u || ((t = ae(t)), (u = o.call(r, t)));
  const f = s.call(r, t);
  return r.set(t, e), u ? ro(e, f) && Fr(r, "set", t, e) : Fr(r, "add", t, e), this;
}
function vg(t) {
  const e = ae(this),
    { has: r, get: o } = Ec(e);
  let s = r.call(e, t);
  s || ((t = ae(t)), (s = r.call(e, t))), o && o.call(e, t);
  const u = e.delete(t);
  return s && Fr(e, "delete", t, void 0), u;
}
function mg() {
  const t = ae(this),
    e = t.size !== 0,
    r = t.clear();
  return e && Fr(t, "clear", void 0, void 0), r;
}
function Sa(t, e) {
  return function (o, s) {
    const u = this,
      f = u.__v_raw,
      h = ae(f),
      d = e ? ph : t ? yh : pl;
    return !t && Nn(h, "iterate", Qi), f.forEach((g, v) => o.call(s, d(g), d(v), u));
  };
}
function ka(t, e, r) {
  return function (...o) {
    const s = this.__v_raw,
      u = ae(s),
      f = Ko(u),
      h = t === "entries" || (t === Symbol.iterator && f),
      d = t === "keys" && f,
      g = s[t](...o),
      v = r ? ph : e ? yh : pl;
    return (
      !e && Nn(u, "iterate", d ? bf : Qi),
      {
        next() {
          const { value: b, done: w } = g.next();
          return w ? { value: b, done: w } : { value: h ? [v(b[0]), v(b[1])] : v(b), done: w };
        },
        [Symbol.iterator]() {
          return this;
        },
      }
    );
  };
}
function ui(t) {
  return function (...e) {
    return t === "delete" ? !1 : this;
  };
}
function _1() {
  const t = {
      get(u) {
        return wa(this, u);
      },
      get size() {
        return _a(this);
      },
      has: xa,
      add: pg,
      set: gg,
      delete: vg,
      clear: mg,
      forEach: Sa(!1, !1),
    },
    e = {
      get(u) {
        return wa(this, u, !1, !0);
      },
      get size() {
        return _a(this);
      },
      has: xa,
      add: pg,
      set: gg,
      delete: vg,
      clear: mg,
      forEach: Sa(!1, !0),
    },
    r = {
      get(u) {
        return wa(this, u, !0);
      },
      get size() {
        return _a(this, !0);
      },
      has(u) {
        return xa.call(this, u, !0);
      },
      add: ui("add"),
      set: ui("set"),
      delete: ui("delete"),
      clear: ui("clear"),
      forEach: Sa(!0, !1),
    },
    o = {
      get(u) {
        return wa(this, u, !0, !0);
      },
      get size() {
        return _a(this, !0);
      },
      has(u) {
        return xa.call(this, u, !0);
      },
      add: ui("add"),
      set: ui("set"),
      delete: ui("delete"),
      clear: ui("clear"),
      forEach: Sa(!0, !0),
    };
  return (
    ["keys", "values", "entries", Symbol.iterator].forEach((u) => {
      (t[u] = ka(u, !1, !1)),
        (r[u] = ka(u, !0, !1)),
        (e[u] = ka(u, !1, !0)),
        (o[u] = ka(u, !0, !0));
    }),
    [t, r, e, o]
  );
}
const [S1, k1, C1, T1] = _1();
function gh(t, e) {
  const r = e ? (t ? T1 : C1) : t ? k1 : S1;
  return (o, s, u) =>
    s === "__v_isReactive"
      ? !t
      : s === "__v_isReadonly"
      ? t
      : s === "__v_raw"
      ? o
      : Reflect.get(le(r, s) && s in o ? r : o, s, u);
}
const E1 = { get: gh(!1, !1) },
  L1 = { get: gh(!1, !0) },
  A1 = { get: gh(!0, !1) },
  qm = new WeakMap(),
  Hm = new WeakMap(),
  Bm = new WeakMap(),
  M1 = new WeakMap();
function N1(t) {
  switch (t) {
    case "Object":
    case "Array":
      return 1;
    case "Map":
    case "Set":
    case "WeakMap":
    case "WeakSet":
      return 2;
    default:
      return 0;
  }
}
function P1(t) {
  return t.__v_skip || !Object.isExtensible(t) ? 0 : N1(t1(t));
}
function Un(t) {
  return rs(t) ? t : vh(t, !1, b1, E1, qm);
}
function Wm(t) {
  return vh(t, !1, x1, L1, Hm);
}
function Lc(t) {
  return vh(t, !0, w1, A1, Bm);
}
function vh(t, e, r, o, s) {
  if (!ye(t) || (t.__v_raw && !(e && t.__v_isReactive))) return t;
  const u = s.get(t);
  if (u) return u;
  const f = P1(t);
  if (f === 0) return t;
  const h = new Proxy(t, f === 2 ? o : r);
  return s.set(t, h), h;
}
function Xo(t) {
  return rs(t) ? Xo(t.__v_raw) : !!(t && t.__v_isReactive);
}
function rs(t) {
  return !!(t && t.__v_isReadonly);
}
function Qa(t) {
  return !!(t && t.__v_isShallow);
}
function Um(t) {
  return Xo(t) || rs(t);
}
function ae(t) {
  const e = t && t.__v_raw;
  return e ? ae(e) : t;
}
function mh(t) {
  return Za(t, "__v_skip", !0), t;
}
const pl = (t) => (ye(t) ? Un(t) : t),
  yh = (t) => (ye(t) ? Lc(t) : t);
function bh(t) {
  Si && ir && ((t = ae(t)), Rm(t.dep || (t.dep = hh())));
}
function wh(t, e) {
  t = ae(t);
  const r = t.dep;
  r && wf(r);
}
function Le(t) {
  return !!(t && t.__v_isRef === !0);
}
function Zt(t) {
  return jm(t, !1);
}
function vs(t) {
  return jm(t, !0);
}
function jm(t, e) {
  return Le(t) ? t : new O1(t, e);
}
class O1 {
  constructor(e, r) {
    (this.__v_isShallow = r),
      (this.dep = void 0),
      (this.__v_isRef = !0),
      (this._rawValue = r ? e : ae(e)),
      (this._value = r ? e : pl(e));
  }
  get value() {
    return bh(this), this._value;
  }
  set value(e) {
    const r = this.__v_isShallow || Qa(e) || rs(e);
    (e = r ? e : ae(e)),
      ro(e, this._rawValue) && ((this._rawValue = e), (this._value = r ? e : pl(e)), wh(this));
  }
}
function U(t) {
  return Le(t) ? t.value : t;
}
const D1 = {
  get: (t, e, r) => U(Reflect.get(t, e, r)),
  set: (t, e, r, o) => {
    const s = t[e];
    return Le(s) && !Le(r) ? ((s.value = r), !0) : Reflect.set(t, e, r, o);
  },
};
function Vm(t) {
  return Xo(t) ? t : new Proxy(t, D1);
}
class $1 {
  constructor(e) {
    (this.dep = void 0), (this.__v_isRef = !0);
    const { get: r, set: o } = e(
      () => bh(this),
      () => wh(this),
    );
    (this._get = r), (this._set = o);
  }
  get value() {
    return this._get();
  }
  set value(e) {
    this._set(e);
  }
}
function R1(t) {
  return new $1(t);
}
function z1(t) {
  const e = It(t) ? new Array(t.length) : {};
  for (const r in t) e[r] = Gm(t, r);
  return e;
}
class I1 {
  constructor(e, r, o) {
    (this._object = e), (this._key = r), (this._defaultValue = o), (this.__v_isRef = !0);
  }
  get value() {
    const e = this._object[this._key];
    return e === void 0 ? this._defaultValue : e;
  }
  set value(e) {
    this._object[this._key] = e;
  }
  get dep() {
    return p1(ae(this._object), this._key);
  }
}
class F1 {
  constructor(e) {
    (this._getter = e), (this.__v_isRef = !0), (this.__v_isReadonly = !0);
  }
  get value() {
    return this._getter();
  }
}
function xh(t, e, r) {
  return Le(t) ? t : jt(t) ? new F1(t) : ye(t) && arguments.length > 1 ? Gm(t, e, r) : Zt(t);
}
function Gm(t, e, r) {
  const o = t[e];
  return Le(o) ? o : new I1(t, e, r);
}
class q1 {
  constructor(e, r, o, s) {
    (this._setter = r),
      (this.dep = void 0),
      (this.__v_isRef = !0),
      (this.__v_isReadonly = !1),
      (this._dirty = !0),
      (this.effect = new dh(e, () => {
        this._dirty || ((this._dirty = !0), wh(this));
      })),
      (this.effect.computed = this),
      (this.effect.active = this._cacheable = !s),
      (this.__v_isReadonly = o);
  }
  get value() {
    const e = ae(this);
    return (
      bh(e), (e._dirty || !e._cacheable) && ((e._dirty = !1), (e._value = e.effect.run())), e._value
    );
  }
  set value(e) {
    this._setter(e);
  }
}
function H1(t, e, r = !1) {
  let o, s;
  const u = jt(t);
  return u ? ((o = t), (s = wr)) : ((o = t.get), (s = t.set)), new q1(o, s, u || !s, r);
}
function ki(t, e, r, o) {
  let s;
  try {
    s = o ? t(...o) : t();
  } catch (u) {
    Nl(u, e, r);
  }
  return s;
}
function jn(t, e, r, o) {
  if (jt(t)) {
    const u = ki(t, e, r, o);
    return (
      u &&
        Tm(u) &&
        u.catch((f) => {
          Nl(f, e, r);
        }),
      u
    );
  }
  const s = [];
  for (let u = 0; u < t.length; u++) s.push(jn(t[u], e, r, o));
  return s;
}
function Nl(t, e, r, o = !0) {
  const s = e ? e.vnode : null;
  if (e) {
    let u = e.parent;
    const f = e.proxy,
      h = r;
    for (; u; ) {
      const g = u.ec;
      if (g) {
        for (let v = 0; v < g.length; v++) if (g[v](t, f, h) === !1) return;
      }
      u = u.parent;
    }
    const d = e.appContext.config.errorHandler;
    if (d) {
      ki(d, null, 10, [t, f, h]);
      return;
    }
  }
  B1(t, r, s, o);
}
function B1(t, e, r, o = !0) {
  console.error(t);
}
let gl = !1,
  xf = !1;
const en = [];
let vr = 0;
const Yo = [];
let $r = null,
  Ki = 0;
const Km = Promise.resolve();
let _h = null;
function Br(t) {
  const e = _h || Km;
  return t ? e.then(this ? t.bind(this) : t) : e;
}
function W1(t) {
  let e = vr + 1,
    r = en.length;
  for (; e < r; ) {
    const o = (e + r) >>> 1,
      s = en[o],
      u = vl(s);
    u < t || (u === t && s.pre) ? (e = o + 1) : (r = o);
  }
  return e;
}
function Sh(t) {
  (!en.length || !en.includes(t, gl && t.allowRecurse ? vr + 1 : vr)) &&
    (t.id == null ? en.push(t) : en.splice(W1(t.id), 0, t), Xm());
}
function Xm() {
  !gl && !xf && ((xf = !0), (_h = Km.then(Zm)));
}
function U1(t) {
  const e = en.indexOf(t);
  e > vr && en.splice(e, 1);
}
function _f(t) {
  It(t) ? Yo.push(...t) : (!$r || !$r.includes(t, t.allowRecurse ? Ki + 1 : Ki)) && Yo.push(t),
    Xm();
}
function yg(t, e = gl ? vr + 1 : 0) {
  for (; e < en.length; e++) {
    const r = en[e];
    r && r.pre && (en.splice(e, 1), e--, r());
  }
}
function Ym(t) {
  if (Yo.length) {
    const e = [...new Set(Yo)];
    if (((Yo.length = 0), $r)) {
      $r.push(...e);
      return;
    }
    for ($r = e, $r.sort((r, o) => vl(r) - vl(o)), Ki = 0; Ki < $r.length; Ki++) $r[Ki]();
    ($r = null), (Ki = 0);
  }
}
const vl = (t) => (t.id == null ? 1 / 0 : t.id),
  j1 = (t, e) => {
    const r = vl(t) - vl(e);
    if (r === 0) {
      if (t.pre && !e.pre) return -1;
      if (e.pre && !t.pre) return 1;
    }
    return r;
  };
function Zm(t) {
  (xf = !1), (gl = !0), en.sort(j1);
  try {
    for (vr = 0; vr < en.length; vr++) {
      const e = en[vr];
      e && e.active !== !1 && ki(e, null, 14);
    }
  } finally {
    (vr = 0), (en.length = 0), Ym(), (gl = !1), (_h = null), (en.length || Yo.length) && Zm();
  }
}
function V1(t, e, ...r) {
  if (t.isUnmounted) return;
  const o = t.vnode.props || we;
  let s = r;
  const u = e.startsWith("update:"),
    f = u && e.slice(7);
  if (f && f in o) {
    const v = `${f === "modelValue" ? "model" : f}Modifiers`,
      { number: b, trim: w } = o[v] || we;
    w && (s = r.map((S) => (Ie(S) ? S.trim() : S))), b && (s = r.map(vf));
  }
  let h,
    d = o[(h = ju(e))] || o[(h = ju(_r(e)))];
  !d && u && (d = o[(h = ju(co(e)))]), d && jn(d, t, 6, s);
  const g = o[h + "Once"];
  if (g) {
    if (!t.emitted) t.emitted = {};
    else if (t.emitted[h]) return;
    (t.emitted[h] = !0), jn(g, t, 6, s);
  }
}
function Jm(t, e, r = !1) {
  const o = e.emitsCache,
    s = o.get(t);
  if (s !== void 0) return s;
  const u = t.emits;
  let f = {},
    h = !1;
  if (!jt(t)) {
    const d = (g) => {
      const v = Jm(g, e, !0);
      v && ((h = !0), ze(f, v));
    };
    !r && e.mixins.length && e.mixins.forEach(d),
      t.extends && d(t.extends),
      t.mixins && t.mixins.forEach(d);
  }
  return !u && !h
    ? (ye(t) && o.set(t, null), null)
    : (It(u) ? u.forEach((d) => (f[d] = null)) : ze(f, u), ye(t) && o.set(t, f), f);
}
function Ac(t, e) {
  return !t || !_c(e)
    ? !1
    : ((e = e.slice(2).replace(/Once$/, "")),
      le(t, e[0].toLowerCase() + e.slice(1)) || le(t, co(e)) || le(t, e));
}
let Je = null,
  Mc = null;
function tc(t) {
  const e = Je;
  return (Je = t), (Mc = (t && t.type.__scopeId) || null), e;
}
function Qm(t) {
  Mc = t;
}
function t0() {
  Mc = null;
}
const G1 = (t) => ee;
function ee(t, e = Je, r) {
  if (!e || t._n) return t;
  const o = (...s) => {
    o._d && Mg(-1);
    const u = tc(e);
    let f;
    try {
      f = t(...s);
    } finally {
      tc(u), o._d && Mg(1);
    }
    return f;
  };
  return (o._n = !0), (o._c = !0), (o._d = !0), o;
}
function Vu(t) {
  const {
    type: e,
    vnode: r,
    proxy: o,
    withProxy: s,
    props: u,
    propsOptions: [f],
    slots: h,
    attrs: d,
    emit: g,
    render: v,
    renderCache: b,
    data: w,
    setupState: S,
    ctx: P,
    inheritAttrs: A,
  } = t;
  let L, T;
  const M = tc(t);
  try {
    if (r.shapeFlag & 4) {
      const E = s || o;
      (L = rr(v.call(E, E, b, u, S, w, P))), (T = d);
    } else {
      const E = e;
      (L = rr(E.length > 1 ? E(u, { attrs: d, slots: h, emit: g }) : E(u, null))),
        (T = e.props ? d : X1(d));
    }
  } catch (E) {
    (cl.length = 0), Nl(E, t, 1), (L = Ft(Mn));
  }
  let R = L;
  if (T && A !== !1) {
    const E = Object.keys(T),
      { shapeFlag: B } = R;
    E.length && B & 7 && (f && E.some(ch) && (T = Y1(T, f)), (R = Ei(R, T)));
  }
  return (
    r.dirs && ((R = Ei(R)), (R.dirs = R.dirs ? R.dirs.concat(r.dirs) : r.dirs)),
    r.transition && (R.transition = r.transition),
    (L = R),
    tc(M),
    L
  );
}
function K1(t) {
  let e;
  for (let r = 0; r < t.length; r++) {
    const o = t[r];
    if (yl(o)) {
      if (o.type !== Mn || o.children === "v-if") {
        if (e) return;
        e = o;
      }
    } else return;
  }
  return e;
}
const X1 = (t) => {
    let e;
    for (const r in t) (r === "class" || r === "style" || _c(r)) && ((e || (e = {}))[r] = t[r]);
    return e;
  },
  Y1 = (t, e) => {
    const r = {};
    for (const o in t) (!ch(o) || !(o.slice(9) in e)) && (r[o] = t[o]);
    return r;
  };
function Z1(t, e, r) {
  const { props: o, children: s, component: u } = t,
    { props: f, children: h, patchFlag: d } = e,
    g = u.emitsOptions;
  if (e.dirs || e.transition) return !0;
  if (r && d >= 0) {
    if (d & 1024) return !0;
    if (d & 16) return o ? bg(o, f, g) : !!f;
    if (d & 8) {
      const v = e.dynamicProps;
      for (let b = 0; b < v.length; b++) {
        const w = v[b];
        if (f[w] !== o[w] && !Ac(g, w)) return !0;
      }
    }
  } else
    return (s || h) && (!h || !h.$stable) ? !0 : o === f ? !1 : o ? (f ? bg(o, f, g) : !0) : !!f;
  return !1;
}
function bg(t, e, r) {
  const o = Object.keys(e);
  if (o.length !== Object.keys(t).length) return !0;
  for (let s = 0; s < o.length; s++) {
    const u = o[s];
    if (e[u] !== t[u] && !Ac(r, u)) return !0;
  }
  return !1;
}
function kh({ vnode: t, parent: e }, r) {
  for (; e && e.subTree === t; ) ((t = e.vnode).el = r), (e = e.parent);
}
const e0 = "components",
  J1 = "directives";
function io(t, e) {
  return n0(e0, t, !0, e) || t;
}
const Q1 = Symbol.for("v-ndc");
function uo(t) {
  return n0(J1, t);
}
function n0(t, e, r = !0, o = !1) {
  const s = Je || Ge;
  if (s) {
    const u = s.type;
    if (t === e0) {
      const h = Z_(u, !1);
      if (h && (h === e || h === _r(e) || h === Tc(_r(e)))) return u;
    }
    const f = wg(s[t] || u[t], e) || wg(s.appContext[t], e);
    return !f && o ? u : f;
  }
}
function wg(t, e) {
  return t && (t[e] || t[_r(e)] || t[Tc(_r(e))]);
}
const t_ = (t) => t.__isSuspense,
  e_ = {
    name: "Suspense",
    __isSuspense: !0,
    process(t, e, r, o, s, u, f, h, d, g) {
      t == null ? r_(e, r, o, s, u, f, h, d, g) : i_(t, e, r, o, s, f, h, d, g);
    },
    hydrate: o_,
    create: Ch,
    normalize: s_,
  },
  n_ = e_;
function ml(t, e) {
  const r = t.props && t.props[e];
  jt(r) && r();
}
function r_(t, e, r, o, s, u, f, h, d) {
  const {
      p: g,
      o: { createElement: v },
    } = d,
    b = v("div"),
    w = (t.suspense = Ch(t, s, o, e, b, r, u, f, h, d));
  g(null, (w.pendingBranch = t.ssContent), b, null, o, w, u, f),
    w.deps > 0
      ? (ml(t, "onPending"),
        ml(t, "onFallback"),
        g(null, t.ssFallback, e, r, o, null, u, f),
        Zo(w, t.ssFallback))
      : w.resolve(!1, !0);
}
function i_(t, e, r, o, s, u, f, h, { p: d, um: g, o: { createElement: v } }) {
  const b = (e.suspense = t.suspense);
  (b.vnode = e), (e.el = t.el);
  const w = e.ssContent,
    S = e.ssFallback,
    { activeBranch: P, pendingBranch: A, isInFallback: L, isHydrating: T } = b;
  if (A)
    (b.pendingBranch = w),
      mr(w, A)
        ? (d(A, w, b.hiddenContainer, null, s, b, u, f, h),
          b.deps <= 0 ? b.resolve() : L && (d(P, S, r, o, s, null, u, f, h), Zo(b, S)))
        : (b.pendingId++,
          T ? ((b.isHydrating = !1), (b.activeBranch = A)) : g(A, s, b),
          (b.deps = 0),
          (b.effects.length = 0),
          (b.hiddenContainer = v("div")),
          L
            ? (d(null, w, b.hiddenContainer, null, s, b, u, f, h),
              b.deps <= 0 ? b.resolve() : (d(P, S, r, o, s, null, u, f, h), Zo(b, S)))
            : P && mr(w, P)
            ? (d(P, w, r, o, s, b, u, f, h), b.resolve(!0))
            : (d(null, w, b.hiddenContainer, null, s, b, u, f, h), b.deps <= 0 && b.resolve()));
  else if (P && mr(w, P)) d(P, w, r, o, s, b, u, f, h), Zo(b, w);
  else if (
    (ml(e, "onPending"),
    (b.pendingBranch = w),
    b.pendingId++,
    d(null, w, b.hiddenContainer, null, s, b, u, f, h),
    b.deps <= 0)
  )
    b.resolve();
  else {
    const { timeout: M, pendingId: R } = b;
    M > 0
      ? setTimeout(() => {
          b.pendingId === R && b.fallback(S);
        }, M)
      : M === 0 && b.fallback(S);
  }
}
function Ch(t, e, r, o, s, u, f, h, d, g, v = !1) {
  const {
    p: b,
    m: w,
    um: S,
    n: P,
    o: { parentNode: A, remove: L },
  } = g;
  let T;
  const M = a_(t);
  M && e != null && e.pendingBranch && ((T = e.pendingId), e.deps++);
  const R = t.props ? Am(t.props.timeout) : void 0,
    E = {
      vnode: t,
      parent: e,
      parentComponent: r,
      isSVG: f,
      container: o,
      hiddenContainer: s,
      anchor: u,
      deps: 0,
      pendingId: 0,
      timeout: typeof R == "number" ? R : -1,
      activeBranch: null,
      pendingBranch: null,
      isInFallback: !0,
      isHydrating: v,
      isUnmounted: !1,
      effects: [],
      resolve(B = !1, K = !1) {
        const {
          vnode: ht,
          activeBranch: Y,
          pendingBranch: nt,
          pendingId: at,
          effects: pt,
          parentComponent: gt,
          container: G,
        } = E;
        let z = !1;
        if (E.isHydrating) E.isHydrating = !1;
        else if (!B) {
          (z = Y && nt.transition && nt.transition.mode === "out-in"),
            z &&
              (Y.transition.afterLeave = () => {
                at === E.pendingId && (w(nt, G, H, 0), _f(pt));
              });
          let { anchor: H } = E;
          Y && ((H = P(Y)), S(Y, gt, E, !0)), z || w(nt, G, H, 0);
        }
        Zo(E, nt), (E.pendingBranch = null), (E.isInFallback = !1);
        let k = E.parent,
          F = !1;
        for (; k; ) {
          if (k.pendingBranch) {
            k.effects.push(...pt), (F = !0);
            break;
          }
          k = k.parent;
        }
        !F && !z && _f(pt),
          (E.effects = []),
          M &&
            e &&
            e.pendingBranch &&
            T === e.pendingId &&
            (e.deps--, e.deps === 0 && !K && e.resolve()),
          ml(ht, "onResolve");
      },
      fallback(B) {
        if (!E.pendingBranch) return;
        const { vnode: K, activeBranch: ht, parentComponent: Y, container: nt, isSVG: at } = E;
        ml(K, "onFallback");
        const pt = P(ht),
          gt = () => {
            E.isInFallback && (b(null, B, nt, pt, Y, null, at, h, d), Zo(E, B));
          },
          G = B.transition && B.transition.mode === "out-in";
        G && (ht.transition.afterLeave = gt), (E.isInFallback = !0), S(ht, Y, null, !0), G || gt();
      },
      move(B, K, ht) {
        E.activeBranch && w(E.activeBranch, B, K, ht), (E.container = B);
      },
      next() {
        return E.activeBranch && P(E.activeBranch);
      },
      registerDep(B, K) {
        const ht = !!E.pendingBranch;
        ht && E.deps++;
        const Y = B.vnode.el;
        B.asyncDep
          .catch((nt) => {
            Nl(nt, B, 0);
          })
          .then((nt) => {
            if (B.isUnmounted || E.isUnmounted || E.pendingId !== B.suspenseId) return;
            B.asyncResolved = !0;
            const { vnode: at } = B;
            Nf(B, nt, !1), Y && (at.el = Y);
            const pt = !Y && B.subTree.el;
            K(B, at, A(Y || B.subTree.el), Y ? null : P(B.subTree), E, f, d),
              pt && L(pt),
              kh(B, at.el),
              ht && --E.deps === 0 && E.resolve();
          });
      },
      unmount(B, K) {
        (E.isUnmounted = !0),
          E.activeBranch && S(E.activeBranch, r, B, K),
          E.pendingBranch && S(E.pendingBranch, r, B, K);
      },
    };
  return E;
}
function o_(t, e, r, o, s, u, f, h, d) {
  const g = (e.suspense = Ch(
      e,
      o,
      r,
      t.parentNode,
      document.createElement("div"),
      null,
      s,
      u,
      f,
      h,
      !0,
    )),
    v = d(t, (g.pendingBranch = e.ssContent), r, g, u, f);
  return g.deps === 0 && g.resolve(!1, !0), v;
}
function s_(t) {
  const { shapeFlag: e, children: r } = t,
    o = e & 32;
  (t.ssContent = xg(o ? r.default : r)), (t.ssFallback = o ? xg(r.fallback) : Ft(Mn));
}
function xg(t) {
  let e;
  if (jt(t)) {
    const r = is && t._c;
    r && ((t._d = !1), st()), (t = t()), r && ((t._d = !0), (e = Wn), b0());
  }
  return (
    It(t) && (t = K1(t)),
    (t = rr(t)),
    e && !t.dynamicChildren && (t.dynamicChildren = e.filter((r) => r !== t)),
    t
  );
}
function l_(t, e) {
  e && e.pendingBranch ? (It(t) ? e.effects.push(...t) : e.effects.push(t)) : _f(t);
}
function Zo(t, e) {
  t.activeBranch = e;
  const { vnode: r, parentComponent: o } = t,
    s = (r.el = e.el);
  o && o.subTree === r && ((o.vnode.el = s), kh(o, s));
}
function a_(t) {
  var e;
  return ((e = t.props) == null ? void 0 : e.suspensible) != null && t.props.suspensible !== !1;
}
function Th(t, e) {
  return Eh(t, null, e);
}
const Ca = {};
function Re(t, e, r) {
  return Eh(t, e, r);
}
function Eh(t, e, { immediate: r, deep: o, flush: s, onTrack: u, onTrigger: f } = we) {
  var h;
  const d = Pm() === ((h = Ge) == null ? void 0 : h.scope) ? Ge : null;
  let g,
    v = !1,
    b = !1;
  if (
    (Le(t)
      ? ((g = () => t.value), (v = Qa(t)))
      : Xo(t)
      ? ((g = () => t), (o = !0))
      : It(t)
      ? ((b = !0),
        (v = t.some((E) => Xo(E) || Qa(E))),
        (g = () =>
          t.map((E) => {
            if (Le(E)) return E.value;
            if (Xo(E)) return Yi(E);
            if (jt(E)) return ki(E, d, 2);
          })))
      : jt(t)
      ? e
        ? (g = () => ki(t, d, 2))
        : (g = () => {
            if (!(d && d.isUnmounted)) return w && w(), jn(t, d, 3, [S]);
          })
      : (g = wr),
    e && o)
  ) {
    const E = g;
    g = () => Yi(E());
  }
  let w,
    S = (E) => {
      w = M.onStop = () => {
        ki(E, d, 4);
      };
    },
    P;
  if (bl)
    if (((S = wr), e ? r && jn(e, d, 3, [g(), b ? [] : void 0, S]) : g(), s === "sync")) {
      const E = tS();
      P = E.__watcherHandles || (E.__watcherHandles = []);
    } else return wr;
  let A = b ? new Array(t.length).fill(Ca) : Ca;
  const L = () => {
    if (M.active)
      if (e) {
        const E = M.run();
        (o || v || (b ? E.some((B, K) => ro(B, A[K])) : ro(E, A))) &&
          (w && w(), jn(e, d, 3, [E, A === Ca ? void 0 : b && A[0] === Ca ? [] : A, S]), (A = E));
      } else M.run();
  };
  L.allowRecurse = !!e;
  let T;
  s === "sync"
    ? (T = L)
    : s === "post"
    ? (T = () => Cn(L, d && d.suspense))
    : ((L.pre = !0), d && (L.id = d.uid), (T = () => Sh(L)));
  const M = new dh(g, T);
  e ? (r ? L() : (A = M.run())) : s === "post" ? Cn(M.run.bind(M), d && d.suspense) : M.run();
  const R = () => {
    M.stop(), d && d.scope && uh(d.scope.effects, M);
  };
  return P && P.push(R), R;
}
function c_(t, e, r) {
  const o = this.proxy,
    s = Ie(t) ? (t.includes(".") ? r0(o, t) : () => o[t]) : t.bind(o, o);
  let u;
  jt(e) ? (u = e) : ((u = e.handler), (r = e));
  const f = Ge;
  os(this);
  const h = Eh(s, u.bind(o), r);
  return f ? os(f) : to(), h;
}
function r0(t, e) {
  const r = e.split(".");
  return () => {
    let o = t;
    for (let s = 0; s < r.length && o; s++) o = o[r[s]];
    return o;
  };
}
function Yi(t, e) {
  if (!ye(t) || t.__v_skip || ((e = e || new Set()), e.has(t))) return t;
  if ((e.add(t), Le(t))) Yi(t.value, e);
  else if (It(t)) for (let r = 0; r < t.length; r++) Yi(t[r], e);
  else if (Cm(t) || Ko(t))
    t.forEach((r) => {
      Yi(r, e);
    });
  else if (Lm(t)) for (const r in t) Yi(t[r], e);
  return t;
}
function nn(t, e) {
  const r = Je;
  if (r === null) return t;
  const o = $c(r) || r.proxy,
    s = t.dirs || (t.dirs = []);
  for (let u = 0; u < e.length; u++) {
    let [f, h, d, g = we] = e[u];
    f &&
      (jt(f) && (f = { mounted: f, updated: f }),
      f.deep && Yi(h),
      s.push({ dir: f, instance: o, value: h, oldValue: void 0, arg: d, modifiers: g }));
  }
  return t;
}
function Wi(t, e, r, o) {
  const s = t.dirs,
    u = e && e.dirs;
  for (let f = 0; f < s.length; f++) {
    const h = s[f];
    u && (h.oldValue = u[f].value);
    let d = h.dir[o];
    d && (ps(), jn(d, r, 8, [t.el, h, t, e]), gs());
  }
}
const gi = Symbol("_leaveCb"),
  Ta = Symbol("_enterCb");
function u_() {
  const t = { isMounted: !1, isLeaving: !1, isUnmounting: !1, leavingVNodes: new Map() };
  return (
    ms(() => {
      t.isMounted = !0;
    }),
    a0(() => {
      t.isUnmounting = !0;
    }),
    t
  );
}
const Bn = [Function, Array],
  i0 = {
    mode: String,
    appear: Boolean,
    persisted: Boolean,
    onBeforeEnter: Bn,
    onEnter: Bn,
    onAfterEnter: Bn,
    onEnterCancelled: Bn,
    onBeforeLeave: Bn,
    onLeave: Bn,
    onAfterLeave: Bn,
    onLeaveCancelled: Bn,
    onBeforeAppear: Bn,
    onAppear: Bn,
    onAfterAppear: Bn,
    onAppearCancelled: Bn,
  },
  f_ = {
    name: "BaseTransition",
    props: i0,
    setup(t, { slots: e }) {
      const r = Pl(),
        o = u_();
      let s;
      return () => {
        const u = e.default && s0(e.default(), !0);
        if (!u || !u.length) return;
        let f = u[0];
        if (u.length > 1) {
          for (const A of u)
            if (A.type !== Mn) {
              f = A;
              break;
            }
        }
        const h = ae(t),
          { mode: d } = h;
        if (o.isLeaving) return Gu(f);
        const g = _g(f);
        if (!g) return Gu(f);
        const v = Sf(g, h, o, r);
        kf(g, v);
        const b = r.subTree,
          w = b && _g(b);
        let S = !1;
        const { getTransitionKey: P } = g.type;
        if (P) {
          const A = P();
          s === void 0 ? (s = A) : A !== s && ((s = A), (S = !0));
        }
        if (w && w.type !== Mn && (!mr(g, w) || S)) {
          const A = Sf(w, h, o, r);
          if ((kf(w, A), d === "out-in"))
            return (
              (o.isLeaving = !0),
              (A.afterLeave = () => {
                (o.isLeaving = !1), r.update.active !== !1 && r.update();
              }),
              Gu(f)
            );
          d === "in-out" &&
            g.type !== Mn &&
            (A.delayLeave = (L, T, M) => {
              const R = o0(o, w);
              (R[String(w.key)] = w),
                (L[gi] = () => {
                  T(), (L[gi] = void 0), delete v.delayedLeave;
                }),
                (v.delayedLeave = M);
            });
        }
        return f;
      };
    },
  },
  h_ = f_;
function o0(t, e) {
  const { leavingVNodes: r } = t;
  let o = r.get(e.type);
  return o || ((o = Object.create(null)), r.set(e.type, o)), o;
}
function Sf(t, e, r, o) {
  const {
      appear: s,
      mode: u,
      persisted: f = !1,
      onBeforeEnter: h,
      onEnter: d,
      onAfterEnter: g,
      onEnterCancelled: v,
      onBeforeLeave: b,
      onLeave: w,
      onAfterLeave: S,
      onLeaveCancelled: P,
      onBeforeAppear: A,
      onAppear: L,
      onAfterAppear: T,
      onAppearCancelled: M,
    } = e,
    R = String(t.key),
    E = o0(r, t),
    B = (Y, nt) => {
      Y && jn(Y, o, 9, nt);
    },
    K = (Y, nt) => {
      const at = nt[1];
      B(Y, nt), It(Y) ? Y.every((pt) => pt.length <= 1) && at() : Y.length <= 1 && at();
    },
    ht = {
      mode: u,
      persisted: f,
      beforeEnter(Y) {
        let nt = h;
        if (!r.isMounted)
          if (s) nt = A || h;
          else return;
        Y[gi] && Y[gi](!0);
        const at = E[R];
        at && mr(t, at) && at.el[gi] && at.el[gi](), B(nt, [Y]);
      },
      enter(Y) {
        let nt = d,
          at = g,
          pt = v;
        if (!r.isMounted)
          if (s) (nt = L || d), (at = T || g), (pt = M || v);
          else return;
        let gt = !1;
        const G = (Y[Ta] = (z) => {
          gt ||
            ((gt = !0),
            z ? B(pt, [Y]) : B(at, [Y]),
            ht.delayedLeave && ht.delayedLeave(),
            (Y[Ta] = void 0));
        });
        nt ? K(nt, [Y, G]) : G();
      },
      leave(Y, nt) {
        const at = String(t.key);
        if ((Y[Ta] && Y[Ta](!0), r.isUnmounting)) return nt();
        B(b, [Y]);
        let pt = !1;
        const gt = (Y[gi] = (G) => {
          pt ||
            ((pt = !0),
            nt(),
            G ? B(P, [Y]) : B(S, [Y]),
            (Y[gi] = void 0),
            E[at] === t && delete E[at]);
        });
        (E[at] = t), w ? K(w, [Y, gt]) : gt();
      },
      clone(Y) {
        return Sf(Y, e, r, o);
      },
    };
  return ht;
}
function Gu(t) {
  if (Nc(t)) return (t = Ei(t)), (t.children = null), t;
}
function _g(t) {
  return Nc(t) ? (t.children ? t.children[0] : void 0) : t;
}
function kf(t, e) {
  t.shapeFlag & 6 && t.component
    ? kf(t.component.subTree, e)
    : t.shapeFlag & 128
    ? ((t.ssContent.transition = e.clone(t.ssContent)),
      (t.ssFallback.transition = e.clone(t.ssFallback)))
    : (t.transition = e);
}
function s0(t, e = !1, r) {
  let o = [],
    s = 0;
  for (let u = 0; u < t.length; u++) {
    let f = t[u];
    const h = r == null ? f.key : String(r) + String(f.key != null ? f.key : u);
    f.type === ne
      ? (f.patchFlag & 128 && s++, (o = o.concat(s0(f.children, e, h))))
      : (e || f.type !== Mn) && o.push(h != null ? Ei(f, { key: h }) : f);
  }
  if (s > 1) for (let u = 0; u < o.length; u++) o[u].patchFlag = -2;
  return o;
} /*! #__NO_SIDE_EFFECTS__ */
function ie(t, e) {
  return jt(t) ? (() => ze({ name: t.name }, e, { setup: t }))() : t;
}
const ll = (t) => !!t.type.__asyncLoader,
  Nc = (t) => t.type.__isKeepAlive;
function d_(t, e) {
  l0(t, "a", e);
}
function p_(t, e) {
  l0(t, "da", e);
}
function l0(t, e, r = Ge) {
  const o =
    t.__wdc ||
    (t.__wdc = () => {
      let s = r;
      for (; s; ) {
        if (s.isDeactivated) return;
        s = s.parent;
      }
      return t();
    });
  if ((Pc(e, o, r), r)) {
    let s = r.parent;
    for (; s && s.parent; ) Nc(s.parent.vnode) && g_(o, e, r, s), (s = s.parent);
  }
}
function g_(t, e, r, o) {
  const s = Pc(e, t, o, !0);
  Lh(() => {
    uh(o[e], s);
  }, r);
}
function Pc(t, e, r = Ge, o = !1) {
  if (r) {
    const s = r[t] || (r[t] = []),
      u =
        e.__weh ||
        (e.__weh = (...f) => {
          if (r.isUnmounted) return;
          ps(), os(r);
          const h = jn(e, r, t, f);
          return to(), gs(), h;
        });
    return o ? s.unshift(u) : s.push(u), u;
  }
}
const Kr = (t) => (e, r = Ge) => (!bl || t === "sp") && Pc(t, (...o) => e(...o), r),
  v_ = Kr("bm"),
  ms = Kr("m"),
  m_ = Kr("bu"),
  y_ = Kr("u"),
  a0 = Kr("bum"),
  Lh = Kr("um"),
  b_ = Kr("sp"),
  w_ = Kr("rtg"),
  x_ = Kr("rtc");
function __(t, e = Ge) {
  Pc("ec", t, e);
}
function Rn(t, e, r, o) {
  let s;
  const u = r && r[o];
  if (It(t) || Ie(t)) {
    s = new Array(t.length);
    for (let f = 0, h = t.length; f < h; f++) s[f] = e(t[f], f, void 0, u && u[f]);
  } else if (typeof t == "number") {
    s = new Array(t);
    for (let f = 0; f < t; f++) s[f] = e(f + 1, f, void 0, u && u[f]);
  } else if (ye(t))
    if (t[Symbol.iterator]) s = Array.from(t, (f, h) => e(f, h, void 0, u && u[h]));
    else {
      const f = Object.keys(t);
      s = new Array(f.length);
      for (let h = 0, d = f.length; h < d; h++) {
        const g = f[h];
        s[h] = e(t[g], g, h, u && u[h]);
      }
    }
  else s = [];
  return r && (r[o] = s), s;
}
function sr(t, e, r = {}, o, s) {
  if (Je.isCE || (Je.parent && ll(Je.parent) && Je.parent.isCE))
    return e !== "default" && (r.name = e), Ft("slot", r, o && o());
  let u = t[e];
  u && u._c && (u._d = !1), st();
  const f = u && c0(u(r)),
    h = te(
      ne,
      { key: r.key || (f && f.key) || `_${e}` },
      f || (o ? o() : []),
      f && t._ === 1 ? 64 : -2,
    );
  return !s && h.scopeId && (h.slotScopeIds = [h.scopeId + "-s"]), u && u._c && (u._d = !0), h;
}
function c0(t) {
  return t.some((e) => (yl(e) ? !(e.type === Mn || (e.type === ne && !c0(e.children))) : !0))
    ? t
    : null;
}
const Cf = (t) => (t ? (_0(t) ? $c(t) || t.proxy : Cf(t.parent)) : null),
  al = ze(Object.create(null), {
    $: (t) => t,
    $el: (t) => t.vnode.el,
    $data: (t) => t.data,
    $props: (t) => t.props,
    $attrs: (t) => t.attrs,
    $slots: (t) => t.slots,
    $refs: (t) => t.refs,
    $parent: (t) => Cf(t.parent),
    $root: (t) => Cf(t.root),
    $emit: (t) => t.emit,
    $options: (t) => Ah(t),
    $forceUpdate: (t) => t.f || (t.f = () => Sh(t.update)),
    $nextTick: (t) => t.n || (t.n = Br.bind(t.proxy)),
    $watch: (t) => c_.bind(t),
  }),
  Ku = (t, e) => t !== we && !t.__isScriptSetup && le(t, e),
  S_ = {
    get({ _: t }, e) {
      const {
        ctx: r,
        setupState: o,
        data: s,
        props: u,
        accessCache: f,
        type: h,
        appContext: d,
      } = t;
      let g;
      if (e[0] !== "$") {
        const S = f[e];
        if (S !== void 0)
          switch (S) {
            case 1:
              return o[e];
            case 2:
              return s[e];
            case 4:
              return r[e];
            case 3:
              return u[e];
          }
        else {
          if (Ku(o, e)) return (f[e] = 1), o[e];
          if (s !== we && le(s, e)) return (f[e] = 2), s[e];
          if ((g = t.propsOptions[0]) && le(g, e)) return (f[e] = 3), u[e];
          if (r !== we && le(r, e)) return (f[e] = 4), r[e];
          Ef && (f[e] = 0);
        }
      }
      const v = al[e];
      let b, w;
      if (v) return e === "$attrs" && Nn(t, "get", e), v(t);
      if ((b = h.__cssModules) && (b = b[e])) return b;
      if (r !== we && le(r, e)) return (f[e] = 4), r[e];
      if (((w = d.config.globalProperties), le(w, e))) return w[e];
    },
    set({ _: t }, e, r) {
      const { data: o, setupState: s, ctx: u } = t;
      return Ku(s, e)
        ? ((s[e] = r), !0)
        : o !== we && le(o, e)
        ? ((o[e] = r), !0)
        : le(t.props, e) || (e[0] === "$" && e.slice(1) in t)
        ? !1
        : ((u[e] = r), !0);
    },
    has(
      { _: { data: t, setupState: e, accessCache: r, ctx: o, appContext: s, propsOptions: u } },
      f,
    ) {
      let h;
      return (
        !!r[f] ||
        (t !== we && le(t, f)) ||
        Ku(e, f) ||
        ((h = u[0]) && le(h, f)) ||
        le(o, f) ||
        le(al, f) ||
        le(s.config.globalProperties, f)
      );
    },
    defineProperty(t, e, r) {
      return (
        r.get != null ? (t._.accessCache[e] = 0) : le(r, "value") && this.set(t, e, r.value, null),
        Reflect.defineProperty(t, e, r)
      );
    },
  };
function k_() {
  return C_().attrs;
}
function u0(t, e, r) {
  const o = Pl();
  if (r && r.local) {
    const s = Zt(t[e]);
    return (
      Re(
        () => t[e],
        (u) => (s.value = u),
      ),
      Re(s, (u) => {
        u !== t[e] && o.emit(`update:${e}`, u);
      }),
      s
    );
  } else
    return {
      __v_isRef: !0,
      get value() {
        return t[e];
      },
      set value(s) {
        o.emit(`update:${e}`, s);
      },
    };
}
function C_() {
  const t = Pl();
  return t.setupContext || (t.setupContext = k0(t));
}
function ec(t) {
  return It(t) ? t.reduce((e, r) => ((e[r] = null), e), {}) : t;
}
function Tf(t, e) {
  return !t || !e ? t || e : It(t) && It(e) ? t.concat(e) : ze({}, ec(t), ec(e));
}
let Ef = !0;
function T_(t) {
  const e = Ah(t),
    r = t.proxy,
    o = t.ctx;
  (Ef = !1), e.beforeCreate && Sg(e.beforeCreate, t, "bc");
  const {
    data: s,
    computed: u,
    methods: f,
    watch: h,
    provide: d,
    inject: g,
    created: v,
    beforeMount: b,
    mounted: w,
    beforeUpdate: S,
    updated: P,
    activated: A,
    deactivated: L,
    beforeDestroy: T,
    beforeUnmount: M,
    destroyed: R,
    unmounted: E,
    render: B,
    renderTracked: K,
    renderTriggered: ht,
    errorCaptured: Y,
    serverPrefetch: nt,
    expose: at,
    inheritAttrs: pt,
    components: gt,
    directives: G,
    filters: z,
  } = e;
  if ((g && E_(g, o, null), f))
    for (const H in f) {
      const J = f[H];
      jt(J) && (o[H] = J.bind(r));
    }
  if (s) {
    const H = s.call(r, r);
    ye(H) && (t.data = Un(H));
  }
  if (((Ef = !0), u))
    for (const H in u) {
      const J = u[H],
        yt = jt(J) ? J.bind(r, r) : jt(J.get) ? J.get.bind(r, r) : wr,
        At = !jt(J) && jt(J.set) ? J.set.bind(r) : wr,
        qt = xt({ get: yt, set: At });
      Object.defineProperty(o, H, {
        enumerable: !0,
        configurable: !0,
        get: () => qt.value,
        set: (Ht) => (qt.value = Ht),
      });
    }
  if (h) for (const H in h) f0(h[H], o, r, H);
  if (d) {
    const H = jt(d) ? d.call(r) : d;
    Reflect.ownKeys(H).forEach((J) => {
      qa(J, H[J]);
    });
  }
  v && Sg(v, t, "c");
  function F(H, J) {
    It(J) ? J.forEach((yt) => H(yt.bind(r))) : J && H(J.bind(r));
  }
  if (
    (F(v_, b),
    F(ms, w),
    F(m_, S),
    F(y_, P),
    F(d_, A),
    F(p_, L),
    F(__, Y),
    F(x_, K),
    F(w_, ht),
    F(a0, M),
    F(Lh, E),
    F(b_, nt),
    It(at))
  )
    if (at.length) {
      const H = t.exposed || (t.exposed = {});
      at.forEach((J) => {
        Object.defineProperty(H, J, { get: () => r[J], set: (yt) => (r[J] = yt) });
      });
    } else t.exposed || (t.exposed = {});
  B && t.render === wr && (t.render = B),
    pt != null && (t.inheritAttrs = pt),
    gt && (t.components = gt),
    G && (t.directives = G);
}
function E_(t, e, r = wr) {
  It(t) && (t = Lf(t));
  for (const o in t) {
    const s = t[o];
    let u;
    ye(s)
      ? "default" in s
        ? (u = qr(s.from || o, s.default, !0))
        : (u = qr(s.from || o))
      : (u = qr(s)),
      Le(u)
        ? Object.defineProperty(e, o, {
            enumerable: !0,
            configurable: !0,
            get: () => u.value,
            set: (f) => (u.value = f),
          })
        : (e[o] = u);
  }
}
function Sg(t, e, r) {
  jn(It(t) ? t.map((o) => o.bind(e.proxy)) : t.bind(e.proxy), e, r);
}
function f0(t, e, r, o) {
  const s = o.includes(".") ? r0(r, o) : () => r[o];
  if (Ie(t)) {
    const u = e[t];
    jt(u) && Re(s, u);
  } else if (jt(t)) Re(s, t.bind(r));
  else if (ye(t))
    if (It(t)) t.forEach((u) => f0(u, e, r, o));
    else {
      const u = jt(t.handler) ? t.handler.bind(r) : e[t.handler];
      jt(u) && Re(s, u, t);
    }
}
function Ah(t) {
  const e = t.type,
    { mixins: r, extends: o } = e,
    {
      mixins: s,
      optionsCache: u,
      config: { optionMergeStrategies: f },
    } = t.appContext,
    h = u.get(e);
  let d;
  return (
    h
      ? (d = h)
      : !s.length && !r && !o
      ? (d = e)
      : ((d = {}), s.length && s.forEach((g) => nc(d, g, f, !0)), nc(d, e, f)),
    ye(e) && u.set(e, d),
    d
  );
}
function nc(t, e, r, o = !1) {
  const { mixins: s, extends: u } = e;
  u && nc(t, u, r, !0), s && s.forEach((f) => nc(t, f, r, !0));
  for (const f in e)
    if (!(o && f === "expose")) {
      const h = L_[f] || (r && r[f]);
      t[f] = h ? h(t[f], e[f]) : e[f];
    }
  return t;
}
const L_ = {
  data: kg,
  props: Cg,
  emits: Cg,
  methods: il,
  computed: il,
  beforeCreate: dn,
  created: dn,
  beforeMount: dn,
  mounted: dn,
  beforeUpdate: dn,
  updated: dn,
  beforeDestroy: dn,
  beforeUnmount: dn,
  destroyed: dn,
  unmounted: dn,
  activated: dn,
  deactivated: dn,
  errorCaptured: dn,
  serverPrefetch: dn,
  components: il,
  directives: il,
  watch: M_,
  provide: kg,
  inject: A_,
};
function kg(t, e) {
  return e
    ? t
      ? function () {
          return ze(jt(t) ? t.call(this, this) : t, jt(e) ? e.call(this, this) : e);
        }
      : e
    : t;
}
function A_(t, e) {
  return il(Lf(t), Lf(e));
}
function Lf(t) {
  if (It(t)) {
    const e = {};
    for (let r = 0; r < t.length; r++) e[t[r]] = t[r];
    return e;
  }
  return t;
}
function dn(t, e) {
  return t ? [...new Set([].concat(t, e))] : e;
}
function il(t, e) {
  return t ? ze(Object.create(null), t, e) : e;
}
function Cg(t, e) {
  return t
    ? It(t) && It(e)
      ? [...new Set([...t, ...e])]
      : ze(Object.create(null), ec(t), ec(e ?? {}))
    : e;
}
function M_(t, e) {
  if (!t) return e;
  if (!e) return t;
  const r = ze(Object.create(null), t);
  for (const o in e) r[o] = dn(t[o], e[o]);
  return r;
}
function h0() {
  return {
    app: null,
    config: {
      isNativeTag: Zx,
      performance: !1,
      globalProperties: {},
      optionMergeStrategies: {},
      errorHandler: void 0,
      warnHandler: void 0,
      compilerOptions: {},
    },
    mixins: [],
    components: {},
    directives: {},
    provides: Object.create(null),
    optionsCache: new WeakMap(),
    propsCache: new WeakMap(),
    emitsCache: new WeakMap(),
  };
}
let N_ = 0;
function P_(t, e) {
  return function (o, s = null) {
    jt(o) || (o = ze({}, o)), s != null && !ye(s) && (s = null);
    const u = h0(),
      f = new WeakSet();
    let h = !1;
    const d = (u.app = {
      _uid: N_++,
      _component: o,
      _props: s,
      _container: null,
      _context: u,
      _instance: null,
      version: eS,
      get config() {
        return u.config;
      },
      set config(g) {},
      use(g, ...v) {
        return (
          f.has(g) ||
            (g && jt(g.install) ? (f.add(g), g.install(d, ...v)) : jt(g) && (f.add(g), g(d, ...v))),
          d
        );
      },
      mixin(g) {
        return u.mixins.includes(g) || u.mixins.push(g), d;
      },
      component(g, v) {
        return v ? ((u.components[g] = v), d) : u.components[g];
      },
      directive(g, v) {
        return v ? ((u.directives[g] = v), d) : u.directives[g];
      },
      mount(g, v, b) {
        if (!h) {
          const w = Ft(o, s);
          return (
            (w.appContext = u),
            v && e ? e(w, g) : t(w, g, b),
            (h = !0),
            (d._container = g),
            (g.__vue_app__ = d),
            $c(w.component) || w.component.proxy
          );
        }
      },
      unmount() {
        h && (t(null, d._container), delete d._container.__vue_app__);
      },
      provide(g, v) {
        return (u.provides[g] = v), d;
      },
      runWithContext(g) {
        rc = d;
        try {
          return g();
        } finally {
          rc = null;
        }
      },
    });
    return d;
  };
}
let rc = null;
function qa(t, e) {
  if (Ge) {
    let r = Ge.provides;
    const o = Ge.parent && Ge.parent.provides;
    o === r && (r = Ge.provides = Object.create(o)), (r[t] = e);
  }
}
function qr(t, e, r = !1) {
  const o = Ge || Je;
  if (o || rc) {
    const s = o
      ? o.parent == null
        ? o.vnode.appContext && o.vnode.appContext.provides
        : o.parent.provides
      : rc._context.provides;
    if (s && t in s) return s[t];
    if (arguments.length > 1) return r && jt(e) ? e.call(o && o.proxy) : e;
  }
}
function O_(t, e, r, o = !1) {
  const s = {},
    u = {};
  Za(u, Dc, 1), (t.propsDefaults = Object.create(null)), d0(t, e, s, u);
  for (const f in t.propsOptions[0]) f in s || (s[f] = void 0);
  r ? (t.props = o ? s : Wm(s)) : t.type.props ? (t.props = s) : (t.props = u), (t.attrs = u);
}
function D_(t, e, r, o) {
  const {
      props: s,
      attrs: u,
      vnode: { patchFlag: f },
    } = t,
    h = ae(s),
    [d] = t.propsOptions;
  let g = !1;
  if ((o || f > 0) && !(f & 16)) {
    if (f & 8) {
      const v = t.vnode.dynamicProps;
      for (let b = 0; b < v.length; b++) {
        let w = v[b];
        if (Ac(t.emitsOptions, w)) continue;
        const S = e[w];
        if (d)
          if (le(u, w)) S !== u[w] && ((u[w] = S), (g = !0));
          else {
            const P = _r(w);
            s[P] = Af(d, h, P, S, t, !1);
          }
        else S !== u[w] && ((u[w] = S), (g = !0));
      }
    }
  } else {
    d0(t, e, s, u) && (g = !0);
    let v;
    for (const b in h)
      (!e || (!le(e, b) && ((v = co(b)) === b || !le(e, v)))) &&
        (d
          ? r && (r[b] !== void 0 || r[v] !== void 0) && (s[b] = Af(d, h, b, void 0, t, !0))
          : delete s[b]);
    if (u !== h) for (const b in u) (!e || !le(e, b)) && (delete u[b], (g = !0));
  }
  g && Fr(t, "set", "$attrs");
}
function d0(t, e, r, o) {
  const [s, u] = t.propsOptions;
  let f = !1,
    h;
  if (e)
    for (let d in e) {
      if (Ia(d)) continue;
      const g = e[d];
      let v;
      s && le(s, (v = _r(d)))
        ? !u || !u.includes(v)
          ? (r[v] = g)
          : ((h || (h = {}))[v] = g)
        : Ac(t.emitsOptions, d) || ((!(d in o) || g !== o[d]) && ((o[d] = g), (f = !0)));
    }
  if (u) {
    const d = ae(r),
      g = h || we;
    for (let v = 0; v < u.length; v++) {
      const b = u[v];
      r[b] = Af(s, d, b, g[b], t, !le(g, b));
    }
  }
  return f;
}
function Af(t, e, r, o, s, u) {
  const f = t[r];
  if (f != null) {
    const h = le(f, "default");
    if (h && o === void 0) {
      const d = f.default;
      if (f.type !== Function && !f.skipFactory && jt(d)) {
        const { propsDefaults: g } = s;
        r in g ? (o = g[r]) : (os(s), (o = g[r] = d.call(null, e)), to());
      } else o = d;
    }
    f[0] && (u && !h ? (o = !1) : f[1] && (o === "" || o === co(r)) && (o = !0));
  }
  return o;
}
function p0(t, e, r = !1) {
  const o = e.propsCache,
    s = o.get(t);
  if (s) return s;
  const u = t.props,
    f = {},
    h = [];
  let d = !1;
  if (!jt(t)) {
    const v = (b) => {
      d = !0;
      const [w, S] = p0(b, e, !0);
      ze(f, w), S && h.push(...S);
    };
    !r && e.mixins.length && e.mixins.forEach(v),
      t.extends && v(t.extends),
      t.mixins && t.mixins.forEach(v);
  }
  if (!u && !d) return ye(t) && o.set(t, Go), Go;
  if (It(u))
    for (let v = 0; v < u.length; v++) {
      const b = _r(u[v]);
      Tg(b) && (f[b] = we);
    }
  else if (u)
    for (const v in u) {
      const b = _r(v);
      if (Tg(b)) {
        const w = u[v],
          S = (f[b] = It(w) || jt(w) ? { type: w } : ze({}, w));
        if (S) {
          const P = Ag(Boolean, S.type),
            A = Ag(String, S.type);
          (S[0] = P > -1), (S[1] = A < 0 || P < A), (P > -1 || le(S, "default")) && h.push(b);
        }
      }
    }
  const g = [f, h];
  return ye(t) && o.set(t, g), g;
}
function Tg(t) {
  return t[0] !== "$";
}
function Eg(t) {
  const e = t && t.toString().match(/^\s*(function|class) (\w+)/);
  return e ? e[2] : t === null ? "null" : "";
}
function Lg(t, e) {
  return Eg(t) === Eg(e);
}
function Ag(t, e) {
  return It(e) ? e.findIndex((r) => Lg(r, t)) : jt(e) && Lg(e, t) ? 0 : -1;
}
const g0 = (t) => t[0] === "_" || t === "$stable",
  Mh = (t) => (It(t) ? t.map(rr) : [rr(t)]),
  $_ = (t, e, r) => {
    if (e._n) return e;
    const o = ee((...s) => Mh(e(...s)), r);
    return (o._c = !1), o;
  },
  v0 = (t, e, r) => {
    const o = t._ctx;
    for (const s in t) {
      if (g0(s)) continue;
      const u = t[s];
      if (jt(u)) e[s] = $_(s, u, o);
      else if (u != null) {
        const f = Mh(u);
        e[s] = () => f;
      }
    }
  },
  m0 = (t, e) => {
    const r = Mh(e);
    t.slots.default = () => r;
  },
  R_ = (t, e) => {
    if (t.vnode.shapeFlag & 32) {
      const r = e._;
      r ? ((t.slots = ae(e)), Za(e, "_", r)) : v0(e, (t.slots = {}));
    } else (t.slots = {}), e && m0(t, e);
    Za(t.slots, Dc, 1);
  },
  z_ = (t, e, r) => {
    const { vnode: o, slots: s } = t;
    let u = !0,
      f = we;
    if (o.shapeFlag & 32) {
      const h = e._;
      h
        ? r && h === 1
          ? (u = !1)
          : (ze(s, e), !r && h === 1 && delete s._)
        : ((u = !e.$stable), v0(e, s)),
        (f = e);
    } else e && (m0(t, e), (f = { default: 1 }));
    if (u) for (const h in s) !g0(h) && f[h] == null && delete s[h];
  };
function Mf(t, e, r, o, s = !1) {
  if (It(t)) {
    t.forEach((w, S) => Mf(w, e && (It(e) ? e[S] : e), r, o, s));
    return;
  }
  if (ll(o) && !s) return;
  const u = o.shapeFlag & 4 ? $c(o.component) || o.component.proxy : o.el,
    f = s ? null : u,
    { i: h, r: d } = t,
    g = e && e.r,
    v = h.refs === we ? (h.refs = {}) : h.refs,
    b = h.setupState;
  if (
    (g != null &&
      g !== d &&
      (Ie(g) ? ((v[g] = null), le(b, g) && (b[g] = null)) : Le(g) && (g.value = null)),
    jt(d))
  )
    ki(d, h, 12, [f, v]);
  else {
    const w = Ie(d),
      S = Le(d);
    if (w || S) {
      const P = () => {
        if (t.f) {
          const A = w ? (le(b, d) ? b[d] : v[d]) : d.value;
          s
            ? It(A) && uh(A, u)
            : It(A)
            ? A.includes(u) || A.push(u)
            : w
            ? ((v[d] = [u]), le(b, d) && (b[d] = v[d]))
            : ((d.value = [u]), t.k && (v[t.k] = d.value));
        } else w ? ((v[d] = f), le(b, d) && (b[d] = f)) : S && ((d.value = f), t.k && (v[t.k] = f));
      };
      f ? ((P.id = -1), Cn(P, r)) : P();
    }
  }
}
const Cn = l_;
function I_(t) {
  return F_(t);
}
function F_(t, e) {
  const r = mf();
  r.__VUE__ = !0;
  const {
      insert: o,
      remove: s,
      patchProp: u,
      createElement: f,
      createText: h,
      createComment: d,
      setText: g,
      setElementText: v,
      parentNode: b,
      nextSibling: w,
      setScopeId: S = wr,
      insertStaticContent: P,
    } = t,
    A = ($, I, V, Q = null, ot = null, ut = null, St = !1, mt = null, ct = !!I.dynamicChildren) => {
      if ($ === I) return;
      $ && !mr($, I) && ((Q = j($)), Ht($, ot, ut, !0), ($ = null)),
        I.patchFlag === -2 && ((ct = !1), (I.dynamicChildren = null));
      const { type: ft, ref: $t, shapeFlag: Nt } = I;
      switch (ft) {
        case Oc:
          L($, I, V, Q);
          break;
        case Mn:
          T($, I, V, Q);
          break;
        case Xu:
          $ == null && M(I, V, Q, St);
          break;
        case ne:
          gt($, I, V, Q, ot, ut, St, mt, ct);
          break;
        default:
          Nt & 1
            ? B($, I, V, Q, ot, ut, St, mt, ct)
            : Nt & 6
            ? G($, I, V, Q, ot, ut, St, mt, ct)
            : (Nt & 64 || Nt & 128) && ft.process($, I, V, Q, ot, ut, St, mt, ct, lt);
      }
      $t != null && ot && Mf($t, $ && $.ref, ut, I || $, !I);
    },
    L = ($, I, V, Q) => {
      if ($ == null) o((I.el = h(I.children)), V, Q);
      else {
        const ot = (I.el = $.el);
        I.children !== $.children && g(ot, I.children);
      }
    },
    T = ($, I, V, Q) => {
      $ == null ? o((I.el = d(I.children || "")), V, Q) : (I.el = $.el);
    },
    M = ($, I, V, Q) => {
      [$.el, $.anchor] = P($.children, I, V, Q, $.el, $.anchor);
    },
    R = ({ el: $, anchor: I }, V, Q) => {
      let ot;
      for (; $ && $ !== I; ) (ot = w($)), o($, V, Q), ($ = ot);
      o(I, V, Q);
    },
    E = ({ el: $, anchor: I }) => {
      let V;
      for (; $ && $ !== I; ) (V = w($)), s($), ($ = V);
      s(I);
    },
    B = ($, I, V, Q, ot, ut, St, mt, ct) => {
      (St = St || I.type === "svg"),
        $ == null ? K(I, V, Q, ot, ut, St, mt, ct) : nt($, I, ot, ut, St, mt, ct);
    },
    K = ($, I, V, Q, ot, ut, St, mt) => {
      let ct, ft;
      const { type: $t, props: Nt, shapeFlag: Dt, transition: Bt, dirs: Kt } = $;
      if (
        ((ct = $.el = f($.type, ut, Nt && Nt.is, Nt)),
        Dt & 8
          ? v(ct, $.children)
          : Dt & 16 && Y($.children, ct, null, Q, ot, ut && $t !== "foreignObject", St, mt),
        Kt && Wi($, null, Q, "created"),
        ht(ct, $, $.scopeId, St, Q),
        Nt)
      ) {
        for (const oe in Nt)
          oe !== "value" && !Ia(oe) && u(ct, oe, null, Nt[oe], ut, $.children, Q, ot, Tt);
        "value" in Nt && u(ct, "value", null, Nt.value),
          (ft = Nt.onVnodeBeforeMount) && gr(ft, Q, $);
      }
      Kt && Wi($, null, Q, "beforeMount");
      const re = q_(ot, Bt);
      re && Bt.beforeEnter(ct),
        o(ct, I, V),
        ((ft = Nt && Nt.onVnodeMounted) || re || Kt) &&
          Cn(() => {
            ft && gr(ft, Q, $), re && Bt.enter(ct), Kt && Wi($, null, Q, "mounted");
          }, ot);
    },
    ht = ($, I, V, Q, ot) => {
      if ((V && S($, V), Q)) for (let ut = 0; ut < Q.length; ut++) S($, Q[ut]);
      if (ot) {
        let ut = ot.subTree;
        if (I === ut) {
          const St = ot.vnode;
          ht($, St, St.scopeId, St.slotScopeIds, ot.parent);
        }
      }
    },
    Y = ($, I, V, Q, ot, ut, St, mt, ct = 0) => {
      for (let ft = ct; ft < $.length; ft++) {
        const $t = ($[ft] = mt ? vi($[ft]) : rr($[ft]));
        A(null, $t, I, V, Q, ot, ut, St, mt);
      }
    },
    nt = ($, I, V, Q, ot, ut, St) => {
      const mt = (I.el = $.el);
      let { patchFlag: ct, dynamicChildren: ft, dirs: $t } = I;
      ct |= $.patchFlag & 16;
      const Nt = $.props || we,
        Dt = I.props || we;
      let Bt;
      V && Ui(V, !1),
        (Bt = Dt.onVnodeBeforeUpdate) && gr(Bt, V, I, $),
        $t && Wi(I, $, V, "beforeUpdate"),
        V && Ui(V, !0);
      const Kt = ot && I.type !== "foreignObject";
      if (
        (ft
          ? at($.dynamicChildren, ft, mt, V, Q, Kt, ut)
          : St || J($, I, mt, null, V, Q, Kt, ut, !1),
        ct > 0)
      ) {
        if (ct & 16) pt(mt, I, Nt, Dt, V, Q, ot);
        else if (
          (ct & 2 && Nt.class !== Dt.class && u(mt, "class", null, Dt.class, ot),
          ct & 4 && u(mt, "style", Nt.style, Dt.style, ot),
          ct & 8)
        ) {
          const re = I.dynamicProps;
          for (let oe = 0; oe < re.length; oe++) {
            const fe = re[oe],
              se = Nt[fe],
              rn = Dt[fe];
            (rn !== se || fe === "value") && u(mt, fe, se, rn, ot, $.children, V, Q, Tt);
          }
        }
        ct & 1 && $.children !== I.children && v(mt, I.children);
      } else !St && ft == null && pt(mt, I, Nt, Dt, V, Q, ot);
      ((Bt = Dt.onVnodeUpdated) || $t) &&
        Cn(() => {
          Bt && gr(Bt, V, I, $), $t && Wi(I, $, V, "updated");
        }, Q);
    },
    at = ($, I, V, Q, ot, ut, St) => {
      for (let mt = 0; mt < I.length; mt++) {
        const ct = $[mt],
          ft = I[mt],
          $t = ct.el && (ct.type === ne || !mr(ct, ft) || ct.shapeFlag & 70) ? b(ct.el) : V;
        A(ct, ft, $t, null, Q, ot, ut, St, !0);
      }
    },
    pt = ($, I, V, Q, ot, ut, St) => {
      if (V !== Q) {
        if (V !== we)
          for (const mt in V)
            !Ia(mt) && !(mt in Q) && u($, mt, V[mt], null, St, I.children, ot, ut, Tt);
        for (const mt in Q) {
          if (Ia(mt)) continue;
          const ct = Q[mt],
            ft = V[mt];
          ct !== ft && mt !== "value" && u($, mt, ft, ct, St, I.children, ot, ut, Tt);
        }
        "value" in Q && u($, "value", V.value, Q.value);
      }
    },
    gt = ($, I, V, Q, ot, ut, St, mt, ct) => {
      const ft = (I.el = $ ? $.el : h("")),
        $t = (I.anchor = $ ? $.anchor : h(""));
      let { patchFlag: Nt, dynamicChildren: Dt, slotScopeIds: Bt } = I;
      Bt && (mt = mt ? mt.concat(Bt) : Bt),
        $ == null
          ? (o(ft, V, Q), o($t, V, Q), Y(I.children, V, $t, ot, ut, St, mt, ct))
          : Nt > 0 && Nt & 64 && Dt && $.dynamicChildren
          ? (at($.dynamicChildren, Dt, V, ot, ut, St, mt),
            (I.key != null || (ot && I === ot.subTree)) && y0($, I, !0))
          : J($, I, V, $t, ot, ut, St, mt, ct);
    },
    G = ($, I, V, Q, ot, ut, St, mt, ct) => {
      (I.slotScopeIds = mt),
        $ == null
          ? I.shapeFlag & 512
            ? ot.ctx.activate(I, V, Q, St, ct)
            : z(I, V, Q, ot, ut, St, ct)
          : k($, I, ct);
    },
    z = ($, I, V, Q, ot, ut, St) => {
      const mt = ($.component = G_($, Q, ot));
      if ((Nc($) && (mt.ctx.renderer = lt), K_(mt), mt.asyncDep)) {
        if ((ot && ot.registerDep(mt, F), !$.el)) {
          const ct = (mt.subTree = Ft(Mn));
          T(null, ct, I, V);
        }
        return;
      }
      F(mt, $, I, V, ot, ut, St);
    },
    k = ($, I, V) => {
      const Q = (I.component = $.component);
      if (Z1($, I, V))
        if (Q.asyncDep && !Q.asyncResolved) {
          H(Q, I, V);
          return;
        } else (Q.next = I), U1(Q.update), Q.update();
      else (I.el = $.el), (Q.vnode = I);
    },
    F = ($, I, V, Q, ot, ut, St) => {
      const mt = () => {
          if ($.isMounted) {
            let { next: $t, bu: Nt, u: Dt, parent: Bt, vnode: Kt } = $,
              re = $t,
              oe;
            Ui($, !1),
              $t ? (($t.el = Kt.el), H($, $t, St)) : ($t = Kt),
              Nt && Fa(Nt),
              (oe = $t.props && $t.props.onVnodeBeforeUpdate) && gr(oe, Bt, $t, Kt),
              Ui($, !0);
            const fe = Vu($),
              se = $.subTree;
            ($.subTree = fe),
              A(se, fe, b(se.el), j(se), $, ot, ut),
              ($t.el = fe.el),
              re === null && kh($, fe.el),
              Dt && Cn(Dt, ot),
              (oe = $t.props && $t.props.onVnodeUpdated) && Cn(() => gr(oe, Bt, $t, Kt), ot);
          } else {
            let $t;
            const { el: Nt, props: Dt } = I,
              { bm: Bt, m: Kt, parent: re } = $,
              oe = ll(I);
            if (
              (Ui($, !1),
              Bt && Fa(Bt),
              !oe && ($t = Dt && Dt.onVnodeBeforeMount) && gr($t, re, I),
              Ui($, !0),
              Nt && Et)
            ) {
              const fe = () => {
                ($.subTree = Vu($)), Et(Nt, $.subTree, $, ot, null);
              };
              oe ? I.type.__asyncLoader().then(() => !$.isUnmounted && fe()) : fe();
            } else {
              const fe = ($.subTree = Vu($));
              A(null, fe, V, Q, $, ot, ut), (I.el = fe.el);
            }
            if ((Kt && Cn(Kt, ot), !oe && ($t = Dt && Dt.onVnodeMounted))) {
              const fe = I;
              Cn(() => gr($t, re, fe), ot);
            }
            (I.shapeFlag & 256 || (re && ll(re.vnode) && re.vnode.shapeFlag & 256)) &&
              $.a &&
              Cn($.a, ot),
              ($.isMounted = !0),
              (I = V = Q = null);
          }
        },
        ct = ($.effect = new dh(mt, () => Sh(ft), $.scope)),
        ft = ($.update = () => ct.run());
      (ft.id = $.uid), Ui($, !0), ft();
    },
    H = ($, I, V) => {
      I.component = $;
      const Q = $.vnode.props;
      ($.vnode = I), ($.next = null), D_($, I.props, Q, V), z_($, I.children, V), ps(), yg(), gs();
    },
    J = ($, I, V, Q, ot, ut, St, mt, ct = !1) => {
      const ft = $ && $.children,
        $t = $ ? $.shapeFlag : 0,
        Nt = I.children,
        { patchFlag: Dt, shapeFlag: Bt } = I;
      if (Dt > 0) {
        if (Dt & 128) {
          At(ft, Nt, V, Q, ot, ut, St, mt, ct);
          return;
        } else if (Dt & 256) {
          yt(ft, Nt, V, Q, ot, ut, St, mt, ct);
          return;
        }
      }
      Bt & 8
        ? ($t & 16 && Tt(ft, ot, ut), Nt !== ft && v(V, Nt))
        : $t & 16
        ? Bt & 16
          ? At(ft, Nt, V, Q, ot, ut, St, mt, ct)
          : Tt(ft, ot, ut, !0)
        : ($t & 8 && v(V, ""), Bt & 16 && Y(Nt, V, Q, ot, ut, St, mt, ct));
    },
    yt = ($, I, V, Q, ot, ut, St, mt, ct) => {
      ($ = $ || Go), (I = I || Go);
      const ft = $.length,
        $t = I.length,
        Nt = Math.min(ft, $t);
      let Dt;
      for (Dt = 0; Dt < Nt; Dt++) {
        const Bt = (I[Dt] = ct ? vi(I[Dt]) : rr(I[Dt]));
        A($[Dt], Bt, V, null, ot, ut, St, mt, ct);
      }
      ft > $t ? Tt($, ot, ut, !0, !1, Nt) : Y(I, V, Q, ot, ut, St, mt, ct, Nt);
    },
    At = ($, I, V, Q, ot, ut, St, mt, ct) => {
      let ft = 0;
      const $t = I.length;
      let Nt = $.length - 1,
        Dt = $t - 1;
      for (; ft <= Nt && ft <= Dt; ) {
        const Bt = $[ft],
          Kt = (I[ft] = ct ? vi(I[ft]) : rr(I[ft]));
        if (mr(Bt, Kt)) A(Bt, Kt, V, null, ot, ut, St, mt, ct);
        else break;
        ft++;
      }
      for (; ft <= Nt && ft <= Dt; ) {
        const Bt = $[Nt],
          Kt = (I[Dt] = ct ? vi(I[Dt]) : rr(I[Dt]));
        if (mr(Bt, Kt)) A(Bt, Kt, V, null, ot, ut, St, mt, ct);
        else break;
        Nt--, Dt--;
      }
      if (ft > Nt) {
        if (ft <= Dt) {
          const Bt = Dt + 1,
            Kt = Bt < $t ? I[Bt].el : Q;
          for (; ft <= Dt; )
            A(null, (I[ft] = ct ? vi(I[ft]) : rr(I[ft])), V, Kt, ot, ut, St, mt, ct), ft++;
        }
      } else if (ft > Dt) for (; ft <= Nt; ) Ht($[ft], ot, ut, !0), ft++;
      else {
        const Bt = ft,
          Kt = ft,
          re = new Map();
        for (ft = Kt; ft <= Dt; ft++) {
          const Ae = (I[ft] = ct ? vi(I[ft]) : rr(I[ft]));
          Ae.key != null && re.set(Ae.key, ft);
        }
        let oe,
          fe = 0;
        const se = Dt - Kt + 1;
        let rn = !1,
          Pn = 0;
        const wn = new Array(se);
        for (ft = 0; ft < se; ft++) wn[ft] = 0;
        for (ft = Bt; ft <= Nt; ft++) {
          const Ae = $[ft];
          if (fe >= se) {
            Ht(Ae, ot, ut, !0);
            continue;
          }
          let xn;
          if (Ae.key != null) xn = re.get(Ae.key);
          else
            for (oe = Kt; oe <= Dt; oe++)
              if (wn[oe - Kt] === 0 && mr(Ae, I[oe])) {
                xn = oe;
                break;
              }
          xn === void 0
            ? Ht(Ae, ot, ut, !0)
            : ((wn[xn - Kt] = ft + 1),
              xn >= Pn ? (Pn = xn) : (rn = !0),
              A(Ae, I[xn], V, null, ot, ut, St, mt, ct),
              fe++);
        }
        const cr = rn ? H_(wn) : Go;
        for (oe = cr.length - 1, ft = se - 1; ft >= 0; ft--) {
          const Ae = Kt + ft,
            xn = I[Ae],
            Yt = Ae + 1 < $t ? I[Ae + 1].el : Q;
          wn[ft] === 0
            ? A(null, xn, V, Yt, ot, ut, St, mt, ct)
            : rn && (oe < 0 || ft !== cr[oe] ? qt(xn, V, Yt, 2) : oe--);
        }
      }
    },
    qt = ($, I, V, Q, ot = null) => {
      const { el: ut, type: St, transition: mt, children: ct, shapeFlag: ft } = $;
      if (ft & 6) {
        qt($.component.subTree, I, V, Q);
        return;
      }
      if (ft & 128) {
        $.suspense.move(I, V, Q);
        return;
      }
      if (ft & 64) {
        St.move($, I, V, lt);
        return;
      }
      if (St === ne) {
        o(ut, I, V);
        for (let Nt = 0; Nt < ct.length; Nt++) qt(ct[Nt], I, V, Q);
        o($.anchor, I, V);
        return;
      }
      if (St === Xu) {
        R($, I, V);
        return;
      }
      if (Q !== 2 && ft & 1 && mt)
        if (Q === 0) mt.beforeEnter(ut), o(ut, I, V), Cn(() => mt.enter(ut), ot);
        else {
          const { leave: Nt, delayLeave: Dt, afterLeave: Bt } = mt,
            Kt = () => o(ut, I, V),
            re = () => {
              Nt(ut, () => {
                Kt(), Bt && Bt();
              });
            };
          Dt ? Dt(ut, Kt, re) : re();
        }
      else o(ut, I, V);
    },
    Ht = ($, I, V, Q = !1, ot = !1) => {
      const {
        type: ut,
        props: St,
        ref: mt,
        children: ct,
        dynamicChildren: ft,
        shapeFlag: $t,
        patchFlag: Nt,
        dirs: Dt,
      } = $;
      if ((mt != null && Mf(mt, null, V, $, !0), $t & 256)) {
        I.ctx.deactivate($);
        return;
      }
      const Bt = $t & 1 && Dt,
        Kt = !ll($);
      let re;
      if ((Kt && (re = St && St.onVnodeBeforeUnmount) && gr(re, I, $), $t & 6))
        Gt($.component, V, Q);
      else {
        if ($t & 128) {
          $.suspense.unmount(V, Q);
          return;
        }
        Bt && Wi($, null, I, "beforeUnmount"),
          $t & 64
            ? $.type.remove($, I, V, ot, lt, Q)
            : ft && (ut !== ne || (Nt > 0 && Nt & 64))
            ? Tt(ft, I, V, !1, !0)
            : ((ut === ne && Nt & 384) || (!ot && $t & 16)) && Tt(ct, I, V),
          Q && Qt($);
      }
      ((Kt && (re = St && St.onVnodeUnmounted)) || Bt) &&
        Cn(() => {
          re && gr(re, I, $), Bt && Wi($, null, I, "unmounted");
        }, V);
    },
    Qt = ($) => {
      const { type: I, el: V, anchor: Q, transition: ot } = $;
      if (I === ne) {
        Jt(V, Q);
        return;
      }
      if (I === Xu) {
        E($);
        return;
      }
      const ut = () => {
        s(V), ot && !ot.persisted && ot.afterLeave && ot.afterLeave();
      };
      if ($.shapeFlag & 1 && ot && !ot.persisted) {
        const { leave: St, delayLeave: mt } = ot,
          ct = () => St(V, ut);
        mt ? mt($.el, ut, ct) : ct();
      } else ut();
    },
    Jt = ($, I) => {
      let V;
      for (; $ !== I; ) (V = w($)), s($), ($ = V);
      s(I);
    },
    Gt = ($, I, V) => {
      const { bum: Q, scope: ot, update: ut, subTree: St, um: mt } = $;
      Q && Fa(Q),
        ot.stop(),
        ut && ((ut.active = !1), Ht(St, $, I, V)),
        mt && Cn(mt, I),
        Cn(() => {
          $.isUnmounted = !0;
        }, I),
        I &&
          I.pendingBranch &&
          !I.isUnmounted &&
          $.asyncDep &&
          !$.asyncResolved &&
          $.suspenseId === I.pendingId &&
          (I.deps--, I.deps === 0 && I.resolve());
    },
    Tt = ($, I, V, Q = !1, ot = !1, ut = 0) => {
      for (let St = ut; St < $.length; St++) Ht($[St], I, V, Q, ot);
    },
    j = ($) =>
      $.shapeFlag & 6
        ? j($.component.subTree)
        : $.shapeFlag & 128
        ? $.suspense.next()
        : w($.anchor || $.el),
    rt = ($, I, V) => {
      $ == null
        ? I._vnode && Ht(I._vnode, null, null, !0)
        : A(I._vnode || null, $, I, null, null, null, V),
        yg(),
        Ym(),
        (I._vnode = $);
    },
    lt = { p: A, um: Ht, m: qt, r: Qt, mt: z, mc: Y, pc: J, pbc: at, n: j, o: t };
  let Mt, Et;
  return e && ([Mt, Et] = e(lt)), { render: rt, hydrate: Mt, createApp: P_(rt, Mt) };
}
function Ui({ effect: t, update: e }, r) {
  t.allowRecurse = e.allowRecurse = r;
}
function q_(t, e) {
  return (!t || (t && !t.pendingBranch)) && e && !e.persisted;
}
function y0(t, e, r = !1) {
  const o = t.children,
    s = e.children;
  if (It(o) && It(s))
    for (let u = 0; u < o.length; u++) {
      const f = o[u];
      let h = s[u];
      h.shapeFlag & 1 &&
        !h.dynamicChildren &&
        ((h.patchFlag <= 0 || h.patchFlag === 32) && ((h = s[u] = vi(s[u])), (h.el = f.el)),
        r || y0(f, h)),
        h.type === Oc && (h.el = f.el);
    }
}
function H_(t) {
  const e = t.slice(),
    r = [0];
  let o, s, u, f, h;
  const d = t.length;
  for (o = 0; o < d; o++) {
    const g = t[o];
    if (g !== 0) {
      if (((s = r[r.length - 1]), t[s] < g)) {
        (e[o] = s), r.push(o);
        continue;
      }
      for (u = 0, f = r.length - 1; u < f; )
        (h = (u + f) >> 1), t[r[h]] < g ? (u = h + 1) : (f = h);
      g < t[r[u]] && (u > 0 && (e[o] = r[u - 1]), (r[u] = o));
    }
  }
  for (u = r.length, f = r[u - 1]; u-- > 0; ) (r[u] = f), (f = e[f]);
  return r;
}
const B_ = (t) => t.__isTeleport,
  ne = Symbol.for("v-fgt"),
  Oc = Symbol.for("v-txt"),
  Mn = Symbol.for("v-cmt"),
  Xu = Symbol.for("v-stc"),
  cl = [];
let Wn = null;
function st(t = !1) {
  cl.push((Wn = t ? null : []));
}
function b0() {
  cl.pop(), (Wn = cl[cl.length - 1] || null);
}
let is = 1;
function Mg(t) {
  is += t;
}
function w0(t) {
  return (t.dynamicChildren = is > 0 ? Wn || Go : null), b0(), is > 0 && Wn && Wn.push(t), t;
}
function kt(t, e, r, o, s, u) {
  return w0(tt(t, e, r, o, s, u, !0));
}
function te(t, e, r, o, s) {
  return w0(Ft(t, e, r, o, s, !0));
}
function yl(t) {
  return t ? t.__v_isVNode === !0 : !1;
}
function mr(t, e) {
  return t.type === e.type && t.key === e.key;
}
const Dc = "__vInternal",
  x0 = ({ key: t }) => t ?? null,
  Ha = ({ ref: t, ref_key: e, ref_for: r }) => (
    typeof t == "number" && (t = "" + t),
    t != null ? (Ie(t) || Le(t) || jt(t) ? { i: Je, r: t, k: e, f: !!r } : t) : null
  );
function tt(t, e = null, r = null, o = 0, s = null, u = t === ne ? 0 : 1, f = !1, h = !1) {
  const d = {
    __v_isVNode: !0,
    __v_skip: !0,
    type: t,
    props: e,
    key: e && x0(e),
    ref: e && Ha(e),
    scopeId: Mc,
    slotScopeIds: null,
    children: r,
    component: null,
    suspense: null,
    ssContent: null,
    ssFallback: null,
    dirs: null,
    transition: null,
    el: null,
    anchor: null,
    target: null,
    targetAnchor: null,
    staticCount: 0,
    shapeFlag: u,
    patchFlag: o,
    dynamicProps: s,
    dynamicChildren: null,
    appContext: null,
    ctx: Je,
  };
  return (
    h ? (Nh(d, r), u & 128 && t.normalize(d)) : r && (d.shapeFlag |= Ie(r) ? 8 : 16),
    is > 0 && !f && Wn && (d.patchFlag > 0 || u & 6) && d.patchFlag !== 32 && Wn.push(d),
    d
  );
}
const Ft = W_;
function W_(t, e = null, r = null, o = 0, s = null, u = !1) {
  if (((!t || t === Q1) && (t = Mn), yl(t))) {
    const h = Ei(t, e, !0);
    return (
      r && Nh(h, r),
      is > 0 && !u && Wn && (h.shapeFlag & 6 ? (Wn[Wn.indexOf(t)] = h) : Wn.push(h)),
      (h.patchFlag |= -2),
      h
    );
  }
  if ((J_(t) && (t = t.__vccOpts), e)) {
    e = U_(e);
    let { class: h, style: d } = e;
    h && !Ie(h) && (e.class = ve(h)),
      ye(d) && (Um(d) && !It(d) && (d = ze({}, d)), (e.style = An(d)));
  }
  const f = Ie(t) ? 1 : t_(t) ? 128 : B_(t) ? 64 : ye(t) ? 4 : jt(t) ? 2 : 0;
  return tt(t, e, r, o, s, f, u, !0);
}
function U_(t) {
  return t ? (Um(t) || Dc in t ? ze({}, t) : t) : null;
}
function Ei(t, e, r = !1) {
  const { props: o, ref: s, patchFlag: u, children: f } = t,
    h = e ? Ci(o || {}, e) : o;
  return {
    __v_isVNode: !0,
    __v_skip: !0,
    type: t.type,
    props: h,
    key: h && x0(h),
    ref: e && e.ref ? (r && s ? (It(s) ? s.concat(Ha(e)) : [s, Ha(e)]) : Ha(e)) : s,
    scopeId: t.scopeId,
    slotScopeIds: t.slotScopeIds,
    children: f,
    target: t.target,
    targetAnchor: t.targetAnchor,
    staticCount: t.staticCount,
    shapeFlag: t.shapeFlag,
    patchFlag: e && t.type !== ne ? (u === -1 ? 16 : u | 16) : u,
    dynamicProps: t.dynamicProps,
    dynamicChildren: t.dynamicChildren,
    appContext: t.appContext,
    dirs: t.dirs,
    transition: t.transition,
    component: t.component,
    suspense: t.suspense,
    ssContent: t.ssContent && Ei(t.ssContent),
    ssFallback: t.ssFallback && Ei(t.ssFallback),
    el: t.el,
    anchor: t.anchor,
    ctx: t.ctx,
    ce: t.ce,
  };
}
function me(t = " ", e = 0) {
  return Ft(Oc, null, t, e);
}
function Vt(t = "", e = !1) {
  return e ? (st(), te(Mn, null, t)) : Ft(Mn, null, t);
}
function rr(t) {
  return t == null || typeof t == "boolean"
    ? Ft(Mn)
    : It(t)
    ? Ft(ne, null, t.slice())
    : typeof t == "object"
    ? vi(t)
    : Ft(Oc, null, String(t));
}
function vi(t) {
  return (t.el === null && t.patchFlag !== -1) || t.memo ? t : Ei(t);
}
function Nh(t, e) {
  let r = 0;
  const { shapeFlag: o } = t;
  if (e == null) e = null;
  else if (It(e)) r = 16;
  else if (typeof e == "object")
    if (o & 65) {
      const s = e.default;
      s && (s._c && (s._d = !1), Nh(t, s()), s._c && (s._d = !0));
      return;
    } else {
      r = 32;
      const s = e._;
      !s && !(Dc in e)
        ? (e._ctx = Je)
        : s === 3 && Je && (Je.slots._ === 1 ? (e._ = 1) : ((e._ = 2), (t.patchFlag |= 1024)));
    }
  else
    jt(e)
      ? ((e = { default: e, _ctx: Je }), (r = 32))
      : ((e = String(e)), o & 64 ? ((r = 16), (e = [me(e)])) : (r = 8));
  (t.children = e), (t.shapeFlag |= r);
}
function Ci(...t) {
  const e = {};
  for (let r = 0; r < t.length; r++) {
    const o = t[r];
    for (const s in o)
      if (s === "class") e.class !== o.class && (e.class = ve([e.class, o.class]));
      else if (s === "style") e.style = An([e.style, o.style]);
      else if (_c(s)) {
        const u = e[s],
          f = o[s];
        f && u !== f && !(It(u) && u.includes(f)) && (e[s] = u ? [].concat(u, f) : f);
      } else s !== "" && (e[s] = o[s]);
  }
  return e;
}
function gr(t, e, r, o = null) {
  jn(t, e, 7, [r, o]);
}
const j_ = h0();
let V_ = 0;
function G_(t, e, r) {
  const o = t.type,
    s = (e ? e.appContext : t.appContext) || j_,
    u = {
      uid: V_++,
      vnode: t,
      type: o,
      parent: e,
      appContext: s,
      root: null,
      next: null,
      subTree: null,
      effect: null,
      update: null,
      scope: new c1(!0),
      render: null,
      proxy: null,
      exposed: null,
      exposeProxy: null,
      withProxy: null,
      provides: e ? e.provides : Object.create(s.provides),
      accessCache: null,
      renderCache: [],
      components: null,
      directives: null,
      propsOptions: p0(o, s),
      emitsOptions: Jm(o, s),
      emit: null,
      emitted: null,
      propsDefaults: we,
      inheritAttrs: o.inheritAttrs,
      ctx: we,
      data: we,
      props: we,
      attrs: we,
      slots: we,
      refs: we,
      setupState: we,
      setupContext: null,
      attrsProxy: null,
      slotsProxy: null,
      suspense: r,
      suspenseId: r ? r.pendingId : 0,
      asyncDep: null,
      asyncResolved: !1,
      isMounted: !1,
      isUnmounted: !1,
      isDeactivated: !1,
      bc: null,
      c: null,
      bm: null,
      m: null,
      bu: null,
      u: null,
      um: null,
      bum: null,
      da: null,
      a: null,
      rtg: null,
      rtc: null,
      ec: null,
      sp: null,
    };
  return (
    (u.ctx = { _: u }), (u.root = e ? e.root : u), (u.emit = V1.bind(null, u)), t.ce && t.ce(u), u
  );
}
let Ge = null;
const Pl = () => Ge || Je;
let Ph,
  qo,
  Ng = "__VUE_INSTANCE_SETTERS__";
(qo = mf()[Ng]) || (qo = mf()[Ng] = []),
  qo.push((t) => (Ge = t)),
  (Ph = (t) => {
    qo.length > 1 ? qo.forEach((e) => e(t)) : qo[0](t);
  });
const os = (t) => {
    Ph(t), t.scope.on();
  },
  to = () => {
    Ge && Ge.scope.off(), Ph(null);
  };
function _0(t) {
  return t.vnode.shapeFlag & 4;
}
let bl = !1;
function K_(t, e = !1) {
  bl = e;
  const { props: r, children: o } = t.vnode,
    s = _0(t);
  O_(t, r, s, e), R_(t, o);
  const u = s ? X_(t, e) : void 0;
  return (bl = !1), u;
}
function X_(t, e) {
  const r = t.type;
  (t.accessCache = Object.create(null)), (t.proxy = mh(new Proxy(t.ctx, S_)));
  const { setup: o } = r;
  if (o) {
    const s = (t.setupContext = o.length > 1 ? k0(t) : null);
    os(t), ps();
    const u = ki(o, t, 0, [t.props, s]);
    if ((gs(), to(), Tm(u))) {
      if ((u.then(to, to), e))
        return u
          .then((f) => {
            Nf(t, f, e);
          })
          .catch((f) => {
            Nl(f, t, 0);
          });
      t.asyncDep = u;
    } else Nf(t, u, e);
  } else S0(t, e);
}
function Nf(t, e, r) {
  jt(e)
    ? t.type.__ssrInlineRender
      ? (t.ssrRender = e)
      : (t.render = e)
    : ye(e) && (t.setupState = Vm(e)),
    S0(t, r);
}
let Pg;
function S0(t, e, r) {
  const o = t.type;
  if (!t.render) {
    if (!e && Pg && !o.render) {
      const s = o.template || Ah(t).template;
      if (s) {
        const { isCustomElement: u, compilerOptions: f } = t.appContext.config,
          { delimiters: h, compilerOptions: d } = o,
          g = ze(ze({ isCustomElement: u, delimiters: h }, f), d);
        o.render = Pg(s, g);
      }
    }
    t.render = o.render || wr;
  }
  {
    os(t), ps();
    try {
      T_(t);
    } finally {
      gs(), to();
    }
  }
}
function Y_(t) {
  return (
    t.attrsProxy ||
    (t.attrsProxy = new Proxy(t.attrs, {
      get(e, r) {
        return Nn(t, "get", "$attrs"), e[r];
      },
    }))
  );
}
function k0(t) {
  const e = (r) => {
    t.exposed = r || {};
  };
  return {
    get attrs() {
      return Y_(t);
    },
    slots: t.slots,
    emit: t.emit,
    expose: e,
  };
}
function $c(t) {
  if (t.exposed)
    return (
      t.exposeProxy ||
      (t.exposeProxy = new Proxy(Vm(mh(t.exposed)), {
        get(e, r) {
          if (r in e) return e[r];
          if (r in al) return al[r](t);
        },
        has(e, r) {
          return r in e || r in al;
        },
      }))
    );
}
function Z_(t, e = !0) {
  return jt(t) ? t.displayName || t.name : t.name || (e && t.__name);
}
function J_(t) {
  return jt(t) && "__vccOpts" in t;
}
const xt = (t, e) => H1(t, e, bl);
function Ol(t, e, r) {
  const o = arguments.length;
  return o === 2
    ? ye(e) && !It(e)
      ? yl(e)
        ? Ft(t, null, [e])
        : Ft(t, e)
      : Ft(t, null, e)
    : (o > 3 ? (r = Array.prototype.slice.call(arguments, 2)) : o === 3 && yl(r) && (r = [r]),
      Ft(t, e, r));
}
const Q_ = Symbol.for("v-scx"),
  tS = () => qr(Q_),
  eS = "3.3.8",
  nS = "http://www.w3.org/2000/svg",
  Xi = typeof document < "u" ? document : null,
  Og = Xi && Xi.createElement("template"),
  rS = {
    insert: (t, e, r) => {
      e.insertBefore(t, r || null);
    },
    remove: (t) => {
      const e = t.parentNode;
      e && e.removeChild(t);
    },
    createElement: (t, e, r, o) => {
      const s = e ? Xi.createElementNS(nS, t) : Xi.createElement(t, r ? { is: r } : void 0);
      return t === "select" && o && o.multiple != null && s.setAttribute("multiple", o.multiple), s;
    },
    createText: (t) => Xi.createTextNode(t),
    createComment: (t) => Xi.createComment(t),
    setText: (t, e) => {
      t.nodeValue = e;
    },
    setElementText: (t, e) => {
      t.textContent = e;
    },
    parentNode: (t) => t.parentNode,
    nextSibling: (t) => t.nextSibling,
    querySelector: (t) => Xi.querySelector(t),
    setScopeId(t, e) {
      t.setAttribute(e, "");
    },
    insertStaticContent(t, e, r, o, s, u) {
      const f = r ? r.previousSibling : e.lastChild;
      if (s && (s === u || s.nextSibling))
        for (; e.insertBefore(s.cloneNode(!0), r), !(s === u || !(s = s.nextSibling)); );
      else {
        Og.innerHTML = o ? `<svg>${t}</svg>` : t;
        const h = Og.content;
        if (o) {
          const d = h.firstChild;
          for (; d.firstChild; ) h.appendChild(d.firstChild);
          h.removeChild(d);
        }
        e.insertBefore(h, r);
      }
      return [f ? f.nextSibling : e.firstChild, r ? r.previousSibling : e.lastChild];
    },
  },
  fi = "transition",
  Zs = "animation",
  wl = Symbol("_vtc"),
  Oh = (t, { slots: e }) => Ol(h_, iS(t), e);
Oh.displayName = "Transition";
const C0 = {
  name: String,
  type: String,
  css: { type: Boolean, default: !0 },
  duration: [String, Number, Object],
  enterFromClass: String,
  enterActiveClass: String,
  enterToClass: String,
  appearFromClass: String,
  appearActiveClass: String,
  appearToClass: String,
  leaveFromClass: String,
  leaveActiveClass: String,
  leaveToClass: String,
};
Oh.props = ze({}, i0, C0);
const ji = (t, e = []) => {
    It(t) ? t.forEach((r) => r(...e)) : t && t(...e);
  },
  Dg = (t) => (t ? (It(t) ? t.some((e) => e.length > 1) : t.length > 1) : !1);
function iS(t) {
  const e = {};
  for (const gt in t) gt in C0 || (e[gt] = t[gt]);
  if (t.css === !1) return e;
  const {
      name: r = "v",
      type: o,
      duration: s,
      enterFromClass: u = `${r}-enter-from`,
      enterActiveClass: f = `${r}-enter-active`,
      enterToClass: h = `${r}-enter-to`,
      appearFromClass: d = u,
      appearActiveClass: g = f,
      appearToClass: v = h,
      leaveFromClass: b = `${r}-leave-from`,
      leaveActiveClass: w = `${r}-leave-active`,
      leaveToClass: S = `${r}-leave-to`,
    } = t,
    P = oS(s),
    A = P && P[0],
    L = P && P[1],
    {
      onBeforeEnter: T,
      onEnter: M,
      onEnterCancelled: R,
      onLeave: E,
      onLeaveCancelled: B,
      onBeforeAppear: K = T,
      onAppear: ht = M,
      onAppearCancelled: Y = R,
    } = e,
    nt = (gt, G, z) => {
      Vi(gt, G ? v : h), Vi(gt, G ? g : f), z && z();
    },
    at = (gt, G) => {
      (gt._isLeaving = !1), Vi(gt, b), Vi(gt, S), Vi(gt, w), G && G();
    },
    pt = (gt) => (G, z) => {
      const k = gt ? ht : M,
        F = () => nt(G, gt, z);
      ji(k, [G, F]),
        $g(() => {
          Vi(G, gt ? d : u), hi(G, gt ? v : h), Dg(k) || Rg(G, o, A, F);
        });
    };
  return ze(e, {
    onBeforeEnter(gt) {
      ji(T, [gt]), hi(gt, u), hi(gt, f);
    },
    onBeforeAppear(gt) {
      ji(K, [gt]), hi(gt, d), hi(gt, g);
    },
    onEnter: pt(!1),
    onAppear: pt(!0),
    onLeave(gt, G) {
      gt._isLeaving = !0;
      const z = () => at(gt, G);
      hi(gt, b),
        aS(),
        hi(gt, w),
        $g(() => {
          gt._isLeaving && (Vi(gt, b), hi(gt, S), Dg(E) || Rg(gt, o, L, z));
        }),
        ji(E, [gt, z]);
    },
    onEnterCancelled(gt) {
      nt(gt, !1), ji(R, [gt]);
    },
    onAppearCancelled(gt) {
      nt(gt, !0), ji(Y, [gt]);
    },
    onLeaveCancelled(gt) {
      at(gt), ji(B, [gt]);
    },
  });
}
function oS(t) {
  if (t == null) return null;
  if (ye(t)) return [Yu(t.enter), Yu(t.leave)];
  {
    const e = Yu(t);
    return [e, e];
  }
}
function Yu(t) {
  return Am(t);
}
function hi(t, e) {
  e.split(/\s+/).forEach((r) => r && t.classList.add(r)), (t[wl] || (t[wl] = new Set())).add(e);
}
function Vi(t, e) {
  e.split(/\s+/).forEach((o) => o && t.classList.remove(o));
  const r = t[wl];
  r && (r.delete(e), r.size || (t[wl] = void 0));
}
function $g(t) {
  requestAnimationFrame(() => {
    requestAnimationFrame(t);
  });
}
let sS = 0;
function Rg(t, e, r, o) {
  const s = (t._endId = ++sS),
    u = () => {
      s === t._endId && o();
    };
  if (r) return setTimeout(u, r);
  const { type: f, timeout: h, propCount: d } = lS(t, e);
  if (!f) return o();
  const g = f + "end";
  let v = 0;
  const b = () => {
      t.removeEventListener(g, w), u();
    },
    w = (S) => {
      S.target === t && ++v >= d && b();
    };
  setTimeout(() => {
    v < d && b();
  }, h + 1),
    t.addEventListener(g, w);
}
function lS(t, e) {
  const r = window.getComputedStyle(t),
    o = (P) => (r[P] || "").split(", "),
    s = o(`${fi}Delay`),
    u = o(`${fi}Duration`),
    f = zg(s, u),
    h = o(`${Zs}Delay`),
    d = o(`${Zs}Duration`),
    g = zg(h, d);
  let v = null,
    b = 0,
    w = 0;
  e === fi
    ? f > 0 && ((v = fi), (b = f), (w = u.length))
    : e === Zs
    ? g > 0 && ((v = Zs), (b = g), (w = d.length))
    : ((b = Math.max(f, g)),
      (v = b > 0 ? (f > g ? fi : Zs) : null),
      (w = v ? (v === fi ? u.length : d.length) : 0));
  const S = v === fi && /\b(transform|all)(,|$)/.test(o(`${fi}Property`).toString());
  return { type: v, timeout: b, propCount: w, hasTransform: S };
}
function zg(t, e) {
  for (; t.length < e.length; ) t = t.concat(t);
  return Math.max(...e.map((r, o) => Ig(r) + Ig(t[o])));
}
function Ig(t) {
  return t === "auto" ? 0 : Number(t.slice(0, -1).replace(",", ".")) * 1e3;
}
function aS() {
  return document.body.offsetHeight;
}
function cS(t, e, r) {
  const o = t[wl];
  o && (e = (e ? [e, ...o] : [...o]).join(" ")),
    e == null ? t.removeAttribute("class") : r ? t.setAttribute("class", e) : (t.className = e);
}
const Dh = Symbol("_vod"),
  Pf = {
    beforeMount(t, { value: e }, { transition: r }) {
      (t[Dh] = t.style.display === "none" ? "" : t.style.display),
        r && e ? r.beforeEnter(t) : Js(t, e);
    },
    mounted(t, { value: e }, { transition: r }) {
      r && e && r.enter(t);
    },
    updated(t, { value: e, oldValue: r }, { transition: o }) {
      !e != !r &&
        (o
          ? e
            ? (o.beforeEnter(t), Js(t, !0), o.enter(t))
            : o.leave(t, () => {
                Js(t, !1);
              })
          : Js(t, e));
    },
    beforeUnmount(t, { value: e }) {
      Js(t, e);
    },
  };
function Js(t, e) {
  t.style.display = e ? t[Dh] : "none";
}
function uS(t, e, r) {
  const o = t.style,
    s = Ie(r);
  if (r && !s) {
    if (e && !Ie(e)) for (const u in e) r[u] == null && Of(o, u, "");
    for (const u in r) Of(o, u, r[u]);
  } else {
    const u = o.display;
    s ? e !== r && (o.cssText = r) : e && t.removeAttribute("style"), Dh in t && (o.display = u);
  }
}
const Fg = /\s*!important$/;
function Of(t, e, r) {
  if (It(r)) r.forEach((o) => Of(t, e, o));
  else if ((r == null && (r = ""), e.startsWith("--"))) t.setProperty(e, r);
  else {
    const o = fS(t, e);
    Fg.test(r) ? t.setProperty(co(o), r.replace(Fg, ""), "important") : (t[o] = r);
  }
}
const qg = ["Webkit", "Moz", "ms"],
  Zu = {};
function fS(t, e) {
  const r = Zu[e];
  if (r) return r;
  let o = _r(e);
  if (o !== "filter" && o in t) return (Zu[e] = o);
  o = Tc(o);
  for (let s = 0; s < qg.length; s++) {
    const u = qg[s] + o;
    if (u in t) return (Zu[e] = u);
  }
  return e;
}
const Hg = "http://www.w3.org/1999/xlink";
function hS(t, e, r, o, s) {
  if (o && e.startsWith("xlink:"))
    r == null ? t.removeAttributeNS(Hg, e.slice(6, e.length)) : t.setAttributeNS(Hg, e, r);
  else {
    const u = a1(e);
    r == null || (u && !Mm(r)) ? t.removeAttribute(e) : t.setAttribute(e, u ? "" : r);
  }
}
function dS(t, e, r, o, s, u, f) {
  if (e === "innerHTML" || e === "textContent") {
    o && f(o, s, u), (t[e] = r ?? "");
    return;
  }
  const h = t.tagName;
  if (e === "value" && h !== "PROGRESS" && !h.includes("-")) {
    t._value = r;
    const g = h === "OPTION" ? t.getAttribute("value") : t.value,
      v = r ?? "";
    g !== v && (t.value = v), r == null && t.removeAttribute(e);
    return;
  }
  let d = !1;
  if (r === "" || r == null) {
    const g = typeof t[e];
    g === "boolean"
      ? (r = Mm(r))
      : r == null && g === "string"
      ? ((r = ""), (d = !0))
      : g === "number" && ((r = 0), (d = !0));
  }
  try {
    t[e] = r;
  } catch {}
  d && t.removeAttribute(e);
}
function Bo(t, e, r, o) {
  t.addEventListener(e, r, o);
}
function pS(t, e, r, o) {
  t.removeEventListener(e, r, o);
}
const Bg = Symbol("_vei");
function gS(t, e, r, o, s = null) {
  const u = t[Bg] || (t[Bg] = {}),
    f = u[e];
  if (o && f) f.value = o;
  else {
    const [h, d] = vS(e);
    if (o) {
      const g = (u[e] = bS(o, s));
      Bo(t, h, g, d);
    } else f && (pS(t, h, f, d), (u[e] = void 0));
  }
}
const Wg = /(?:Once|Passive|Capture)$/;
function vS(t) {
  let e;
  if (Wg.test(t)) {
    e = {};
    let o;
    for (; (o = t.match(Wg)); )
      (t = t.slice(0, t.length - o[0].length)), (e[o[0].toLowerCase()] = !0);
  }
  return [t[2] === ":" ? t.slice(3) : co(t.slice(2)), e];
}
let Ju = 0;
const mS = Promise.resolve(),
  yS = () => Ju || (mS.then(() => (Ju = 0)), (Ju = Date.now()));
function bS(t, e) {
  const r = (o) => {
    if (!o._vts) o._vts = Date.now();
    else if (o._vts <= r.attached) return;
    jn(wS(o, r.value), e, 5, [o]);
  };
  return (r.value = t), (r.attached = yS()), r;
}
function wS(t, e) {
  if (It(e)) {
    const r = t.stopImmediatePropagation;
    return (
      (t.stopImmediatePropagation = () => {
        r.call(t), (t._stopped = !0);
      }),
      e.map((o) => (s) => !s._stopped && o && o(s))
    );
  } else return e;
}
const Ug = /^on[a-z]/,
  xS = (t, e, r, o, s = !1, u, f, h, d) => {
    e === "class"
      ? cS(t, o, s)
      : e === "style"
      ? uS(t, r, o)
      : _c(e)
      ? ch(e) || gS(t, e, r, o, f)
      : (
          e[0] === "."
            ? ((e = e.slice(1)), !0)
            : e[0] === "^"
            ? ((e = e.slice(1)), !1)
            : _S(t, e, o, s)
        )
      ? dS(t, e, o, u, f, h, d)
      : (e === "true-value" ? (t._trueValue = o) : e === "false-value" && (t._falseValue = o),
        hS(t, e, o, s));
  };
function _S(t, e, r, o) {
  return o
    ? !!(e === "innerHTML" || e === "textContent" || (e in t && Ug.test(e) && jt(r)))
    : e === "spellcheck" ||
      e === "draggable" ||
      e === "translate" ||
      e === "form" ||
      (e === "list" && t.tagName === "INPUT") ||
      (e === "type" && t.tagName === "TEXTAREA") ||
      (Ug.test(e) && Ie(r))
    ? !1
    : e in t;
}
const jg = (t) => {
  const e = t.props["onUpdate:modelValue"] || !1;
  return It(e) ? (r) => Fa(e, r) : e;
};
function SS(t) {
  t.target.composing = !0;
}
function Vg(t) {
  const e = t.target;
  e.composing && ((e.composing = !1), e.dispatchEvent(new Event("input")));
}
const Qu = Symbol("_assign"),
  kS = {
    created(t, { modifiers: { lazy: e, trim: r, number: o } }, s) {
      t[Qu] = jg(s);
      const u = o || (s.props && s.props.type === "number");
      Bo(t, e ? "change" : "input", (f) => {
        if (f.target.composing) return;
        let h = t.value;
        r && (h = h.trim()), u && (h = vf(h)), t[Qu](h);
      }),
        r &&
          Bo(t, "change", () => {
            t.value = t.value.trim();
          }),
        e || (Bo(t, "compositionstart", SS), Bo(t, "compositionend", Vg), Bo(t, "change", Vg));
    },
    mounted(t, { value: e }) {
      t.value = e ?? "";
    },
    beforeUpdate(t, { value: e, modifiers: { lazy: r, trim: o, number: s } }, u) {
      if (
        ((t[Qu] = jg(u)),
        t.composing ||
          (document.activeElement === t &&
            t.type !== "range" &&
            (r ||
              (o && t.value.trim() === e) ||
              ((s || t.type === "number") && vf(t.value) === e))))
      )
        return;
      const f = e ?? "";
      t.value !== f && (t.value = f);
    },
  },
  CS = {
    esc: "escape",
    space: " ",
    up: "arrow-up",
    left: "arrow-left",
    right: "arrow-right",
    down: "arrow-down",
    delete: "backspace",
  },
  Df = (t, e) => (r) => {
    if (!("key" in r)) return;
    const o = co(r.key);
    if (e.some((s) => s === o || CS[s] === o)) return t(r);
  },
  TS = ze({ patchProp: xS }, rS);
let Gg;
function ES() {
  return Gg || (Gg = I_(TS));
}
const T0 = (...t) => {
  const e = ES().createApp(...t),
    { mount: r } = e;
  return (
    (e.mount = (o) => {
      const s = LS(o);
      if (!s) return;
      const u = e._component;
      !jt(u) && !u.render && !u.template && (u.template = s.innerHTML), (s.innerHTML = "");
      const f = r(s, !1, s instanceof SVGElement);
      return (
        s instanceof Element && (s.removeAttribute("v-cloak"), s.setAttribute("data-v-app", "")), f
      );
    }),
    e
  );
};
function LS(t) {
  return Ie(t) ? document.querySelector(t) : t;
}
const fo = (t, e) => {
    const r = t.__vccOpts || t;
    for (const [o, s] of e) r[o] = s;
    return r;
  },
  AS = {};
function MS(t, e) {
  const r = io("RouterView");
  return st(), te(r);
}
const NS = fo(AS, [["render", MS]]); /*!
 * vue-router v4.2.5
 * (c) 2023 Eduardo San Martin Morote
 * @license MIT
 */
const Wo = typeof window < "u";
function PS(t) {
  return t.__esModule || t[Symbol.toStringTag] === "Module";
}
const pe = Object.assign;
function tf(t, e) {
  const r = {};
  for (const o in e) {
    const s = e[o];
    r[o] = lr(s) ? s.map(t) : t(s);
  }
  return r;
}
const ul = () => {},
  lr = Array.isArray,
  OS = /\/$/,
  DS = (t) => t.replace(OS, "");
function ef(t, e, r = "/") {
  let o,
    s = {},
    u = "",
    f = "";
  const h = e.indexOf("#");
  let d = e.indexOf("?");
  return (
    h < d && h >= 0 && (d = -1),
    d > -1 && ((o = e.slice(0, d)), (u = e.slice(d + 1, h > -1 ? h : e.length)), (s = t(u))),
    h > -1 && ((o = o || e.slice(0, h)), (f = e.slice(h, e.length))),
    (o = IS(o ?? e, r)),
    { fullPath: o + (u && "?") + u + f, path: o, query: s, hash: f }
  );
}
function $S(t, e) {
  const r = e.query ? t(e.query) : "";
  return e.path + (r && "?") + r + (e.hash || "");
}
function Kg(t, e) {
  return !e || !t.toLowerCase().startsWith(e.toLowerCase()) ? t : t.slice(e.length) || "/";
}
function RS(t, e, r) {
  const o = e.matched.length - 1,
    s = r.matched.length - 1;
  return (
    o > -1 &&
    o === s &&
    ss(e.matched[o], r.matched[s]) &&
    E0(e.params, r.params) &&
    t(e.query) === t(r.query) &&
    e.hash === r.hash
  );
}
function ss(t, e) {
  return (t.aliasOf || t) === (e.aliasOf || e);
}
function E0(t, e) {
  if (Object.keys(t).length !== Object.keys(e).length) return !1;
  for (const r in t) if (!zS(t[r], e[r])) return !1;
  return !0;
}
function zS(t, e) {
  return lr(t) ? Xg(t, e) : lr(e) ? Xg(e, t) : t === e;
}
function Xg(t, e) {
  return lr(e)
    ? t.length === e.length && t.every((r, o) => r === e[o])
    : t.length === 1 && t[0] === e;
}
function IS(t, e) {
  if (t.startsWith("/")) return t;
  if (!t) return e;
  const r = e.split("/"),
    o = t.split("/"),
    s = o[o.length - 1];
  (s === ".." || s === ".") && o.push("");
  let u = r.length - 1,
    f,
    h;
  for (f = 0; f < o.length; f++)
    if (((h = o[f]), h !== "."))
      if (h === "..") u > 1 && u--;
      else break;
  return r.slice(0, u).join("/") + "/" + o.slice(f - (f === o.length ? 1 : 0)).join("/");
}
var xl;
(function (t) {
  (t.pop = "pop"), (t.push = "push");
})(xl || (xl = {}));
var fl;
(function (t) {
  (t.back = "back"), (t.forward = "forward"), (t.unknown = "");
})(fl || (fl = {}));
function FS(t) {
  if (!t)
    if (Wo) {
      const e = document.querySelector("base");
      (t = (e && e.getAttribute("href")) || "/"), (t = t.replace(/^\w+:\/\/[^\/]+/, ""));
    } else t = "/";
  return t[0] !== "/" && t[0] !== "#" && (t = "/" + t), DS(t);
}
const qS = /^[^#]+#/;
function HS(t, e) {
  return t.replace(qS, "#") + e;
}
function BS(t, e) {
  const r = document.documentElement.getBoundingClientRect(),
    o = t.getBoundingClientRect();
  return {
    behavior: e.behavior,
    left: o.left - r.left - (e.left || 0),
    top: o.top - r.top - (e.top || 0),
  };
}
const Rc = () => ({ left: window.pageXOffset, top: window.pageYOffset });
function WS(t) {
  let e;
  if ("el" in t) {
    const r = t.el,
      o = typeof r == "string" && r.startsWith("#"),
      s =
        typeof r == "string"
          ? o
            ? document.getElementById(r.slice(1))
            : document.querySelector(r)
          : r;
    if (!s) return;
    e = BS(s, t);
  } else e = t;
  "scrollBehavior" in document.documentElement.style
    ? window.scrollTo(e)
    : window.scrollTo(
        e.left != null ? e.left : window.pageXOffset,
        e.top != null ? e.top : window.pageYOffset,
      );
}
function Yg(t, e) {
  return (history.state ? history.state.position - e : -1) + t;
}
const $f = new Map();
function US(t, e) {
  $f.set(t, e);
}
function jS(t) {
  const e = $f.get(t);
  return $f.delete(t), e;
}
let VS = () => location.protocol + "//" + location.host;
function L0(t, e) {
  const { pathname: r, search: o, hash: s } = e,
    u = t.indexOf("#");
  if (u > -1) {
    let h = s.includes(t.slice(u)) ? t.slice(u).length : 1,
      d = s.slice(h);
    return d[0] !== "/" && (d = "/" + d), Kg(d, "");
  }
  return Kg(r, t) + o + s;
}
function GS(t, e, r, o) {
  let s = [],
    u = [],
    f = null;
  const h = ({ state: w }) => {
    const S = L0(t, location),
      P = r.value,
      A = e.value;
    let L = 0;
    if (w) {
      if (((r.value = S), (e.value = w), f && f === P)) {
        f = null;
        return;
      }
      L = A ? w.position - A.position : 0;
    } else o(S);
    s.forEach((T) => {
      T(r.value, P, {
        delta: L,
        type: xl.pop,
        direction: L ? (L > 0 ? fl.forward : fl.back) : fl.unknown,
      });
    });
  };
  function d() {
    f = r.value;
  }
  function g(w) {
    s.push(w);
    const S = () => {
      const P = s.indexOf(w);
      P > -1 && s.splice(P, 1);
    };
    return u.push(S), S;
  }
  function v() {
    const { history: w } = window;
    w.state && w.replaceState(pe({}, w.state, { scroll: Rc() }), "");
  }
  function b() {
    for (const w of u) w();
    (u = []),
      window.removeEventListener("popstate", h),
      window.removeEventListener("beforeunload", v);
  }
  return (
    window.addEventListener("popstate", h),
    window.addEventListener("beforeunload", v, { passive: !0 }),
    { pauseListeners: d, listen: g, destroy: b }
  );
}
function Zg(t, e, r, o = !1, s = !1) {
  return {
    back: t,
    current: e,
    forward: r,
    replaced: o,
    position: window.history.length,
    scroll: s ? Rc() : null,
  };
}
function KS(t) {
  const { history: e, location: r } = window,
    o = { value: L0(t, r) },
    s = { value: e.state };
  s.value ||
    u(
      o.value,
      {
        back: null,
        current: o.value,
        forward: null,
        position: e.length - 1,
        replaced: !0,
        scroll: null,
      },
      !0,
    );
  function u(d, g, v) {
    const b = t.indexOf("#"),
      w = b > -1 ? (r.host && document.querySelector("base") ? t : t.slice(b)) + d : VS() + t + d;
    try {
      e[v ? "replaceState" : "pushState"](g, "", w), (s.value = g);
    } catch (S) {
      console.error(S), r[v ? "replace" : "assign"](w);
    }
  }
  function f(d, g) {
    const v = pe({}, e.state, Zg(s.value.back, d, s.value.forward, !0), g, {
      position: s.value.position,
    });
    u(d, v, !0), (o.value = d);
  }
  function h(d, g) {
    const v = pe({}, s.value, e.state, { forward: d, scroll: Rc() });
    u(v.current, v, !0);
    const b = pe({}, Zg(o.value, d, null), { position: v.position + 1 }, g);
    u(d, b, !1), (o.value = d);
  }
  return { location: o, state: s, push: h, replace: f };
}
function XS(t) {
  t = FS(t);
  const e = KS(t),
    r = GS(t, e.state, e.location, e.replace);
  function o(u, f = !0) {
    f || r.pauseListeners(), history.go(u);
  }
  const s = pe({ location: "", base: t, go: o, createHref: HS.bind(null, t) }, e, r);
  return (
    Object.defineProperty(s, "location", { enumerable: !0, get: () => e.location.value }),
    Object.defineProperty(s, "state", { enumerable: !0, get: () => e.state.value }),
    s
  );
}
function YS(t) {
  return (
    (t = location.host ? t || location.pathname + location.search : ""),
    t.includes("#") || (t += "#"),
    XS(t)
  );
}
function ZS(t) {
  return typeof t == "string" || (t && typeof t == "object");
}
function A0(t) {
  return typeof t == "string" || typeof t == "symbol";
}
const di = {
    path: "/",
    name: void 0,
    params: {},
    query: {},
    hash: "",
    fullPath: "/",
    matched: [],
    meta: {},
    redirectedFrom: void 0,
  },
  M0 = Symbol("");
var Jg;
(function (t) {
  (t[(t.aborted = 4)] = "aborted"),
    (t[(t.cancelled = 8)] = "cancelled"),
    (t[(t.duplicated = 16)] = "duplicated");
})(Jg || (Jg = {}));
function ls(t, e) {
  return pe(new Error(), { type: t, [M0]: !0 }, e);
}
function Or(t, e) {
  return t instanceof Error && M0 in t && (e == null || !!(t.type & e));
}
const Qg = "[^/]+?",
  JS = { sensitive: !1, strict: !1, start: !0, end: !0 },
  QS = /[.+*?^${}()[\]/\\]/g;
function tk(t, e) {
  const r = pe({}, JS, e),
    o = [];
  let s = r.start ? "^" : "";
  const u = [];
  for (const g of t) {
    const v = g.length ? [] : [90];
    r.strict && !g.length && (s += "/");
    for (let b = 0; b < g.length; b++) {
      const w = g[b];
      let S = 40 + (r.sensitive ? 0.25 : 0);
      if (w.type === 0) b || (s += "/"), (s += w.value.replace(QS, "\\$&")), (S += 40);
      else if (w.type === 1) {
        const { value: P, repeatable: A, optional: L, regexp: T } = w;
        u.push({ name: P, repeatable: A, optional: L });
        const M = T || Qg;
        if (M !== Qg) {
          S += 10;
          try {
            new RegExp(`(${M})`);
          } catch (E) {
            throw new Error(`Invalid custom RegExp for param "${P}" (${M}): ` + E.message);
          }
        }
        let R = A ? `((?:${M})(?:/(?:${M}))*)` : `(${M})`;
        b || (R = L && g.length < 2 ? `(?:/${R})` : "/" + R),
          L && (R += "?"),
          (s += R),
          (S += 20),
          L && (S += -8),
          A && (S += -20),
          M === ".*" && (S += -50);
      }
      v.push(S);
    }
    o.push(v);
  }
  if (r.strict && r.end) {
    const g = o.length - 1;
    o[g][o[g].length - 1] += 0.7000000000000001;
  }
  r.strict || (s += "/?"), r.end ? (s += "$") : r.strict && (s += "(?:/|$)");
  const f = new RegExp(s, r.sensitive ? "" : "i");
  function h(g) {
    const v = g.match(f),
      b = {};
    if (!v) return null;
    for (let w = 1; w < v.length; w++) {
      const S = v[w] || "",
        P = u[w - 1];
      b[P.name] = S && P.repeatable ? S.split("/") : S;
    }
    return b;
  }
  function d(g) {
    let v = "",
      b = !1;
    for (const w of t) {
      (!b || !v.endsWith("/")) && (v += "/"), (b = !1);
      for (const S of w)
        if (S.type === 0) v += S.value;
        else if (S.type === 1) {
          const { value: P, repeatable: A, optional: L } = S,
            T = P in g ? g[P] : "";
          if (lr(T) && !A)
            throw new Error(
              `Provided param "${P}" is an array but it is not repeatable (* or + modifiers)`,
            );
          const M = lr(T) ? T.join("/") : T;
          if (!M)
            if (L) w.length < 2 && (v.endsWith("/") ? (v = v.slice(0, -1)) : (b = !0));
            else throw new Error(`Missing required param "${P}"`);
          v += M;
        }
    }
    return v || "/";
  }
  return { re: f, score: o, keys: u, parse: h, stringify: d };
}
function ek(t, e) {
  let r = 0;
  for (; r < t.length && r < e.length; ) {
    const o = e[r] - t[r];
    if (o) return o;
    r++;
  }
  return t.length < e.length
    ? t.length === 1 && t[0] === 40 + 40
      ? -1
      : 1
    : t.length > e.length
    ? e.length === 1 && e[0] === 40 + 40
      ? 1
      : -1
    : 0;
}
function nk(t, e) {
  let r = 0;
  const o = t.score,
    s = e.score;
  for (; r < o.length && r < s.length; ) {
    const u = ek(o[r], s[r]);
    if (u) return u;
    r++;
  }
  if (Math.abs(s.length - o.length) === 1) {
    if (tv(o)) return 1;
    if (tv(s)) return -1;
  }
  return s.length - o.length;
}
function tv(t) {
  const e = t[t.length - 1];
  return t.length > 0 && e[e.length - 1] < 0;
}
const rk = { type: 0, value: "" },
  ik = /[a-zA-Z0-9_]/;
function ok(t) {
  if (!t) return [[]];
  if (t === "/") return [[rk]];
  if (!t.startsWith("/")) throw new Error(`Invalid path "${t}"`);
  function e(S) {
    throw new Error(`ERR (${r})/"${g}": ${S}`);
  }
  let r = 0,
    o = r;
  const s = [];
  let u;
  function f() {
    u && s.push(u), (u = []);
  }
  let h = 0,
    d,
    g = "",
    v = "";
  function b() {
    g &&
      (r === 0
        ? u.push({ type: 0, value: g })
        : r === 1 || r === 2 || r === 3
        ? (u.length > 1 &&
            (d === "*" || d === "+") &&
            e(`A repeatable param (${g}) must be alone in its segment. eg: '/:ids+.`),
          u.push({
            type: 1,
            value: g,
            regexp: v,
            repeatable: d === "*" || d === "+",
            optional: d === "*" || d === "?",
          }))
        : e("Invalid state to consume buffer"),
      (g = ""));
  }
  function w() {
    g += d;
  }
  for (; h < t.length; ) {
    if (((d = t[h++]), d === "\\" && r !== 2)) {
      (o = r), (r = 4);
      continue;
    }
    switch (r) {
      case 0:
        d === "/" ? (g && b(), f()) : d === ":" ? (b(), (r = 1)) : w();
        break;
      case 4:
        w(), (r = o);
        break;
      case 1:
        d === "("
          ? (r = 2)
          : ik.test(d)
          ? w()
          : (b(), (r = 0), d !== "*" && d !== "?" && d !== "+" && h--);
        break;
      case 2:
        d === ")" ? (v[v.length - 1] == "\\" ? (v = v.slice(0, -1) + d) : (r = 3)) : (v += d);
        break;
      case 3:
        b(), (r = 0), d !== "*" && d !== "?" && d !== "+" && h--, (v = "");
        break;
      default:
        e("Unknown state");
        break;
    }
  }
  return r === 2 && e(`Unfinished custom RegExp for param "${g}"`), b(), f(), s;
}
function sk(t, e, r) {
  const o = tk(ok(t.path), r),
    s = pe(o, { record: t, parent: e, children: [], alias: [] });
  return e && !s.record.aliasOf == !e.record.aliasOf && e.children.push(s), s;
}
function lk(t, e) {
  const r = [],
    o = new Map();
  e = rv({ strict: !1, end: !0, sensitive: !1 }, e);
  function s(v) {
    return o.get(v);
  }
  function u(v, b, w) {
    const S = !w,
      P = ak(v);
    P.aliasOf = w && w.record;
    const A = rv(e, v),
      L = [P];
    if ("alias" in v) {
      const R = typeof v.alias == "string" ? [v.alias] : v.alias;
      for (const E of R)
        L.push(
          pe({}, P, {
            components: w ? w.record.components : P.components,
            path: E,
            aliasOf: w ? w.record : P,
          }),
        );
    }
    let T, M;
    for (const R of L) {
      const { path: E } = R;
      if (b && E[0] !== "/") {
        const B = b.record.path,
          K = B[B.length - 1] === "/" ? "" : "/";
        R.path = b.record.path + (E && K + E);
      }
      if (
        ((T = sk(R, b, A)),
        w
          ? w.alias.push(T)
          : ((M = M || T), M !== T && M.alias.push(T), S && v.name && !nv(T) && f(v.name)),
        P.children)
      ) {
        const B = P.children;
        for (let K = 0; K < B.length; K++) u(B[K], T, w && w.children[K]);
      }
      (w = w || T),
        ((T.record.components && Object.keys(T.record.components).length) ||
          T.record.name ||
          T.record.redirect) &&
          d(T);
    }
    return M
      ? () => {
          f(M);
        }
      : ul;
  }
  function f(v) {
    if (A0(v)) {
      const b = o.get(v);
      b && (o.delete(v), r.splice(r.indexOf(b), 1), b.children.forEach(f), b.alias.forEach(f));
    } else {
      const b = r.indexOf(v);
      b > -1 &&
        (r.splice(b, 1),
        v.record.name && o.delete(v.record.name),
        v.children.forEach(f),
        v.alias.forEach(f));
    }
  }
  function h() {
    return r;
  }
  function d(v) {
    let b = 0;
    for (
      ;
      b < r.length && nk(v, r[b]) >= 0 && (v.record.path !== r[b].record.path || !N0(v, r[b]));
    )
      b++;
    r.splice(b, 0, v), v.record.name && !nv(v) && o.set(v.record.name, v);
  }
  function g(v, b) {
    let w,
      S = {},
      P,
      A;
    if ("name" in v && v.name) {
      if (((w = o.get(v.name)), !w)) throw ls(1, { location: v });
      (A = w.record.name),
        (S = pe(
          ev(
            b.params,
            w.keys.filter((M) => !M.optional).map((M) => M.name),
          ),
          v.params &&
            ev(
              v.params,
              w.keys.map((M) => M.name),
            ),
        )),
        (P = w.stringify(S));
    } else if ("path" in v)
      (P = v.path), (w = r.find((M) => M.re.test(P))), w && ((S = w.parse(P)), (A = w.record.name));
    else {
      if (((w = b.name ? o.get(b.name) : r.find((M) => M.re.test(b.path))), !w))
        throw ls(1, { location: v, currentLocation: b });
      (A = w.record.name), (S = pe({}, b.params, v.params)), (P = w.stringify(S));
    }
    const L = [];
    let T = w;
    for (; T; ) L.unshift(T.record), (T = T.parent);
    return { name: A, path: P, params: S, matched: L, meta: uk(L) };
  }
  return (
    t.forEach((v) => u(v)),
    { addRoute: u, resolve: g, removeRoute: f, getRoutes: h, getRecordMatcher: s }
  );
}
function ev(t, e) {
  const r = {};
  for (const o of e) o in t && (r[o] = t[o]);
  return r;
}
function ak(t) {
  return {
    path: t.path,
    redirect: t.redirect,
    name: t.name,
    meta: t.meta || {},
    aliasOf: void 0,
    beforeEnter: t.beforeEnter,
    props: ck(t),
    children: t.children || [],
    instances: {},
    leaveGuards: new Set(),
    updateGuards: new Set(),
    enterCallbacks: {},
    components: "components" in t ? t.components || null : t.component && { default: t.component },
  };
}
function ck(t) {
  const e = {},
    r = t.props || !1;
  if ("component" in t) e.default = r;
  else for (const o in t.components) e[o] = typeof r == "object" ? r[o] : r;
  return e;
}
function nv(t) {
  for (; t; ) {
    if (t.record.aliasOf) return !0;
    t = t.parent;
  }
  return !1;
}
function uk(t) {
  return t.reduce((e, r) => pe(e, r.meta), {});
}
function rv(t, e) {
  const r = {};
  for (const o in t) r[o] = o in e ? e[o] : t[o];
  return r;
}
function N0(t, e) {
  return e.children.some((r) => r === t || N0(t, r));
}
const P0 = /#/g,
  fk = /&/g,
  hk = /\//g,
  dk = /=/g,
  pk = /\?/g,
  O0 = /\+/g,
  gk = /%5B/g,
  vk = /%5D/g,
  D0 = /%5E/g,
  mk = /%60/g,
  $0 = /%7B/g,
  yk = /%7C/g,
  R0 = /%7D/g,
  bk = /%20/g;
function $h(t) {
  return encodeURI("" + t)
    .replace(yk, "|")
    .replace(gk, "[")
    .replace(vk, "]");
}
function wk(t) {
  return $h(t).replace($0, "{").replace(R0, "}").replace(D0, "^");
}
function Rf(t) {
  return $h(t)
    .replace(O0, "%2B")
    .replace(bk, "+")
    .replace(P0, "%23")
    .replace(fk, "%26")
    .replace(mk, "`")
    .replace($0, "{")
    .replace(R0, "}")
    .replace(D0, "^");
}
function xk(t) {
  return Rf(t).replace(dk, "%3D");
}
function _k(t) {
  return $h(t).replace(P0, "%23").replace(pk, "%3F");
}
function Sk(t) {
  return t == null ? "" : _k(t).replace(hk, "%2F");
}
function ic(t) {
  try {
    return decodeURIComponent("" + t);
  } catch {}
  return "" + t;
}
function kk(t) {
  const e = {};
  if (t === "" || t === "?") return e;
  const o = (t[0] === "?" ? t.slice(1) : t).split("&");
  for (let s = 0; s < o.length; ++s) {
    const u = o[s].replace(O0, " "),
      f = u.indexOf("="),
      h = ic(f < 0 ? u : u.slice(0, f)),
      d = f < 0 ? null : ic(u.slice(f + 1));
    if (h in e) {
      let g = e[h];
      lr(g) || (g = e[h] = [g]), g.push(d);
    } else e[h] = d;
  }
  return e;
}
function iv(t) {
  let e = "";
  for (let r in t) {
    const o = t[r];
    if (((r = xk(r)), o == null)) {
      o !== void 0 && (e += (e.length ? "&" : "") + r);
      continue;
    }
    (lr(o) ? o.map((u) => u && Rf(u)) : [o && Rf(o)]).forEach((u) => {
      u !== void 0 && ((e += (e.length ? "&" : "") + r), u != null && (e += "=" + u));
    });
  }
  return e;
}
function Ck(t) {
  const e = {};
  for (const r in t) {
    const o = t[r];
    o !== void 0 &&
      (e[r] = lr(o) ? o.map((s) => (s == null ? null : "" + s)) : o == null ? o : "" + o);
  }
  return e;
}
const Tk = Symbol(""),
  ov = Symbol(""),
  Rh = Symbol(""),
  z0 = Symbol(""),
  zf = Symbol("");
function Qs() {
  let t = [];
  function e(o) {
    return (
      t.push(o),
      () => {
        const s = t.indexOf(o);
        s > -1 && t.splice(s, 1);
      }
    );
  }
  function r() {
    t = [];
  }
  return { add: e, list: () => t.slice(), reset: r };
}
function mi(t, e, r, o, s) {
  const u = o && (o.enterCallbacks[s] = o.enterCallbacks[s] || []);
  return () =>
    new Promise((f, h) => {
      const d = (b) => {
          b === !1
            ? h(ls(4, { from: r, to: e }))
            : b instanceof Error
            ? h(b)
            : ZS(b)
            ? h(ls(2, { from: e, to: b }))
            : (u && o.enterCallbacks[s] === u && typeof b == "function" && u.push(b), f());
        },
        g = t.call(o && o.instances[s], e, r, d);
      let v = Promise.resolve(g);
      t.length < 3 && (v = v.then(d)), v.catch((b) => h(b));
    });
}
function nf(t, e, r, o) {
  const s = [];
  for (const u of t)
    for (const f in u.components) {
      let h = u.components[f];
      if (!(e !== "beforeRouteEnter" && !u.instances[f]))
        if (Ek(h)) {
          const g = (h.__vccOpts || h)[e];
          g && s.push(mi(g, r, o, u, f));
        } else {
          let d = h();
          s.push(() =>
            d.then((g) => {
              if (!g)
                return Promise.reject(
                  new Error(`Couldn't resolve component "${f}" at "${u.path}"`),
                );
              const v = PS(g) ? g.default : g;
              u.components[f] = v;
              const w = (v.__vccOpts || v)[e];
              return w && mi(w, r, o, u, f)();
            }),
          );
        }
    }
  return s;
}
function Ek(t) {
  return typeof t == "object" || "displayName" in t || "props" in t || "__vccOpts" in t;
}
function sv(t) {
  const e = qr(Rh),
    r = qr(z0),
    o = xt(() => e.resolve(U(t.to))),
    s = xt(() => {
      const { matched: d } = o.value,
        { length: g } = d,
        v = d[g - 1],
        b = r.matched;
      if (!v || !b.length) return -1;
      const w = b.findIndex(ss.bind(null, v));
      if (w > -1) return w;
      const S = lv(d[g - 2]);
      return g > 1 && lv(v) === S && b[b.length - 1].path !== S
        ? b.findIndex(ss.bind(null, d[g - 2]))
        : w;
    }),
    u = xt(() => s.value > -1 && Nk(r.params, o.value.params)),
    f = xt(() => s.value > -1 && s.value === r.matched.length - 1 && E0(r.params, o.value.params));
  function h(d = {}) {
    return Mk(d) ? e[U(t.replace) ? "replace" : "push"](U(t.to)).catch(ul) : Promise.resolve();
  }
  return { route: o, href: xt(() => o.value.href), isActive: u, isExactActive: f, navigate: h };
}
const Lk = ie({
    name: "RouterLink",
    compatConfig: { MODE: 3 },
    props: {
      to: { type: [String, Object], required: !0 },
      replace: Boolean,
      activeClass: String,
      exactActiveClass: String,
      custom: Boolean,
      ariaCurrentValue: { type: String, default: "page" },
    },
    useLink: sv,
    setup(t, { slots: e }) {
      const r = Un(sv(t)),
        { options: o } = qr(Rh),
        s = xt(() => ({
          [av(t.activeClass, o.linkActiveClass, "router-link-active")]: r.isActive,
          [av(t.exactActiveClass, o.linkExactActiveClass, "router-link-exact-active")]:
            r.isExactActive,
        }));
      return () => {
        const u = e.default && e.default(r);
        return t.custom
          ? u
          : Ol(
              "a",
              {
                "aria-current": r.isExactActive ? t.ariaCurrentValue : null,
                href: r.href,
                onClick: r.navigate,
                class: s.value,
              },
              u,
            );
      };
    },
  }),
  Ak = Lk;
function Mk(t) {
  if (
    !(t.metaKey || t.altKey || t.ctrlKey || t.shiftKey) &&
    !t.defaultPrevented &&
    !(t.button !== void 0 && t.button !== 0)
  ) {
    if (t.currentTarget && t.currentTarget.getAttribute) {
      const e = t.currentTarget.getAttribute("target");
      if (/\b_blank\b/i.test(e)) return;
    }
    return t.preventDefault && t.preventDefault(), !0;
  }
}
function Nk(t, e) {
  for (const r in e) {
    const o = e[r],
      s = t[r];
    if (typeof o == "string") {
      if (o !== s) return !1;
    } else if (!lr(s) || s.length !== o.length || o.some((u, f) => u !== s[f])) return !1;
  }
  return !0;
}
function lv(t) {
  return t ? (t.aliasOf ? t.aliasOf.path : t.path) : "";
}
const av = (t, e, r) => t ?? e ?? r,
  Pk = ie({
    name: "RouterView",
    inheritAttrs: !1,
    props: { name: { type: String, default: "default" }, route: Object },
    compatConfig: { MODE: 3 },
    setup(t, { attrs: e, slots: r }) {
      const o = qr(zf),
        s = xt(() => t.route || o.value),
        u = qr(ov, 0),
        f = xt(() => {
          let g = U(u);
          const { matched: v } = s.value;
          let b;
          for (; (b = v[g]) && !b.components; ) g++;
          return g;
        }),
        h = xt(() => s.value.matched[f.value]);
      qa(
        ov,
        xt(() => f.value + 1),
      ),
        qa(Tk, h),
        qa(zf, s);
      const d = Zt();
      return (
        Re(
          () => [d.value, h.value, t.name],
          ([g, v, b], [w, S, P]) => {
            v &&
              ((v.instances[b] = g),
              S &&
                S !== v &&
                g &&
                g === w &&
                (v.leaveGuards.size || (v.leaveGuards = S.leaveGuards),
                v.updateGuards.size || (v.updateGuards = S.updateGuards))),
              g && v && (!S || !ss(v, S) || !w) && (v.enterCallbacks[b] || []).forEach((A) => A(g));
          },
          { flush: "post" },
        ),
        () => {
          const g = s.value,
            v = t.name,
            b = h.value,
            w = b && b.components[v];
          if (!w) return cv(r.default, { Component: w, route: g });
          const S = b.props[v],
            P = S ? (S === !0 ? g.params : typeof S == "function" ? S(g) : S) : null,
            L = Ol(
              w,
              pe({}, P, e, {
                onVnodeUnmounted: (T) => {
                  T.component.isUnmounted && (b.instances[v] = null);
                },
                ref: d,
              }),
            );
          return cv(r.default, { Component: L, route: g }) || L;
        }
      );
    },
  });
function cv(t, e) {
  if (!t) return null;
  const r = t(e);
  return r.length === 1 ? r[0] : r;
}
const Ok = Pk;
function Dk(t) {
  const e = lk(t.routes, t),
    r = t.parseQuery || kk,
    o = t.stringifyQuery || iv,
    s = t.history,
    u = Qs(),
    f = Qs(),
    h = Qs(),
    d = vs(di);
  let g = di;
  Wo &&
    t.scrollBehavior &&
    "scrollRestoration" in history &&
    (history.scrollRestoration = "manual");
  const v = tf.bind(null, (j) => "" + j),
    b = tf.bind(null, Sk),
    w = tf.bind(null, ic);
  function S(j, rt) {
    let lt, Mt;
    return A0(j) ? ((lt = e.getRecordMatcher(j)), (Mt = rt)) : (Mt = j), e.addRoute(Mt, lt);
  }
  function P(j) {
    const rt = e.getRecordMatcher(j);
    rt && e.removeRoute(rt);
  }
  function A() {
    return e.getRoutes().map((j) => j.record);
  }
  function L(j) {
    return !!e.getRecordMatcher(j);
  }
  function T(j, rt) {
    if (((rt = pe({}, rt || d.value)), typeof j == "string")) {
      const V = ef(r, j, rt.path),
        Q = e.resolve({ path: V.path }, rt),
        ot = s.createHref(V.fullPath);
      return pe(V, Q, { params: w(Q.params), hash: ic(V.hash), redirectedFrom: void 0, href: ot });
    }
    let lt;
    if ("path" in j) lt = pe({}, j, { path: ef(r, j.path, rt.path).path });
    else {
      const V = pe({}, j.params);
      for (const Q in V) V[Q] == null && delete V[Q];
      (lt = pe({}, j, { params: b(V) })), (rt.params = b(rt.params));
    }
    const Mt = e.resolve(lt, rt),
      Et = j.hash || "";
    Mt.params = v(w(Mt.params));
    const $ = $S(o, pe({}, j, { hash: wk(Et), path: Mt.path })),
      I = s.createHref($);
    return pe({ fullPath: $, hash: Et, query: o === iv ? Ck(j.query) : j.query || {} }, Mt, {
      redirectedFrom: void 0,
      href: I,
    });
  }
  function M(j) {
    return typeof j == "string" ? ef(r, j, d.value.path) : pe({}, j);
  }
  function R(j, rt) {
    if (g !== j) return ls(8, { from: rt, to: j });
  }
  function E(j) {
    return ht(j);
  }
  function B(j) {
    return E(pe(M(j), { replace: !0 }));
  }
  function K(j) {
    const rt = j.matched[j.matched.length - 1];
    if (rt && rt.redirect) {
      const { redirect: lt } = rt;
      let Mt = typeof lt == "function" ? lt(j) : lt;
      return (
        typeof Mt == "string" &&
          ((Mt = Mt.includes("?") || Mt.includes("#") ? (Mt = M(Mt)) : { path: Mt }),
          (Mt.params = {})),
        pe({ query: j.query, hash: j.hash, params: "path" in Mt ? {} : j.params }, Mt)
      );
    }
  }
  function ht(j, rt) {
    const lt = (g = T(j)),
      Mt = d.value,
      Et = j.state,
      $ = j.force,
      I = j.replace === !0,
      V = K(lt);
    if (V)
      return ht(
        pe(M(V), { state: typeof V == "object" ? pe({}, Et, V.state) : Et, force: $, replace: I }),
        rt || lt,
      );
    const Q = lt;
    Q.redirectedFrom = rt;
    let ot;
    return (
      !$ && RS(o, Mt, lt) && ((ot = ls(16, { to: Q, from: Mt })), qt(Mt, Mt, !0, !1)),
      (ot ? Promise.resolve(ot) : at(Q, Mt))
        .catch((ut) => (Or(ut) ? (Or(ut, 2) ? ut : At(ut)) : J(ut, Q, Mt)))
        .then((ut) => {
          if (ut) {
            if (Or(ut, 2))
              return ht(
                pe({ replace: I }, M(ut.to), {
                  state: typeof ut.to == "object" ? pe({}, Et, ut.to.state) : Et,
                  force: $,
                }),
                rt || Q,
              );
          } else ut = gt(Q, Mt, !0, I, Et);
          return pt(Q, Mt, ut), ut;
        })
    );
  }
  function Y(j, rt) {
    const lt = R(j, rt);
    return lt ? Promise.reject(lt) : Promise.resolve();
  }
  function nt(j) {
    const rt = Jt.values().next().value;
    return rt && typeof rt.runWithContext == "function" ? rt.runWithContext(j) : j();
  }
  function at(j, rt) {
    let lt;
    const [Mt, Et, $] = $k(j, rt);
    lt = nf(Mt.reverse(), "beforeRouteLeave", j, rt);
    for (const V of Mt)
      V.leaveGuards.forEach((Q) => {
        lt.push(mi(Q, j, rt));
      });
    const I = Y.bind(null, j, rt);
    return (
      lt.push(I),
      Tt(lt)
        .then(() => {
          lt = [];
          for (const V of u.list()) lt.push(mi(V, j, rt));
          return lt.push(I), Tt(lt);
        })
        .then(() => {
          lt = nf(Et, "beforeRouteUpdate", j, rt);
          for (const V of Et)
            V.updateGuards.forEach((Q) => {
              lt.push(mi(Q, j, rt));
            });
          return lt.push(I), Tt(lt);
        })
        .then(() => {
          lt = [];
          for (const V of $)
            if (V.beforeEnter)
              if (lr(V.beforeEnter)) for (const Q of V.beforeEnter) lt.push(mi(Q, j, rt));
              else lt.push(mi(V.beforeEnter, j, rt));
          return lt.push(I), Tt(lt);
        })
        .then(
          () => (
            j.matched.forEach((V) => (V.enterCallbacks = {})),
            (lt = nf($, "beforeRouteEnter", j, rt)),
            lt.push(I),
            Tt(lt)
          ),
        )
        .then(() => {
          lt = [];
          for (const V of f.list()) lt.push(mi(V, j, rt));
          return lt.push(I), Tt(lt);
        })
        .catch((V) => (Or(V, 8) ? V : Promise.reject(V)))
    );
  }
  function pt(j, rt, lt) {
    h.list().forEach((Mt) => nt(() => Mt(j, rt, lt)));
  }
  function gt(j, rt, lt, Mt, Et) {
    const $ = R(j, rt);
    if ($) return $;
    const I = rt === di,
      V = Wo ? history.state : {};
    lt &&
      (Mt || I
        ? s.replace(j.fullPath, pe({ scroll: I && V && V.scroll }, Et))
        : s.push(j.fullPath, Et)),
      (d.value = j),
      qt(j, rt, lt, I),
      At();
  }
  let G;
  function z() {
    G ||
      (G = s.listen((j, rt, lt) => {
        if (!Gt.listening) return;
        const Mt = T(j),
          Et = K(Mt);
        if (Et) {
          ht(pe(Et, { replace: !0 }), Mt).catch(ul);
          return;
        }
        g = Mt;
        const $ = d.value;
        Wo && US(Yg($.fullPath, lt.delta), Rc()),
          at(Mt, $)
            .catch((I) =>
              Or(I, 12)
                ? I
                : Or(I, 2)
                ? (ht(I.to, Mt)
                    .then((V) => {
                      Or(V, 20) && !lt.delta && lt.type === xl.pop && s.go(-1, !1);
                    })
                    .catch(ul),
                  Promise.reject())
                : (lt.delta && s.go(-lt.delta, !1), J(I, Mt, $)),
            )
            .then((I) => {
              (I = I || gt(Mt, $, !1)),
                I &&
                  (lt.delta && !Or(I, 8)
                    ? s.go(-lt.delta, !1)
                    : lt.type === xl.pop && Or(I, 20) && s.go(-1, !1)),
                pt(Mt, $, I);
            })
            .catch(ul);
      }));
  }
  let k = Qs(),
    F = Qs(),
    H;
  function J(j, rt, lt) {
    At(j);
    const Mt = F.list();
    return Mt.length ? Mt.forEach((Et) => Et(j, rt, lt)) : console.error(j), Promise.reject(j);
  }
  function yt() {
    return H && d.value !== di
      ? Promise.resolve()
      : new Promise((j, rt) => {
          k.add([j, rt]);
        });
  }
  function At(j) {
    return H || ((H = !j), z(), k.list().forEach(([rt, lt]) => (j ? lt(j) : rt())), k.reset()), j;
  }
  function qt(j, rt, lt, Mt) {
    const { scrollBehavior: Et } = t;
    if (!Wo || !Et) return Promise.resolve();
    const $ =
      (!lt && jS(Yg(j.fullPath, 0))) ||
      ((Mt || !lt) && history.state && history.state.scroll) ||
      null;
    return Br()
      .then(() => Et(j, rt, $))
      .then((I) => I && WS(I))
      .catch((I) => J(I, j, rt));
  }
  const Ht = (j) => s.go(j);
  let Qt;
  const Jt = new Set(),
    Gt = {
      currentRoute: d,
      listening: !0,
      addRoute: S,
      removeRoute: P,
      hasRoute: L,
      getRoutes: A,
      resolve: T,
      options: t,
      push: E,
      replace: B,
      go: Ht,
      back: () => Ht(-1),
      forward: () => Ht(1),
      beforeEach: u.add,
      beforeResolve: f.add,
      afterEach: h.add,
      onError: F.add,
      isReady: yt,
      install(j) {
        const rt = this;
        j.component("RouterLink", Ak),
          j.component("RouterView", Ok),
          (j.config.globalProperties.$router = rt),
          Object.defineProperty(j.config.globalProperties, "$route", {
            enumerable: !0,
            get: () => U(d),
          }),
          Wo && !Qt && d.value === di && ((Qt = !0), E(s.location).catch((Et) => {}));
        const lt = {};
        for (const Et in di)
          Object.defineProperty(lt, Et, { get: () => d.value[Et], enumerable: !0 });
        j.provide(Rh, rt), j.provide(z0, Wm(lt)), j.provide(zf, d);
        const Mt = j.unmount;
        Jt.add(j),
          (j.unmount = function () {
            Jt.delete(j),
              Jt.size < 1 && ((g = di), G && G(), (G = null), (d.value = di), (Qt = !1), (H = !1)),
              Mt();
          });
      },
    };
  function Tt(j) {
    return j.reduce((rt, lt) => rt.then(() => nt(lt)), Promise.resolve());
  }
  return Gt;
}
function $k(t, e) {
  const r = [],
    o = [],
    s = [],
    u = Math.max(e.matched.length, t.matched.length);
  for (let f = 0; f < u; f++) {
    const h = e.matched[f];
    h && (t.matched.find((g) => ss(g, h)) ? o.push(h) : r.push(h));
    const d = t.matched[f];
    d && (e.matched.find((g) => ss(g, d)) || s.push(d));
  }
  return [r, o, s];
}
function Xr(t) {
  return t.split("-")[0];
}
function Jo(t) {
  return t.split("-")[1];
}
function Dl(t) {
  return ["top", "bottom"].includes(Xr(t)) ? "x" : "y";
}
function zh(t) {
  return t === "y" ? "height" : "width";
}
function uv(t) {
  let { reference: e, floating: r, placement: o } = t;
  const s = e.x + e.width / 2 - r.width / 2,
    u = e.y + e.height / 2 - r.height / 2;
  let f;
  switch (Xr(o)) {
    case "top":
      f = { x: s, y: e.y - r.height };
      break;
    case "bottom":
      f = { x: s, y: e.y + e.height };
      break;
    case "right":
      f = { x: e.x + e.width, y: u };
      break;
    case "left":
      f = { x: e.x - r.width, y: u };
      break;
    default:
      f = { x: e.x, y: e.y };
  }
  const h = Dl(o),
    d = zh(h);
  switch (Jo(o)) {
    case "start":
      f[h] = f[h] - (e[d] / 2 - r[d] / 2);
      break;
    case "end":
      f[h] = f[h] + (e[d] / 2 - r[d] / 2);
      break;
  }
  return f;
}
const Rk = async (t, e, r) => {
  const { placement: o = "bottom", strategy: s = "absolute", middleware: u = [], platform: f } = r;
  let h = await f.getElementRects({ reference: t, floating: e, strategy: s }),
    { x: d, y: g } = uv({ ...h, placement: o }),
    v = o,
    b = {};
  for (let w = 0; w < u.length; w++) {
    const { name: S, fn: P } = u[w],
      {
        x: A,
        y: L,
        data: T,
        reset: M,
      } = await P({
        x: d,
        y: g,
        initialPlacement: o,
        placement: v,
        strategy: s,
        middlewareData: b,
        rects: h,
        platform: f,
        elements: { reference: t, floating: e },
      });
    if (((d = A ?? d), (g = L ?? g), (b = { ...b, [S]: T ?? {} }), M)) {
      typeof M == "object" &&
        (M.placement && (v = M.placement),
        M.rects &&
          (h =
            M.rects === !0
              ? await f.getElementRects({ reference: t, floating: e, strategy: s })
              : M.rects),
        ({ x: d, y: g } = uv({ ...h, placement: v }))),
        (w = -1);
      continue;
    }
  }
  return { x: d, y: g, placement: v, strategy: s, middlewareData: b };
};
function zk(t) {
  return { top: 0, right: 0, bottom: 0, left: 0, ...t };
}
function I0(t) {
  return typeof t != "number" ? zk(t) : { top: t, right: t, bottom: t, left: t };
}
function If(t) {
  return { ...t, top: t.y, left: t.x, right: t.x + t.width, bottom: t.y + t.height };
}
async function zc(t, e) {
  e === void 0 && (e = {});
  const { x: r, y: o, platform: s, rects: u, elements: f, strategy: h } = t,
    {
      boundary: d = "clippingParents",
      rootBoundary: g = "viewport",
      elementContext: v = "floating",
      altBoundary: b = !1,
      padding: w = 0,
    } = e,
    S = I0(w),
    A = f[b ? (v === "floating" ? "reference" : "floating") : v],
    L = await s.getClippingClientRect({
      element: (await s.isElement(A))
        ? A
        : A.contextElement || (await s.getDocumentElement({ element: f.floating })),
      boundary: d,
      rootBoundary: g,
    }),
    T = If(
      await s.convertOffsetParentRelativeRectToViewportRelativeRect({
        rect: v === "floating" ? { ...u.floating, x: r, y: o } : u.reference,
        offsetParent: await s.getOffsetParent({ element: f.floating }),
        strategy: h,
      }),
    );
  return {
    top: L.top - T.top + S.top,
    bottom: T.bottom - L.bottom + S.bottom,
    left: L.left - T.left + S.left,
    right: T.right - L.right + S.right,
  };
}
const Ik = Math.min,
  Gi = Math.max;
function Ff(t, e, r) {
  return Gi(t, Ik(e, r));
}
const Fk = (t) => ({
    name: "arrow",
    options: t,
    async fn(e) {
      const { element: r, padding: o = 0 } = t ?? {},
        { x: s, y: u, placement: f, rects: h, platform: d } = e;
      if (r == null) return {};
      const g = I0(o),
        v = { x: s, y: u },
        b = Xr(f),
        w = Dl(b),
        S = zh(w),
        P = await d.getDimensions({ element: r }),
        A = w === "y" ? "top" : "left",
        L = w === "y" ? "bottom" : "right",
        T = h.reference[S] + h.reference[w] - v[w] - h.floating[S],
        M = v[w] - h.reference[w],
        R = await d.getOffsetParent({ element: r }),
        E = R ? (w === "y" ? R.clientHeight || 0 : R.clientWidth || 0) : 0,
        B = T / 2 - M / 2,
        K = g[A],
        ht = E - P[S] - g[L],
        Y = E / 2 - P[S] / 2 + B,
        nt = Ff(K, Y, ht);
      return { data: { [w]: nt, centerOffset: Y - nt } };
    },
  }),
  qk = { left: "right", right: "left", bottom: "top", top: "bottom" };
function oc(t) {
  return t.replace(/left|right|bottom|top/g, (e) => qk[e]);
}
function F0(t, e) {
  const r = Jo(t) === "start",
    o = Dl(t),
    s = zh(o);
  let u = o === "x" ? (r ? "right" : "left") : r ? "bottom" : "top";
  return e.reference[s] > e.floating[s] && (u = oc(u)), { main: u, cross: oc(u) };
}
const Hk = { start: "end", end: "start" };
function qf(t) {
  return t.replace(/start|end/g, (e) => Hk[e]);
}
const Bk = ["top", "right", "bottom", "left"],
  Wk = Bk.reduce((t, e) => t.concat(e, e + "-start", e + "-end"), []);
function Uk(t, e, r) {
  return (
    t
      ? [...r.filter((s) => Jo(s) === t), ...r.filter((s) => Jo(s) !== t)]
      : r.filter((s) => Xr(s) === s)
  ).filter((s) => (t ? Jo(s) === t || (e ? qf(s) !== s : !1) : !0));
}
const jk = function (t) {
  return (
    t === void 0 && (t = {}),
    {
      name: "autoPlacement",
      options: t,
      async fn(e) {
        var r, o, s, u, f, h;
        const { x: d, y: g, rects: v, middlewareData: b, placement: w } = e,
          { alignment: S = null, allowedPlacements: P = Wk, autoAlignment: A = !0, ...L } = t;
        if ((r = b.autoPlacement) != null && r.skip) return {};
        const T = Uk(S, A, P),
          M = await zc(e, L),
          R = (o = (s = b.autoPlacement) == null ? void 0 : s.index) != null ? o : 0,
          E = T[R],
          { main: B, cross: K } = F0(E, v);
        if (w !== E) return { x: d, y: g, reset: { placement: T[0] } };
        const ht = [M[Xr(E)], M[B], M[K]],
          Y = [
            ...((u = (f = b.autoPlacement) == null ? void 0 : f.overflows) != null ? u : []),
            { placement: E, overflows: ht },
          ],
          nt = T[R + 1];
        if (nt) return { data: { index: R + 1, overflows: Y }, reset: { placement: nt } };
        const at = Y.slice().sort((gt, G) => gt.overflows[0] - G.overflows[0]),
          pt =
            (h = at.find((gt) => {
              let { overflows: G } = gt;
              return G.every((z) => z <= 0);
            })) == null
              ? void 0
              : h.placement;
        return { data: { skip: !0 }, reset: { placement: pt ?? at[0].placement } };
      },
    }
  );
};
function Vk(t) {
  const e = oc(t);
  return [qf(t), e, qf(e)];
}
const Gk = function (t) {
  return (
    t === void 0 && (t = {}),
    {
      name: "flip",
      options: t,
      async fn(e) {
        var r, o;
        const { placement: s, middlewareData: u, rects: f, initialPlacement: h } = e;
        if ((r = u.flip) != null && r.skip) return {};
        const {
            mainAxis: d = !0,
            crossAxis: g = !0,
            fallbackPlacements: v,
            fallbackStrategy: b = "bestFit",
            flipAlignment: w = !0,
            ...S
          } = t,
          P = Xr(s),
          L = v || (P === h || !w ? [oc(h)] : Vk(h)),
          T = [h, ...L],
          M = await zc(e, S),
          R = [];
        let E = ((o = u.flip) == null ? void 0 : o.overflows) || [];
        if ((d && R.push(M[P]), g)) {
          const { main: Y, cross: nt } = F0(s, f);
          R.push(M[Y], M[nt]);
        }
        if (((E = [...E, { placement: s, overflows: R }]), !R.every((Y) => Y <= 0))) {
          var B, K;
          const Y = ((B = (K = u.flip) == null ? void 0 : K.index) != null ? B : 0) + 1,
            nt = T[Y];
          if (nt) return { data: { index: Y, overflows: E }, reset: { placement: nt } };
          let at = "bottom";
          switch (b) {
            case "bestFit": {
              var ht;
              const pt =
                (ht = E.slice().sort(
                  (gt, G) =>
                    gt.overflows.filter((z) => z > 0).reduce((z, k) => z + k, 0) -
                    G.overflows.filter((z) => z > 0).reduce((z, k) => z + k, 0),
                )[0]) == null
                  ? void 0
                  : ht.placement;
              pt && (at = pt);
              break;
            }
            case "initialPlacement":
              at = h;
              break;
          }
          return { data: { skip: !0 }, reset: { placement: at } };
        }
        return {};
      },
    }
  );
};
function Kk(t) {
  let { placement: e, rects: r, value: o } = t;
  const s = Xr(e),
    u = ["left", "top"].includes(s) ? -1 : 1,
    f = typeof o == "function" ? o({ ...r, placement: e }) : o,
    { mainAxis: h, crossAxis: d } =
      typeof f == "number" ? { mainAxis: f, crossAxis: 0 } : { mainAxis: 0, crossAxis: 0, ...f };
  return Dl(s) === "x" ? { x: d, y: h * u } : { x: h * u, y: d };
}
const Xk = function (t) {
  return (
    t === void 0 && (t = 0),
    {
      name: "offset",
      options: t,
      fn(e) {
        const { x: r, y: o, placement: s, rects: u } = e,
          f = Kk({ placement: s, rects: u, value: t });
        return { x: r + f.x, y: o + f.y, data: f };
      },
    }
  );
};
function Yk(t) {
  return t === "x" ? "y" : "x";
}
const Zk = function (t) {
    return (
      t === void 0 && (t = {}),
      {
        name: "shift",
        options: t,
        async fn(e) {
          const { x: r, y: o, placement: s } = e,
            {
              mainAxis: u = !0,
              crossAxis: f = !1,
              limiter: h = {
                fn: (L) => {
                  let { x: T, y: M } = L;
                  return { x: T, y: M };
                },
              },
              ...d
            } = t,
            g = { x: r, y: o },
            v = await zc(e, d),
            b = Dl(Xr(s)),
            w = Yk(b);
          let S = g[b],
            P = g[w];
          if (u) {
            const L = b === "y" ? "top" : "left",
              T = b === "y" ? "bottom" : "right",
              M = S + v[L],
              R = S - v[T];
            S = Ff(M, S, R);
          }
          if (f) {
            const L = w === "y" ? "top" : "left",
              T = w === "y" ? "bottom" : "right",
              M = P + v[L],
              R = P - v[T];
            P = Ff(M, P, R);
          }
          const A = h.fn({ ...e, [b]: S, [w]: P });
          return { ...A, data: { x: A.x - r, y: A.y - o } };
        },
      }
    );
  },
  Jk = function (t) {
    return (
      t === void 0 && (t = {}),
      {
        name: "size",
        options: t,
        async fn(e) {
          var r;
          const { placement: o, rects: s, middlewareData: u } = e,
            { apply: f, ...h } = t;
          if ((r = u.size) != null && r.skip) return {};
          const d = await zc(e, h),
            g = Xr(o),
            v = Jo(o) === "end";
          let b, w;
          g === "top" || g === "bottom"
            ? ((b = g), (w = v ? "left" : "right"))
            : ((w = g), (b = v ? "top" : "bottom"));
          const S = Gi(d.left, 0),
            P = Gi(d.right, 0),
            A = Gi(d.top, 0),
            L = Gi(d.bottom, 0),
            T = {
              height:
                s.floating.height -
                (["left", "right"].includes(o)
                  ? 2 * (A !== 0 || L !== 0 ? A + L : Gi(d.top, d.bottom))
                  : d[b]),
              width:
                s.floating.width -
                (["top", "bottom"].includes(o)
                  ? 2 * (S !== 0 || P !== 0 ? S + P : Gi(d.left, d.right))
                  : d[w]),
            };
          return f == null || f({ ...T, ...s }), { data: { skip: !0 }, reset: { rects: !0 } };
        },
      }
    );
  };
function Ih(t) {
  return (t == null ? void 0 : t.toString()) === "[object Window]";
}
function Mi(t) {
  if (t == null) return window;
  if (!Ih(t)) {
    const e = t.ownerDocument;
    return (e && e.defaultView) || window;
  }
  return t;
}
function Ic(t) {
  return Mi(t).getComputedStyle(t);
}
function Wr(t) {
  return Ih(t) ? "" : t ? (t.nodeName || "").toLowerCase() : "";
}
function Ur(t) {
  return t instanceof Mi(t).HTMLElement;
}
function sc(t) {
  return t instanceof Mi(t).Element;
}
function Qk(t) {
  return t instanceof Mi(t).Node;
}
function q0(t) {
  const e = Mi(t).ShadowRoot;
  return t instanceof e || t instanceof ShadowRoot;
}
function Fc(t) {
  const { overflow: e, overflowX: r, overflowY: o } = Ic(t);
  return /auto|scroll|overlay|hidden/.test(e + o + r);
}
function tC(t) {
  return ["table", "td", "th"].includes(Wr(t));
}
function H0(t) {
  const e = navigator.userAgent.toLowerCase().includes("firefox"),
    r = Ic(t);
  return (
    r.transform !== "none" ||
    r.perspective !== "none" ||
    r.contain === "paint" ||
    ["transform", "perspective"].includes(r.willChange) ||
    (e && r.willChange === "filter") ||
    (e && (r.filter ? r.filter !== "none" : !1))
  );
}
const fv = Math.min,
  hl = Math.max,
  lc = Math.round;
function as(t, e) {
  e === void 0 && (e = !1);
  const r = t.getBoundingClientRect();
  let o = 1,
    s = 1;
  return (
    e &&
      Ur(t) &&
      ((o = (t.offsetWidth > 0 && lc(r.width) / t.offsetWidth) || 1),
      (s = (t.offsetHeight > 0 && lc(r.height) / t.offsetHeight) || 1)),
    {
      width: r.width / o,
      height: r.height / s,
      top: r.top / s,
      right: r.right / o,
      bottom: r.bottom / s,
      left: r.left / o,
      x: r.left / o,
      y: r.top / s,
    }
  );
}
function Ni(t) {
  return ((Qk(t) ? t.ownerDocument : t.document) || window.document).documentElement;
}
function qc(t) {
  return Ih(t)
    ? { scrollLeft: t.pageXOffset, scrollTop: t.pageYOffset }
    : { scrollLeft: t.scrollLeft, scrollTop: t.scrollTop };
}
function B0(t) {
  return as(Ni(t)).left + qc(t).scrollLeft;
}
function eC(t) {
  const e = as(t);
  return lc(e.width) !== t.offsetWidth || lc(e.height) !== t.offsetHeight;
}
function nC(t, e, r) {
  const o = Ur(e),
    s = Ni(e),
    u = as(t, o && eC(e));
  let f = { scrollLeft: 0, scrollTop: 0 };
  const h = { x: 0, y: 0 };
  if (o || (!o && r !== "fixed"))
    if (((Wr(e) !== "body" || Fc(s)) && (f = qc(e)), Ur(e))) {
      const d = as(e, !0);
      (h.x = d.x + e.clientLeft), (h.y = d.y + e.clientTop);
    } else s && (h.x = B0(s));
  return {
    x: u.left + f.scrollLeft - h.x,
    y: u.top + f.scrollTop - h.y,
    width: u.width,
    height: u.height,
  };
}
function Hc(t) {
  return Wr(t) === "html" ? t : t.assignedSlot || t.parentNode || (q0(t) ? t.host : null) || Ni(t);
}
function hv(t) {
  return !Ur(t) || getComputedStyle(t).position === "fixed" ? null : t.offsetParent;
}
function rC(t) {
  let e = Hc(t);
  for (; Ur(e) && !["html", "body"].includes(Wr(e)); ) {
    if (H0(e)) return e;
    e = e.parentNode;
  }
  return null;
}
function Hf(t) {
  const e = Mi(t);
  let r = hv(t);
  for (; r && tC(r) && getComputedStyle(r).position === "static"; ) r = hv(r);
  return r &&
    (Wr(r) === "html" || (Wr(r) === "body" && getComputedStyle(r).position === "static" && !H0(r)))
    ? e
    : r || rC(t) || e;
}
function dv(t) {
  return { width: t.offsetWidth, height: t.offsetHeight };
}
function iC(t) {
  let { rect: e, offsetParent: r, strategy: o } = t;
  const s = Ur(r),
    u = Ni(r);
  if (r === u) return e;
  let f = { scrollLeft: 0, scrollTop: 0 };
  const h = { x: 0, y: 0 };
  if ((s || (!s && o !== "fixed")) && ((Wr(r) !== "body" || Fc(u)) && (f = qc(r)), Ur(r))) {
    const d = as(r, !0);
    (h.x = d.x + r.clientLeft), (h.y = d.y + r.clientTop);
  }
  return { ...e, x: e.x - f.scrollLeft + h.x, y: e.y - f.scrollTop + h.y };
}
function oC(t) {
  const e = Mi(t),
    r = Ni(t),
    o = e.visualViewport;
  let s = r.clientWidth,
    u = r.clientHeight,
    f = 0,
    h = 0;
  return (
    o &&
      ((s = o.width),
      (u = o.height),
      Math.abs(e.innerWidth / o.scale - o.width) < 0.01 && ((f = o.offsetLeft), (h = o.offsetTop))),
    { width: s, height: u, x: f, y: h }
  );
}
function sC(t) {
  var e;
  const r = Ni(t),
    o = qc(t),
    s = (e = t.ownerDocument) == null ? void 0 : e.body,
    u = hl(r.scrollWidth, r.clientWidth, s ? s.scrollWidth : 0, s ? s.clientWidth : 0),
    f = hl(r.scrollHeight, r.clientHeight, s ? s.scrollHeight : 0, s ? s.clientHeight : 0);
  let h = -o.scrollLeft + B0(t);
  const d = -o.scrollTop;
  return (
    Ic(s || r).direction === "rtl" && (h += hl(r.clientWidth, s ? s.clientWidth : 0) - u),
    { width: u, height: f, x: h, y: d }
  );
}
function W0(t) {
  return ["html", "body", "#document"].includes(Wr(t))
    ? t.ownerDocument.body
    : Ur(t) && Fc(t)
    ? t
    : W0(Hc(t));
}
function ac(t, e) {
  var r;
  e === void 0 && (e = []);
  const o = W0(t),
    s = o === ((r = t.ownerDocument) == null ? void 0 : r.body),
    u = Mi(o),
    f = s ? [u].concat(u.visualViewport || [], Fc(o) ? o : []) : o,
    h = e.concat(f);
  return s ? h : h.concat(ac(Hc(f)));
}
function lC(t, e) {
  const r = e.getRootNode == null ? void 0 : e.getRootNode();
  if (t.contains(e)) return !0;
  if (r && q0(r)) {
    let o = e;
    do {
      if (o && t === o) return !0;
      o = o.parentNode || o.host;
    } while (o);
  }
  return !1;
}
function aC(t) {
  const e = as(t),
    r = e.top + t.clientTop,
    o = e.left + t.clientLeft;
  return {
    top: r,
    left: o,
    x: o,
    y: r,
    right: o + t.clientWidth,
    bottom: r + t.clientHeight,
    width: t.clientWidth,
    height: t.clientHeight,
  };
}
function pv(t, e) {
  return e === "viewport" ? If(oC(t)) : sc(e) ? aC(e) : If(sC(Ni(t)));
}
function cC(t) {
  const e = ac(Hc(t)),
    o = ["absolute", "fixed"].includes(Ic(t).position) && Ur(t) ? Hf(t) : t;
  return sc(o) ? e.filter((s) => sc(s) && lC(s, o) && Wr(s) !== "body") : [];
}
function uC(t) {
  let { element: e, boundary: r, rootBoundary: o } = t;
  const u = [...(r === "clippingParents" ? cC(e) : [].concat(r)), o],
    f = u[0],
    h = u.reduce((d, g) => {
      const v = pv(e, g);
      return (
        (d.top = hl(v.top, d.top)),
        (d.right = fv(v.right, d.right)),
        (d.bottom = fv(v.bottom, d.bottom)),
        (d.left = hl(v.left, d.left)),
        d
      );
    }, pv(e, f));
  return (
    (h.width = h.right - h.left), (h.height = h.bottom - h.top), (h.x = h.left), (h.y = h.top), h
  );
}
const fC = {
    getElementRects: (t) => {
      let { reference: e, floating: r, strategy: o } = t;
      return { reference: nC(e, Hf(r), o), floating: { ...dv(r), x: 0, y: 0 } };
    },
    convertOffsetParentRelativeRectToViewportRelativeRect: (t) => iC(t),
    getOffsetParent: (t) => {
      let { element: e } = t;
      return Hf(e);
    },
    isElement: (t) => sc(t),
    getDocumentElement: (t) => {
      let { element: e } = t;
      return Ni(e);
    },
    getClippingClientRect: (t) => uC(t),
    getDimensions: (t) => {
      let { element: e } = t;
      return dv(e);
    },
    getClientRects: (t) => {
      let { element: e } = t;
      return e.getClientRects();
    },
  },
  hC = (t, e, r) => Rk(t, e, { platform: fC, ...r });
var dC = Object.defineProperty,
  pC = Object.defineProperties,
  gC = Object.getOwnPropertyDescriptors,
  gv = Object.getOwnPropertySymbols,
  vC = Object.prototype.hasOwnProperty,
  mC = Object.prototype.propertyIsEnumerable,
  vv = (t, e, r) =>
    e in t ? dC(t, e, { enumerable: !0, configurable: !0, writable: !0, value: r }) : (t[e] = r),
  zr = (t, e) => {
    for (var r in e || (e = {})) vC.call(e, r) && vv(t, r, e[r]);
    if (gv) for (var r of gv(e)) mC.call(e, r) && vv(t, r, e[r]);
    return t;
  },
  $l = (t, e) => pC(t, gC(e));
function U0(t, e) {
  for (const r in e)
    Object.prototype.hasOwnProperty.call(e, r) &&
      (typeof e[r] == "object" && t[r] ? U0(t[r], e[r]) : (t[r] = e[r]));
}
const eo = {
  disabled: !1,
  distance: 5,
  skidding: 0,
  container: "body",
  boundary: void 0,
  instantMove: !1,
  disposeTimeout: 5e3,
  popperTriggers: [],
  strategy: "absolute",
  preventOverflow: !0,
  flip: !0,
  shift: !0,
  overflowPadding: 0,
  arrowPadding: 0,
  arrowOverflow: !0,
  themes: {
    tooltip: {
      placement: "top",
      triggers: ["hover", "focus", "touch"],
      hideTriggers: (t) => [...t, "click"],
      delay: { show: 200, hide: 0 },
      handleResize: !1,
      html: !1,
      loadingContent: "...",
    },
    dropdown: {
      placement: "bottom",
      triggers: ["click"],
      delay: 0,
      handleResize: !0,
      autoHide: !0,
    },
    menu: {
      $extend: "dropdown",
      triggers: ["hover", "focus"],
      popperTriggers: ["hover", "focus"],
      delay: { show: 0, hide: 400 },
    },
  },
};
function cs(t, e) {
  let r = eo.themes[t] || {},
    o;
  do
    (o = r[e]),
      typeof o > "u"
        ? r.$extend
          ? (r = eo.themes[r.$extend] || {})
          : ((r = null), (o = eo[e]))
        : (r = null);
  while (r);
  return o;
}
function yC(t) {
  const e = [t];
  let r = eo.themes[t] || {};
  do r.$extend && !r.$resetCss ? (e.push(r.$extend), (r = eo.themes[r.$extend] || {})) : (r = null);
  while (r);
  return e.map((o) => `v-popper--theme-${o}`);
}
let us = !1;
if (typeof window < "u") {
  us = !1;
  try {
    const t = Object.defineProperty({}, "passive", {
      get() {
        us = !0;
      },
    });
    window.addEventListener("test", null, t);
  } catch {}
}
let j0 = !1;
typeof window < "u" &&
  typeof navigator < "u" &&
  (j0 = /iPad|iPhone|iPod/.test(navigator.userAgent) && !window.MSStream);
const V0 = ["auto", "top", "bottom", "left", "right"].reduce(
    (t, e) => t.concat([e, `${e}-start`, `${e}-end`]),
    [],
  ),
  mv = { hover: "mouseenter", focus: "focus", click: "click", touch: "touchstart" },
  yv = { hover: "mouseleave", focus: "blur", click: "click", touch: "touchend" };
function bC(t, e) {
  const r = t.indexOf(e);
  r !== -1 && t.splice(r, 1);
}
function rf() {
  return new Promise((t) =>
    requestAnimationFrame(() => {
      requestAnimationFrame(t);
    }),
  );
}
const br = [];
let Ho = null,
  Bf = function () {};
typeof window < "u" && (Bf = window.Element);
function de(t) {
  return function (e) {
    return cs(e.theme, t);
  };
}
var G0 = () =>
  ie({
    name: "VPopper",
    props: {
      theme: { type: String, required: !0 },
      targetNodes: { type: Function, required: !0 },
      referenceNode: { type: Function, required: !0 },
      popperNode: { type: Function, required: !0 },
      shown: { type: Boolean, default: !1 },
      showGroup: { type: String, default: null },
      ariaId: { default: null },
      disabled: { type: Boolean, default: de("disabled") },
      placement: { type: String, default: de("placement"), validator: (t) => V0.includes(t) },
      delay: { type: [String, Number, Object], default: de("delay") },
      distance: { type: [Number, String], default: de("distance") },
      skidding: { type: [Number, String], default: de("skidding") },
      triggers: { type: Array, default: de("triggers") },
      showTriggers: { type: [Array, Function], default: de("showTriggers") },
      hideTriggers: { type: [Array, Function], default: de("hideTriggers") },
      popperTriggers: { type: Array, default: de("popperTriggers") },
      popperShowTriggers: { type: [Array, Function], default: de("popperShowTriggers") },
      popperHideTriggers: { type: [Array, Function], default: de("popperHideTriggers") },
      container: { type: [String, Object, Bf, Boolean], default: de("container") },
      boundary: { type: [String, Bf], default: de("boundary") },
      strategy: {
        type: String,
        validator: (t) => ["absolute", "fixed"].includes(t),
        default: de("strategy"),
      },
      autoHide: { type: Boolean, default: de("autoHide") },
      handleResize: { type: Boolean, default: de("handleResize") },
      instantMove: { type: Boolean, default: de("instantMove") },
      eagerMount: { type: Boolean, default: de("eagerMount") },
      popperClass: { type: [String, Array, Object], default: de("popperClass") },
      computeTransformOrigin: { type: Boolean, default: de("computeTransformOrigin") },
      autoMinSize: { type: Boolean, default: de("autoMinSize") },
      autoMaxSize: { type: Boolean, default: de("autoMaxSize") },
      preventOverflow: { type: Boolean, default: de("preventOverflow") },
      overflowPadding: { type: [Number, String], default: de("overflowPadding") },
      arrowPadding: { type: [Number, String], default: de("arrowPadding") },
      arrowOverflow: { type: Boolean, default: de("arrowOverflow") },
      flip: { type: Boolean, default: de("flip") },
      shift: { type: Boolean, default: de("shift") },
      shiftCrossAxis: { type: Boolean, default: de("shiftCrossAxis") },
    },
    emits: [
      "show",
      "hide",
      "update:shown",
      "apply-show",
      "apply-hide",
      "close-group",
      "close-directive",
      "auto-hide",
      "resize",
      "dispose",
    ],
    data() {
      return {
        isShown: !1,
        isMounted: !1,
        skipTransition: !1,
        classes: { showFrom: !1, showTo: !1, hideFrom: !1, hideTo: !0 },
        result: {
          x: 0,
          y: 0,
          placement: "",
          strategy: this.strategy,
          arrow: { x: 0, y: 0, centerOffset: 0 },
          transformOrigin: null,
        },
      };
    },
    computed: {
      popperId() {
        return this.ariaId != null ? this.ariaId : this.randomId;
      },
      shouldMountContent() {
        return this.eagerMount || this.isMounted;
      },
      slotData() {
        return {
          popperId: this.popperId,
          isShown: this.isShown,
          shouldMountContent: this.shouldMountContent,
          skipTransition: this.skipTransition,
          autoHide: this.autoHide,
          show: this.show,
          hide: this.hide,
          handleResize: this.handleResize,
          onResize: this.onResize,
          classes: $l(zr({}, this.classes), { popperClass: this.popperClass }),
          result: this.result,
        };
      },
    },
    watch: zr(
      {
        shown: "$_autoShowHide",
        disabled(t) {
          t ? this.dispose() : this.init();
        },
        async container() {
          this.isShown && (this.$_ensureTeleport(), await this.$_computePosition());
        },
        triggers() {
          this.$_isDisposed || (this.$_removeEventListeners(), this.$_addEventListeners());
        },
      },
      [
        "placement",
        "distance",
        "skidding",
        "boundary",
        "strategy",
        "overflowPadding",
        "arrowPadding",
        "preventOverflow",
        "shift",
        "shiftCrossAxis",
        "flip",
      ].reduce((t, e) => ((t[e] = "$_computePosition"), t), {}),
    ),
    created() {
      (this.$_isDisposed = !0),
        (this.randomId = `popper_${[Math.random(), Date.now()]
          .map((t) => t.toString(36).substring(2, 10))
          .join("_")}`);
    },
    mounted() {
      this.init(), this.$_detachPopperNode();
    },
    activated() {
      this.$_autoShowHide();
    },
    deactivated() {
      this.hide();
    },
    beforeUnmount() {
      this.dispose();
    },
    methods: {
      show({ event: t = null, skipDelay: e = !1, force: r = !1 } = {}) {
        (r || !this.disabled) &&
          (this.$_scheduleShow(t, e),
          this.$emit("show"),
          (this.$_showFrameLocked = !0),
          requestAnimationFrame(() => {
            this.$_showFrameLocked = !1;
          })),
          this.$emit("update:shown", !0);
      },
      hide({ event: t = null, skipDelay: e = !1 } = {}) {
        this.$_scheduleHide(t, e), this.$emit("hide"), this.$emit("update:shown", !1);
      },
      init() {
        this.$_isDisposed &&
          ((this.$_isDisposed = !1),
          (this.isMounted = !1),
          (this.$_events = []),
          (this.$_preventShow = !1),
          (this.$_referenceNode = this.referenceNode()),
          (this.$_targetNodes = this.targetNodes().filter((t) => t.nodeType === t.ELEMENT_NODE)),
          (this.$_popperNode = this.popperNode()),
          (this.$_innerNode = this.$_popperNode.querySelector(".v-popper__inner")),
          (this.$_arrowNode = this.$_popperNode.querySelector(".v-popper__arrow-container")),
          this.$_swapTargetAttrs("title", "data-original-title"),
          this.$_detachPopperNode(),
          this.triggers.length && this.$_addEventListeners(),
          this.shown && this.show());
      },
      dispose() {
        this.$_isDisposed ||
          ((this.$_isDisposed = !0),
          this.$_removeEventListeners(),
          this.hide({ skipDelay: !0 }),
          this.$_detachPopperNode(),
          (this.isMounted = !1),
          (this.isShown = !1),
          this.$_swapTargetAttrs("data-original-title", "title"),
          this.$emit("dispose"));
      },
      async onResize() {
        this.isShown && (await this.$_computePosition(), this.$emit("resize"));
      },
      async $_computePosition() {
        var t;
        if (this.$_isDisposed) return;
        const e = { strategy: this.strategy, middleware: [] };
        (this.distance || this.skidding) &&
          e.middleware.push(Xk({ mainAxis: this.distance, crossAxis: this.skidding }));
        const r = this.placement.startsWith("auto");
        r
          ? e.middleware.push(
              jk({ alignment: (t = this.placement.split("-")[1]) != null ? t : "" }),
            )
          : (e.placement = this.placement),
          this.preventOverflow &&
            (this.shift &&
              e.middleware.push(
                Zk({
                  padding: this.overflowPadding,
                  boundary: this.boundary,
                  crossAxis: this.shiftCrossAxis,
                }),
              ),
            !r &&
              this.flip &&
              e.middleware.push(Gk({ padding: this.overflowPadding, boundary: this.boundary }))),
          e.middleware.push(Fk({ element: this.$_arrowNode, padding: this.arrowPadding })),
          this.arrowOverflow &&
            e.middleware.push({
              name: "arrowOverflow",
              fn: ({ placement: s, rects: u, middlewareData: f }) => {
                let h;
                const { centerOffset: d } = f.arrow;
                return (
                  s.startsWith("top") || s.startsWith("bottom")
                    ? (h = Math.abs(d) > u.reference.width / 2)
                    : (h = Math.abs(d) > u.reference.height / 2),
                  { data: { overflow: h } }
                );
              },
            }),
          this.autoMinSize &&
            e.middleware.push({
              name: "autoMinSize",
              fn: ({ rects: s, placement: u, middlewareData: f }) => {
                var h;
                if ((h = f.autoMinSize) != null && h.skip) return {};
                let d, g;
                return (
                  u.startsWith("top") || u.startsWith("bottom")
                    ? (d = s.reference.width)
                    : (g = s.reference.height),
                  (this.$_innerNode.style.minWidth = d != null ? `${d}px` : null),
                  (this.$_innerNode.style.minHeight = g != null ? `${g}px` : null),
                  { data: { skip: !0 }, reset: { rects: !0 } }
                );
              },
            }),
          this.autoMaxSize &&
            e.middleware.push(
              Jk({
                boundary: this.boundary,
                padding: this.overflowPadding,
                apply: ({ width: s, height: u }) => {
                  (this.$_innerNode.style.maxWidth = s != null ? `${s}px` : null),
                    (this.$_innerNode.style.maxHeight = u != null ? `${u}px` : null);
                },
              }),
            );
        const o = await hC(this.$_referenceNode, this.$_popperNode, e);
        Object.assign(this.result, {
          x: o.x,
          y: o.y,
          placement: o.placement,
          strategy: o.strategy,
          arrow: zr(zr({}, o.middlewareData.arrow), o.middlewareData.arrowOverflow),
        });
      },
      $_scheduleShow(t = null, e = !1) {
        if (
          ((this.$_hideInProgress = !1),
          clearTimeout(this.$_scheduleTimer),
          Ho && this.instantMove && Ho.instantMove)
        ) {
          Ho.$_applyHide(!0), this.$_applyShow(!0);
          return;
        }
        e
          ? this.$_applyShow()
          : (this.$_scheduleTimer = setTimeout(
              this.$_applyShow.bind(this),
              this.$_computeDelay("show"),
            ));
      },
      $_scheduleHide(t = null, e = !1) {
        (this.$_hideInProgress = !0),
          clearTimeout(this.$_scheduleTimer),
          this.isShown && (Ho = this),
          e
            ? this.$_applyHide()
            : (this.$_scheduleTimer = setTimeout(
                this.$_applyHide.bind(this),
                this.$_computeDelay("hide"),
              ));
      },
      $_computeDelay(t) {
        const e = this.delay;
        return parseInt((e && e[t]) || e || 0);
      },
      async $_applyShow(t = !1) {
        clearTimeout(this.$_disposeTimer),
          clearTimeout(this.$_scheduleTimer),
          (this.skipTransition = t),
          !this.isShown &&
            (this.$_ensureTeleport(),
            await rf(),
            await this.$_computePosition(),
            await this.$_applyShowEffect());
      },
      async $_applyShowEffect() {
        if (this.$_hideInProgress) return;
        if (this.computeTransformOrigin) {
          const e = this.$_referenceNode.getBoundingClientRect(),
            r = this.$_popperNode.querySelector(".v-popper__wrapper"),
            o = r.parentNode.getBoundingClientRect(),
            s = e.x + e.width / 2 - (o.left + r.offsetLeft),
            u = e.y + e.height / 2 - (o.top + r.offsetTop);
          this.result.transformOrigin = `${s}px ${u}px`;
        }
        (this.isShown = !0),
          this.$_applyAttrsToTarget({ "aria-describedby": this.popperId, "data-popper-shown": "" });
        const t = this.showGroup;
        if (t) {
          let e;
          for (let r = 0; r < br.length; r++)
            (e = br[r]), e.showGroup !== t && (e.hide(), e.$emit("close-group"));
        }
        br.push(this),
          this.$emit("apply-show"),
          (this.classes.showFrom = !0),
          (this.classes.showTo = !1),
          (this.classes.hideFrom = !1),
          (this.classes.hideTo = !1),
          await rf(),
          (this.classes.showFrom = !1),
          (this.classes.showTo = !0);
      },
      async $_applyHide(t = !1) {
        if ((clearTimeout(this.$_scheduleTimer), !this.isShown)) return;
        (this.skipTransition = t),
          bC(br, this),
          Ho === this && (Ho = null),
          (this.isShown = !1),
          this.$_applyAttrsToTarget({ "aria-describedby": void 0, "data-popper-shown": void 0 }),
          clearTimeout(this.$_disposeTimer);
        const e = cs(this.theme, "disposeTimeout");
        e !== null &&
          (this.$_disposeTimer = setTimeout(() => {
            this.$_popperNode && (this.$_detachPopperNode(), (this.isMounted = !1));
          }, e)),
          this.$emit("apply-hide"),
          (this.classes.showFrom = !1),
          (this.classes.showTo = !1),
          (this.classes.hideFrom = !0),
          (this.classes.hideTo = !1),
          await rf(),
          (this.classes.hideFrom = !1),
          (this.classes.hideTo = !0);
      },
      $_autoShowHide() {
        this.shown ? this.show() : this.hide();
      },
      $_ensureTeleport() {
        if (this.$_isDisposed) return;
        let t = this.container;
        if (
          (typeof t == "string"
            ? (t = window.document.querySelector(t))
            : t === !1 && (t = this.$_targetNodes[0].parentNode),
          !t)
        )
          throw new Error("No container for popover: " + this.container);
        t.appendChild(this.$_popperNode), (this.isMounted = !0);
      },
      $_addEventListeners() {
        const t = (s, u, f) => {
            this.$_events.push({ targetNodes: s, eventType: u, handler: f }),
              s.forEach((h) => h.addEventListener(u, f, us ? { passive: !0 } : void 0));
          },
          e = (s, u, f, h, d) => {
            let g = f;
            h != null && (g = typeof h == "function" ? h(g) : h),
              g.forEach((v) => {
                const b = u[v];
                b && t(s, b, d);
              });
          },
          r = (s) => {
            (this.isShown && !this.$_hideInProgress) ||
              ((s.usedByTooltip = !0), !this.$_preventShow && this.show({ event: s }));
          };
        e(this.$_targetNodes, mv, this.triggers, this.showTriggers, r),
          e([this.$_popperNode], mv, this.popperTriggers, this.popperShowTriggers, r);
        const o = (s) => {
          s.usedByTooltip || this.hide({ event: s });
        };
        e(this.$_targetNodes, yv, this.triggers, this.hideTriggers, o),
          e([this.$_popperNode], yv, this.popperTriggers, this.popperHideTriggers, o),
          t([...ac(this.$_referenceNode), ...ac(this.$_popperNode)], "scroll", () => {
            this.$_computePosition();
          });
      },
      $_removeEventListeners() {
        this.$_events.forEach(({ targetNodes: t, eventType: e, handler: r }) => {
          t.forEach((o) => o.removeEventListener(e, r));
        }),
          (this.$_events = []);
      },
      $_handleGlobalClose(t, e = !1) {
        this.$_showFrameLocked ||
          (this.hide({ event: t }),
          t.closePopover ? this.$emit("close-directive") : this.$emit("auto-hide"),
          e &&
            ((this.$_preventShow = !0),
            setTimeout(() => {
              this.$_preventShow = !1;
            }, 300)));
      },
      $_detachPopperNode() {
        this.$_popperNode.parentNode && this.$_popperNode.parentNode.removeChild(this.$_popperNode);
      },
      $_swapTargetAttrs(t, e) {
        for (const r of this.$_targetNodes) {
          const o = r.getAttribute(t);
          o && (r.removeAttribute(t), r.setAttribute(e, o));
        }
      },
      $_applyAttrsToTarget(t) {
        for (const e of this.$_targetNodes)
          for (const r in t) {
            const o = t[r];
            o == null ? e.removeAttribute(r) : e.setAttribute(r, o);
          }
      },
    },
    render() {
      return this.$slots.default(this.slotData);
    },
  });
typeof document < "u" &&
  typeof window < "u" &&
  (j0
    ? (document.addEventListener("touchstart", bv, us ? { passive: !0, capture: !0 } : !0),
      document.addEventListener("touchend", xC, us ? { passive: !0, capture: !0 } : !0))
    : (window.addEventListener("mousedown", bv, !0), window.addEventListener("click", wC, !0)),
  window.addEventListener("resize", _C));
function bv(t) {
  for (let e = 0; e < br.length; e++) {
    const r = br[e],
      o = r.popperNode();
    r.$_mouseDownContains = o.contains(t.target);
  }
}
function wC(t) {
  K0(t);
}
function xC(t) {
  K0(t, !0);
}
function K0(t, e = !1) {
  for (let r = 0; r < br.length; r++) {
    const o = br[r],
      s = o.popperNode(),
      u = o.$_mouseDownContains || s.contains(t.target);
    requestAnimationFrame(() => {
      (t.closeAllPopover || (t.closePopover && u) || (o.autoHide && !u)) &&
        o.$_handleGlobalClose(t, e);
    });
  }
}
function _C(t) {
  for (let e = 0; e < br.length; e++) br[e].$_computePosition(t);
}
function SC() {
  var t = window.navigator.userAgent,
    e = t.indexOf("MSIE ");
  if (e > 0) return parseInt(t.substring(e + 5, t.indexOf(".", e)), 10);
  var r = t.indexOf("Trident/");
  if (r > 0) {
    var o = t.indexOf("rv:");
    return parseInt(t.substring(o + 3, t.indexOf(".", o)), 10);
  }
  var s = t.indexOf("Edge/");
  return s > 0 ? parseInt(t.substring(s + 5, t.indexOf(".", s)), 10) : -1;
}
let Ba;
function Wf() {
  Wf.init || ((Wf.init = !0), (Ba = SC() !== -1));
}
var Bc = {
  name: "ResizeObserver",
  props: {
    emitOnMount: { type: Boolean, default: !1 },
    ignoreWidth: { type: Boolean, default: !1 },
    ignoreHeight: { type: Boolean, default: !1 },
  },
  emits: ["notify"],
  mounted() {
    Wf(),
      Br(() => {
        (this._w = this.$el.offsetWidth),
          (this._h = this.$el.offsetHeight),
          this.emitOnMount && this.emitSize();
      });
    const t = document.createElement("object");
    (this._resizeObject = t),
      t.setAttribute("aria-hidden", "true"),
      t.setAttribute("tabindex", -1),
      (t.onload = this.addResizeHandlers),
      (t.type = "text/html"),
      Ba && this.$el.appendChild(t),
      (t.data = "about:blank"),
      Ba || this.$el.appendChild(t);
  },
  beforeUnmount() {
    this.removeResizeHandlers();
  },
  methods: {
    compareAndNotify() {
      ((!this.ignoreWidth && this._w !== this.$el.offsetWidth) ||
        (!this.ignoreHeight && this._h !== this.$el.offsetHeight)) &&
        ((this._w = this.$el.offsetWidth), (this._h = this.$el.offsetHeight), this.emitSize());
    },
    emitSize() {
      this.$emit("notify", { width: this._w, height: this._h });
    },
    addResizeHandlers() {
      this._resizeObject.contentDocument.defaultView.addEventListener(
        "resize",
        this.compareAndNotify,
      ),
        this.compareAndNotify();
    },
    removeResizeHandlers() {
      this._resizeObject &&
        this._resizeObject.onload &&
        (!Ba &&
          this._resizeObject.contentDocument &&
          this._resizeObject.contentDocument.defaultView.removeEventListener(
            "resize",
            this.compareAndNotify,
          ),
        this.$el.removeChild(this._resizeObject),
        (this._resizeObject.onload = null),
        (this._resizeObject = null));
    },
  },
};
const kC = G1();
Qm("data-v-b329ee4c");
const CC = { class: "resize-observer", tabindex: "-1" };
t0();
const TC = kC((t, e, r, o, s, u) => (st(), te("div", CC)));
Bc.render = TC;
Bc.__scopeId = "data-v-b329ee4c";
Bc.__file = "src/components/ResizeObserver.vue";
var X0 = {
    computed: {
      themeClass() {
        return yC(this.theme);
      },
    },
  },
  Fh = (t, e) => {
    const r = t.__vccOpts || t;
    for (const [o, s] of e) r[o] = s;
    return r;
  };
const EC = ie({
    name: "VPopperContent",
    components: { ResizeObserver: Bc },
    mixins: [X0],
    props: {
      popperId: String,
      theme: String,
      shown: Boolean,
      mounted: Boolean,
      skipTransition: Boolean,
      autoHide: Boolean,
      handleResize: Boolean,
      classes: Object,
      result: Object,
    },
    emits: ["hide", "resize"],
    methods: {
      toPx(t) {
        return t != null && !isNaN(t) ? `${t}px` : null;
      },
    },
  }),
  LC = ["id", "aria-hidden", "tabindex", "data-popper-placement"],
  AC = { ref: "inner", class: "v-popper__inner" },
  MC = tt("div", { class: "v-popper__arrow-outer" }, null, -1),
  NC = tt("div", { class: "v-popper__arrow-inner" }, null, -1),
  PC = [MC, NC];
function OC(t, e, r, o, s, u) {
  const f = io("ResizeObserver");
  return (
    st(),
    kt(
      "div",
      {
        id: t.popperId,
        ref: "popover",
        class: ve([
          "v-popper__popper",
          [
            t.themeClass,
            t.classes.popperClass,
            {
              "v-popper__popper--shown": t.shown,
              "v-popper__popper--hidden": !t.shown,
              "v-popper__popper--show-from": t.classes.showFrom,
              "v-popper__popper--show-to": t.classes.showTo,
              "v-popper__popper--hide-from": t.classes.hideFrom,
              "v-popper__popper--hide-to": t.classes.hideTo,
              "v-popper__popper--skip-transition": t.skipTransition,
              "v-popper__popper--arrow-overflow": t.result.arrow.overflow,
            },
          ],
        ]),
        style: An({
          position: t.result.strategy,
          transform: `translate3d(${Math.round(t.result.x)}px,${Math.round(t.result.y)}px,0)`,
        }),
        "aria-hidden": t.shown ? "false" : "true",
        tabindex: t.autoHide ? 0 : void 0,
        "data-popper-placement": t.result.placement,
        onKeyup: e[1] || (e[1] = Df((h) => t.autoHide && t.$emit("hide"), ["esc"])),
      },
      [
        tt(
          "div",
          { class: "v-popper__wrapper", style: An({ transformOrigin: t.result.transformOrigin }) },
          [
            tt(
              "div",
              AC,
              [
                t.mounted
                  ? (st(),
                    kt(
                      ne,
                      { key: 0 },
                      [
                        tt("div", null, [sr(t.$slots, "default")]),
                        t.handleResize
                          ? (st(),
                            te(f, {
                              key: 0,
                              onNotify: e[0] || (e[0] = (h) => t.$emit("resize", h)),
                            }))
                          : Vt("", !0),
                      ],
                      64,
                    ))
                  : Vt("", !0),
              ],
              512,
            ),
            tt(
              "div",
              {
                ref: "arrow",
                class: "v-popper__arrow-container",
                style: An({ left: t.toPx(t.result.arrow.x), top: t.toPx(t.result.arrow.y) }),
              },
              PC,
              4,
            ),
          ],
          4,
        ),
      ],
      46,
      LC,
    )
  );
}
var Y0 = Fh(EC, [["render", OC]]),
  Z0 = {
    methods: {
      show(...t) {
        return this.$refs.popper.show(...t);
      },
      hide(...t) {
        return this.$refs.popper.hide(...t);
      },
      dispose(...t) {
        return this.$refs.popper.dispose(...t);
      },
      onResize(...t) {
        return this.$refs.popper.onResize(...t);
      },
    },
  };
const DC = ie({
  name: "VPopperWrapper",
  components: { Popper: G0(), PopperContent: Y0 },
  mixins: [Z0, X0],
  inheritAttrs: !1,
  props: { theme: { type: String, default: null } },
  computed: {
    finalTheme() {
      var t;
      return (t = this.theme) != null ? t : this.$options.vPopperTheme;
    },
    popperAttrs() {
      const t = zr({}, this.$attrs);
      return delete t.class, delete t.style, t;
    },
  },
  methods: {
    getTargetNodes() {
      return Array.from(this.$refs.reference.children).filter(
        (t) => t !== this.$refs.popperContent.$el,
      );
    },
  },
});
function $C(t, e, r, o, s, u) {
  const f = io("PopperContent"),
    h = io("Popper");
  return (
    st(),
    te(
      h,
      Ci({ ref: "popper" }, t.popperAttrs, {
        theme: t.finalTheme,
        "target-nodes": t.getTargetNodes,
        "reference-node": () => t.$refs.reference,
        "popper-node": () => t.$refs.popperContent.$el,
      }),
      {
        default: ee(
          ({
            popperId: d,
            isShown: g,
            shouldMountContent: v,
            skipTransition: b,
            autoHide: w,
            show: S,
            hide: P,
            handleResize: A,
            onResize: L,
            classes: T,
            result: M,
          }) => [
            tt(
              "div",
              {
                ref: "reference",
                class: ve(["v-popper", [t.$attrs.class, t.themeClass, { "v-popper--shown": g }]]),
                style: An(t.$attrs.style),
              },
              [
                sr(t.$slots, "default", { shown: g, show: S, hide: P }),
                Ft(
                  f,
                  {
                    ref: "popperContent",
                    "popper-id": d,
                    theme: t.finalTheme,
                    shown: g,
                    mounted: v,
                    "skip-transition": b,
                    "auto-hide": w,
                    "handle-resize": A,
                    classes: T,
                    result: M,
                    onHide: P,
                    onResize: L,
                  },
                  { default: ee(() => [sr(t.$slots, "popper", { shown: g, hide: P })]), _: 2 },
                  1032,
                  [
                    "popper-id",
                    "theme",
                    "shown",
                    "mounted",
                    "skip-transition",
                    "auto-hide",
                    "handle-resize",
                    "classes",
                    "result",
                    "onHide",
                    "onResize",
                  ],
                ),
              ],
              6,
            ),
          ],
        ),
        _: 3,
      },
      16,
      ["theme", "target-nodes", "reference-node", "popper-node"],
    )
  );
}
var qh = Fh(DC, [["render", $C]]);
const wv = ie($l(zr({}, qh), { name: "VDropdown", vPopperTheme: "dropdown" })),
  xv = ie($l(zr({}, qh), { name: "VMenu", vPopperTheme: "menu" })),
  Uf = ie($l(zr({}, qh), { name: "VTooltip", vPopperTheme: "tooltip" })),
  RC = ie({
    name: "VTooltipDirective",
    components: { Popper: G0(), PopperContent: Y0 },
    mixins: [Z0],
    inheritAttrs: !1,
    props: {
      theme: { type: String, default: "tooltip" },
      html: { type: Boolean, default: (t) => cs(t.theme, "html") },
      content: { type: [String, Number, Function], default: null },
      loadingContent: { type: String, default: (t) => cs(t.theme, "loadingContent") },
    },
    data() {
      return { asyncContent: null };
    },
    computed: {
      isContentAsync() {
        return typeof this.content == "function";
      },
      loading() {
        return this.isContentAsync && this.asyncContent == null;
      },
      finalContent() {
        return this.isContentAsync
          ? this.loading
            ? this.loadingContent
            : this.asyncContent
          : this.content;
      },
    },
    watch: {
      content: {
        handler() {
          this.fetchContent(!0);
        },
        immediate: !0,
      },
      async finalContent() {
        await this.$nextTick(), this.$refs.popper.onResize();
      },
    },
    created() {
      this.$_fetchId = 0;
    },
    methods: {
      fetchContent(t) {
        if (
          typeof this.content == "function" &&
          this.$_isShown &&
          (t || (!this.$_loading && this.asyncContent == null))
        ) {
          (this.asyncContent = null), (this.$_loading = !0);
          const e = ++this.$_fetchId,
            r = this.content(this);
          r.then ? r.then((o) => this.onResult(e, o)) : this.onResult(e, r);
        }
      },
      onResult(t, e) {
        t === this.$_fetchId && ((this.$_loading = !1), (this.asyncContent = e));
      },
      onShow() {
        (this.$_isShown = !0), this.fetchContent();
      },
      onHide() {
        this.$_isShown = !1;
      },
    },
  }),
  zC = ["innerHTML"],
  IC = ["textContent"];
function FC(t, e, r, o, s, u) {
  const f = io("PopperContent"),
    h = io("Popper");
  return (
    st(),
    te(
      h,
      Ci({ ref: "popper" }, t.$attrs, {
        theme: t.theme,
        "popper-node": () => t.$refs.popperContent.$el,
        onApplyShow: t.onShow,
        onApplyHide: t.onHide,
      }),
      {
        default: ee(
          ({
            popperId: d,
            isShown: g,
            shouldMountContent: v,
            skipTransition: b,
            autoHide: w,
            hide: S,
            handleResize: P,
            onResize: A,
            classes: L,
            result: T,
          }) => [
            Ft(
              f,
              {
                ref: "popperContent",
                class: ve({ "v-popper--tooltip-loading": t.loading }),
                "popper-id": d,
                theme: t.theme,
                shown: g,
                mounted: v,
                "skip-transition": b,
                "auto-hide": w,
                "handle-resize": P,
                classes: L,
                result: T,
                onHide: S,
                onResize: A,
              },
              {
                default: ee(() => [
                  t.html
                    ? (st(), kt("div", { key: 0, innerHTML: t.finalContent }, null, 8, zC))
                    : (st(), kt("div", { key: 1, textContent: Ut(t.finalContent) }, null, 8, IC)),
                ]),
                _: 2,
              },
              1032,
              [
                "class",
                "popper-id",
                "theme",
                "shown",
                "mounted",
                "skip-transition",
                "auto-hide",
                "handle-resize",
                "classes",
                "result",
                "onHide",
                "onResize",
              ],
            ),
          ],
        ),
        _: 1,
      },
      16,
      ["theme", "popper-node", "onApplyShow", "onApplyHide"],
    )
  );
}
var qC = Fh(RC, [["render", FC]]);
const J0 = "v-popper--has-tooltip";
function HC(t, e) {
  let r = t.placement;
  if (!r && e) for (const o of V0) e[o] && (r = o);
  return r || (r = cs(t.theme || "tooltip", "placement")), r;
}
function Q0(t, e, r) {
  let o;
  const s = typeof e;
  return (
    s === "string" ? (o = { content: e }) : e && s === "object" ? (o = e) : (o = { content: !1 }),
    (o.placement = HC(o, r)),
    (o.targetNodes = () => [t]),
    (o.referenceNode = () => t),
    o
  );
}
let of,
  _l,
  BC = 0;
function WC() {
  if (of) return;
  (_l = Zt([])),
    (of = T0({
      name: "VTooltipDirectiveApp",
      setup() {
        return { directives: _l };
      },
      render() {
        return this.directives.map((e) =>
          Ol(qC, $l(zr({}, e.options), { shown: e.shown.value || e.options.shown, key: e.id })),
        );
      },
      devtools: { hide: !0 },
    }));
  const t = document.createElement("div");
  document.body.appendChild(t), of.mount(t);
}
function ty(t, e, r) {
  WC();
  const o = Zt(Q0(t, e, r)),
    s = Zt(!1),
    u = { id: BC++, options: o, shown: s };
  return (
    _l.value.push(u),
    t.classList && t.classList.add(J0),
    (t.$_popper = {
      options: o,
      item: u,
      show() {
        s.value = !0;
      },
      hide() {
        s.value = !1;
      },
    })
  );
}
function Hh(t) {
  if (t.$_popper) {
    const e = _l.value.indexOf(t.$_popper.item);
    e !== -1 && _l.value.splice(e, 1),
      delete t.$_popper,
      delete t.$_popperOldShown,
      delete t.$_popperMountTarget;
  }
  t.classList && t.classList.remove(J0);
}
function _v(t, { value: e, oldValue: r, modifiers: o }) {
  const s = Q0(t, e, o);
  if (!s.content || cs(s.theme || "tooltip", "disabled")) Hh(t);
  else {
    let u;
    t.$_popper ? ((u = t.$_popper), (u.options.value = s)) : (u = ty(t, e, o)),
      typeof e.shown < "u" &&
        e.shown !== t.$_popperOldShown &&
        ((t.$_popperOldShown = e.shown), e.shown ? u.show() : u.hide());
  }
}
var ey = {
  beforeMount: _v,
  updated: _v,
  beforeUnmount(t) {
    Hh(t);
  },
};
function Sv(t) {
  t.addEventListener("click", ny), t.addEventListener("touchstart", ry, us ? { passive: !0 } : !1);
}
function kv(t) {
  t.removeEventListener("click", ny),
    t.removeEventListener("touchstart", ry),
    t.removeEventListener("touchend", iy),
    t.removeEventListener("touchcancel", oy);
}
function ny(t) {
  const e = t.currentTarget;
  (t.closePopover = !e.$_vclosepopover_touch),
    (t.closeAllPopover = e.$_closePopoverModifiers && !!e.$_closePopoverModifiers.all);
}
function ry(t) {
  if (t.changedTouches.length === 1) {
    const e = t.currentTarget;
    e.$_vclosepopover_touch = !0;
    const r = t.changedTouches[0];
    (e.$_vclosepopover_touchPoint = r),
      e.addEventListener("touchend", iy),
      e.addEventListener("touchcancel", oy);
  }
}
function iy(t) {
  const e = t.currentTarget;
  if (((e.$_vclosepopover_touch = !1), t.changedTouches.length === 1)) {
    const r = t.changedTouches[0],
      o = e.$_vclosepopover_touchPoint;
    (t.closePopover = Math.abs(r.screenY - o.screenY) < 20 && Math.abs(r.screenX - o.screenX) < 20),
      (t.closeAllPopover = e.$_closePopoverModifiers && !!e.$_closePopoverModifiers.all);
  }
}
function oy(t) {
  const e = t.currentTarget;
  e.$_vclosepopover_touch = !1;
}
var UC = {
  beforeMount(t, { value: e, modifiers: r }) {
    (t.$_closePopoverModifiers = r), (typeof e > "u" || e) && Sv(t);
  },
  updated(t, { value: e, oldValue: r, modifiers: o }) {
    (t.$_closePopoverModifiers = o), e !== r && (typeof e > "u" || e ? Sv(t) : kv(t));
  },
  beforeUnmount(t) {
    kv(t);
  },
};
const jC = ey,
  VC = Uf;
function GC(t, e = {}) {
  t.$_vTooltipInstalled ||
    ((t.$_vTooltipInstalled = !0),
    U0(eo, e),
    t.directive("tooltip", ey),
    t.directive("close-popper", UC),
    t.component("v-tooltip", Uf),
    t.component("VTooltip", Uf),
    t.component("v-dropdown", wv),
    t.component("VDropdown", wv),
    t.component("v-menu", xv),
    t.component("VMenu", xv));
}
const sy = { version: "2.0.0-y.0", install: GC, options: eo },
  KC = 6e4;
function ly(t) {
  return t;
}
const XC = ly,
  { clearTimeout: YC, setTimeout: ZC } = globalThis,
  JC = Math.random.bind(Math);
function QC(t, e) {
  const {
      post: r,
      on: o,
      eventNames: s = [],
      serialize: u = ly,
      deserialize: f = XC,
      resolver: h,
      timeout: d = KC,
    } = e,
    g = new Map();
  let v;
  const b = new Proxy(
    {},
    {
      get(w, S) {
        if (S === "$functions") return t;
        const P = (...L) => {
          r(u({ m: S, a: L, t: "q" }));
        };
        if (s.includes(S)) return (P.asEvent = P), P;
        const A = async (...L) => (
          await v,
          new Promise((T, M) => {
            var B, K;
            const R = eT();
            let E;
            d >= 0 &&
              (E =
                (K = (B = ZC(() => {
                  M(new Error(`[birpc] timeout on calling "${S}"`)), g.delete(R);
                }, d)).unref) == null
                  ? void 0
                  : K.call(B)),
              g.set(R, { resolve: T, reject: M, timeoutId: E }),
              r(u({ m: S, a: L, i: R, t: "q" }));
          })
        );
        return (A.asEvent = P), A;
      },
    },
  );
  return (
    (v = o(async (w, ...S) => {
      const P = f(w);
      if (P.t === "q") {
        const { m: A, a: L } = P;
        let T, M;
        const R = h ? h(A, t[A]) : t[A];
        if (!R) M = new Error(`[birpc] function "${A}" not found`);
        else
          try {
            T = await R.apply(b, L);
          } catch (E) {
            M = E;
          }
        P.i && (M && e.onError && e.onError(M, A, L), r(u({ t: "s", i: P.i, r: T, e: M }), ...S));
      } else {
        const { i: A, r: L, e: T } = P,
          M = g.get(A);
        M && (YC(M.timeoutId), T ? M.reject(T) : M.resolve(L)), g.delete(A);
      }
    })),
    b
  );
}
const tT = "useandom-26T198340PX75pxJACKVERYMINDBUSHWOLF_GQZbfghjklqvwyzrict";
function eT(t = 21) {
  let e = "",
    r = t;
  for (; r--; ) e += tT[(JC() * 64) | 0];
  return e;
} /*! (c) 2020 Andrea Giammarchi */
const { parse: nT, stringify: rT } = JSON,
  { keys: iT } = Object,
  Sl = String,
  ay = "string",
  Cv = {},
  cc = "object",
  cy = (t, e) => e,
  oT = (t) => (t instanceof Sl ? Sl(t) : t),
  sT = (t, e) => (typeof e === ay ? new Sl(e) : e),
  uy = (t, e, r, o) => {
    const s = [];
    for (let u = iT(r), { length: f } = u, h = 0; h < f; h++) {
      const d = u[h],
        g = r[d];
      if (g instanceof Sl) {
        const v = t[g];
        typeof v === cc && !e.has(v)
          ? (e.add(v), (r[d] = Cv), s.push({ k: d, a: [t, e, v, o] }))
          : (r[d] = o.call(r, d, v));
      } else r[d] !== Cv && (r[d] = o.call(r, d, g));
    }
    for (let { length: u } = s, f = 0; f < u; f++) {
      const { k: h, a: d } = s[f];
      r[h] = o.call(r, h, uy.apply(null, d));
    }
    return r;
  },
  Tv = (t, e, r) => {
    const o = Sl(e.push(r) - 1);
    return t.set(r, o), o;
  },
  jf = (t, e) => {
    const r = nT(t, sT).map(oT),
      o = r[0],
      s = e || cy,
      u = typeof o === cc && o ? uy(r, new Set(), o, s) : o;
    return s.call({ "": u }, "", u);
  },
  lT = (t, e, r) => {
    const o =
        e && typeof e === cc ? (v, b) => (v === "" || -1 < e.indexOf(v) ? b : void 0) : e || cy,
      s = new Map(),
      u = [],
      f = [];
    let h = +Tv(s, u, o.call({ "": t }, "", t)),
      d = !h;
    for (; h < u.length; ) (d = !0), (f[h] = rT(u[h++], g, r));
    return "[" + f.join(",") + "]";
    function g(v, b) {
      if (d) return (d = !d), b;
      const w = o.call(this, v, b);
      switch (typeof w) {
        case cc:
          if (w === null) return w;
        case ay:
          return s.get(w) || Tv(s, u, w);
      }
      return w;
    }
  };
function aT(t = "") {
  return !t || !t.includes("\\") ? t : t.replace(/\\/g, "/");
}
const cT = /^[/\\](?![/\\])|^[/\\]{2}(?!\.)|^[A-Za-z]:[/\\]/;
function uT() {
  return typeof process < "u" ? process.cwd().replace(/\\/g, "/") : "/";
}
const Ev = function (...t) {
  t = t.map((o) => aT(o));
  let e = "",
    r = !1;
  for (let o = t.length - 1; o >= -1 && !r; o--) {
    const s = o >= 0 ? t[o] : uT();
    !s || s.length === 0 || ((e = `${s}/${e}`), (r = Lv(s)));
  }
  return (e = fT(e, !r)), r && !Lv(e) ? `/${e}` : e.length > 0 ? e : ".";
};
function fT(t, e) {
  let r = "",
    o = 0,
    s = -1,
    u = 0,
    f = null;
  for (let h = 0; h <= t.length; ++h) {
    if (h < t.length) f = t[h];
    else {
      if (f === "/") break;
      f = "/";
    }
    if (f === "/") {
      if (!(s === h - 1 || u === 1))
        if (u === 2) {
          if (r.length < 2 || o !== 2 || r[r.length - 1] !== "." || r[r.length - 2] !== ".") {
            if (r.length > 2) {
              const d = r.lastIndexOf("/");
              d === -1
                ? ((r = ""), (o = 0))
                : ((r = r.slice(0, d)), (o = r.length - 1 - r.lastIndexOf("/"))),
                (s = h),
                (u = 0);
              continue;
            } else if (r.length > 0) {
              (r = ""), (o = 0), (s = h), (u = 0);
              continue;
            }
          }
          e && ((r += r.length > 0 ? "/.." : ".."), (o = 2));
        } else
          r.length > 0 ? (r += `/${t.slice(s + 1, h)}`) : (r = t.slice(s + 1, h)), (o = h - s - 1);
      (s = h), (u = 0);
    } else f === "." && u !== -1 ? ++u : (u = -1);
  }
  return r;
}
const Lv = function (t) {
    return cT.test(t);
  },
  hT = function (t, e) {
    const r = Ev(t).split("/"),
      o = Ev(e).split("/"),
      s = [...r];
    for (const u of s) {
      if (o[0] !== u) break;
      r.shift(), o.shift();
    }
    return [...r.map(() => ".."), ...o].join("/");
  };
function dT(t) {
  return typeof AggregateError < "u" && t instanceof AggregateError
    ? !0
    : t instanceof Error && "errors" in t;
}
class fy {
  constructor() {
    ci(this, "filesMap", new Map());
    ci(this, "pathsSet", new Set());
    ci(this, "browserTestPromises", new Map());
    ci(this, "idMap", new Map());
    ci(this, "taskFileMap", new WeakMap());
    ci(this, "errorsSet", new Set());
    ci(this, "processTimeoutCauses", new Set());
  }
  catchError(e, r) {
    if (dT(e)) return e.errors.forEach((s) => this.catchError(s, r));
    e === Object(e) ? (e.type = r) : (e = { type: r, message: e });
    const o = e;
    if (o && typeof o == "object" && o.code === "VITEST_PENDING") {
      const s = this.idMap.get(o.taskId);
      s &&
        ((s.mode = "skip"), s.result ?? (s.result = { state: "skip" }), (s.result.state = "skip"));
      return;
    }
    this.errorsSet.add(e);
  }
  clearErrors() {
    this.errorsSet.clear();
  }
  getUnhandledErrors() {
    return Array.from(this.errorsSet.values());
  }
  addProcessTimeoutCause(e) {
    this.processTimeoutCauses.add(e);
  }
  getProcessTimeoutCauses() {
    return Array.from(this.processTimeoutCauses.values());
  }
  getPaths() {
    return Array.from(this.pathsSet);
  }
  getFiles(e) {
    return e
      ? e
          .map((r) => this.filesMap.get(r))
          .filter(Boolean)
          .flat()
      : Array.from(this.filesMap.values()).flat();
  }
  getFilepaths() {
    return Array.from(this.filesMap.keys());
  }
  getFailedFilepaths() {
    return this.getFiles()
      .filter((e) => {
        var r;
        return ((r = e.result) == null ? void 0 : r.state) === "fail";
      })
      .map((e) => e.filepath);
  }
  collectPaths(e = []) {
    e.forEach((r) => {
      this.pathsSet.add(r);
    });
  }
  collectFiles(e = []) {
    e.forEach((r) => {
      const s = (this.filesMap.get(r.filepath) || []).filter(
        (u) => u.projectName !== r.projectName,
      );
      s.push(r), this.filesMap.set(r.filepath, s), this.updateId(r);
    });
  }
  clearFiles(e, r = []) {
    const o = e;
    r.forEach((s) => {
      const u = this.filesMap.get(s);
      if (!u) return;
      const f = u.filter((h) => h.projectName !== o.config.name);
      f.length ? this.filesMap.set(s, f) : this.filesMap.delete(s);
    });
  }
  updateId(e) {
    this.idMap.get(e.id) !== e &&
      (this.idMap.set(e.id, e),
      e.type === "suite" &&
        e.tasks.forEach((r) => {
          this.updateId(r);
        }));
  }
  updateTasks(e) {
    for (const [r, o, s] of e) {
      const u = this.idMap.get(r);
      u &&
        ((u.result = o),
        (u.meta = s),
        (o == null ? void 0 : o.state) === "skip" && (u.mode = "skip"));
    }
  }
  updateUserLog(e) {
    const r = e.taskId && this.idMap.get(e.taskId);
    r && (r.logs || (r.logs = []), r.logs.push(e));
  }
  getCountOfFailedTests() {
    return Array.from(this.idMap.values()).filter((e) => {
      var r;
      return ((r = e.result) == null ? void 0 : r.state) === "fail";
    }).length;
  }
  cancelFiles(e, r, o) {
    this.collectFiles(
      e.map((s) => ({
        filepath: s,
        name: hT(r, s),
        id: s,
        mode: "skip",
        type: "suite",
        result: { state: "skip" },
        meta: {},
        tasks: [],
        projectName: o,
      })),
    );
  }
}
var oo =
  typeof globalThis < "u"
    ? globalThis
    : typeof window < "u"
    ? window
    : typeof global < "u"
    ? global
    : typeof self < "u"
    ? self
    : {};
function hy(t) {
  return t && t.__esModule && Object.prototype.hasOwnProperty.call(t, "default") ? t.default : t;
}
function dy(t) {
  return t != null;
}
function py(t) {
  return t == null && (t = []), Array.isArray(t) ? t : [t];
}
function Vf(t) {
  return t.type === "test" || t.type === "custom";
}
function gy(t) {
  const e = [],
    r = py(t);
  for (const o of r)
    if (Vf(o)) e.push(o);
    else for (const s of o.tasks) Vf(s) ? e.push(s) : e.push(...gy(s));
  return e;
}
function Bh(t = []) {
  return py(t).flatMap((e) => (Vf(e) ? [e] : [e, ...Bh(e.tasks)]));
}
function pT(t) {
  const e = [t.name];
  let r = t;
  for (; (r != null && r.suite) || (r != null && r.file); )
    (r = r.suite || r.file), r != null && r.name && e.unshift(r.name);
  return e;
}
function Wc(t) {
  return gy(t).some((e) => {
    var r, o;
    return (o = (r = e.result) == null ? void 0 : r.errors) == null
      ? void 0
      : o.some(
          (s) =>
            typeof (s == null ? void 0 : s.message) == "string" &&
            s.message.match(/Snapshot .* mismatched/),
        );
  });
}
function gT(t, e = {}) {
  const {
    handlers: r = {},
    autoReconnect: o = !0,
    reconnectInterval: s = 2e3,
    reconnectTries: u = 10,
    reactive: f = (T) => T,
    WebSocketConstructor: h = globalThis.WebSocket,
  } = e;
  let d = u;
  const g = f({ ws: new h(t), state: new fy(), waitForConnection: L, reconnect: P });
  (g.state.filesMap = f(g.state.filesMap)), (g.state.idMap = f(g.state.idMap));
  let v;
  const b = {
      onPathsCollected(T) {
        var M;
        g.state.collectPaths(T), (M = r.onPathsCollected) == null || M.call(r, T);
      },
      onCollected(T) {
        var M;
        g.state.collectFiles(T), (M = r.onCollected) == null || M.call(r, T);
      },
      onTaskUpdate(T) {
        var M;
        g.state.updateTasks(T), (M = r.onTaskUpdate) == null || M.call(r, T);
      },
      onUserConsoleLog(T) {
        g.state.updateUserLog(T);
      },
      onFinished(T, M) {
        var R;
        (R = r.onFinished) == null || R.call(r, T, M);
      },
      onCancel(T) {
        var M;
        (M = r.onCancel) == null || M.call(r, T);
      },
    },
    w = { post: (T) => g.ws.send(T), on: (T) => (v = T), serialize: lT, deserialize: jf };
  g.rpc = QC(b, w);
  let S;
  function P(T = !1) {
    T && (d = u), (g.ws = new h(t)), A();
  }
  function A() {
    (S = new Promise((T) => {
      g.ws.addEventListener("open", () => {
        (d = u), T();
      });
    })),
      g.ws.addEventListener("message", (T) => {
        v(T.data);
      }),
      g.ws.addEventListener("close", () => {
        (d -= 1), o && d > 0 && setTimeout(P, s);
      });
  }
  A();
  function L() {
    return S;
  }
  return g;
}
const vT = location.port,
  mT = [location.hostname, vT].filter(Boolean).join(":"),
  yT = `${location.protocol === "https:" ? "wss:" : "ws:"}//${mT}/__vitest_api__`,
  jr = !!window.METADATA_PATH;
var vy = {},
  Hr = {};
const bT = "Á",
  wT = "á",
  xT = "Ă",
  _T = "ă",
  ST = "∾",
  kT = "∿",
  CT = "∾̳",
  TT = "Â",
  ET = "â",
  LT = "´",
  AT = "А",
  MT = "а",
  NT = "Æ",
  PT = "æ",
  OT = "⁡",
  DT = "𝔄",
  $T = "𝔞",
  RT = "À",
  zT = "à",
  IT = "ℵ",
  FT = "ℵ",
  qT = "Α",
  HT = "α",
  BT = "Ā",
  WT = "ā",
  UT = "⨿",
  jT = "&",
  VT = "&",
  GT = "⩕",
  KT = "⩓",
  XT = "∧",
  YT = "⩜",
  ZT = "⩘",
  JT = "⩚",
  QT = "∠",
  tE = "⦤",
  eE = "∠",
  nE = "⦨",
  rE = "⦩",
  iE = "⦪",
  oE = "⦫",
  sE = "⦬",
  lE = "⦭",
  aE = "⦮",
  cE = "⦯",
  uE = "∡",
  fE = "∟",
  hE = "⊾",
  dE = "⦝",
  pE = "∢",
  gE = "Å",
  vE = "⍼",
  mE = "Ą",
  yE = "ą",
  bE = "𝔸",
  wE = "𝕒",
  xE = "⩯",
  _E = "≈",
  SE = "⩰",
  kE = "≊",
  CE = "≋",
  TE = "'",
  EE = "⁡",
  LE = "≈",
  AE = "≊",
  ME = "Å",
  NE = "å",
  PE = "𝒜",
  OE = "𝒶",
  DE = "≔",
  $E = "*",
  RE = "≈",
  zE = "≍",
  IE = "Ã",
  FE = "ã",
  qE = "Ä",
  HE = "ä",
  BE = "∳",
  WE = "⨑",
  UE = "≌",
  jE = "϶",
  VE = "‵",
  GE = "∽",
  KE = "⋍",
  XE = "∖",
  YE = "⫧",
  ZE = "⊽",
  JE = "⌅",
  QE = "⌆",
  tL = "⌅",
  eL = "⎵",
  nL = "⎶",
  rL = "≌",
  iL = "Б",
  oL = "б",
  sL = "„",
  lL = "∵",
  aL = "∵",
  cL = "∵",
  uL = "⦰",
  fL = "϶",
  hL = "ℬ",
  dL = "ℬ",
  pL = "Β",
  gL = "β",
  vL = "ℶ",
  mL = "≬",
  yL = "𝔅",
  bL = "𝔟",
  wL = "⋂",
  xL = "◯",
  _L = "⋃",
  SL = "⨀",
  kL = "⨁",
  CL = "⨂",
  TL = "⨆",
  EL = "★",
  LL = "▽",
  AL = "△",
  ML = "⨄",
  NL = "⋁",
  PL = "⋀",
  OL = "⤍",
  DL = "⧫",
  $L = "▪",
  RL = "▴",
  zL = "▾",
  IL = "◂",
  FL = "▸",
  qL = "␣",
  HL = "▒",
  BL = "░",
  WL = "▓",
  UL = "█",
  jL = "=⃥",
  VL = "≡⃥",
  GL = "⫭",
  KL = "⌐",
  XL = "𝔹",
  YL = "𝕓",
  ZL = "⊥",
  JL = "⊥",
  QL = "⋈",
  tA = "⧉",
  eA = "┐",
  nA = "╕",
  rA = "╖",
  iA = "╗",
  oA = "┌",
  sA = "╒",
  lA = "╓",
  aA = "╔",
  cA = "─",
  uA = "═",
  fA = "┬",
  hA = "╤",
  dA = "╥",
  pA = "╦",
  gA = "┴",
  vA = "╧",
  mA = "╨",
  yA = "╩",
  bA = "⊟",
  wA = "⊞",
  xA = "⊠",
  _A = "┘",
  SA = "╛",
  kA = "╜",
  CA = "╝",
  TA = "└",
  EA = "╘",
  LA = "╙",
  AA = "╚",
  MA = "│",
  NA = "║",
  PA = "┼",
  OA = "╪",
  DA = "╫",
  $A = "╬",
  RA = "┤",
  zA = "╡",
  IA = "╢",
  FA = "╣",
  qA = "├",
  HA = "╞",
  BA = "╟",
  WA = "╠",
  UA = "‵",
  jA = "˘",
  VA = "˘",
  GA = "¦",
  KA = "𝒷",
  XA = "ℬ",
  YA = "⁏",
  ZA = "∽",
  JA = "⋍",
  QA = "⧅",
  tM = "\\",
  eM = "⟈",
  nM = "•",
  rM = "•",
  iM = "≎",
  oM = "⪮",
  sM = "≏",
  lM = "≎",
  aM = "≏",
  cM = "Ć",
  uM = "ć",
  fM = "⩄",
  hM = "⩉",
  dM = "⩋",
  pM = "∩",
  gM = "⋒",
  vM = "⩇",
  mM = "⩀",
  yM = "ⅅ",
  bM = "∩︀",
  wM = "⁁",
  xM = "ˇ",
  _M = "ℭ",
  SM = "⩍",
  kM = "Č",
  CM = "č",
  TM = "Ç",
  EM = "ç",
  LM = "Ĉ",
  AM = "ĉ",
  MM = "∰",
  NM = "⩌",
  PM = "⩐",
  OM = "Ċ",
  DM = "ċ",
  $M = "¸",
  RM = "¸",
  zM = "⦲",
  IM = "¢",
  FM = "·",
  qM = "·",
  HM = "𝔠",
  BM = "ℭ",
  WM = "Ч",
  UM = "ч",
  jM = "✓",
  VM = "✓",
  GM = "Χ",
  KM = "χ",
  XM = "ˆ",
  YM = "≗",
  ZM = "↺",
  JM = "↻",
  QM = "⊛",
  tN = "⊚",
  eN = "⊝",
  nN = "⊙",
  rN = "®",
  iN = "Ⓢ",
  oN = "⊖",
  sN = "⊕",
  lN = "⊗",
  aN = "○",
  cN = "⧃",
  uN = "≗",
  fN = "⨐",
  hN = "⫯",
  dN = "⧂",
  pN = "∲",
  gN = "”",
  vN = "’",
  mN = "♣",
  yN = "♣",
  bN = ":",
  wN = "∷",
  xN = "⩴",
  _N = "≔",
  SN = "≔",
  kN = ",",
  CN = "@",
  TN = "∁",
  EN = "∘",
  LN = "∁",
  AN = "ℂ",
  MN = "≅",
  NN = "⩭",
  PN = "≡",
  ON = "∮",
  DN = "∯",
  $N = "∮",
  RN = "𝕔",
  zN = "ℂ",
  IN = "∐",
  FN = "∐",
  qN = "©",
  HN = "©",
  BN = "℗",
  WN = "∳",
  UN = "↵",
  jN = "✗",
  VN = "⨯",
  GN = "𝒞",
  KN = "𝒸",
  XN = "⫏",
  YN = "⫑",
  ZN = "⫐",
  JN = "⫒",
  QN = "⋯",
  tP = "⤸",
  eP = "⤵",
  nP = "⋞",
  rP = "⋟",
  iP = "↶",
  oP = "⤽",
  sP = "⩈",
  lP = "⩆",
  aP = "≍",
  cP = "∪",
  uP = "⋓",
  fP = "⩊",
  hP = "⊍",
  dP = "⩅",
  pP = "∪︀",
  gP = "↷",
  vP = "⤼",
  mP = "⋞",
  yP = "⋟",
  bP = "⋎",
  wP = "⋏",
  xP = "¤",
  _P = "↶",
  SP = "↷",
  kP = "⋎",
  CP = "⋏",
  TP = "∲",
  EP = "∱",
  LP = "⌭",
  AP = "†",
  MP = "‡",
  NP = "ℸ",
  PP = "↓",
  OP = "↡",
  DP = "⇓",
  $P = "‐",
  RP = "⫤",
  zP = "⊣",
  IP = "⤏",
  FP = "˝",
  qP = "Ď",
  HP = "ď",
  BP = "Д",
  WP = "д",
  UP = "‡",
  jP = "⇊",
  VP = "ⅅ",
  GP = "ⅆ",
  KP = "⤑",
  XP = "⩷",
  YP = "°",
  ZP = "∇",
  JP = "Δ",
  QP = "δ",
  tO = "⦱",
  eO = "⥿",
  nO = "𝔇",
  rO = "𝔡",
  iO = "⥥",
  oO = "⇃",
  sO = "⇂",
  lO = "´",
  aO = "˙",
  cO = "˝",
  uO = "`",
  fO = "˜",
  hO = "⋄",
  dO = "⋄",
  pO = "⋄",
  gO = "♦",
  vO = "♦",
  mO = "¨",
  yO = "ⅆ",
  bO = "ϝ",
  wO = "⋲",
  xO = "÷",
  _O = "÷",
  SO = "⋇",
  kO = "⋇",
  CO = "Ђ",
  TO = "ђ",
  EO = "⌞",
  LO = "⌍",
  AO = "$",
  MO = "𝔻",
  NO = "𝕕",
  PO = "¨",
  OO = "˙",
  DO = "⃜",
  $O = "≐",
  RO = "≑",
  zO = "≐",
  IO = "∸",
  FO = "∔",
  qO = "⊡",
  HO = "⌆",
  BO = "∯",
  WO = "¨",
  UO = "⇓",
  jO = "⇐",
  VO = "⇔",
  GO = "⫤",
  KO = "⟸",
  XO = "⟺",
  YO = "⟹",
  ZO = "⇒",
  JO = "⊨",
  QO = "⇑",
  tD = "⇕",
  eD = "∥",
  nD = "⤓",
  rD = "↓",
  iD = "↓",
  oD = "⇓",
  sD = "⇵",
  lD = "̑",
  aD = "⇊",
  cD = "⇃",
  uD = "⇂",
  fD = "⥐",
  hD = "⥞",
  dD = "⥖",
  pD = "↽",
  gD = "⥟",
  vD = "⥗",
  mD = "⇁",
  yD = "↧",
  bD = "⊤",
  wD = "⤐",
  xD = "⌟",
  _D = "⌌",
  SD = "𝒟",
  kD = "𝒹",
  CD = "Ѕ",
  TD = "ѕ",
  ED = "⧶",
  LD = "Đ",
  AD = "đ",
  MD = "⋱",
  ND = "▿",
  PD = "▾",
  OD = "⇵",
  DD = "⥯",
  $D = "⦦",
  RD = "Џ",
  zD = "џ",
  ID = "⟿",
  FD = "É",
  qD = "é",
  HD = "⩮",
  BD = "Ě",
  WD = "ě",
  UD = "Ê",
  jD = "ê",
  VD = "≖",
  GD = "≕",
  KD = "Э",
  XD = "э",
  YD = "⩷",
  ZD = "Ė",
  JD = "ė",
  QD = "≑",
  t$ = "ⅇ",
  e$ = "≒",
  n$ = "𝔈",
  r$ = "𝔢",
  i$ = "⪚",
  o$ = "È",
  s$ = "è",
  l$ = "⪖",
  a$ = "⪘",
  c$ = "⪙",
  u$ = "∈",
  f$ = "⏧",
  h$ = "ℓ",
  d$ = "⪕",
  p$ = "⪗",
  g$ = "Ē",
  v$ = "ē",
  m$ = "∅",
  y$ = "∅",
  b$ = "◻",
  w$ = "∅",
  x$ = "▫",
  _$ = " ",
  S$ = " ",
  k$ = " ",
  C$ = "Ŋ",
  T$ = "ŋ",
  E$ = " ",
  L$ = "Ę",
  A$ = "ę",
  M$ = "𝔼",
  N$ = "𝕖",
  P$ = "⋕",
  O$ = "⧣",
  D$ = "⩱",
  $$ = "ε",
  R$ = "Ε",
  z$ = "ε",
  I$ = "ϵ",
  F$ = "≖",
  q$ = "≕",
  H$ = "≂",
  B$ = "⪖",
  W$ = "⪕",
  U$ = "⩵",
  j$ = "=",
  V$ = "≂",
  G$ = "≟",
  K$ = "⇌",
  X$ = "≡",
  Y$ = "⩸",
  Z$ = "⧥",
  J$ = "⥱",
  Q$ = "≓",
  tR = "ℯ",
  eR = "ℰ",
  nR = "≐",
  rR = "⩳",
  iR = "≂",
  oR = "Η",
  sR = "η",
  lR = "Ð",
  aR = "ð",
  cR = "Ë",
  uR = "ë",
  fR = "€",
  hR = "!",
  dR = "∃",
  pR = "∃",
  gR = "ℰ",
  vR = "ⅇ",
  mR = "ⅇ",
  yR = "≒",
  bR = "Ф",
  wR = "ф",
  xR = "♀",
  _R = "ﬃ",
  SR = "ﬀ",
  kR = "ﬄ",
  CR = "𝔉",
  TR = "𝔣",
  ER = "ﬁ",
  LR = "◼",
  AR = "▪",
  MR = "fj",
  NR = "♭",
  PR = "ﬂ",
  OR = "▱",
  DR = "ƒ",
  $R = "𝔽",
  RR = "𝕗",
  zR = "∀",
  IR = "∀",
  FR = "⋔",
  qR = "⫙",
  HR = "ℱ",
  BR = "⨍",
  WR = "½",
  UR = "⅓",
  jR = "¼",
  VR = "⅕",
  GR = "⅙",
  KR = "⅛",
  XR = "⅔",
  YR = "⅖",
  ZR = "¾",
  JR = "⅗",
  QR = "⅜",
  t2 = "⅘",
  e2 = "⅚",
  n2 = "⅝",
  r2 = "⅞",
  i2 = "⁄",
  o2 = "⌢",
  s2 = "𝒻",
  l2 = "ℱ",
  a2 = "ǵ",
  c2 = "Γ",
  u2 = "γ",
  f2 = "Ϝ",
  h2 = "ϝ",
  d2 = "⪆",
  p2 = "Ğ",
  g2 = "ğ",
  v2 = "Ģ",
  m2 = "Ĝ",
  y2 = "ĝ",
  b2 = "Г",
  w2 = "г",
  x2 = "Ġ",
  _2 = "ġ",
  S2 = "≥",
  k2 = "≧",
  C2 = "⪌",
  T2 = "⋛",
  E2 = "≥",
  L2 = "≧",
  A2 = "⩾",
  M2 = "⪩",
  N2 = "⩾",
  P2 = "⪀",
  O2 = "⪂",
  D2 = "⪄",
  $2 = "⋛︀",
  R2 = "⪔",
  z2 = "𝔊",
  I2 = "𝔤",
  F2 = "≫",
  q2 = "⋙",
  H2 = "⋙",
  B2 = "ℷ",
  W2 = "Ѓ",
  U2 = "ѓ",
  j2 = "⪥",
  V2 = "≷",
  G2 = "⪒",
  K2 = "⪤",
  X2 = "⪊",
  Y2 = "⪊",
  Z2 = "⪈",
  J2 = "≩",
  Q2 = "⪈",
  tz = "≩",
  ez = "⋧",
  nz = "𝔾",
  rz = "𝕘",
  iz = "`",
  oz = "≥",
  sz = "⋛",
  lz = "≧",
  az = "⪢",
  cz = "≷",
  uz = "⩾",
  fz = "≳",
  hz = "𝒢",
  dz = "ℊ",
  pz = "≳",
  gz = "⪎",
  vz = "⪐",
  mz = "⪧",
  yz = "⩺",
  bz = ">",
  wz = ">",
  xz = "≫",
  _z = "⋗",
  Sz = "⦕",
  kz = "⩼",
  Cz = "⪆",
  Tz = "⥸",
  Ez = "⋗",
  Lz = "⋛",
  Az = "⪌",
  Mz = "≷",
  Nz = "≳",
  Pz = "≩︀",
  Oz = "≩︀",
  Dz = "ˇ",
  $z = " ",
  Rz = "½",
  zz = "ℋ",
  Iz = "Ъ",
  Fz = "ъ",
  qz = "⥈",
  Hz = "↔",
  Bz = "⇔",
  Wz = "↭",
  Uz = "^",
  jz = "ℏ",
  Vz = "Ĥ",
  Gz = "ĥ",
  Kz = "♥",
  Xz = "♥",
  Yz = "…",
  Zz = "⊹",
  Jz = "𝔥",
  Qz = "ℌ",
  tI = "ℋ",
  eI = "⤥",
  nI = "⤦",
  rI = "⇿",
  iI = "∻",
  oI = "↩",
  sI = "↪",
  lI = "𝕙",
  aI = "ℍ",
  cI = "―",
  uI = "─",
  fI = "𝒽",
  hI = "ℋ",
  dI = "ℏ",
  pI = "Ħ",
  gI = "ħ",
  vI = "≎",
  mI = "≏",
  yI = "⁃",
  bI = "‐",
  wI = "Í",
  xI = "í",
  _I = "⁣",
  SI = "Î",
  kI = "î",
  CI = "И",
  TI = "и",
  EI = "İ",
  LI = "Е",
  AI = "е",
  MI = "¡",
  NI = "⇔",
  PI = "𝔦",
  OI = "ℑ",
  DI = "Ì",
  $I = "ì",
  RI = "ⅈ",
  zI = "⨌",
  II = "∭",
  FI = "⧜",
  qI = "℩",
  HI = "Ĳ",
  BI = "ĳ",
  WI = "Ī",
  UI = "ī",
  jI = "ℑ",
  VI = "ⅈ",
  GI = "ℐ",
  KI = "ℑ",
  XI = "ı",
  YI = "ℑ",
  ZI = "⊷",
  JI = "Ƶ",
  QI = "⇒",
  tF = "℅",
  eF = "∞",
  nF = "⧝",
  rF = "ı",
  iF = "⊺",
  oF = "∫",
  sF = "∬",
  lF = "ℤ",
  aF = "∫",
  cF = "⊺",
  uF = "⋂",
  fF = "⨗",
  hF = "⨼",
  dF = "⁣",
  pF = "⁢",
  gF = "Ё",
  vF = "ё",
  mF = "Į",
  yF = "į",
  bF = "𝕀",
  wF = "𝕚",
  xF = "Ι",
  _F = "ι",
  SF = "⨼",
  kF = "¿",
  CF = "𝒾",
  TF = "ℐ",
  EF = "∈",
  LF = "⋵",
  AF = "⋹",
  MF = "⋴",
  NF = "⋳",
  PF = "∈",
  OF = "⁢",
  DF = "Ĩ",
  $F = "ĩ",
  RF = "І",
  zF = "і",
  IF = "Ï",
  FF = "ï",
  qF = "Ĵ",
  HF = "ĵ",
  BF = "Й",
  WF = "й",
  UF = "𝔍",
  jF = "𝔧",
  VF = "ȷ",
  GF = "𝕁",
  KF = "𝕛",
  XF = "𝒥",
  YF = "𝒿",
  ZF = "Ј",
  JF = "ј",
  QF = "Є",
  tq = "є",
  eq = "Κ",
  nq = "κ",
  rq = "ϰ",
  iq = "Ķ",
  oq = "ķ",
  sq = "К",
  lq = "к",
  aq = "𝔎",
  cq = "𝔨",
  uq = "ĸ",
  fq = "Х",
  hq = "х",
  dq = "Ќ",
  pq = "ќ",
  gq = "𝕂",
  vq = "𝕜",
  mq = "𝒦",
  yq = "𝓀",
  bq = "⇚",
  wq = "Ĺ",
  xq = "ĺ",
  _q = "⦴",
  Sq = "ℒ",
  kq = "Λ",
  Cq = "λ",
  Tq = "⟨",
  Eq = "⟪",
  Lq = "⦑",
  Aq = "⟨",
  Mq = "⪅",
  Nq = "ℒ",
  Pq = "«",
  Oq = "⇤",
  Dq = "⤟",
  $q = "←",
  Rq = "↞",
  zq = "⇐",
  Iq = "⤝",
  Fq = "↩",
  qq = "↫",
  Hq = "⤹",
  Bq = "⥳",
  Wq = "↢",
  Uq = "⤙",
  jq = "⤛",
  Vq = "⪫",
  Gq = "⪭",
  Kq = "⪭︀",
  Xq = "⤌",
  Yq = "⤎",
  Zq = "❲",
  Jq = "{",
  Qq = "[",
  tH = "⦋",
  eH = "⦏",
  nH = "⦍",
  rH = "Ľ",
  iH = "ľ",
  oH = "Ļ",
  sH = "ļ",
  lH = "⌈",
  aH = "{",
  cH = "Л",
  uH = "л",
  fH = "⤶",
  hH = "“",
  dH = "„",
  pH = "⥧",
  gH = "⥋",
  vH = "↲",
  mH = "≤",
  yH = "≦",
  bH = "⟨",
  wH = "⇤",
  xH = "←",
  _H = "←",
  SH = "⇐",
  kH = "⇆",
  CH = "↢",
  TH = "⌈",
  EH = "⟦",
  LH = "⥡",
  AH = "⥙",
  MH = "⇃",
  NH = "⌊",
  PH = "↽",
  OH = "↼",
  DH = "⇇",
  $H = "↔",
  RH = "↔",
  zH = "⇔",
  IH = "⇆",
  FH = "⇋",
  qH = "↭",
  HH = "⥎",
  BH = "↤",
  WH = "⊣",
  UH = "⥚",
  jH = "⋋",
  VH = "⧏",
  GH = "⊲",
  KH = "⊴",
  XH = "⥑",
  YH = "⥠",
  ZH = "⥘",
  JH = "↿",
  QH = "⥒",
  tB = "↼",
  eB = "⪋",
  nB = "⋚",
  rB = "≤",
  iB = "≦",
  oB = "⩽",
  sB = "⪨",
  lB = "⩽",
  aB = "⩿",
  cB = "⪁",
  uB = "⪃",
  fB = "⋚︀",
  hB = "⪓",
  dB = "⪅",
  pB = "⋖",
  gB = "⋚",
  vB = "⪋",
  mB = "⋚",
  yB = "≦",
  bB = "≶",
  wB = "≶",
  xB = "⪡",
  _B = "≲",
  SB = "⩽",
  kB = "≲",
  CB = "⥼",
  TB = "⌊",
  EB = "𝔏",
  LB = "𝔩",
  AB = "≶",
  MB = "⪑",
  NB = "⥢",
  PB = "↽",
  OB = "↼",
  DB = "⥪",
  $B = "▄",
  RB = "Љ",
  zB = "љ",
  IB = "⇇",
  FB = "≪",
  qB = "⋘",
  HB = "⌞",
  BB = "⇚",
  WB = "⥫",
  UB = "◺",
  jB = "Ŀ",
  VB = "ŀ",
  GB = "⎰",
  KB = "⎰",
  XB = "⪉",
  YB = "⪉",
  ZB = "⪇",
  JB = "≨",
  QB = "⪇",
  t3 = "≨",
  e3 = "⋦",
  n3 = "⟬",
  r3 = "⇽",
  i3 = "⟦",
  o3 = "⟵",
  s3 = "⟵",
  l3 = "⟸",
  a3 = "⟷",
  c3 = "⟷",
  u3 = "⟺",
  f3 = "⟼",
  h3 = "⟶",
  d3 = "⟶",
  p3 = "⟹",
  g3 = "↫",
  v3 = "↬",
  m3 = "⦅",
  y3 = "𝕃",
  b3 = "𝕝",
  w3 = "⨭",
  x3 = "⨴",
  _3 = "∗",
  S3 = "_",
  k3 = "↙",
  C3 = "↘",
  T3 = "◊",
  E3 = "◊",
  L3 = "⧫",
  A3 = "(",
  M3 = "⦓",
  N3 = "⇆",
  P3 = "⌟",
  O3 = "⇋",
  D3 = "⥭",
  $3 = "‎",
  R3 = "⊿",
  z3 = "‹",
  I3 = "𝓁",
  F3 = "ℒ",
  q3 = "↰",
  H3 = "↰",
  B3 = "≲",
  W3 = "⪍",
  U3 = "⪏",
  j3 = "[",
  V3 = "‘",
  G3 = "‚",
  K3 = "Ł",
  X3 = "ł",
  Y3 = "⪦",
  Z3 = "⩹",
  J3 = "<",
  Q3 = "<",
  t5 = "≪",
  e5 = "⋖",
  n5 = "⋋",
  r5 = "⋉",
  i5 = "⥶",
  o5 = "⩻",
  s5 = "◃",
  l5 = "⊴",
  a5 = "◂",
  c5 = "⦖",
  u5 = "⥊",
  f5 = "⥦",
  h5 = "≨︀",
  d5 = "≨︀",
  p5 = "¯",
  g5 = "♂",
  v5 = "✠",
  m5 = "✠",
  y5 = "↦",
  b5 = "↦",
  w5 = "↧",
  x5 = "↤",
  _5 = "↥",
  S5 = "▮",
  k5 = "⨩",
  C5 = "М",
  T5 = "м",
  E5 = "—",
  L5 = "∺",
  A5 = "∡",
  M5 = " ",
  N5 = "ℳ",
  P5 = "𝔐",
  O5 = "𝔪",
  D5 = "℧",
  $5 = "µ",
  R5 = "*",
  z5 = "⫰",
  I5 = "∣",
  F5 = "·",
  q5 = "⊟",
  H5 = "−",
  B5 = "∸",
  W5 = "⨪",
  U5 = "∓",
  j5 = "⫛",
  V5 = "…",
  G5 = "∓",
  K5 = "⊧",
  X5 = "𝕄",
  Y5 = "𝕞",
  Z5 = "∓",
  J5 = "𝓂",
  Q5 = "ℳ",
  t8 = "∾",
  e8 = "Μ",
  n8 = "μ",
  r8 = "⊸",
  i8 = "⊸",
  o8 = "∇",
  s8 = "Ń",
  l8 = "ń",
  a8 = "∠⃒",
  c8 = "≉",
  u8 = "⩰̸",
  f8 = "≋̸",
  h8 = "ŉ",
  d8 = "≉",
  p8 = "♮",
  g8 = "ℕ",
  v8 = "♮",
  m8 = " ",
  y8 = "≎̸",
  b8 = "≏̸",
  w8 = "⩃",
  x8 = "Ň",
  _8 = "ň",
  S8 = "Ņ",
  k8 = "ņ",
  C8 = "≇",
  T8 = "⩭̸",
  E8 = "⩂",
  L8 = "Н",
  A8 = "н",
  M8 = "–",
  N8 = "⤤",
  P8 = "↗",
  O8 = "⇗",
  D8 = "↗",
  $8 = "≠",
  R8 = "≐̸",
  z8 = "​",
  I8 = "​",
  F8 = "​",
  q8 = "​",
  H8 = "≢",
  B8 = "⤨",
  W8 = "≂̸",
  U8 = "≫",
  j8 = "≪",
  V8 = `
`,
  G8 = "∄",
  K8 = "∄",
  X8 = "𝔑",
  Y8 = "𝔫",
  Z8 = "≧̸",
  J8 = "≱",
  Q8 = "≱",
  tW = "≧̸",
  eW = "⩾̸",
  nW = "⩾̸",
  rW = "⋙̸",
  iW = "≵",
  oW = "≫⃒",
  sW = "≯",
  lW = "≯",
  aW = "≫̸",
  cW = "↮",
  uW = "⇎",
  fW = "⫲",
  hW = "∋",
  dW = "⋼",
  pW = "⋺",
  gW = "∋",
  vW = "Њ",
  mW = "њ",
  yW = "↚",
  bW = "⇍",
  wW = "‥",
  xW = "≦̸",
  _W = "≰",
  SW = "↚",
  kW = "⇍",
  CW = "↮",
  TW = "⇎",
  EW = "≰",
  LW = "≦̸",
  AW = "⩽̸",
  MW = "⩽̸",
  NW = "≮",
  PW = "⋘̸",
  OW = "≴",
  DW = "≪⃒",
  $W = "≮",
  RW = "⋪",
  zW = "⋬",
  IW = "≪̸",
  FW = "∤",
  qW = "⁠",
  HW = " ",
  BW = "𝕟",
  WW = "ℕ",
  UW = "⫬",
  jW = "¬",
  VW = "≢",
  GW = "≭",
  KW = "∦",
  XW = "∉",
  YW = "≠",
  ZW = "≂̸",
  JW = "∄",
  QW = "≯",
  tU = "≱",
  eU = "≧̸",
  nU = "≫̸",
  rU = "≹",
  iU = "⩾̸",
  oU = "≵",
  sU = "≎̸",
  lU = "≏̸",
  aU = "∉",
  cU = "⋵̸",
  uU = "⋹̸",
  fU = "∉",
  hU = "⋷",
  dU = "⋶",
  pU = "⧏̸",
  gU = "⋪",
  vU = "⋬",
  mU = "≮",
  yU = "≰",
  bU = "≸",
  wU = "≪̸",
  xU = "⩽̸",
  _U = "≴",
  SU = "⪢̸",
  kU = "⪡̸",
  CU = "∌",
  TU = "∌",
  EU = "⋾",
  LU = "⋽",
  AU = "⊀",
  MU = "⪯̸",
  NU = "⋠",
  PU = "∌",
  OU = "⧐̸",
  DU = "⋫",
  $U = "⋭",
  RU = "⊏̸",
  zU = "⋢",
  IU = "⊐̸",
  FU = "⋣",
  qU = "⊂⃒",
  HU = "⊈",
  BU = "⊁",
  WU = "⪰̸",
  UU = "⋡",
  jU = "≿̸",
  VU = "⊃⃒",
  GU = "⊉",
  KU = "≁",
  XU = "≄",
  YU = "≇",
  ZU = "≉",
  JU = "∤",
  QU = "∦",
  t4 = "∦",
  e4 = "⫽⃥",
  n4 = "∂̸",
  r4 = "⨔",
  i4 = "⊀",
  o4 = "⋠",
  s4 = "⊀",
  l4 = "⪯̸",
  a4 = "⪯̸",
  c4 = "⤳̸",
  u4 = "↛",
  f4 = "⇏",
  h4 = "↝̸",
  d4 = "↛",
  p4 = "⇏",
  g4 = "⋫",
  v4 = "⋭",
  m4 = "⊁",
  y4 = "⋡",
  b4 = "⪰̸",
  w4 = "𝒩",
  x4 = "𝓃",
  _4 = "∤",
  S4 = "∦",
  k4 = "≁",
  C4 = "≄",
  T4 = "≄",
  E4 = "∤",
  L4 = "∦",
  A4 = "⋢",
  M4 = "⋣",
  N4 = "⊄",
  P4 = "⫅̸",
  O4 = "⊈",
  D4 = "⊂⃒",
  $4 = "⊈",
  R4 = "⫅̸",
  z4 = "⊁",
  I4 = "⪰̸",
  F4 = "⊅",
  q4 = "⫆̸",
  H4 = "⊉",
  B4 = "⊃⃒",
  W4 = "⊉",
  U4 = "⫆̸",
  j4 = "≹",
  V4 = "Ñ",
  G4 = "ñ",
  K4 = "≸",
  X4 = "⋪",
  Y4 = "⋬",
  Z4 = "⋫",
  J4 = "⋭",
  Q4 = "Ν",
  t6 = "ν",
  e6 = "#",
  n6 = "№",
  r6 = " ",
  i6 = "≍⃒",
  o6 = "⊬",
  s6 = "⊭",
  l6 = "⊮",
  a6 = "⊯",
  c6 = "≥⃒",
  u6 = ">⃒",
  f6 = "⤄",
  h6 = "⧞",
  d6 = "⤂",
  p6 = "≤⃒",
  g6 = "<⃒",
  v6 = "⊴⃒",
  m6 = "⤃",
  y6 = "⊵⃒",
  b6 = "∼⃒",
  w6 = "⤣",
  x6 = "↖",
  _6 = "⇖",
  S6 = "↖",
  k6 = "⤧",
  C6 = "Ó",
  T6 = "ó",
  E6 = "⊛",
  L6 = "Ô",
  A6 = "ô",
  M6 = "⊚",
  N6 = "О",
  P6 = "о",
  O6 = "⊝",
  D6 = "Ő",
  $6 = "ő",
  R6 = "⨸",
  z6 = "⊙",
  I6 = "⦼",
  F6 = "Œ",
  q6 = "œ",
  H6 = "⦿",
  B6 = "𝔒",
  W6 = "𝔬",
  U6 = "˛",
  j6 = "Ò",
  V6 = "ò",
  G6 = "⧁",
  K6 = "⦵",
  X6 = "Ω",
  Y6 = "∮",
  Z6 = "↺",
  J6 = "⦾",
  Q6 = "⦻",
  tj = "‾",
  ej = "⧀",
  nj = "Ō",
  rj = "ō",
  ij = "Ω",
  oj = "ω",
  sj = "Ο",
  lj = "ο",
  aj = "⦶",
  cj = "⊖",
  uj = "𝕆",
  fj = "𝕠",
  hj = "⦷",
  dj = "“",
  pj = "‘",
  gj = "⦹",
  vj = "⊕",
  mj = "↻",
  yj = "⩔",
  bj = "∨",
  wj = "⩝",
  xj = "ℴ",
  _j = "ℴ",
  Sj = "ª",
  kj = "º",
  Cj = "⊶",
  Tj = "⩖",
  Ej = "⩗",
  Lj = "⩛",
  Aj = "Ⓢ",
  Mj = "𝒪",
  Nj = "ℴ",
  Pj = "Ø",
  Oj = "ø",
  Dj = "⊘",
  $j = "Õ",
  Rj = "õ",
  zj = "⨶",
  Ij = "⨷",
  Fj = "⊗",
  qj = "Ö",
  Hj = "ö",
  Bj = "⌽",
  Wj = "‾",
  Uj = "⏞",
  jj = "⎴",
  Vj = "⏜",
  Gj = "¶",
  Kj = "∥",
  Xj = "∥",
  Yj = "⫳",
  Zj = "⫽",
  Jj = "∂",
  Qj = "∂",
  t9 = "П",
  e9 = "п",
  n9 = "%",
  r9 = ".",
  i9 = "‰",
  o9 = "⊥",
  s9 = "‱",
  l9 = "𝔓",
  a9 = "𝔭",
  c9 = "Φ",
  u9 = "φ",
  f9 = "ϕ",
  h9 = "ℳ",
  d9 = "☎",
  p9 = "Π",
  g9 = "π",
  v9 = "⋔",
  m9 = "ϖ",
  y9 = "ℏ",
  b9 = "ℎ",
  w9 = "ℏ",
  x9 = "⨣",
  _9 = "⊞",
  S9 = "⨢",
  k9 = "+",
  C9 = "∔",
  T9 = "⨥",
  E9 = "⩲",
  L9 = "±",
  A9 = "±",
  M9 = "⨦",
  N9 = "⨧",
  P9 = "±",
  O9 = "ℌ",
  D9 = "⨕",
  $9 = "𝕡",
  R9 = "ℙ",
  z9 = "£",
  I9 = "⪷",
  F9 = "⪻",
  q9 = "≺",
  H9 = "≼",
  B9 = "⪷",
  W9 = "≺",
  U9 = "≼",
  j9 = "≺",
  V9 = "⪯",
  G9 = "≼",
  K9 = "≾",
  X9 = "⪯",
  Y9 = "⪹",
  Z9 = "⪵",
  J9 = "⋨",
  Q9 = "⪯",
  tV = "⪳",
  eV = "≾",
  nV = "′",
  rV = "″",
  iV = "ℙ",
  oV = "⪹",
  sV = "⪵",
  lV = "⋨",
  aV = "∏",
  cV = "∏",
  uV = "⌮",
  fV = "⌒",
  hV = "⌓",
  dV = "∝",
  pV = "∝",
  gV = "∷",
  vV = "∝",
  mV = "≾",
  yV = "⊰",
  bV = "𝒫",
  wV = "𝓅",
  xV = "Ψ",
  _V = "ψ",
  SV = " ",
  kV = "𝔔",
  CV = "𝔮",
  TV = "⨌",
  EV = "𝕢",
  LV = "ℚ",
  AV = "⁗",
  MV = "𝒬",
  NV = "𝓆",
  PV = "ℍ",
  OV = "⨖",
  DV = "?",
  $V = "≟",
  RV = '"',
  zV = '"',
  IV = "⇛",
  FV = "∽̱",
  qV = "Ŕ",
  HV = "ŕ",
  BV = "√",
  WV = "⦳",
  UV = "⟩",
  jV = "⟫",
  VV = "⦒",
  GV = "⦥",
  KV = "⟩",
  XV = "»",
  YV = "⥵",
  ZV = "⇥",
  JV = "⤠",
  QV = "⤳",
  tG = "→",
  eG = "↠",
  nG = "⇒",
  rG = "⤞",
  iG = "↪",
  oG = "↬",
  sG = "⥅",
  lG = "⥴",
  aG = "⤖",
  cG = "↣",
  uG = "↝",
  fG = "⤚",
  hG = "⤜",
  dG = "∶",
  pG = "ℚ",
  gG = "⤍",
  vG = "⤏",
  mG = "⤐",
  yG = "❳",
  bG = "}",
  wG = "]",
  xG = "⦌",
  _G = "⦎",
  SG = "⦐",
  kG = "Ř",
  CG = "ř",
  TG = "Ŗ",
  EG = "ŗ",
  LG = "⌉",
  AG = "}",
  MG = "Р",
  NG = "р",
  PG = "⤷",
  OG = "⥩",
  DG = "”",
  $G = "”",
  RG = "↳",
  zG = "ℜ",
  IG = "ℛ",
  FG = "ℜ",
  qG = "ℝ",
  HG = "ℜ",
  BG = "▭",
  WG = "®",
  UG = "®",
  jG = "∋",
  VG = "⇋",
  GG = "⥯",
  KG = "⥽",
  XG = "⌋",
  YG = "𝔯",
  ZG = "ℜ",
  JG = "⥤",
  QG = "⇁",
  t7 = "⇀",
  e7 = "⥬",
  n7 = "Ρ",
  r7 = "ρ",
  i7 = "ϱ",
  o7 = "⟩",
  s7 = "⇥",
  l7 = "→",
  a7 = "→",
  c7 = "⇒",
  u7 = "⇄",
  f7 = "↣",
  h7 = "⌉",
  d7 = "⟧",
  p7 = "⥝",
  g7 = "⥕",
  v7 = "⇂",
  m7 = "⌋",
  y7 = "⇁",
  b7 = "⇀",
  w7 = "⇄",
  x7 = "⇌",
  _7 = "⇉",
  S7 = "↝",
  k7 = "↦",
  C7 = "⊢",
  T7 = "⥛",
  E7 = "⋌",
  L7 = "⧐",
  A7 = "⊳",
  M7 = "⊵",
  N7 = "⥏",
  P7 = "⥜",
  O7 = "⥔",
  D7 = "↾",
  $7 = "⥓",
  R7 = "⇀",
  z7 = "˚",
  I7 = "≓",
  F7 = "⇄",
  q7 = "⇌",
  H7 = "‏",
  B7 = "⎱",
  W7 = "⎱",
  U7 = "⫮",
  j7 = "⟭",
  V7 = "⇾",
  G7 = "⟧",
  K7 = "⦆",
  X7 = "𝕣",
  Y7 = "ℝ",
  Z7 = "⨮",
  J7 = "⨵",
  Q7 = "⥰",
  tK = ")",
  eK = "⦔",
  nK = "⨒",
  rK = "⇉",
  iK = "⇛",
  oK = "›",
  sK = "𝓇",
  lK = "ℛ",
  aK = "↱",
  cK = "↱",
  uK = "]",
  fK = "’",
  hK = "’",
  dK = "⋌",
  pK = "⋊",
  gK = "▹",
  vK = "⊵",
  mK = "▸",
  yK = "⧎",
  bK = "⧴",
  wK = "⥨",
  xK = "℞",
  _K = "Ś",
  SK = "ś",
  kK = "‚",
  CK = "⪸",
  TK = "Š",
  EK = "š",
  LK = "⪼",
  AK = "≻",
  MK = "≽",
  NK = "⪰",
  PK = "⪴",
  OK = "Ş",
  DK = "ş",
  $K = "Ŝ",
  RK = "ŝ",
  zK = "⪺",
  IK = "⪶",
  FK = "⋩",
  qK = "⨓",
  HK = "≿",
  BK = "С",
  WK = "с",
  UK = "⊡",
  jK = "⋅",
  VK = "⩦",
  GK = "⤥",
  KK = "↘",
  XK = "⇘",
  YK = "↘",
  ZK = "§",
  JK = ";",
  QK = "⤩",
  tX = "∖",
  eX = "∖",
  nX = "✶",
  rX = "𝔖",
  iX = "𝔰",
  oX = "⌢",
  sX = "♯",
  lX = "Щ",
  aX = "щ",
  cX = "Ш",
  uX = "ш",
  fX = "↓",
  hX = "←",
  dX = "∣",
  pX = "∥",
  gX = "→",
  vX = "↑",
  mX = "­",
  yX = "Σ",
  bX = "σ",
  wX = "ς",
  xX = "ς",
  _X = "∼",
  SX = "⩪",
  kX = "≃",
  CX = "≃",
  TX = "⪞",
  EX = "⪠",
  LX = "⪝",
  AX = "⪟",
  MX = "≆",
  NX = "⨤",
  PX = "⥲",
  OX = "←",
  DX = "∘",
  $X = "∖",
  RX = "⨳",
  zX = "⧤",
  IX = "∣",
  FX = "⌣",
  qX = "⪪",
  HX = "⪬",
  BX = "⪬︀",
  WX = "Ь",
  UX = "ь",
  jX = "⌿",
  VX = "⧄",
  GX = "/",
  KX = "𝕊",
  XX = "𝕤",
  YX = "♠",
  ZX = "♠",
  JX = "∥",
  QX = "⊓",
  tY = "⊓︀",
  eY = "⊔",
  nY = "⊔︀",
  rY = "√",
  iY = "⊏",
  oY = "⊑",
  sY = "⊏",
  lY = "⊑",
  aY = "⊐",
  cY = "⊒",
  uY = "⊐",
  fY = "⊒",
  hY = "□",
  dY = "□",
  pY = "⊓",
  gY = "⊏",
  vY = "⊑",
  mY = "⊐",
  yY = "⊒",
  bY = "⊔",
  wY = "▪",
  xY = "□",
  _Y = "▪",
  SY = "→",
  kY = "𝒮",
  CY = "𝓈",
  TY = "∖",
  EY = "⌣",
  LY = "⋆",
  AY = "⋆",
  MY = "☆",
  NY = "★",
  PY = "ϵ",
  OY = "ϕ",
  DY = "¯",
  $Y = "⊂",
  RY = "⋐",
  zY = "⪽",
  IY = "⫅",
  FY = "⊆",
  qY = "⫃",
  HY = "⫁",
  BY = "⫋",
  WY = "⊊",
  UY = "⪿",
  jY = "⥹",
  VY = "⊂",
  GY = "⋐",
  KY = "⊆",
  XY = "⫅",
  YY = "⊆",
  ZY = "⊊",
  JY = "⫋",
  QY = "⫇",
  tZ = "⫕",
  eZ = "⫓",
  nZ = "⪸",
  rZ = "≻",
  iZ = "≽",
  oZ = "≻",
  sZ = "⪰",
  lZ = "≽",
  aZ = "≿",
  cZ = "⪰",
  uZ = "⪺",
  fZ = "⪶",
  hZ = "⋩",
  dZ = "≿",
  pZ = "∋",
  gZ = "∑",
  vZ = "∑",
  mZ = "♪",
  yZ = "¹",
  bZ = "²",
  wZ = "³",
  xZ = "⊃",
  _Z = "⋑",
  SZ = "⪾",
  kZ = "⫘",
  CZ = "⫆",
  TZ = "⊇",
  EZ = "⫄",
  LZ = "⊃",
  AZ = "⊇",
  MZ = "⟉",
  NZ = "⫗",
  PZ = "⥻",
  OZ = "⫂",
  DZ = "⫌",
  $Z = "⊋",
  RZ = "⫀",
  zZ = "⊃",
  IZ = "⋑",
  FZ = "⊇",
  qZ = "⫆",
  HZ = "⊋",
  BZ = "⫌",
  WZ = "⫈",
  UZ = "⫔",
  jZ = "⫖",
  VZ = "⤦",
  GZ = "↙",
  KZ = "⇙",
  XZ = "↙",
  YZ = "⤪",
  ZZ = "ß",
  JZ = "	",
  QZ = "⌖",
  tJ = "Τ",
  eJ = "τ",
  nJ = "⎴",
  rJ = "Ť",
  iJ = "ť",
  oJ = "Ţ",
  sJ = "ţ",
  lJ = "Т",
  aJ = "т",
  cJ = "⃛",
  uJ = "⌕",
  fJ = "𝔗",
  hJ = "𝔱",
  dJ = "∴",
  pJ = "∴",
  gJ = "∴",
  vJ = "Θ",
  mJ = "θ",
  yJ = "ϑ",
  bJ = "ϑ",
  wJ = "≈",
  xJ = "∼",
  _J = "  ",
  SJ = " ",
  kJ = " ",
  CJ = "≈",
  TJ = "∼",
  EJ = "Þ",
  LJ = "þ",
  AJ = "˜",
  MJ = "∼",
  NJ = "≃",
  PJ = "≅",
  OJ = "≈",
  DJ = "⨱",
  $J = "⊠",
  RJ = "×",
  zJ = "⨰",
  IJ = "∭",
  FJ = "⤨",
  qJ = "⌶",
  HJ = "⫱",
  BJ = "⊤",
  WJ = "𝕋",
  UJ = "𝕥",
  jJ = "⫚",
  VJ = "⤩",
  GJ = "‴",
  KJ = "™",
  XJ = "™",
  YJ = "▵",
  ZJ = "▿",
  JJ = "◃",
  QJ = "⊴",
  tQ = "≜",
  eQ = "▹",
  nQ = "⊵",
  rQ = "◬",
  iQ = "≜",
  oQ = "⨺",
  sQ = "⃛",
  lQ = "⨹",
  aQ = "⧍",
  cQ = "⨻",
  uQ = "⏢",
  fQ = "𝒯",
  hQ = "𝓉",
  dQ = "Ц",
  pQ = "ц",
  gQ = "Ћ",
  vQ = "ћ",
  mQ = "Ŧ",
  yQ = "ŧ",
  bQ = "≬",
  wQ = "↞",
  xQ = "↠",
  _Q = "Ú",
  SQ = "ú",
  kQ = "↑",
  CQ = "↟",
  TQ = "⇑",
  EQ = "⥉",
  LQ = "Ў",
  AQ = "ў",
  MQ = "Ŭ",
  NQ = "ŭ",
  PQ = "Û",
  OQ = "û",
  DQ = "У",
  $Q = "у",
  RQ = "⇅",
  zQ = "Ű",
  IQ = "ű",
  FQ = "⥮",
  qQ = "⥾",
  HQ = "𝔘",
  BQ = "𝔲",
  WQ = "Ù",
  UQ = "ù",
  jQ = "⥣",
  VQ = "↿",
  GQ = "↾",
  KQ = "▀",
  XQ = "⌜",
  YQ = "⌜",
  ZQ = "⌏",
  JQ = "◸",
  QQ = "Ū",
  ttt = "ū",
  ett = "¨",
  ntt = "_",
  rtt = "⏟",
  itt = "⎵",
  ott = "⏝",
  stt = "⋃",
  ltt = "⊎",
  att = "Ų",
  ctt = "ų",
  utt = "𝕌",
  ftt = "𝕦",
  htt = "⤒",
  dtt = "↑",
  ptt = "↑",
  gtt = "⇑",
  vtt = "⇅",
  mtt = "↕",
  ytt = "↕",
  btt = "⇕",
  wtt = "⥮",
  xtt = "↿",
  _tt = "↾",
  Stt = "⊎",
  ktt = "↖",
  Ctt = "↗",
  Ttt = "υ",
  Ett = "ϒ",
  Ltt = "ϒ",
  Att = "Υ",
  Mtt = "υ",
  Ntt = "↥",
  Ptt = "⊥",
  Ott = "⇈",
  Dtt = "⌝",
  $tt = "⌝",
  Rtt = "⌎",
  ztt = "Ů",
  Itt = "ů",
  Ftt = "◹",
  qtt = "𝒰",
  Htt = "𝓊",
  Btt = "⋰",
  Wtt = "Ũ",
  Utt = "ũ",
  jtt = "▵",
  Vtt = "▴",
  Gtt = "⇈",
  Ktt = "Ü",
  Xtt = "ü",
  Ytt = "⦧",
  Ztt = "⦜",
  Jtt = "ϵ",
  Qtt = "ϰ",
  tet = "∅",
  eet = "ϕ",
  net = "ϖ",
  ret = "∝",
  iet = "↕",
  oet = "⇕",
  set = "ϱ",
  aet = "ς",
  cet = "⊊︀",
  uet = "⫋︀",
  fet = "⊋︀",
  het = "⫌︀",
  det = "ϑ",
  pet = "⊲",
  get = "⊳",
  vet = "⫨",
  met = "⫫",
  yet = "⫩",
  bet = "В",
  wet = "в",
  xet = "⊢",
  _et = "⊨",
  ket = "⊩",
  Cet = "⊫",
  Tet = "⫦",
  Eet = "⊻",
  Let = "∨",
  Aet = "⋁",
  Met = "≚",
  Net = "⋮",
  Pet = "|",
  Oet = "‖",
  Det = "|",
  $et = "‖",
  Ret = "∣",
  zet = "|",
  Iet = "❘",
  Fet = "≀",
  qet = " ",
  Het = "𝔙",
  Bet = "𝔳",
  Wet = "⊲",
  Uet = "⊂⃒",
  jet = "⊃⃒",
  Vet = "𝕍",
  Get = "𝕧",
  Ket = "∝",
  Xet = "⊳",
  Yet = "𝒱",
  Zet = "𝓋",
  Jet = "⫋︀",
  Qet = "⊊︀",
  tnt = "⫌︀",
  ent = "⊋︀",
  nnt = "⊪",
  rnt = "⦚",
  int = "Ŵ",
  ont = "ŵ",
  snt = "⩟",
  lnt = "∧",
  ant = "⋀",
  cnt = "≙",
  unt = "℘",
  fnt = "𝔚",
  hnt = "𝔴",
  dnt = "𝕎",
  pnt = "𝕨",
  gnt = "℘",
  vnt = "≀",
  mnt = "≀",
  ynt = "𝒲",
  bnt = "𝓌",
  wnt = "⋂",
  xnt = "◯",
  _nt = "⋃",
  Snt = "▽",
  knt = "𝔛",
  Cnt = "𝔵",
  Tnt = "⟷",
  Ent = "⟺",
  Lnt = "Ξ",
  Ant = "ξ",
  Mnt = "⟵",
  Nnt = "⟸",
  Pnt = "⟼",
  Ont = "⋻",
  Dnt = "⨀",
  $nt = "𝕏",
  Rnt = "𝕩",
  znt = "⨁",
  Int = "⨂",
  Fnt = "⟶",
  qnt = "⟹",
  Hnt = "𝒳",
  Bnt = "𝓍",
  Wnt = "⨆",
  Unt = "⨄",
  jnt = "△",
  Vnt = "⋁",
  Gnt = "⋀",
  Knt = "Ý",
  Xnt = "ý",
  Ynt = "Я",
  Znt = "я",
  Jnt = "Ŷ",
  Qnt = "ŷ",
  trt = "Ы",
  ert = "ы",
  nrt = "¥",
  rrt = "𝔜",
  irt = "𝔶",
  ort = "Ї",
  srt = "ї",
  lrt = "𝕐",
  art = "𝕪",
  crt = "𝒴",
  urt = "𝓎",
  frt = "Ю",
  hrt = "ю",
  drt = "ÿ",
  prt = "Ÿ",
  grt = "Ź",
  vrt = "ź",
  mrt = "Ž",
  yrt = "ž",
  brt = "З",
  wrt = "з",
  xrt = "Ż",
  _rt = "ż",
  Srt = "ℨ",
  krt = "​",
  Crt = "Ζ",
  Trt = "ζ",
  Ert = "𝔷",
  Lrt = "ℨ",
  Art = "Ж",
  Mrt = "ж",
  Nrt = "⇝",
  Prt = "𝕫",
  Ort = "ℤ",
  Drt = "𝒵",
  $rt = "𝓏",
  Rrt = "‍",
  zrt = "‌",
  my = {
    Aacute: bT,
    aacute: wT,
    Abreve: xT,
    abreve: _T,
    ac: ST,
    acd: kT,
    acE: CT,
    Acirc: TT,
    acirc: ET,
    acute: LT,
    Acy: AT,
    acy: MT,
    AElig: NT,
    aelig: PT,
    af: OT,
    Afr: DT,
    afr: $T,
    Agrave: RT,
    agrave: zT,
    alefsym: IT,
    aleph: FT,
    Alpha: qT,
    alpha: HT,
    Amacr: BT,
    amacr: WT,
    amalg: UT,
    amp: jT,
    AMP: VT,
    andand: GT,
    And: KT,
    and: XT,
    andd: YT,
    andslope: ZT,
    andv: JT,
    ang: QT,
    ange: tE,
    angle: eE,
    angmsdaa: nE,
    angmsdab: rE,
    angmsdac: iE,
    angmsdad: oE,
    angmsdae: sE,
    angmsdaf: lE,
    angmsdag: aE,
    angmsdah: cE,
    angmsd: uE,
    angrt: fE,
    angrtvb: hE,
    angrtvbd: dE,
    angsph: pE,
    angst: gE,
    angzarr: vE,
    Aogon: mE,
    aogon: yE,
    Aopf: bE,
    aopf: wE,
    apacir: xE,
    ap: _E,
    apE: SE,
    ape: kE,
    apid: CE,
    apos: TE,
    ApplyFunction: EE,
    approx: LE,
    approxeq: AE,
    Aring: ME,
    aring: NE,
    Ascr: PE,
    ascr: OE,
    Assign: DE,
    ast: $E,
    asymp: RE,
    asympeq: zE,
    Atilde: IE,
    atilde: FE,
    Auml: qE,
    auml: HE,
    awconint: BE,
    awint: WE,
    backcong: UE,
    backepsilon: jE,
    backprime: VE,
    backsim: GE,
    backsimeq: KE,
    Backslash: XE,
    Barv: YE,
    barvee: ZE,
    barwed: JE,
    Barwed: QE,
    barwedge: tL,
    bbrk: eL,
    bbrktbrk: nL,
    bcong: rL,
    Bcy: iL,
    bcy: oL,
    bdquo: sL,
    becaus: lL,
    because: aL,
    Because: cL,
    bemptyv: uL,
    bepsi: fL,
    bernou: hL,
    Bernoullis: dL,
    Beta: pL,
    beta: gL,
    beth: vL,
    between: mL,
    Bfr: yL,
    bfr: bL,
    bigcap: wL,
    bigcirc: xL,
    bigcup: _L,
    bigodot: SL,
    bigoplus: kL,
    bigotimes: CL,
    bigsqcup: TL,
    bigstar: EL,
    bigtriangledown: LL,
    bigtriangleup: AL,
    biguplus: ML,
    bigvee: NL,
    bigwedge: PL,
    bkarow: OL,
    blacklozenge: DL,
    blacksquare: $L,
    blacktriangle: RL,
    blacktriangledown: zL,
    blacktriangleleft: IL,
    blacktriangleright: FL,
    blank: qL,
    blk12: HL,
    blk14: BL,
    blk34: WL,
    block: UL,
    bne: jL,
    bnequiv: VL,
    bNot: GL,
    bnot: KL,
    Bopf: XL,
    bopf: YL,
    bot: ZL,
    bottom: JL,
    bowtie: QL,
    boxbox: tA,
    boxdl: eA,
    boxdL: nA,
    boxDl: rA,
    boxDL: iA,
    boxdr: oA,
    boxdR: sA,
    boxDr: lA,
    boxDR: aA,
    boxh: cA,
    boxH: uA,
    boxhd: fA,
    boxHd: hA,
    boxhD: dA,
    boxHD: pA,
    boxhu: gA,
    boxHu: vA,
    boxhU: mA,
    boxHU: yA,
    boxminus: bA,
    boxplus: wA,
    boxtimes: xA,
    boxul: _A,
    boxuL: SA,
    boxUl: kA,
    boxUL: CA,
    boxur: TA,
    boxuR: EA,
    boxUr: LA,
    boxUR: AA,
    boxv: MA,
    boxV: NA,
    boxvh: PA,
    boxvH: OA,
    boxVh: DA,
    boxVH: $A,
    boxvl: RA,
    boxvL: zA,
    boxVl: IA,
    boxVL: FA,
    boxvr: qA,
    boxvR: HA,
    boxVr: BA,
    boxVR: WA,
    bprime: UA,
    breve: jA,
    Breve: VA,
    brvbar: GA,
    bscr: KA,
    Bscr: XA,
    bsemi: YA,
    bsim: ZA,
    bsime: JA,
    bsolb: QA,
    bsol: tM,
    bsolhsub: eM,
    bull: nM,
    bullet: rM,
    bump: iM,
    bumpE: oM,
    bumpe: sM,
    Bumpeq: lM,
    bumpeq: aM,
    Cacute: cM,
    cacute: uM,
    capand: fM,
    capbrcup: hM,
    capcap: dM,
    cap: pM,
    Cap: gM,
    capcup: vM,
    capdot: mM,
    CapitalDifferentialD: yM,
    caps: bM,
    caret: wM,
    caron: xM,
    Cayleys: _M,
    ccaps: SM,
    Ccaron: kM,
    ccaron: CM,
    Ccedil: TM,
    ccedil: EM,
    Ccirc: LM,
    ccirc: AM,
    Cconint: MM,
    ccups: NM,
    ccupssm: PM,
    Cdot: OM,
    cdot: DM,
    cedil: $M,
    Cedilla: RM,
    cemptyv: zM,
    cent: IM,
    centerdot: FM,
    CenterDot: qM,
    cfr: HM,
    Cfr: BM,
    CHcy: WM,
    chcy: UM,
    check: jM,
    checkmark: VM,
    Chi: GM,
    chi: KM,
    circ: XM,
    circeq: YM,
    circlearrowleft: ZM,
    circlearrowright: JM,
    circledast: QM,
    circledcirc: tN,
    circleddash: eN,
    CircleDot: nN,
    circledR: rN,
    circledS: iN,
    CircleMinus: oN,
    CirclePlus: sN,
    CircleTimes: lN,
    cir: aN,
    cirE: cN,
    cire: uN,
    cirfnint: fN,
    cirmid: hN,
    cirscir: dN,
    ClockwiseContourIntegral: pN,
    CloseCurlyDoubleQuote: gN,
    CloseCurlyQuote: vN,
    clubs: mN,
    clubsuit: yN,
    colon: bN,
    Colon: wN,
    Colone: xN,
    colone: _N,
    coloneq: SN,
    comma: kN,
    commat: CN,
    comp: TN,
    compfn: EN,
    complement: LN,
    complexes: AN,
    cong: MN,
    congdot: NN,
    Congruent: PN,
    conint: ON,
    Conint: DN,
    ContourIntegral: $N,
    copf: RN,
    Copf: zN,
    coprod: IN,
    Coproduct: FN,
    copy: qN,
    COPY: HN,
    copysr: BN,
    CounterClockwiseContourIntegral: WN,
    crarr: UN,
    cross: jN,
    Cross: VN,
    Cscr: GN,
    cscr: KN,
    csub: XN,
    csube: YN,
    csup: ZN,
    csupe: JN,
    ctdot: QN,
    cudarrl: tP,
    cudarrr: eP,
    cuepr: nP,
    cuesc: rP,
    cularr: iP,
    cularrp: oP,
    cupbrcap: sP,
    cupcap: lP,
    CupCap: aP,
    cup: cP,
    Cup: uP,
    cupcup: fP,
    cupdot: hP,
    cupor: dP,
    cups: pP,
    curarr: gP,
    curarrm: vP,
    curlyeqprec: mP,
    curlyeqsucc: yP,
    curlyvee: bP,
    curlywedge: wP,
    curren: xP,
    curvearrowleft: _P,
    curvearrowright: SP,
    cuvee: kP,
    cuwed: CP,
    cwconint: TP,
    cwint: EP,
    cylcty: LP,
    dagger: AP,
    Dagger: MP,
    daleth: NP,
    darr: PP,
    Darr: OP,
    dArr: DP,
    dash: $P,
    Dashv: RP,
    dashv: zP,
    dbkarow: IP,
    dblac: FP,
    Dcaron: qP,
    dcaron: HP,
    Dcy: BP,
    dcy: WP,
    ddagger: UP,
    ddarr: jP,
    DD: VP,
    dd: GP,
    DDotrahd: KP,
    ddotseq: XP,
    deg: YP,
    Del: ZP,
    Delta: JP,
    delta: QP,
    demptyv: tO,
    dfisht: eO,
    Dfr: nO,
    dfr: rO,
    dHar: iO,
    dharl: oO,
    dharr: sO,
    DiacriticalAcute: lO,
    DiacriticalDot: aO,
    DiacriticalDoubleAcute: cO,
    DiacriticalGrave: uO,
    DiacriticalTilde: fO,
    diam: hO,
    diamond: dO,
    Diamond: pO,
    diamondsuit: gO,
    diams: vO,
    die: mO,
    DifferentialD: yO,
    digamma: bO,
    disin: wO,
    div: xO,
    divide: _O,
    divideontimes: SO,
    divonx: kO,
    DJcy: CO,
    djcy: TO,
    dlcorn: EO,
    dlcrop: LO,
    dollar: AO,
    Dopf: MO,
    dopf: NO,
    Dot: PO,
    dot: OO,
    DotDot: DO,
    doteq: $O,
    doteqdot: RO,
    DotEqual: zO,
    dotminus: IO,
    dotplus: FO,
    dotsquare: qO,
    doublebarwedge: HO,
    DoubleContourIntegral: BO,
    DoubleDot: WO,
    DoubleDownArrow: UO,
    DoubleLeftArrow: jO,
    DoubleLeftRightArrow: VO,
    DoubleLeftTee: GO,
    DoubleLongLeftArrow: KO,
    DoubleLongLeftRightArrow: XO,
    DoubleLongRightArrow: YO,
    DoubleRightArrow: ZO,
    DoubleRightTee: JO,
    DoubleUpArrow: QO,
    DoubleUpDownArrow: tD,
    DoubleVerticalBar: eD,
    DownArrowBar: nD,
    downarrow: rD,
    DownArrow: iD,
    Downarrow: oD,
    DownArrowUpArrow: sD,
    DownBreve: lD,
    downdownarrows: aD,
    downharpoonleft: cD,
    downharpoonright: uD,
    DownLeftRightVector: fD,
    DownLeftTeeVector: hD,
    DownLeftVectorBar: dD,
    DownLeftVector: pD,
    DownRightTeeVector: gD,
    DownRightVectorBar: vD,
    DownRightVector: mD,
    DownTeeArrow: yD,
    DownTee: bD,
    drbkarow: wD,
    drcorn: xD,
    drcrop: _D,
    Dscr: SD,
    dscr: kD,
    DScy: CD,
    dscy: TD,
    dsol: ED,
    Dstrok: LD,
    dstrok: AD,
    dtdot: MD,
    dtri: ND,
    dtrif: PD,
    duarr: OD,
    duhar: DD,
    dwangle: $D,
    DZcy: RD,
    dzcy: zD,
    dzigrarr: ID,
    Eacute: FD,
    eacute: qD,
    easter: HD,
    Ecaron: BD,
    ecaron: WD,
    Ecirc: UD,
    ecirc: jD,
    ecir: VD,
    ecolon: GD,
    Ecy: KD,
    ecy: XD,
    eDDot: YD,
    Edot: ZD,
    edot: JD,
    eDot: QD,
    ee: t$,
    efDot: e$,
    Efr: n$,
    efr: r$,
    eg: i$,
    Egrave: o$,
    egrave: s$,
    egs: l$,
    egsdot: a$,
    el: c$,
    Element: u$,
    elinters: f$,
    ell: h$,
    els: d$,
    elsdot: p$,
    Emacr: g$,
    emacr: v$,
    empty: m$,
    emptyset: y$,
    EmptySmallSquare: b$,
    emptyv: w$,
    EmptyVerySmallSquare: x$,
    emsp13: _$,
    emsp14: S$,
    emsp: k$,
    ENG: C$,
    eng: T$,
    ensp: E$,
    Eogon: L$,
    eogon: A$,
    Eopf: M$,
    eopf: N$,
    epar: P$,
    eparsl: O$,
    eplus: D$,
    epsi: $$,
    Epsilon: R$,
    epsilon: z$,
    epsiv: I$,
    eqcirc: F$,
    eqcolon: q$,
    eqsim: H$,
    eqslantgtr: B$,
    eqslantless: W$,
    Equal: U$,
    equals: j$,
    EqualTilde: V$,
    equest: G$,
    Equilibrium: K$,
    equiv: X$,
    equivDD: Y$,
    eqvparsl: Z$,
    erarr: J$,
    erDot: Q$,
    escr: tR,
    Escr: eR,
    esdot: nR,
    Esim: rR,
    esim: iR,
    Eta: oR,
    eta: sR,
    ETH: lR,
    eth: aR,
    Euml: cR,
    euml: uR,
    euro: fR,
    excl: hR,
    exist: dR,
    Exists: pR,
    expectation: gR,
    exponentiale: vR,
    ExponentialE: mR,
    fallingdotseq: yR,
    Fcy: bR,
    fcy: wR,
    female: xR,
    ffilig: _R,
    fflig: SR,
    ffllig: kR,
    Ffr: CR,
    ffr: TR,
    filig: ER,
    FilledSmallSquare: LR,
    FilledVerySmallSquare: AR,
    fjlig: MR,
    flat: NR,
    fllig: PR,
    fltns: OR,
    fnof: DR,
    Fopf: $R,
    fopf: RR,
    forall: zR,
    ForAll: IR,
    fork: FR,
    forkv: qR,
    Fouriertrf: HR,
    fpartint: BR,
    frac12: WR,
    frac13: UR,
    frac14: jR,
    frac15: VR,
    frac16: GR,
    frac18: KR,
    frac23: XR,
    frac25: YR,
    frac34: ZR,
    frac35: JR,
    frac38: QR,
    frac45: t2,
    frac56: e2,
    frac58: n2,
    frac78: r2,
    frasl: i2,
    frown: o2,
    fscr: s2,
    Fscr: l2,
    gacute: a2,
    Gamma: c2,
    gamma: u2,
    Gammad: f2,
    gammad: h2,
    gap: d2,
    Gbreve: p2,
    gbreve: g2,
    Gcedil: v2,
    Gcirc: m2,
    gcirc: y2,
    Gcy: b2,
    gcy: w2,
    Gdot: x2,
    gdot: _2,
    ge: S2,
    gE: k2,
    gEl: C2,
    gel: T2,
    geq: E2,
    geqq: L2,
    geqslant: A2,
    gescc: M2,
    ges: N2,
    gesdot: P2,
    gesdoto: O2,
    gesdotol: D2,
    gesl: $2,
    gesles: R2,
    Gfr: z2,
    gfr: I2,
    gg: F2,
    Gg: q2,
    ggg: H2,
    gimel: B2,
    GJcy: W2,
    gjcy: U2,
    gla: j2,
    gl: V2,
    glE: G2,
    glj: K2,
    gnap: X2,
    gnapprox: Y2,
    gne: Z2,
    gnE: J2,
    gneq: Q2,
    gneqq: tz,
    gnsim: ez,
    Gopf: nz,
    gopf: rz,
    grave: iz,
    GreaterEqual: oz,
    GreaterEqualLess: sz,
    GreaterFullEqual: lz,
    GreaterGreater: az,
    GreaterLess: cz,
    GreaterSlantEqual: uz,
    GreaterTilde: fz,
    Gscr: hz,
    gscr: dz,
    gsim: pz,
    gsime: gz,
    gsiml: vz,
    gtcc: mz,
    gtcir: yz,
    gt: bz,
    GT: wz,
    Gt: xz,
    gtdot: _z,
    gtlPar: Sz,
    gtquest: kz,
    gtrapprox: Cz,
    gtrarr: Tz,
    gtrdot: Ez,
    gtreqless: Lz,
    gtreqqless: Az,
    gtrless: Mz,
    gtrsim: Nz,
    gvertneqq: Pz,
    gvnE: Oz,
    Hacek: Dz,
    hairsp: $z,
    half: Rz,
    hamilt: zz,
    HARDcy: Iz,
    hardcy: Fz,
    harrcir: qz,
    harr: Hz,
    hArr: Bz,
    harrw: Wz,
    Hat: Uz,
    hbar: jz,
    Hcirc: Vz,
    hcirc: Gz,
    hearts: Kz,
    heartsuit: Xz,
    hellip: Yz,
    hercon: Zz,
    hfr: Jz,
    Hfr: Qz,
    HilbertSpace: tI,
    hksearow: eI,
    hkswarow: nI,
    hoarr: rI,
    homtht: iI,
    hookleftarrow: oI,
    hookrightarrow: sI,
    hopf: lI,
    Hopf: aI,
    horbar: cI,
    HorizontalLine: uI,
    hscr: fI,
    Hscr: hI,
    hslash: dI,
    Hstrok: pI,
    hstrok: gI,
    HumpDownHump: vI,
    HumpEqual: mI,
    hybull: yI,
    hyphen: bI,
    Iacute: wI,
    iacute: xI,
    ic: _I,
    Icirc: SI,
    icirc: kI,
    Icy: CI,
    icy: TI,
    Idot: EI,
    IEcy: LI,
    iecy: AI,
    iexcl: MI,
    iff: NI,
    ifr: PI,
    Ifr: OI,
    Igrave: DI,
    igrave: $I,
    ii: RI,
    iiiint: zI,
    iiint: II,
    iinfin: FI,
    iiota: qI,
    IJlig: HI,
    ijlig: BI,
    Imacr: WI,
    imacr: UI,
    image: jI,
    ImaginaryI: VI,
    imagline: GI,
    imagpart: KI,
    imath: XI,
    Im: YI,
    imof: ZI,
    imped: JI,
    Implies: QI,
    incare: tF,
    in: "∈",
    infin: eF,
    infintie: nF,
    inodot: rF,
    intcal: iF,
    int: oF,
    Int: sF,
    integers: lF,
    Integral: aF,
    intercal: cF,
    Intersection: uF,
    intlarhk: fF,
    intprod: hF,
    InvisibleComma: dF,
    InvisibleTimes: pF,
    IOcy: gF,
    iocy: vF,
    Iogon: mF,
    iogon: yF,
    Iopf: bF,
    iopf: wF,
    Iota: xF,
    iota: _F,
    iprod: SF,
    iquest: kF,
    iscr: CF,
    Iscr: TF,
    isin: EF,
    isindot: LF,
    isinE: AF,
    isins: MF,
    isinsv: NF,
    isinv: PF,
    it: OF,
    Itilde: DF,
    itilde: $F,
    Iukcy: RF,
    iukcy: zF,
    Iuml: IF,
    iuml: FF,
    Jcirc: qF,
    jcirc: HF,
    Jcy: BF,
    jcy: WF,
    Jfr: UF,
    jfr: jF,
    jmath: VF,
    Jopf: GF,
    jopf: KF,
    Jscr: XF,
    jscr: YF,
    Jsercy: ZF,
    jsercy: JF,
    Jukcy: QF,
    jukcy: tq,
    Kappa: eq,
    kappa: nq,
    kappav: rq,
    Kcedil: iq,
    kcedil: oq,
    Kcy: sq,
    kcy: lq,
    Kfr: aq,
    kfr: cq,
    kgreen: uq,
    KHcy: fq,
    khcy: hq,
    KJcy: dq,
    kjcy: pq,
    Kopf: gq,
    kopf: vq,
    Kscr: mq,
    kscr: yq,
    lAarr: bq,
    Lacute: wq,
    lacute: xq,
    laemptyv: _q,
    lagran: Sq,
    Lambda: kq,
    lambda: Cq,
    lang: Tq,
    Lang: Eq,
    langd: Lq,
    langle: Aq,
    lap: Mq,
    Laplacetrf: Nq,
    laquo: Pq,
    larrb: Oq,
    larrbfs: Dq,
    larr: $q,
    Larr: Rq,
    lArr: zq,
    larrfs: Iq,
    larrhk: Fq,
    larrlp: qq,
    larrpl: Hq,
    larrsim: Bq,
    larrtl: Wq,
    latail: Uq,
    lAtail: jq,
    lat: Vq,
    late: Gq,
    lates: Kq,
    lbarr: Xq,
    lBarr: Yq,
    lbbrk: Zq,
    lbrace: Jq,
    lbrack: Qq,
    lbrke: tH,
    lbrksld: eH,
    lbrkslu: nH,
    Lcaron: rH,
    lcaron: iH,
    Lcedil: oH,
    lcedil: sH,
    lceil: lH,
    lcub: aH,
    Lcy: cH,
    lcy: uH,
    ldca: fH,
    ldquo: hH,
    ldquor: dH,
    ldrdhar: pH,
    ldrushar: gH,
    ldsh: vH,
    le: mH,
    lE: yH,
    LeftAngleBracket: bH,
    LeftArrowBar: wH,
    leftarrow: xH,
    LeftArrow: _H,
    Leftarrow: SH,
    LeftArrowRightArrow: kH,
    leftarrowtail: CH,
    LeftCeiling: TH,
    LeftDoubleBracket: EH,
    LeftDownTeeVector: LH,
    LeftDownVectorBar: AH,
    LeftDownVector: MH,
    LeftFloor: NH,
    leftharpoondown: PH,
    leftharpoonup: OH,
    leftleftarrows: DH,
    leftrightarrow: $H,
    LeftRightArrow: RH,
    Leftrightarrow: zH,
    leftrightarrows: IH,
    leftrightharpoons: FH,
    leftrightsquigarrow: qH,
    LeftRightVector: HH,
    LeftTeeArrow: BH,
    LeftTee: WH,
    LeftTeeVector: UH,
    leftthreetimes: jH,
    LeftTriangleBar: VH,
    LeftTriangle: GH,
    LeftTriangleEqual: KH,
    LeftUpDownVector: XH,
    LeftUpTeeVector: YH,
    LeftUpVectorBar: ZH,
    LeftUpVector: JH,
    LeftVectorBar: QH,
    LeftVector: tB,
    lEg: eB,
    leg: nB,
    leq: rB,
    leqq: iB,
    leqslant: oB,
    lescc: sB,
    les: lB,
    lesdot: aB,
    lesdoto: cB,
    lesdotor: uB,
    lesg: fB,
    lesges: hB,
    lessapprox: dB,
    lessdot: pB,
    lesseqgtr: gB,
    lesseqqgtr: vB,
    LessEqualGreater: mB,
    LessFullEqual: yB,
    LessGreater: bB,
    lessgtr: wB,
    LessLess: xB,
    lesssim: _B,
    LessSlantEqual: SB,
    LessTilde: kB,
    lfisht: CB,
    lfloor: TB,
    Lfr: EB,
    lfr: LB,
    lg: AB,
    lgE: MB,
    lHar: NB,
    lhard: PB,
    lharu: OB,
    lharul: DB,
    lhblk: $B,
    LJcy: RB,
    ljcy: zB,
    llarr: IB,
    ll: FB,
    Ll: qB,
    llcorner: HB,
    Lleftarrow: BB,
    llhard: WB,
    lltri: UB,
    Lmidot: jB,
    lmidot: VB,
    lmoustache: GB,
    lmoust: KB,
    lnap: XB,
    lnapprox: YB,
    lne: ZB,
    lnE: JB,
    lneq: QB,
    lneqq: t3,
    lnsim: e3,
    loang: n3,
    loarr: r3,
    lobrk: i3,
    longleftarrow: o3,
    LongLeftArrow: s3,
    Longleftarrow: l3,
    longleftrightarrow: a3,
    LongLeftRightArrow: c3,
    Longleftrightarrow: u3,
    longmapsto: f3,
    longrightarrow: h3,
    LongRightArrow: d3,
    Longrightarrow: p3,
    looparrowleft: g3,
    looparrowright: v3,
    lopar: m3,
    Lopf: y3,
    lopf: b3,
    loplus: w3,
    lotimes: x3,
    lowast: _3,
    lowbar: S3,
    LowerLeftArrow: k3,
    LowerRightArrow: C3,
    loz: T3,
    lozenge: E3,
    lozf: L3,
    lpar: A3,
    lparlt: M3,
    lrarr: N3,
    lrcorner: P3,
    lrhar: O3,
    lrhard: D3,
    lrm: $3,
    lrtri: R3,
    lsaquo: z3,
    lscr: I3,
    Lscr: F3,
    lsh: q3,
    Lsh: H3,
    lsim: B3,
    lsime: W3,
    lsimg: U3,
    lsqb: j3,
    lsquo: V3,
    lsquor: G3,
    Lstrok: K3,
    lstrok: X3,
    ltcc: Y3,
    ltcir: Z3,
    lt: J3,
    LT: Q3,
    Lt: t5,
    ltdot: e5,
    lthree: n5,
    ltimes: r5,
    ltlarr: i5,
    ltquest: o5,
    ltri: s5,
    ltrie: l5,
    ltrif: a5,
    ltrPar: c5,
    lurdshar: u5,
    luruhar: f5,
    lvertneqq: h5,
    lvnE: d5,
    macr: p5,
    male: g5,
    malt: v5,
    maltese: m5,
    Map: "⤅",
    map: y5,
    mapsto: b5,
    mapstodown: w5,
    mapstoleft: x5,
    mapstoup: _5,
    marker: S5,
    mcomma: k5,
    Mcy: C5,
    mcy: T5,
    mdash: E5,
    mDDot: L5,
    measuredangle: A5,
    MediumSpace: M5,
    Mellintrf: N5,
    Mfr: P5,
    mfr: O5,
    mho: D5,
    micro: $5,
    midast: R5,
    midcir: z5,
    mid: I5,
    middot: F5,
    minusb: q5,
    minus: H5,
    minusd: B5,
    minusdu: W5,
    MinusPlus: U5,
    mlcp: j5,
    mldr: V5,
    mnplus: G5,
    models: K5,
    Mopf: X5,
    mopf: Y5,
    mp: Z5,
    mscr: J5,
    Mscr: Q5,
    mstpos: t8,
    Mu: e8,
    mu: n8,
    multimap: r8,
    mumap: i8,
    nabla: o8,
    Nacute: s8,
    nacute: l8,
    nang: a8,
    nap: c8,
    napE: u8,
    napid: f8,
    napos: h8,
    napprox: d8,
    natural: p8,
    naturals: g8,
    natur: v8,
    nbsp: m8,
    nbump: y8,
    nbumpe: b8,
    ncap: w8,
    Ncaron: x8,
    ncaron: _8,
    Ncedil: S8,
    ncedil: k8,
    ncong: C8,
    ncongdot: T8,
    ncup: E8,
    Ncy: L8,
    ncy: A8,
    ndash: M8,
    nearhk: N8,
    nearr: P8,
    neArr: O8,
    nearrow: D8,
    ne: $8,
    nedot: R8,
    NegativeMediumSpace: z8,
    NegativeThickSpace: I8,
    NegativeThinSpace: F8,
    NegativeVeryThinSpace: q8,
    nequiv: H8,
    nesear: B8,
    nesim: W8,
    NestedGreaterGreater: U8,
    NestedLessLess: j8,
    NewLine: V8,
    nexist: G8,
    nexists: K8,
    Nfr: X8,
    nfr: Y8,
    ngE: Z8,
    nge: J8,
    ngeq: Q8,
    ngeqq: tW,
    ngeqslant: eW,
    nges: nW,
    nGg: rW,
    ngsim: iW,
    nGt: oW,
    ngt: sW,
    ngtr: lW,
    nGtv: aW,
    nharr: cW,
    nhArr: uW,
    nhpar: fW,
    ni: hW,
    nis: dW,
    nisd: pW,
    niv: gW,
    NJcy: vW,
    njcy: mW,
    nlarr: yW,
    nlArr: bW,
    nldr: wW,
    nlE: xW,
    nle: _W,
    nleftarrow: SW,
    nLeftarrow: kW,
    nleftrightarrow: CW,
    nLeftrightarrow: TW,
    nleq: EW,
    nleqq: LW,
    nleqslant: AW,
    nles: MW,
    nless: NW,
    nLl: PW,
    nlsim: OW,
    nLt: DW,
    nlt: $W,
    nltri: RW,
    nltrie: zW,
    nLtv: IW,
    nmid: FW,
    NoBreak: qW,
    NonBreakingSpace: HW,
    nopf: BW,
    Nopf: WW,
    Not: UW,
    not: jW,
    NotCongruent: VW,
    NotCupCap: GW,
    NotDoubleVerticalBar: KW,
    NotElement: XW,
    NotEqual: YW,
    NotEqualTilde: ZW,
    NotExists: JW,
    NotGreater: QW,
    NotGreaterEqual: tU,
    NotGreaterFullEqual: eU,
    NotGreaterGreater: nU,
    NotGreaterLess: rU,
    NotGreaterSlantEqual: iU,
    NotGreaterTilde: oU,
    NotHumpDownHump: sU,
    NotHumpEqual: lU,
    notin: aU,
    notindot: cU,
    notinE: uU,
    notinva: fU,
    notinvb: hU,
    notinvc: dU,
    NotLeftTriangleBar: pU,
    NotLeftTriangle: gU,
    NotLeftTriangleEqual: vU,
    NotLess: mU,
    NotLessEqual: yU,
    NotLessGreater: bU,
    NotLessLess: wU,
    NotLessSlantEqual: xU,
    NotLessTilde: _U,
    NotNestedGreaterGreater: SU,
    NotNestedLessLess: kU,
    notni: CU,
    notniva: TU,
    notnivb: EU,
    notnivc: LU,
    NotPrecedes: AU,
    NotPrecedesEqual: MU,
    NotPrecedesSlantEqual: NU,
    NotReverseElement: PU,
    NotRightTriangleBar: OU,
    NotRightTriangle: DU,
    NotRightTriangleEqual: $U,
    NotSquareSubset: RU,
    NotSquareSubsetEqual: zU,
    NotSquareSuperset: IU,
    NotSquareSupersetEqual: FU,
    NotSubset: qU,
    NotSubsetEqual: HU,
    NotSucceeds: BU,
    NotSucceedsEqual: WU,
    NotSucceedsSlantEqual: UU,
    NotSucceedsTilde: jU,
    NotSuperset: VU,
    NotSupersetEqual: GU,
    NotTilde: KU,
    NotTildeEqual: XU,
    NotTildeFullEqual: YU,
    NotTildeTilde: ZU,
    NotVerticalBar: JU,
    nparallel: QU,
    npar: t4,
    nparsl: e4,
    npart: n4,
    npolint: r4,
    npr: i4,
    nprcue: o4,
    nprec: s4,
    npreceq: l4,
    npre: a4,
    nrarrc: c4,
    nrarr: u4,
    nrArr: f4,
    nrarrw: h4,
    nrightarrow: d4,
    nRightarrow: p4,
    nrtri: g4,
    nrtrie: v4,
    nsc: m4,
    nsccue: y4,
    nsce: b4,
    Nscr: w4,
    nscr: x4,
    nshortmid: _4,
    nshortparallel: S4,
    nsim: k4,
    nsime: C4,
    nsimeq: T4,
    nsmid: E4,
    nspar: L4,
    nsqsube: A4,
    nsqsupe: M4,
    nsub: N4,
    nsubE: P4,
    nsube: O4,
    nsubset: D4,
    nsubseteq: $4,
    nsubseteqq: R4,
    nsucc: z4,
    nsucceq: I4,
    nsup: F4,
    nsupE: q4,
    nsupe: H4,
    nsupset: B4,
    nsupseteq: W4,
    nsupseteqq: U4,
    ntgl: j4,
    Ntilde: V4,
    ntilde: G4,
    ntlg: K4,
    ntriangleleft: X4,
    ntrianglelefteq: Y4,
    ntriangleright: Z4,
    ntrianglerighteq: J4,
    Nu: Q4,
    nu: t6,
    num: e6,
    numero: n6,
    numsp: r6,
    nvap: i6,
    nvdash: o6,
    nvDash: s6,
    nVdash: l6,
    nVDash: a6,
    nvge: c6,
    nvgt: u6,
    nvHarr: f6,
    nvinfin: h6,
    nvlArr: d6,
    nvle: p6,
    nvlt: g6,
    nvltrie: v6,
    nvrArr: m6,
    nvrtrie: y6,
    nvsim: b6,
    nwarhk: w6,
    nwarr: x6,
    nwArr: _6,
    nwarrow: S6,
    nwnear: k6,
    Oacute: C6,
    oacute: T6,
    oast: E6,
    Ocirc: L6,
    ocirc: A6,
    ocir: M6,
    Ocy: N6,
    ocy: P6,
    odash: O6,
    Odblac: D6,
    odblac: $6,
    odiv: R6,
    odot: z6,
    odsold: I6,
    OElig: F6,
    oelig: q6,
    ofcir: H6,
    Ofr: B6,
    ofr: W6,
    ogon: U6,
    Ograve: j6,
    ograve: V6,
    ogt: G6,
    ohbar: K6,
    ohm: X6,
    oint: Y6,
    olarr: Z6,
    olcir: J6,
    olcross: Q6,
    oline: tj,
    olt: ej,
    Omacr: nj,
    omacr: rj,
    Omega: ij,
    omega: oj,
    Omicron: sj,
    omicron: lj,
    omid: aj,
    ominus: cj,
    Oopf: uj,
    oopf: fj,
    opar: hj,
    OpenCurlyDoubleQuote: dj,
    OpenCurlyQuote: pj,
    operp: gj,
    oplus: vj,
    orarr: mj,
    Or: yj,
    or: bj,
    ord: wj,
    order: xj,
    orderof: _j,
    ordf: Sj,
    ordm: kj,
    origof: Cj,
    oror: Tj,
    orslope: Ej,
    orv: Lj,
    oS: Aj,
    Oscr: Mj,
    oscr: Nj,
    Oslash: Pj,
    oslash: Oj,
    osol: Dj,
    Otilde: $j,
    otilde: Rj,
    otimesas: zj,
    Otimes: Ij,
    otimes: Fj,
    Ouml: qj,
    ouml: Hj,
    ovbar: Bj,
    OverBar: Wj,
    OverBrace: Uj,
    OverBracket: jj,
    OverParenthesis: Vj,
    para: Gj,
    parallel: Kj,
    par: Xj,
    parsim: Yj,
    parsl: Zj,
    part: Jj,
    PartialD: Qj,
    Pcy: t9,
    pcy: e9,
    percnt: n9,
    period: r9,
    permil: i9,
    perp: o9,
    pertenk: s9,
    Pfr: l9,
    pfr: a9,
    Phi: c9,
    phi: u9,
    phiv: f9,
    phmmat: h9,
    phone: d9,
    Pi: p9,
    pi: g9,
    pitchfork: v9,
    piv: m9,
    planck: y9,
    planckh: b9,
    plankv: w9,
    plusacir: x9,
    plusb: _9,
    pluscir: S9,
    plus: k9,
    plusdo: C9,
    plusdu: T9,
    pluse: E9,
    PlusMinus: L9,
    plusmn: A9,
    plussim: M9,
    plustwo: N9,
    pm: P9,
    Poincareplane: O9,
    pointint: D9,
    popf: $9,
    Popf: R9,
    pound: z9,
    prap: I9,
    Pr: F9,
    pr: q9,
    prcue: H9,
    precapprox: B9,
    prec: W9,
    preccurlyeq: U9,
    Precedes: j9,
    PrecedesEqual: V9,
    PrecedesSlantEqual: G9,
    PrecedesTilde: K9,
    preceq: X9,
    precnapprox: Y9,
    precneqq: Z9,
    precnsim: J9,
    pre: Q9,
    prE: tV,
    precsim: eV,
    prime: nV,
    Prime: rV,
    primes: iV,
    prnap: oV,
    prnE: sV,
    prnsim: lV,
    prod: aV,
    Product: cV,
    profalar: uV,
    profline: fV,
    profsurf: hV,
    prop: dV,
    Proportional: pV,
    Proportion: gV,
    propto: vV,
    prsim: mV,
    prurel: yV,
    Pscr: bV,
    pscr: wV,
    Psi: xV,
    psi: _V,
    puncsp: SV,
    Qfr: kV,
    qfr: CV,
    qint: TV,
    qopf: EV,
    Qopf: LV,
    qprime: AV,
    Qscr: MV,
    qscr: NV,
    quaternions: PV,
    quatint: OV,
    quest: DV,
    questeq: $V,
    quot: RV,
    QUOT: zV,
    rAarr: IV,
    race: FV,
    Racute: qV,
    racute: HV,
    radic: BV,
    raemptyv: WV,
    rang: UV,
    Rang: jV,
    rangd: VV,
    range: GV,
    rangle: KV,
    raquo: XV,
    rarrap: YV,
    rarrb: ZV,
    rarrbfs: JV,
    rarrc: QV,
    rarr: tG,
    Rarr: eG,
    rArr: nG,
    rarrfs: rG,
    rarrhk: iG,
    rarrlp: oG,
    rarrpl: sG,
    rarrsim: lG,
    Rarrtl: aG,
    rarrtl: cG,
    rarrw: uG,
    ratail: fG,
    rAtail: hG,
    ratio: dG,
    rationals: pG,
    rbarr: gG,
    rBarr: vG,
    RBarr: mG,
    rbbrk: yG,
    rbrace: bG,
    rbrack: wG,
    rbrke: xG,
    rbrksld: _G,
    rbrkslu: SG,
    Rcaron: kG,
    rcaron: CG,
    Rcedil: TG,
    rcedil: EG,
    rceil: LG,
    rcub: AG,
    Rcy: MG,
    rcy: NG,
    rdca: PG,
    rdldhar: OG,
    rdquo: DG,
    rdquor: $G,
    rdsh: RG,
    real: zG,
    realine: IG,
    realpart: FG,
    reals: qG,
    Re: HG,
    rect: BG,
    reg: WG,
    REG: UG,
    ReverseElement: jG,
    ReverseEquilibrium: VG,
    ReverseUpEquilibrium: GG,
    rfisht: KG,
    rfloor: XG,
    rfr: YG,
    Rfr: ZG,
    rHar: JG,
    rhard: QG,
    rharu: t7,
    rharul: e7,
    Rho: n7,
    rho: r7,
    rhov: i7,
    RightAngleBracket: o7,
    RightArrowBar: s7,
    rightarrow: l7,
    RightArrow: a7,
    Rightarrow: c7,
    RightArrowLeftArrow: u7,
    rightarrowtail: f7,
    RightCeiling: h7,
    RightDoubleBracket: d7,
    RightDownTeeVector: p7,
    RightDownVectorBar: g7,
    RightDownVector: v7,
    RightFloor: m7,
    rightharpoondown: y7,
    rightharpoonup: b7,
    rightleftarrows: w7,
    rightleftharpoons: x7,
    rightrightarrows: _7,
    rightsquigarrow: S7,
    RightTeeArrow: k7,
    RightTee: C7,
    RightTeeVector: T7,
    rightthreetimes: E7,
    RightTriangleBar: L7,
    RightTriangle: A7,
    RightTriangleEqual: M7,
    RightUpDownVector: N7,
    RightUpTeeVector: P7,
    RightUpVectorBar: O7,
    RightUpVector: D7,
    RightVectorBar: $7,
    RightVector: R7,
    ring: z7,
    risingdotseq: I7,
    rlarr: F7,
    rlhar: q7,
    rlm: H7,
    rmoustache: B7,
    rmoust: W7,
    rnmid: U7,
    roang: j7,
    roarr: V7,
    robrk: G7,
    ropar: K7,
    ropf: X7,
    Ropf: Y7,
    roplus: Z7,
    rotimes: J7,
    RoundImplies: Q7,
    rpar: tK,
    rpargt: eK,
    rppolint: nK,
    rrarr: rK,
    Rrightarrow: iK,
    rsaquo: oK,
    rscr: sK,
    Rscr: lK,
    rsh: aK,
    Rsh: cK,
    rsqb: uK,
    rsquo: fK,
    rsquor: hK,
    rthree: dK,
    rtimes: pK,
    rtri: gK,
    rtrie: vK,
    rtrif: mK,
    rtriltri: yK,
    RuleDelayed: bK,
    ruluhar: wK,
    rx: xK,
    Sacute: _K,
    sacute: SK,
    sbquo: kK,
    scap: CK,
    Scaron: TK,
    scaron: EK,
    Sc: LK,
    sc: AK,
    sccue: MK,
    sce: NK,
    scE: PK,
    Scedil: OK,
    scedil: DK,
    Scirc: $K,
    scirc: RK,
    scnap: zK,
    scnE: IK,
    scnsim: FK,
    scpolint: qK,
    scsim: HK,
    Scy: BK,
    scy: WK,
    sdotb: UK,
    sdot: jK,
    sdote: VK,
    searhk: GK,
    searr: KK,
    seArr: XK,
    searrow: YK,
    sect: ZK,
    semi: JK,
    seswar: QK,
    setminus: tX,
    setmn: eX,
    sext: nX,
    Sfr: rX,
    sfr: iX,
    sfrown: oX,
    sharp: sX,
    SHCHcy: lX,
    shchcy: aX,
    SHcy: cX,
    shcy: uX,
    ShortDownArrow: fX,
    ShortLeftArrow: hX,
    shortmid: dX,
    shortparallel: pX,
    ShortRightArrow: gX,
    ShortUpArrow: vX,
    shy: mX,
    Sigma: yX,
    sigma: bX,
    sigmaf: wX,
    sigmav: xX,
    sim: _X,
    simdot: SX,
    sime: kX,
    simeq: CX,
    simg: TX,
    simgE: EX,
    siml: LX,
    simlE: AX,
    simne: MX,
    simplus: NX,
    simrarr: PX,
    slarr: OX,
    SmallCircle: DX,
    smallsetminus: $X,
    smashp: RX,
    smeparsl: zX,
    smid: IX,
    smile: FX,
    smt: qX,
    smte: HX,
    smtes: BX,
    SOFTcy: WX,
    softcy: UX,
    solbar: jX,
    solb: VX,
    sol: GX,
    Sopf: KX,
    sopf: XX,
    spades: YX,
    spadesuit: ZX,
    spar: JX,
    sqcap: QX,
    sqcaps: tY,
    sqcup: eY,
    sqcups: nY,
    Sqrt: rY,
    sqsub: iY,
    sqsube: oY,
    sqsubset: sY,
    sqsubseteq: lY,
    sqsup: aY,
    sqsupe: cY,
    sqsupset: uY,
    sqsupseteq: fY,
    square: hY,
    Square: dY,
    SquareIntersection: pY,
    SquareSubset: gY,
    SquareSubsetEqual: vY,
    SquareSuperset: mY,
    SquareSupersetEqual: yY,
    SquareUnion: bY,
    squarf: wY,
    squ: xY,
    squf: _Y,
    srarr: SY,
    Sscr: kY,
    sscr: CY,
    ssetmn: TY,
    ssmile: EY,
    sstarf: LY,
    Star: AY,
    star: MY,
    starf: NY,
    straightepsilon: PY,
    straightphi: OY,
    strns: DY,
    sub: $Y,
    Sub: RY,
    subdot: zY,
    subE: IY,
    sube: FY,
    subedot: qY,
    submult: HY,
    subnE: BY,
    subne: WY,
    subplus: UY,
    subrarr: jY,
    subset: VY,
    Subset: GY,
    subseteq: KY,
    subseteqq: XY,
    SubsetEqual: YY,
    subsetneq: ZY,
    subsetneqq: JY,
    subsim: QY,
    subsub: tZ,
    subsup: eZ,
    succapprox: nZ,
    succ: rZ,
    succcurlyeq: iZ,
    Succeeds: oZ,
    SucceedsEqual: sZ,
    SucceedsSlantEqual: lZ,
    SucceedsTilde: aZ,
    succeq: cZ,
    succnapprox: uZ,
    succneqq: fZ,
    succnsim: hZ,
    succsim: dZ,
    SuchThat: pZ,
    sum: gZ,
    Sum: vZ,
    sung: mZ,
    sup1: yZ,
    sup2: bZ,
    sup3: wZ,
    sup: xZ,
    Sup: _Z,
    supdot: SZ,
    supdsub: kZ,
    supE: CZ,
    supe: TZ,
    supedot: EZ,
    Superset: LZ,
    SupersetEqual: AZ,
    suphsol: MZ,
    suphsub: NZ,
    suplarr: PZ,
    supmult: OZ,
    supnE: DZ,
    supne: $Z,
    supplus: RZ,
    supset: zZ,
    Supset: IZ,
    supseteq: FZ,
    supseteqq: qZ,
    supsetneq: HZ,
    supsetneqq: BZ,
    supsim: WZ,
    supsub: UZ,
    supsup: jZ,
    swarhk: VZ,
    swarr: GZ,
    swArr: KZ,
    swarrow: XZ,
    swnwar: YZ,
    szlig: ZZ,
    Tab: JZ,
    target: QZ,
    Tau: tJ,
    tau: eJ,
    tbrk: nJ,
    Tcaron: rJ,
    tcaron: iJ,
    Tcedil: oJ,
    tcedil: sJ,
    Tcy: lJ,
    tcy: aJ,
    tdot: cJ,
    telrec: uJ,
    Tfr: fJ,
    tfr: hJ,
    there4: dJ,
    therefore: pJ,
    Therefore: gJ,
    Theta: vJ,
    theta: mJ,
    thetasym: yJ,
    thetav: bJ,
    thickapprox: wJ,
    thicksim: xJ,
    ThickSpace: _J,
    ThinSpace: SJ,
    thinsp: kJ,
    thkap: CJ,
    thksim: TJ,
    THORN: EJ,
    thorn: LJ,
    tilde: AJ,
    Tilde: MJ,
    TildeEqual: NJ,
    TildeFullEqual: PJ,
    TildeTilde: OJ,
    timesbar: DJ,
    timesb: $J,
    times: RJ,
    timesd: zJ,
    tint: IJ,
    toea: FJ,
    topbot: qJ,
    topcir: HJ,
    top: BJ,
    Topf: WJ,
    topf: UJ,
    topfork: jJ,
    tosa: VJ,
    tprime: GJ,
    trade: KJ,
    TRADE: XJ,
    triangle: YJ,
    triangledown: ZJ,
    triangleleft: JJ,
    trianglelefteq: QJ,
    triangleq: tQ,
    triangleright: eQ,
    trianglerighteq: nQ,
    tridot: rQ,
    trie: iQ,
    triminus: oQ,
    TripleDot: sQ,
    triplus: lQ,
    trisb: aQ,
    tritime: cQ,
    trpezium: uQ,
    Tscr: fQ,
    tscr: hQ,
    TScy: dQ,
    tscy: pQ,
    TSHcy: gQ,
    tshcy: vQ,
    Tstrok: mQ,
    tstrok: yQ,
    twixt: bQ,
    twoheadleftarrow: wQ,
    twoheadrightarrow: xQ,
    Uacute: _Q,
    uacute: SQ,
    uarr: kQ,
    Uarr: CQ,
    uArr: TQ,
    Uarrocir: EQ,
    Ubrcy: LQ,
    ubrcy: AQ,
    Ubreve: MQ,
    ubreve: NQ,
    Ucirc: PQ,
    ucirc: OQ,
    Ucy: DQ,
    ucy: $Q,
    udarr: RQ,
    Udblac: zQ,
    udblac: IQ,
    udhar: FQ,
    ufisht: qQ,
    Ufr: HQ,
    ufr: BQ,
    Ugrave: WQ,
    ugrave: UQ,
    uHar: jQ,
    uharl: VQ,
    uharr: GQ,
    uhblk: KQ,
    ulcorn: XQ,
    ulcorner: YQ,
    ulcrop: ZQ,
    ultri: JQ,
    Umacr: QQ,
    umacr: ttt,
    uml: ett,
    UnderBar: ntt,
    UnderBrace: rtt,
    UnderBracket: itt,
    UnderParenthesis: ott,
    Union: stt,
    UnionPlus: ltt,
    Uogon: att,
    uogon: ctt,
    Uopf: utt,
    uopf: ftt,
    UpArrowBar: htt,
    uparrow: dtt,
    UpArrow: ptt,
    Uparrow: gtt,
    UpArrowDownArrow: vtt,
    updownarrow: mtt,
    UpDownArrow: ytt,
    Updownarrow: btt,
    UpEquilibrium: wtt,
    upharpoonleft: xtt,
    upharpoonright: _tt,
    uplus: Stt,
    UpperLeftArrow: ktt,
    UpperRightArrow: Ctt,
    upsi: Ttt,
    Upsi: Ett,
    upsih: Ltt,
    Upsilon: Att,
    upsilon: Mtt,
    UpTeeArrow: Ntt,
    UpTee: Ptt,
    upuparrows: Ott,
    urcorn: Dtt,
    urcorner: $tt,
    urcrop: Rtt,
    Uring: ztt,
    uring: Itt,
    urtri: Ftt,
    Uscr: qtt,
    uscr: Htt,
    utdot: Btt,
    Utilde: Wtt,
    utilde: Utt,
    utri: jtt,
    utrif: Vtt,
    uuarr: Gtt,
    Uuml: Ktt,
    uuml: Xtt,
    uwangle: Ytt,
    vangrt: Ztt,
    varepsilon: Jtt,
    varkappa: Qtt,
    varnothing: tet,
    varphi: eet,
    varpi: net,
    varpropto: ret,
    varr: iet,
    vArr: oet,
    varrho: set,
    varsigma: aet,
    varsubsetneq: cet,
    varsubsetneqq: uet,
    varsupsetneq: fet,
    varsupsetneqq: het,
    vartheta: det,
    vartriangleleft: pet,
    vartriangleright: get,
    vBar: vet,
    Vbar: met,
    vBarv: yet,
    Vcy: bet,
    vcy: wet,
    vdash: xet,
    vDash: _et,
    Vdash: ket,
    VDash: Cet,
    Vdashl: Tet,
    veebar: Eet,
    vee: Let,
    Vee: Aet,
    veeeq: Met,
    vellip: Net,
    verbar: Pet,
    Verbar: Oet,
    vert: Det,
    Vert: $et,
    VerticalBar: Ret,
    VerticalLine: zet,
    VerticalSeparator: Iet,
    VerticalTilde: Fet,
    VeryThinSpace: qet,
    Vfr: Het,
    vfr: Bet,
    vltri: Wet,
    vnsub: Uet,
    vnsup: jet,
    Vopf: Vet,
    vopf: Get,
    vprop: Ket,
    vrtri: Xet,
    Vscr: Yet,
    vscr: Zet,
    vsubnE: Jet,
    vsubne: Qet,
    vsupnE: tnt,
    vsupne: ent,
    Vvdash: nnt,
    vzigzag: rnt,
    Wcirc: int,
    wcirc: ont,
    wedbar: snt,
    wedge: lnt,
    Wedge: ant,
    wedgeq: cnt,
    weierp: unt,
    Wfr: fnt,
    wfr: hnt,
    Wopf: dnt,
    wopf: pnt,
    wp: gnt,
    wr: vnt,
    wreath: mnt,
    Wscr: ynt,
    wscr: bnt,
    xcap: wnt,
    xcirc: xnt,
    xcup: _nt,
    xdtri: Snt,
    Xfr: knt,
    xfr: Cnt,
    xharr: Tnt,
    xhArr: Ent,
    Xi: Lnt,
    xi: Ant,
    xlarr: Mnt,
    xlArr: Nnt,
    xmap: Pnt,
    xnis: Ont,
    xodot: Dnt,
    Xopf: $nt,
    xopf: Rnt,
    xoplus: znt,
    xotime: Int,
    xrarr: Fnt,
    xrArr: qnt,
    Xscr: Hnt,
    xscr: Bnt,
    xsqcup: Wnt,
    xuplus: Unt,
    xutri: jnt,
    xvee: Vnt,
    xwedge: Gnt,
    Yacute: Knt,
    yacute: Xnt,
    YAcy: Ynt,
    yacy: Znt,
    Ycirc: Jnt,
    ycirc: Qnt,
    Ycy: trt,
    ycy: ert,
    yen: nrt,
    Yfr: rrt,
    yfr: irt,
    YIcy: ort,
    yicy: srt,
    Yopf: lrt,
    yopf: art,
    Yscr: crt,
    yscr: urt,
    YUcy: frt,
    yucy: hrt,
    yuml: drt,
    Yuml: prt,
    Zacute: grt,
    zacute: vrt,
    Zcaron: mrt,
    zcaron: yrt,
    Zcy: brt,
    zcy: wrt,
    Zdot: xrt,
    zdot: _rt,
    zeetrf: Srt,
    ZeroWidthSpace: krt,
    Zeta: Crt,
    zeta: Trt,
    zfr: Ert,
    Zfr: Lrt,
    ZHcy: Art,
    zhcy: Mrt,
    zigrarr: Nrt,
    zopf: Prt,
    Zopf: Ort,
    Zscr: Drt,
    zscr: $rt,
    zwj: Rrt,
    zwnj: zrt,
  },
  Irt = "Á",
  Frt = "á",
  qrt = "Â",
  Hrt = "â",
  Brt = "´",
  Wrt = "Æ",
  Urt = "æ",
  jrt = "À",
  Vrt = "à",
  Grt = "&",
  Krt = "&",
  Xrt = "Å",
  Yrt = "å",
  Zrt = "Ã",
  Jrt = "ã",
  Qrt = "Ä",
  tit = "ä",
  eit = "¦",
  nit = "Ç",
  rit = "ç",
  iit = "¸",
  oit = "¢",
  sit = "©",
  lit = "©",
  ait = "¤",
  cit = "°",
  uit = "÷",
  fit = "É",
  hit = "é",
  dit = "Ê",
  pit = "ê",
  git = "È",
  vit = "è",
  mit = "Ð",
  yit = "ð",
  bit = "Ë",
  wit = "ë",
  xit = "½",
  _it = "¼",
  Sit = "¾",
  kit = ">",
  Cit = ">",
  Tit = "Í",
  Eit = "í",
  Lit = "Î",
  Ait = "î",
  Mit = "¡",
  Nit = "Ì",
  Pit = "ì",
  Oit = "¿",
  Dit = "Ï",
  $it = "ï",
  Rit = "«",
  zit = "<",
  Iit = "<",
  Fit = "¯",
  qit = "µ",
  Hit = "·",
  Bit = " ",
  Wit = "¬",
  Uit = "Ñ",
  jit = "ñ",
  Vit = "Ó",
  Git = "ó",
  Kit = "Ô",
  Xit = "ô",
  Yit = "Ò",
  Zit = "ò",
  Jit = "ª",
  Qit = "º",
  tot = "Ø",
  eot = "ø",
  not = "Õ",
  rot = "õ",
  iot = "Ö",
  oot = "ö",
  sot = "¶",
  lot = "±",
  aot = "£",
  cot = '"',
  uot = '"',
  fot = "»",
  hot = "®",
  dot = "®",
  pot = "§",
  got = "­",
  vot = "¹",
  mot = "²",
  yot = "³",
  bot = "ß",
  wot = "Þ",
  xot = "þ",
  _ot = "×",
  Sot = "Ú",
  kot = "ú",
  Cot = "Û",
  Tot = "û",
  Eot = "Ù",
  Lot = "ù",
  Aot = "¨",
  Mot = "Ü",
  Not = "ü",
  Pot = "Ý",
  Oot = "ý",
  Dot = "¥",
  $ot = "ÿ",
  Rot = {
    Aacute: Irt,
    aacute: Frt,
    Acirc: qrt,
    acirc: Hrt,
    acute: Brt,
    AElig: Wrt,
    aelig: Urt,
    Agrave: jrt,
    agrave: Vrt,
    amp: Grt,
    AMP: Krt,
    Aring: Xrt,
    aring: Yrt,
    Atilde: Zrt,
    atilde: Jrt,
    Auml: Qrt,
    auml: tit,
    brvbar: eit,
    Ccedil: nit,
    ccedil: rit,
    cedil: iit,
    cent: oit,
    copy: sit,
    COPY: lit,
    curren: ait,
    deg: cit,
    divide: uit,
    Eacute: fit,
    eacute: hit,
    Ecirc: dit,
    ecirc: pit,
    Egrave: git,
    egrave: vit,
    ETH: mit,
    eth: yit,
    Euml: bit,
    euml: wit,
    frac12: xit,
    frac14: _it,
    frac34: Sit,
    gt: kit,
    GT: Cit,
    Iacute: Tit,
    iacute: Eit,
    Icirc: Lit,
    icirc: Ait,
    iexcl: Mit,
    Igrave: Nit,
    igrave: Pit,
    iquest: Oit,
    Iuml: Dit,
    iuml: $it,
    laquo: Rit,
    lt: zit,
    LT: Iit,
    macr: Fit,
    micro: qit,
    middot: Hit,
    nbsp: Bit,
    not: Wit,
    Ntilde: Uit,
    ntilde: jit,
    Oacute: Vit,
    oacute: Git,
    Ocirc: Kit,
    ocirc: Xit,
    Ograve: Yit,
    ograve: Zit,
    ordf: Jit,
    ordm: Qit,
    Oslash: tot,
    oslash: eot,
    Otilde: not,
    otilde: rot,
    Ouml: iot,
    ouml: oot,
    para: sot,
    plusmn: lot,
    pound: aot,
    quot: cot,
    QUOT: uot,
    raquo: fot,
    reg: hot,
    REG: dot,
    sect: pot,
    shy: got,
    sup1: vot,
    sup2: mot,
    sup3: yot,
    szlig: bot,
    THORN: wot,
    thorn: xot,
    times: _ot,
    Uacute: Sot,
    uacute: kot,
    Ucirc: Cot,
    ucirc: Tot,
    Ugrave: Eot,
    ugrave: Lot,
    uml: Aot,
    Uuml: Mot,
    uuml: Not,
    Yacute: Pot,
    yacute: Oot,
    yen: Dot,
    yuml: $ot,
  },
  zot = "&",
  Iot = "'",
  Fot = ">",
  qot = "<",
  Hot = '"',
  yy = { amp: zot, apos: Iot, gt: Fot, lt: qot, quot: Hot };
var Wh = {};
const Bot = {
  0: 65533,
  128: 8364,
  130: 8218,
  131: 402,
  132: 8222,
  133: 8230,
  134: 8224,
  135: 8225,
  136: 710,
  137: 8240,
  138: 352,
  139: 8249,
  140: 338,
  142: 381,
  145: 8216,
  146: 8217,
  147: 8220,
  148: 8221,
  149: 8226,
  150: 8211,
  151: 8212,
  152: 732,
  153: 8482,
  154: 353,
  155: 8250,
  156: 339,
  158: 382,
  159: 376,
};
var Wot =
  (oo && oo.__importDefault) ||
  function (t) {
    return t && t.__esModule ? t : { default: t };
  };
Object.defineProperty(Wh, "__esModule", { value: !0 });
var Av = Wot(Bot),
  Uot =
    String.fromCodePoint ||
    function (t) {
      var e = "";
      return (
        t > 65535 &&
          ((t -= 65536),
          (e += String.fromCharCode(((t >>> 10) & 1023) | 55296)),
          (t = 56320 | (t & 1023))),
        (e += String.fromCharCode(t)),
        e
      );
    };
function jot(t) {
  return (t >= 55296 && t <= 57343) || t > 1114111
    ? "�"
    : (t in Av.default && (t = Av.default[t]), Uot(t));
}
Wh.default = jot;
var Uc =
  (oo && oo.__importDefault) ||
  function (t) {
    return t && t.__esModule ? t : { default: t };
  };
Object.defineProperty(Hr, "__esModule", { value: !0 });
Hr.decodeHTML = Hr.decodeHTMLStrict = Hr.decodeXML = void 0;
var Gf = Uc(my),
  Vot = Uc(Rot),
  Got = Uc(yy),
  Mv = Uc(Wh),
  Kot = /&(?:[a-zA-Z0-9]+|#[xX][\da-fA-F]+|#\d+);/g;
Hr.decodeXML = by(Got.default);
Hr.decodeHTMLStrict = by(Gf.default);
function by(t) {
  var e = wy(t);
  return function (r) {
    return String(r).replace(Kot, e);
  };
}
var Nv = function (t, e) {
  return t < e ? 1 : -1;
};
Hr.decodeHTML = (function () {
  for (
    var t = Object.keys(Vot.default).sort(Nv), e = Object.keys(Gf.default).sort(Nv), r = 0, o = 0;
    r < e.length;
    r++
  )
    t[o] === e[r] ? ((e[r] += ";?"), o++) : (e[r] += ";");
  var s = new RegExp("&(?:" + e.join("|") + "|#[xX][\\da-fA-F]+;?|#\\d+;?)", "g"),
    u = wy(Gf.default);
  function f(h) {
    return h.substr(-1) !== ";" && (h += ";"), u(h);
  }
  return function (h) {
    return String(h).replace(s, f);
  };
})();
function wy(t) {
  return function (r) {
    if (r.charAt(1) === "#") {
      var o = r.charAt(2);
      return o === "X" || o === "x"
        ? Mv.default(parseInt(r.substr(3), 16))
        : Mv.default(parseInt(r.substr(2), 10));
    }
    return t[r.slice(1, -1)] || r;
  };
}
var zn = {},
  xy =
    (oo && oo.__importDefault) ||
    function (t) {
      return t && t.__esModule ? t : { default: t };
    };
Object.defineProperty(zn, "__esModule", { value: !0 });
zn.escapeUTF8 = zn.escape = zn.encodeNonAsciiHTML = zn.encodeHTML = zn.encodeXML = void 0;
var Xot = xy(yy),
  _y = ky(Xot.default),
  Sy = Cy(_y);
zn.encodeXML = Ly(_y);
var Yot = xy(my),
  Uh = ky(Yot.default),
  Zot = Cy(Uh);
zn.encodeHTML = Qot(Uh, Zot);
zn.encodeNonAsciiHTML = Ly(Uh);
function ky(t) {
  return Object.keys(t)
    .sort()
    .reduce(function (e, r) {
      return (e[t[r]] = "&" + r + ";"), e;
    }, {});
}
function Cy(t) {
  for (var e = [], r = [], o = 0, s = Object.keys(t); o < s.length; o++) {
    var u = s[o];
    u.length === 1 ? e.push("\\" + u) : r.push(u);
  }
  e.sort();
  for (var f = 0; f < e.length - 1; f++) {
    for (var h = f; h < e.length - 1 && e[h].charCodeAt(1) + 1 === e[h + 1].charCodeAt(1); ) h += 1;
    var d = 1 + h - f;
    d < 3 || e.splice(f, d, e[f] + "-" + e[h]);
  }
  return r.unshift("[" + e.join("") + "]"), new RegExp(r.join("|"), "g");
}
var Ty =
    /(?:[\x80-\uD7FF\uE000-\uFFFF]|[\uD800-\uDBFF][\uDC00-\uDFFF]|[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?:[^\uD800-\uDBFF]|^)[\uDC00-\uDFFF])/g,
  Jot =
    String.prototype.codePointAt != null
      ? function (t) {
          return t.codePointAt(0);
        }
      : function (t) {
          return (t.charCodeAt(0) - 55296) * 1024 + t.charCodeAt(1) - 56320 + 65536;
        };
function jc(t) {
  return "&#x" + (t.length > 1 ? Jot(t) : t.charCodeAt(0)).toString(16).toUpperCase() + ";";
}
function Qot(t, e) {
  return function (r) {
    return r
      .replace(e, function (o) {
        return t[o];
      })
      .replace(Ty, jc);
  };
}
var Ey = new RegExp(Sy.source + "|" + Ty.source, "g");
function tst(t) {
  return t.replace(Ey, jc);
}
zn.escape = tst;
function est(t) {
  return t.replace(Sy, jc);
}
zn.escapeUTF8 = est;
function Ly(t) {
  return function (e) {
    return e.replace(Ey, function (r) {
      return t[r] || jc(r);
    });
  };
}
(function (t) {
  Object.defineProperty(t, "__esModule", { value: !0 }),
    (t.decodeXMLStrict =
      t.decodeHTML5Strict =
      t.decodeHTML4Strict =
      t.decodeHTML5 =
      t.decodeHTML4 =
      t.decodeHTMLStrict =
      t.decodeHTML =
      t.decodeXML =
      t.encodeHTML5 =
      t.encodeHTML4 =
      t.escapeUTF8 =
      t.escape =
      t.encodeNonAsciiHTML =
      t.encodeHTML =
      t.encodeXML =
      t.encode =
      t.decodeStrict =
      t.decode =
        void 0);
  var e = Hr,
    r = zn;
  function o(d, g) {
    return (!g || g <= 0 ? e.decodeXML : e.decodeHTML)(d);
  }
  t.decode = o;
  function s(d, g) {
    return (!g || g <= 0 ? e.decodeXML : e.decodeHTMLStrict)(d);
  }
  t.decodeStrict = s;
  function u(d, g) {
    return (!g || g <= 0 ? r.encodeXML : r.encodeHTML)(d);
  }
  t.encode = u;
  var f = zn;
  Object.defineProperty(t, "encodeXML", {
    enumerable: !0,
    get: function () {
      return f.encodeXML;
    },
  }),
    Object.defineProperty(t, "encodeHTML", {
      enumerable: !0,
      get: function () {
        return f.encodeHTML;
      },
    }),
    Object.defineProperty(t, "encodeNonAsciiHTML", {
      enumerable: !0,
      get: function () {
        return f.encodeNonAsciiHTML;
      },
    }),
    Object.defineProperty(t, "escape", {
      enumerable: !0,
      get: function () {
        return f.escape;
      },
    }),
    Object.defineProperty(t, "escapeUTF8", {
      enumerable: !0,
      get: function () {
        return f.escapeUTF8;
      },
    }),
    Object.defineProperty(t, "encodeHTML4", {
      enumerable: !0,
      get: function () {
        return f.encodeHTML;
      },
    }),
    Object.defineProperty(t, "encodeHTML5", {
      enumerable: !0,
      get: function () {
        return f.encodeHTML;
      },
    });
  var h = Hr;
  Object.defineProperty(t, "decodeXML", {
    enumerable: !0,
    get: function () {
      return h.decodeXML;
    },
  }),
    Object.defineProperty(t, "decodeHTML", {
      enumerable: !0,
      get: function () {
        return h.decodeHTML;
      },
    }),
    Object.defineProperty(t, "decodeHTMLStrict", {
      enumerable: !0,
      get: function () {
        return h.decodeHTMLStrict;
      },
    }),
    Object.defineProperty(t, "decodeHTML4", {
      enumerable: !0,
      get: function () {
        return h.decodeHTML;
      },
    }),
    Object.defineProperty(t, "decodeHTML5", {
      enumerable: !0,
      get: function () {
        return h.decodeHTML;
      },
    }),
    Object.defineProperty(t, "decodeHTML4Strict", {
      enumerable: !0,
      get: function () {
        return h.decodeHTMLStrict;
      },
    }),
    Object.defineProperty(t, "decodeHTML5Strict", {
      enumerable: !0,
      get: function () {
        return h.decodeHTMLStrict;
      },
    }),
    Object.defineProperty(t, "decodeXMLStrict", {
      enumerable: !0,
      get: function () {
        return h.decodeXML;
      },
    });
})(vy);
function nst(t, e) {
  if (!(t instanceof e)) throw new TypeError("Cannot call a class as a function");
}
function Pv(t, e) {
  for (var r = 0; r < e.length; r++) {
    var o = e[r];
    (o.enumerable = o.enumerable || !1),
      (o.configurable = !0),
      "value" in o && (o.writable = !0),
      Object.defineProperty(t, o.key, o);
  }
}
function rst(t, e, r) {
  return e && Pv(t.prototype, e), r && Pv(t, r), t;
}
function Ay(t, e) {
  var r = (typeof Symbol < "u" && t[Symbol.iterator]) || t["@@iterator"];
  if (!r) {
    if (Array.isArray(t) || (r = ist(t)) || (e && t && typeof t.length == "number")) {
      r && (t = r);
      var o = 0,
        s = function () {};
      return {
        s,
        n: function () {
          return o >= t.length ? { done: !0 } : { done: !1, value: t[o++] };
        },
        e: function (g) {
          throw g;
        },
        f: s,
      };
    }
    throw new TypeError(`Invalid attempt to iterate non-iterable instance.
In order to be iterable, non-array objects must have a [Symbol.iterator]() method.`);
  }
  var u = !0,
    f = !1,
    h;
  return {
    s: function () {
      r = r.call(t);
    },
    n: function () {
      var g = r.next();
      return (u = g.done), g;
    },
    e: function (g) {
      (f = !0), (h = g);
    },
    f: function () {
      try {
        !u && r.return != null && r.return();
      } finally {
        if (f) throw h;
      }
    },
  };
}
function ist(t, e) {
  if (t) {
    if (typeof t == "string") return Ov(t, e);
    var r = Object.prototype.toString.call(t).slice(8, -1);
    if ((r === "Object" && t.constructor && (r = t.constructor.name), r === "Map" || r === "Set"))
      return Array.from(t);
    if (r === "Arguments" || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(r)) return Ov(t, e);
  }
}
function Ov(t, e) {
  (e == null || e > t.length) && (e = t.length);
  for (var r = 0, o = new Array(e); r < e; r++) o[r] = t[r];
  return o;
}
var ost = vy,
  Dv = { fg: "#FFF", bg: "#000", newline: !1, escapeXML: !1, stream: !1, colors: sst() };
function sst() {
  var t = {
    0: "#000",
    1: "#A00",
    2: "#0A0",
    3: "#A50",
    4: "#00A",
    5: "#A0A",
    6: "#0AA",
    7: "#AAA",
    8: "#555",
    9: "#F55",
    10: "#5F5",
    11: "#FF5",
    12: "#55F",
    13: "#F5F",
    14: "#5FF",
    15: "#FFF",
  };
  return (
    Ea(0, 5).forEach(function (e) {
      Ea(0, 5).forEach(function (r) {
        Ea(0, 5).forEach(function (o) {
          return lst(e, r, o, t);
        });
      });
    }),
    Ea(0, 23).forEach(function (e) {
      var r = e + 232,
        o = My(e * 10 + 8);
      t[r] = "#" + o + o + o;
    }),
    t
  );
}
function lst(t, e, r, o) {
  var s = 16 + t * 36 + e * 6 + r,
    u = t > 0 ? t * 40 + 55 : 0,
    f = e > 0 ? e * 40 + 55 : 0,
    h = r > 0 ? r * 40 + 55 : 0;
  o[s] = ast([u, f, h]);
}
function My(t) {
  for (var e = t.toString(16); e.length < 2; ) e = "0" + e;
  return e;
}
function ast(t) {
  var e = [],
    r = Ay(t),
    o;
  try {
    for (r.s(); !(o = r.n()).done; ) {
      var s = o.value;
      e.push(My(s));
    }
  } catch (u) {
    r.e(u);
  } finally {
    r.f();
  }
  return "#" + e.join("");
}
function $v(t, e, r, o) {
  var s;
  return (
    e === "text"
      ? (s = hst(r, o))
      : e === "display"
      ? (s = ust(t, r, o))
      : e === "xterm256Foreground"
      ? (s = Ua(t, o.colors[r]))
      : e === "xterm256Background"
      ? (s = ja(t, o.colors[r]))
      : e === "rgb" && (s = cst(t, r)),
    s
  );
}
function cst(t, e) {
  e = e.substring(2).slice(0, -1);
  var r = +e.substr(0, 2),
    o = e.substring(5).split(";"),
    s = o
      .map(function (u) {
        return ("0" + Number(u).toString(16)).substr(-2);
      })
      .join("");
  return Wa(t, (r === 38 ? "color:#" : "background-color:#") + s);
}
function ust(t, e, r) {
  e = parseInt(e, 10);
  var o = {
      "-1": function () {
        return "<br/>";
      },
      0: function () {
        return t.length && Ny(t);
      },
      1: function () {
        return wi(t, "b");
      },
      3: function () {
        return wi(t, "i");
      },
      4: function () {
        return wi(t, "u");
      },
      8: function () {
        return Wa(t, "display:none");
      },
      9: function () {
        return wi(t, "strike");
      },
      22: function () {
        return Wa(t, "font-weight:normal;text-decoration:none;font-style:normal");
      },
      23: function () {
        return zv(t, "i");
      },
      24: function () {
        return zv(t, "u");
      },
      39: function () {
        return Ua(t, r.fg);
      },
      49: function () {
        return ja(t, r.bg);
      },
      53: function () {
        return Wa(t, "text-decoration:overline");
      },
    },
    s;
  return (
    o[e]
      ? (s = o[e]())
      : 4 < e && e < 7
      ? (s = wi(t, "blink"))
      : 29 < e && e < 38
      ? (s = Ua(t, r.colors[e - 30]))
      : 39 < e && e < 48
      ? (s = ja(t, r.colors[e - 40]))
      : 89 < e && e < 98
      ? (s = Ua(t, r.colors[8 + (e - 90)]))
      : 99 < e && e < 108 && (s = ja(t, r.colors[8 + (e - 100)])),
    s
  );
}
function Ny(t) {
  var e = t.slice(0);
  return (
    (t.length = 0),
    e
      .reverse()
      .map(function (r) {
        return "</" + r + ">";
      })
      .join("")
  );
}
function Ea(t, e) {
  for (var r = [], o = t; o <= e; o++) r.push(o);
  return r;
}
function fst(t) {
  return function (e) {
    return (t === null || e.category !== t) && t !== "all";
  };
}
function Rv(t) {
  t = parseInt(t, 10);
  var e = null;
  return (
    t === 0
      ? (e = "all")
      : t === 1
      ? (e = "bold")
      : 2 < t && t < 5
      ? (e = "underline")
      : 4 < t && t < 7
      ? (e = "blink")
      : t === 8
      ? (e = "hide")
      : t === 9
      ? (e = "strike")
      : (29 < t && t < 38) || t === 39 || (89 < t && t < 98)
      ? (e = "foreground-color")
      : ((39 < t && t < 48) || t === 49 || (99 < t && t < 108)) && (e = "background-color"),
    e
  );
}
function hst(t, e) {
  return e.escapeXML ? ost.encodeXML(t) : t;
}
function wi(t, e, r) {
  return r || (r = ""), t.push(e), "<".concat(e).concat(r ? ' style="'.concat(r, '"') : "", ">");
}
function Wa(t, e) {
  return wi(t, "span", e);
}
function Ua(t, e) {
  return wi(t, "span", "color:" + e);
}
function ja(t, e) {
  return wi(t, "span", "background-color:" + e);
}
function zv(t, e) {
  var r;
  if ((t.slice(-1)[0] === e && (r = t.pop()), r)) return "</" + e + ">";
}
function dst(t, e, r) {
  var o = !1,
    s = 3;
  function u() {
    return "";
  }
  function f(B, K) {
    return r("xterm256Foreground", K), "";
  }
  function h(B, K) {
    return r("xterm256Background", K), "";
  }
  function d(B) {
    return e.newline ? r("display", -1) : r("text", B), "";
  }
  function g(B, K) {
    (o = !0), K.trim().length === 0 && (K = "0"), (K = K.trimRight(";").split(";"));
    var ht = Ay(K),
      Y;
    try {
      for (ht.s(); !(Y = ht.n()).done; ) {
        var nt = Y.value;
        r("display", nt);
      }
    } catch (at) {
      ht.e(at);
    } finally {
      ht.f();
    }
    return "";
  }
  function v(B) {
    return r("text", B), "";
  }
  function b(B) {
    return r("rgb", B), "";
  }
  var w = [
    { pattern: /^\x08+/, sub: u },
    { pattern: /^\x1b\[[012]?K/, sub: u },
    { pattern: /^\x1b\[\(B/, sub: u },
    { pattern: /^\x1b\[[34]8;2;\d+;\d+;\d+m/, sub: b },
    { pattern: /^\x1b\[38;5;(\d+)m/, sub: f },
    { pattern: /^\x1b\[48;5;(\d+)m/, sub: h },
    { pattern: /^\n/, sub: d },
    { pattern: /^\r+\n/, sub: d },
    { pattern: /^\r/, sub: d },
    { pattern: /^\x1b\[((?:\d{1,3};?)+|)m/, sub: g },
    { pattern: /^\x1b\[\d?J/, sub: u },
    { pattern: /^\x1b\[\d{0,3};\d{0,3}f/, sub: u },
    { pattern: /^\x1b\[?[\d;]{0,3}/, sub: u },
    { pattern: /^(([^\x1b\x08\r\n])+)/, sub: v },
  ];
  function S(B, K) {
    (K > s && o) || ((o = !1), (t = t.replace(B.pattern, B.sub)));
  }
  var P = [],
    A = t,
    L = A.length;
  t: for (; L > 0; ) {
    for (var T = 0, M = 0, R = w.length; M < R; T = ++M) {
      var E = w[T];
      if ((S(E, T), t.length !== L)) {
        L = t.length;
        continue t;
      }
    }
    if (t.length === L) break;
    P.push(0), (L = t.length);
  }
  return P;
}
function pst(t, e, r) {
  return (
    e !== "text" && ((t = t.filter(fst(Rv(r)))), t.push({ token: e, data: r, category: Rv(r) })), t
  );
}
var gst = (function () {
    function t(e) {
      nst(this, t),
        (e = e || {}),
        e.colors && (e.colors = Object.assign({}, Dv.colors, e.colors)),
        (this.options = Object.assign({}, Dv, e)),
        (this.stack = []),
        (this.stickyStack = []);
    }
    return (
      rst(t, [
        {
          key: "toHtml",
          value: function (r) {
            var o = this;
            r = typeof r == "string" ? [r] : r;
            var s = this.stack,
              u = this.options,
              f = [];
            return (
              this.stickyStack.forEach(function (h) {
                var d = $v(s, h.token, h.data, u);
                d && f.push(d);
              }),
              dst(r.join(""), u, function (h, d) {
                var g = $v(s, h, d, u);
                g && f.push(g), u.stream && (o.stickyStack = pst(o.stickyStack, h, d));
              }),
              s.length && f.push(Ny(s)),
              f.join("")
            );
          },
        },
      ]),
      t
    );
  })(),
  vst = gst;
const mst = hy(vst);
function yst(t = "") {
  return !t || !t.includes("\\") ? t : t.replace(/\\/g, "/");
}
const bst = /^[/\\](?![/\\])|^[/\\]{2}(?!\.)|^[A-Za-z]:[/\\]/;
function wst() {
  return typeof process < "u" ? process.cwd().replace(/\\/g, "/") : "/";
}
const xst = function (...t) {
  t = t.map((o) => yst(o));
  let e = "",
    r = !1;
  for (let o = t.length - 1; o >= -1 && !r; o--) {
    const s = o >= 0 ? t[o] : wst();
    !s || s.length === 0 || ((e = `${s}/${e}`), (r = Iv(s)));
  }
  return (e = _st(e, !r)), r && !Iv(e) ? `/${e}` : e.length > 0 ? e : ".";
};
function _st(t, e) {
  let r = "",
    o = 0,
    s = -1,
    u = 0,
    f = null;
  for (let h = 0; h <= t.length; ++h) {
    if (h < t.length) f = t[h];
    else {
      if (f === "/") break;
      f = "/";
    }
    if (f === "/") {
      if (!(s === h - 1 || u === 1))
        if (u === 2) {
          if (r.length < 2 || o !== 2 || r[r.length - 1] !== "." || r[r.length - 2] !== ".") {
            if (r.length > 2) {
              const d = r.lastIndexOf("/");
              d === -1
                ? ((r = ""), (o = 0))
                : ((r = r.slice(0, d)), (o = r.length - 1 - r.lastIndexOf("/"))),
                (s = h),
                (u = 0);
              continue;
            } else if (r.length > 0) {
              (r = ""), (o = 0), (s = h), (u = 0);
              continue;
            }
          }
          e && ((r += r.length > 0 ? "/.." : ".."), (o = 2));
        } else
          r.length > 0 ? (r += `/${t.slice(s + 1, h)}`) : (r = t.slice(s + 1, h)), (o = h - s - 1);
      (s = h), (u = 0);
    } else f === "." && u !== -1 ? ++u : (u = -1);
  }
  return r;
}
const Iv = function (t) {
    return bst.test(t);
  },
  Sst = ",".charCodeAt(0),
  Fv = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/",
  kst = new Uint8Array(64),
  Py = new Uint8Array(128);
for (let t = 0; t < Fv.length; t++) {
  const e = Fv.charCodeAt(t);
  (kst[t] = e), (Py[e] = t);
}
function Cst(t) {
  const e = new Int32Array(5),
    r = [];
  let o = 0;
  do {
    const s = Tst(t, o),
      u = [];
    let f = !0,
      h = 0;
    e[0] = 0;
    for (let d = o; d < s; d++) {
      let g;
      d = tl(t, d, e, 0);
      const v = e[0];
      v < h && (f = !1),
        (h = v),
        qv(t, d, s)
          ? ((d = tl(t, d, e, 1)),
            (d = tl(t, d, e, 2)),
            (d = tl(t, d, e, 3)),
            qv(t, d, s)
              ? ((d = tl(t, d, e, 4)), (g = [v, e[1], e[2], e[3], e[4]]))
              : (g = [v, e[1], e[2], e[3]]))
          : (g = [v]),
        u.push(g);
    }
    f || Est(u), r.push(u), (o = s + 1);
  } while (o <= t.length);
  return r;
}
function Tst(t, e) {
  const r = t.indexOf(";", e);
  return r === -1 ? t.length : r;
}
function tl(t, e, r, o) {
  let s = 0,
    u = 0,
    f = 0;
  do {
    const d = t.charCodeAt(e++);
    (f = Py[d]), (s |= (f & 31) << u), (u += 5);
  } while (f & 32);
  const h = s & 1;
  return (s >>>= 1), h && (s = -2147483648 | -s), (r[o] += s), e;
}
function qv(t, e, r) {
  return e >= r ? !1 : t.charCodeAt(e) !== Sst;
}
function Est(t) {
  t.sort(Lst);
}
function Lst(t, e) {
  return t[0] - e[0];
}
const Ast = /^[\w+.-]+:\/\//,
  Mst = /^([\w+.-]+:)\/\/([^@/#?]*@)?([^:/#?]*)(:\d+)?(\/[^#?]*)?(\?[^#]*)?(#.*)?/,
  Nst = /^file:(?:\/\/((?![a-z]:)[^/#?]*)?)?(\/?[^#?]*)(\?[^#]*)?(#.*)?/i;
var De;
(function (t) {
  (t[(t.Empty = 1)] = "Empty"),
    (t[(t.Hash = 2)] = "Hash"),
    (t[(t.Query = 3)] = "Query"),
    (t[(t.RelativePath = 4)] = "RelativePath"),
    (t[(t.AbsolutePath = 5)] = "AbsolutePath"),
    (t[(t.SchemeRelative = 6)] = "SchemeRelative"),
    (t[(t.Absolute = 7)] = "Absolute");
})(De || (De = {}));
function Pst(t) {
  return Ast.test(t);
}
function Ost(t) {
  return t.startsWith("//");
}
function Oy(t) {
  return t.startsWith("/");
}
function Dst(t) {
  return t.startsWith("file:");
}
function Hv(t) {
  return /^[.?#]/.test(t);
}
function La(t) {
  const e = Mst.exec(t);
  return Dy(e[1], e[2] || "", e[3], e[4] || "", e[5] || "/", e[6] || "", e[7] || "");
}
function $st(t) {
  const e = Nst.exec(t),
    r = e[2];
  return Dy("file:", "", e[1] || "", "", Oy(r) ? r : "/" + r, e[3] || "", e[4] || "");
}
function Dy(t, e, r, o, s, u, f) {
  return { scheme: t, user: e, host: r, port: o, path: s, query: u, hash: f, type: De.Absolute };
}
function Bv(t) {
  if (Ost(t)) {
    const r = La("http:" + t);
    return (r.scheme = ""), (r.type = De.SchemeRelative), r;
  }
  if (Oy(t)) {
    const r = La("http://foo.com" + t);
    return (r.scheme = ""), (r.host = ""), (r.type = De.AbsolutePath), r;
  }
  if (Dst(t)) return $st(t);
  if (Pst(t)) return La(t);
  const e = La("http://foo.com/" + t);
  return (
    (e.scheme = ""),
    (e.host = ""),
    (e.type = t
      ? t.startsWith("?")
        ? De.Query
        : t.startsWith("#")
        ? De.Hash
        : De.RelativePath
      : De.Empty),
    e
  );
}
function Rst(t) {
  if (t.endsWith("/..")) return t;
  const e = t.lastIndexOf("/");
  return t.slice(0, e + 1);
}
function zst(t, e) {
  $y(e, e.type), t.path === "/" ? (t.path = e.path) : (t.path = Rst(e.path) + t.path);
}
function $y(t, e) {
  const r = e <= De.RelativePath,
    o = t.path.split("/");
  let s = 1,
    u = 0,
    f = !1;
  for (let d = 1; d < o.length; d++) {
    const g = o[d];
    if (!g) {
      f = !0;
      continue;
    }
    if (((f = !1), g !== ".")) {
      if (g === "..") {
        u ? ((f = !0), u--, s--) : r && (o[s++] = g);
        continue;
      }
      (o[s++] = g), u++;
    }
  }
  let h = "";
  for (let d = 1; d < s; d++) h += "/" + o[d];
  (!h || (f && !h.endsWith("/.."))) && (h += "/"), (t.path = h);
}
function Ist(t, e) {
  if (!t && !e) return "";
  const r = Bv(t);
  let o = r.type;
  if (e && o !== De.Absolute) {
    const u = Bv(e),
      f = u.type;
    switch (o) {
      case De.Empty:
        r.hash = u.hash;
      case De.Hash:
        r.query = u.query;
      case De.Query:
      case De.RelativePath:
        zst(r, u);
      case De.AbsolutePath:
        (r.user = u.user), (r.host = u.host), (r.port = u.port);
      case De.SchemeRelative:
        r.scheme = u.scheme;
    }
    f > o && (o = f);
  }
  $y(r, o);
  const s = r.query + r.hash;
  switch (o) {
    case De.Hash:
    case De.Query:
      return s;
    case De.RelativePath: {
      const u = r.path.slice(1);
      return u ? (Hv(e || t) && !Hv(u) ? "./" + u + s : u + s) : s || ".";
    }
    case De.AbsolutePath:
      return r.path + s;
    default:
      return r.scheme + "//" + r.user + r.host + r.port + r.path + s;
  }
}
function Wv(t, e) {
  return e && !e.endsWith("/") && (e += "/"), Ist(t, e);
}
function Fst(t) {
  if (!t) return "";
  const e = t.lastIndexOf("/");
  return t.slice(0, e + 1);
}
const Li = 0,
  qst = 1,
  Hst = 2,
  Bst = 3,
  Wst = 4;
function Ust(t, e) {
  const r = Uv(t, 0);
  if (r === t.length) return t;
  e || (t = t.slice());
  for (let o = r; o < t.length; o = Uv(t, o + 1)) t[o] = Vst(t[o], e);
  return t;
}
function Uv(t, e) {
  for (let r = e; r < t.length; r++) if (!jst(t[r])) return r;
  return t.length;
}
function jst(t) {
  for (let e = 1; e < t.length; e++) if (t[e][Li] < t[e - 1][Li]) return !1;
  return !0;
}
function Vst(t, e) {
  return e || (t = t.slice()), t.sort(Gst);
}
function Gst(t, e) {
  return t[Li] - e[Li];
}
let uc = !1;
function Kst(t, e, r, o) {
  for (; r <= o; ) {
    const s = r + ((o - r) >> 1),
      u = t[s][Li] - e;
    if (u === 0) return (uc = !0), s;
    u < 0 ? (r = s + 1) : (o = s - 1);
  }
  return (uc = !1), r - 1;
}
function Xst(t, e, r) {
  for (let o = r + 1; o < t.length && t[o][Li] === e; r = o++);
  return r;
}
function Yst(t, e, r) {
  for (let o = r - 1; o >= 0 && t[o][Li] === e; r = o--);
  return r;
}
function Zst() {
  return { lastKey: -1, lastNeedle: -1, lastIndex: -1 };
}
function Jst(t, e, r, o) {
  const { lastKey: s, lastNeedle: u, lastIndex: f } = r;
  let h = 0,
    d = t.length - 1;
  if (o === s) {
    if (e === u) return (uc = f !== -1 && t[f][Li] === e), f;
    e >= u ? (h = f === -1 ? 0 : f) : (d = f);
  }
  return (r.lastKey = o), (r.lastNeedle = e), (r.lastIndex = Kst(t, e, h, d));
}
const Qst = "`line` must be greater than 0 (lines start at line 1)",
  tlt = "`column` must be greater than or equal to 0 (columns start at column 0)",
  jv = -1,
  elt = 1;
let Vv, Ry;
class nlt {
  constructor(e, r) {
    const o = typeof e == "string";
    if (!o && e._decodedMemo) return e;
    const s = o ? JSON.parse(e) : e,
      { version: u, file: f, names: h, sourceRoot: d, sources: g, sourcesContent: v } = s;
    (this.version = u),
      (this.file = f),
      (this.names = h || []),
      (this.sourceRoot = d),
      (this.sources = g),
      (this.sourcesContent = v);
    const b = Wv(d || "", Fst(r));
    this.resolvedSources = g.map((S) => Wv(S || "", b));
    const { mappings: w } = s;
    typeof w == "string"
      ? ((this._encoded = w), (this._decoded = void 0))
      : ((this._encoded = void 0), (this._decoded = Ust(w, o))),
      (this._decodedMemo = Zst()),
      (this._bySources = void 0),
      (this._bySourceMemos = void 0);
  }
}
(Vv = (t) => t._decoded || (t._decoded = Cst(t._encoded))),
  (Ry = (t, { line: e, column: r, bias: o }) => {
    if ((e--, e < 0)) throw new Error(Qst);
    if (r < 0) throw new Error(tlt);
    const s = Vv(t);
    if (e >= s.length) return Aa(null, null, null, null);
    const u = s[e],
      f = rlt(u, t._decodedMemo, e, r, o || elt);
    if (f === -1) return Aa(null, null, null, null);
    const h = u[f];
    if (h.length === 1) return Aa(null, null, null, null);
    const { names: d, resolvedSources: g } = t;
    return Aa(g[h[qst]], h[Hst] + 1, h[Bst], h.length === 5 ? d[h[Wst]] : null);
  });
function Aa(t, e, r, o) {
  return { source: t, line: e, column: r, name: o };
}
function rlt(t, e, r, o, s) {
  let u = Jst(t, o, e, r);
  return (
    uc ? (u = (s === jv ? Xst : Yst)(t, o, u)) : s === jv && u++,
    u === -1 || u === t.length ? -1 : u
  );
}
const zy = /^\s*at .*(\S+:\d+|\(native\))/m,
  ilt = /^(eval@)?(\[native code])?$/,
  olt = [
    "node:internal",
    /\/packages\/\w+\/dist\//,
    /\/@vitest\/\w+\/dist\//,
    "/vitest/dist/",
    "/vitest/src/",
    "/vite-node/dist/",
    "/vite-node/src/",
    "/node_modules/chai/",
    "/node_modules/tinypool/",
    "/node_modules/tinyspy/",
    "/deps/chai.js",
    /__vitest_browser__/,
  ];
function Iy(t) {
  if (!t.includes(":")) return [t];
  const r = /(.+?)(?::(\d+))?(?::(\d+))?$/.exec(t.replace(/^\(|\)$/g, ""));
  if (!r) return [t];
  let o = r[1];
  return (
    (o.startsWith("http:") || o.startsWith("https:")) && (o = new URL(o).pathname),
    o.startsWith("/@fs/") &&
      (o = o.slice(typeof process < "u" && process.platform === "win32" ? 5 : 4)),
    [o, r[2] || void 0, r[3] || void 0]
  );
}
function slt(t) {
  let e = t.trim();
  if (
    ilt.test(e) ||
    (e.includes(" > eval") &&
      (e = e.replace(/ line (\d+)(?: > eval line \d+)* > eval:\d+:\d+/g, ":$1")),
    !e.includes("@") && !e.includes(":"))
  )
    return null;
  const r = /((.*".+"[^@]*)?[^@]*)(?:@)/,
    o = e.match(r),
    s = o && o[1] ? o[1] : void 0,
    [u, f, h] = Iy(e.replace(r, ""));
  return !u || !f || !h
    ? null
    : { file: u, method: s || "", line: Number.parseInt(f), column: Number.parseInt(h) };
}
function llt(t) {
  let e = t.trim();
  if (!zy.test(e)) return null;
  e.includes("(eval ") &&
    (e = e.replace(/eval code/g, "eval").replace(/(\(eval at [^()]*)|(,.*$)/g, ""));
  let r = e.replace(/^\s+/, "").replace(/\(eval code/g, "(").replace(/^.*?\s+/, "");
  const o = r.match(/ (\(.+\)$)/);
  r = o ? r.replace(o[0], "") : r;
  const [s, u, f] = Iy(o ? o[1] : r);
  let h = (o && r) || "",
    d = s && ["eval", "<anonymous>"].includes(s) ? void 0 : s;
  return !d || !u || !f
    ? null
    : (h.startsWith("async ") && (h = h.slice(6)),
      d.startsWith("file://") && (d = d.slice(7)),
      (d = xst(d)),
      h && (h = h.replace(/__vite_ssr_import_\d+__\./g, "")),
      { method: h, file: d, line: Number.parseInt(u), column: Number.parseInt(f) });
}
function alt(t, e = {}) {
  const { ignoreStackEntries: r = olt } = e;
  let o = zy.test(t) ? ult(t) : clt(t);
  return (
    r.length && (o = o.filter((s) => !r.some((u) => s.file.match(u)))),
    o.map((s) => {
      var u;
      const f = (u = e.getSourceMap) == null ? void 0 : u.call(e, s.file);
      if (!f || typeof f != "object" || !f.version) return s;
      const h = new nlt(f),
        { line: d, column: g } = Ry(h, s);
      return d != null && g != null ? { ...s, line: d, column: g } : s;
    })
  );
}
function clt(t) {
  return t
    .split(`
`)
    .map((e) => slt(e))
    .filter(dy);
}
function ult(t) {
  return t
    .split(`
`)
    .map((e) => llt(e))
    .filter(dy);
}
function flt(t, e) {
  return e && t.endsWith(e);
}
async function Fy(t, e, r) {
  const o = encodeURI(`${t}:${e}:${r}`);
  await fetch(`/__open-in-editor?file=${o}`);
}
function jh(t) {
  return new mst({ fg: t ? "#FFF" : "#000", bg: t ? "#000" : "#FFF" });
}
function hlt(t) {
  return t === null || (typeof t != "function" && typeof t != "object");
}
function qy(t) {
  let e = t;
  if ((hlt(t) && (e = { message: String(e).split(/\n/g)[0], stack: String(e), name: "" }), !t)) {
    const r = new Error("unknown error");
    e = { message: r.message, stack: r.stack, name: "" };
  }
  return (e.stacks = alt(e.stack || e.stackStr || "", { ignoreStackEntries: [] })), e;
}
function Vh(t) {
  return Pm() ? (f1(t), !0) : !1;
}
function Sr(t) {
  return typeof t == "function" ? t() : U(t);
}
const dlt = typeof window < "u" && typeof document < "u";
typeof WorkerGlobalScope < "u" && globalThis instanceof WorkerGlobalScope;
const plt = Object.prototype.toString,
  glt = (t) => plt.call(t) === "[object Object]",
  so = () => {};
function Gh(t, e) {
  function r(...o) {
    return new Promise((s, u) => {
      Promise.resolve(t(() => e.apply(this, o), { fn: e, thisArg: this, args: o }))
        .then(s)
        .catch(u);
    });
  }
  return r;
}
const Hy = (t) => t();
function By(t, e = {}) {
  let r,
    o,
    s = so;
  const u = (h) => {
    clearTimeout(h), s(), (s = so);
  };
  return (h) => {
    const d = Sr(t),
      g = Sr(e.maxWait);
    return (
      r && u(r),
      d <= 0 || (g !== void 0 && g <= 0)
        ? (o && (u(o), (o = null)), Promise.resolve(h()))
        : new Promise((v, b) => {
            (s = e.rejectOnCancel ? b : v),
              g &&
                !o &&
                (o = setTimeout(() => {
                  r && u(r), (o = null), v(h());
                }, g)),
              (r = setTimeout(() => {
                o && u(o), (o = null), v(h());
              }, d));
          })
    );
  };
}
function vlt(t, e = !0, r = !0, o = !1) {
  let s = 0,
    u,
    f = !0,
    h = so,
    d;
  const g = () => {
    u && (clearTimeout(u), (u = void 0), h(), (h = so));
  };
  return (b) => {
    const w = Sr(t),
      S = Date.now() - s,
      P = () => (d = b());
    return (
      g(),
      w <= 0
        ? ((s = Date.now()), P())
        : (S > w && (r || !f)
            ? ((s = Date.now()), P())
            : e &&
              (d = new Promise((A, L) => {
                (h = o ? L : A),
                  (u = setTimeout(() => {
                    (s = Date.now()), (f = !0), A(P()), g();
                  }, Math.max(0, w - S)));
              })),
          !r && !u && (u = setTimeout(() => (f = !0), w)),
          (f = !1),
          d)
    );
  };
}
function mlt(t = Hy) {
  const e = Zt(!0);
  function r() {
    e.value = !1;
  }
  function o() {
    e.value = !0;
  }
  const s = (...u) => {
    e.value && t(...u);
  };
  return { isActive: Lc(e), pause: r, resume: o, eventFilter: s };
}
function ylt(...t) {
  if (t.length !== 1) return xh(...t);
  const e = t[0];
  return typeof e == "function" ? Lc(R1(() => ({ get: e, set: so }))) : Zt(e);
}
function Gv(t, e = 200, r = {}) {
  return Gh(By(e, r), t);
}
function blt(t, e = 200, r = !1, o = !0, s = !1) {
  return Gh(vlt(e, r, o, s), t);
}
function wlt(t, e = 200, r = !0, o = !0) {
  if (e <= 0) return t;
  const s = Zt(t.value),
    u = blt(
      () => {
        s.value = t.value;
      },
      e,
      r,
      o,
    );
  return Re(t, () => u()), s;
}
function Wy(t, e, r = {}) {
  const { eventFilter: o = Hy, ...s } = r;
  return Re(t, Gh(o, e), s);
}
function Uy(t, e, r = {}) {
  const { eventFilter: o, ...s } = r,
    { eventFilter: u, pause: f, resume: h, isActive: d } = mlt(o);
  return { stop: Wy(t, e, { ...s, eventFilter: u }), pause: f, resume: h, isActive: d };
}
function Kh(t, e = !0) {
  Pl() ? ms(t) : e ? t() : Br(t);
}
function xlt(t = !1, e = {}) {
  const { truthyValue: r = !0, falsyValue: o = !1 } = e,
    s = Le(t),
    u = Zt(t);
  function f(h) {
    if (arguments.length) return (u.value = h), u.value;
    {
      const d = Sr(r);
      return (u.value = u.value === d ? Sr(o) : d), u.value;
    }
  }
  return s ? f : [u, f];
}
function _lt(t, e, r = {}) {
  const { debounce: o = 0, maxWait: s = void 0, ...u } = r;
  return Wy(t, e, { ...u, eventFilter: By(o, { maxWait: s }) });
}
function Slt(t, e, r) {
  const o = Re(t, (...s) => (Br(() => o()), e(...s)), r);
  return o;
}
function klt(t, e, r) {
  let o;
  Le(r) ? (o = { evaluating: r }) : (o = r || {});
  const { lazy: s = !1, evaluating: u = void 0, shallow: f = !0, onError: h = so } = o,
    d = Zt(!s),
    g = f ? vs(e) : Zt(e);
  let v = 0;
  return (
    Th(async (b) => {
      if (!d.value) return;
      v++;
      const w = v;
      let S = !1;
      u &&
        Promise.resolve().then(() => {
          u.value = !0;
        });
      try {
        const P = await t((A) => {
          b(() => {
            u && (u.value = !1), S || A();
          });
        });
        w === v && (g.value = P);
      } catch (P) {
        h(P);
      } finally {
        u && w === v && (u.value = !1), (S = !0);
      }
    }),
    s ? xt(() => ((d.value = !0), g.value)) : g
  );
}
function fc(t) {
  var e;
  const r = Sr(t);
  return (e = r == null ? void 0 : r.$el) != null ? e : r;
}
const Vr = dlt ? window : void 0;
function fs(...t) {
  let e, r, o, s;
  if (
    (typeof t[0] == "string" || Array.isArray(t[0])
      ? (([r, o, s] = t), (e = Vr))
      : ([e, r, o, s] = t),
    !e)
  )
    return so;
  Array.isArray(r) || (r = [r]), Array.isArray(o) || (o = [o]);
  const u = [],
    f = () => {
      u.forEach((v) => v()), (u.length = 0);
    },
    h = (v, b, w, S) => (v.addEventListener(b, w, S), () => v.removeEventListener(b, w, S)),
    d = Re(
      () => [fc(e), Sr(s)],
      ([v, b]) => {
        if ((f(), !v)) return;
        const w = glt(b) ? { ...b } : b;
        u.push(...r.flatMap((S) => o.map((P) => h(v, S, P, w))));
      },
      { immediate: !0, flush: "post" },
    ),
    g = () => {
      d(), f();
    };
  return Vh(g), g;
}
function Clt(t) {
  return typeof t == "function"
    ? t
    : typeof t == "string"
    ? (e) => e.key === t
    : Array.isArray(t)
    ? (e) => t.includes(e.key)
    : () => !0;
}
function Tlt(...t) {
  let e,
    r,
    o = {};
  t.length === 3
    ? ((e = t[0]), (r = t[1]), (o = t[2]))
    : t.length === 2
    ? typeof t[1] == "object"
      ? ((e = !0), (r = t[0]), (o = t[1]))
      : ((e = t[0]), (r = t[1]))
    : ((e = !0), (r = t[0]));
  const { target: s = Vr, eventName: u = "keydown", passive: f = !1, dedupe: h = !1 } = o,
    d = Clt(e);
  return fs(
    s,
    u,
    (v) => {
      (v.repeat && Sr(h)) || (d(v) && r(v));
    },
    f,
  );
}
function Elt() {
  const t = Zt(!1);
  return (
    Pl() &&
      ms(() => {
        t.value = !0;
      }),
    t
  );
}
function jy(t) {
  const e = Elt();
  return xt(() => (e.value, !!t()));
}
function Vy(t, e = {}) {
  const { window: r = Vr } = e,
    o = jy(() => r && "matchMedia" in r && typeof r.matchMedia == "function");
  let s;
  const u = Zt(!1),
    f = (g) => {
      u.value = g.matches;
    },
    h = () => {
      s && ("removeEventListener" in s ? s.removeEventListener("change", f) : s.removeListener(f));
    },
    d = Th(() => {
      o.value &&
        (h(),
        (s = r.matchMedia(Sr(t))),
        "addEventListener" in s ? s.addEventListener("change", f) : s.addListener(f),
        (u.value = s.matches));
    });
  return (
    Vh(() => {
      d(), h(), (s = void 0);
    }),
    u
  );
}
const Ma =
    typeof globalThis < "u"
      ? globalThis
      : typeof window < "u"
      ? window
      : typeof global < "u"
      ? global
      : typeof self < "u"
      ? self
      : {},
  Na = "__vueuse_ssr_handlers__",
  Llt = Alt();
function Alt() {
  return Na in Ma || (Ma[Na] = Ma[Na] || {}), Ma[Na];
}
function Gy(t, e) {
  return Llt[t] || e;
}
function Mlt(t) {
  return t == null
    ? "any"
    : t instanceof Set
    ? "set"
    : t instanceof Map
    ? "map"
    : t instanceof Date
    ? "date"
    : typeof t == "boolean"
    ? "boolean"
    : typeof t == "string"
    ? "string"
    : typeof t == "object"
    ? "object"
    : Number.isNaN(t)
    ? "any"
    : "number";
}
const Nlt = {
    boolean: { read: (t) => t === "true", write: (t) => String(t) },
    object: { read: (t) => JSON.parse(t), write: (t) => JSON.stringify(t) },
    number: { read: (t) => Number.parseFloat(t), write: (t) => String(t) },
    any: { read: (t) => t, write: (t) => String(t) },
    string: { read: (t) => t, write: (t) => String(t) },
    map: {
      read: (t) => new Map(JSON.parse(t)),
      write: (t) => JSON.stringify(Array.from(t.entries())),
    },
    set: { read: (t) => new Set(JSON.parse(t)), write: (t) => JSON.stringify(Array.from(t)) },
    date: { read: (t) => new Date(t), write: (t) => t.toISOString() },
  },
  Kv = "vueuse-storage";
function Plt(t, e, r, o = {}) {
  var s;
  const {
      flush: u = "pre",
      deep: f = !0,
      listenToStorageChanges: h = !0,
      writeDefaults: d = !0,
      mergeDefaults: g = !1,
      shallow: v,
      window: b = Vr,
      eventFilter: w,
      onError: S = (nt) => {
        console.error(nt);
      },
      initOnMounted: P,
    } = o,
    A = (v ? vs : Zt)(typeof e == "function" ? e() : e);
  if (!r)
    try {
      r = Gy("getDefaultStorage", () => {
        var nt;
        return (nt = Vr) == null ? void 0 : nt.localStorage;
      })();
    } catch (nt) {
      S(nt);
    }
  if (!r) return A;
  const L = Sr(e),
    T = Mlt(L),
    M = (s = o.serializer) != null ? s : Nlt[T],
    { pause: R, resume: E } = Uy(A, () => B(A.value), { flush: u, deep: f, eventFilter: w });
  return (
    b &&
      h &&
      Kh(() => {
        fs(b, "storage", Y), fs(b, Kv, ht), P && Y();
      }),
    P || Y(),
    A
  );
  function B(nt) {
    try {
      if (nt == null) r.removeItem(t);
      else {
        const at = M.write(nt),
          pt = r.getItem(t);
        pt !== at &&
          (r.setItem(t, at),
          b &&
            b.dispatchEvent(
              new CustomEvent(Kv, {
                detail: { key: t, oldValue: pt, newValue: at, storageArea: r },
              }),
            ));
      }
    } catch (at) {
      S(at);
    }
  }
  function K(nt) {
    const at = nt ? nt.newValue : r.getItem(t);
    if (at == null) return d && L !== null && r.setItem(t, M.write(L)), L;
    if (!nt && g) {
      const pt = M.read(at);
      return typeof g == "function"
        ? g(pt, L)
        : T === "object" && !Array.isArray(pt)
        ? { ...L, ...pt }
        : pt;
    } else return typeof at != "string" ? at : M.read(at);
  }
  function ht(nt) {
    Y(nt.detail);
  }
  function Y(nt) {
    if (!(nt && nt.storageArea !== r)) {
      if (nt && nt.key == null) {
        A.value = L;
        return;
      }
      if (!(nt && nt.key !== t)) {
        R();
        try {
          (nt == null ? void 0 : nt.newValue) !== M.write(A.value) && (A.value = K(nt));
        } catch (at) {
          S(at);
        } finally {
          nt ? Br(E) : E();
        }
      }
    }
  }
}
function Olt(t) {
  return Vy("(prefers-color-scheme: dark)", t);
}
function Dlt(t = {}) {
  const {
      selector: e = "html",
      attribute: r = "class",
      initialValue: o = "auto",
      window: s = Vr,
      storage: u,
      storageKey: f = "vueuse-color-scheme",
      listenToStorageChanges: h = !0,
      storageRef: d,
      emitAuto: g,
      disableTransition: v = !0,
    } = t,
    b = { auto: "", light: "light", dark: "dark", ...(t.modes || {}) },
    w = Olt({ window: s }),
    S = xt(() => (w.value ? "dark" : "light")),
    P = d || (f == null ? ylt(o) : Plt(f, o, u, { window: s, listenToStorageChanges: h })),
    A = xt(() => (P.value === "auto" ? S.value : P.value)),
    L = Gy("updateHTMLAttrs", (E, B, K) => {
      const ht = typeof E == "string" ? (s == null ? void 0 : s.document.querySelector(E)) : fc(E);
      if (!ht) return;
      let Y;
      if (v) {
        Y = s.document.createElement("style");
        const nt =
          "*,*::before,*::after{-webkit-transition:none!important;-moz-transition:none!important;-o-transition:none!important;-ms-transition:none!important;transition:none!important}";
        Y.appendChild(document.createTextNode(nt)), s.document.head.appendChild(Y);
      }
      if (B === "class") {
        const nt = K.split(/\s/g);
        Object.values(b)
          .flatMap((at) => (at || "").split(/\s/g))
          .filter(Boolean)
          .forEach((at) => {
            nt.includes(at) ? ht.classList.add(at) : ht.classList.remove(at);
          });
      } else ht.setAttribute(B, K);
      v && (s.getComputedStyle(Y).opacity, document.head.removeChild(Y));
    });
  function T(E) {
    var B;
    L(e, r, (B = b[E]) != null ? B : E);
  }
  function M(E) {
    t.onChanged ? t.onChanged(E, T) : T(E);
  }
  Re(A, M, { flush: "post", immediate: !0 }), Kh(() => M(A.value));
  const R = xt({
    get() {
      return g ? P.value : A.value;
    },
    set(E) {
      P.value = E;
    },
  });
  try {
    return Object.assign(R, { store: P, system: S, state: A });
  } catch {
    return R;
  }
}
function $lt(t = {}) {
  const { valueDark: e = "dark", valueLight: r = "" } = t,
    o = Dlt({
      ...t,
      onChanged: (u, f) => {
        var h;
        t.onChanged ? (h = t.onChanged) == null || h.call(t, u === "dark", f, u) : f(u);
      },
      modes: { dark: e, light: r },
    });
  return xt({
    get() {
      return o.value === "dark";
    },
    set(u) {
      const f = u ? "dark" : "light";
      o.system.value === f ? (o.value = "auto") : (o.value = f);
    },
  });
}
function Rlt(t, e, r = {}) {
  const { window: o = Vr, ...s } = r;
  let u;
  const f = jy(() => o && "ResizeObserver" in o),
    h = () => {
      u && (u.disconnect(), (u = void 0));
    },
    d = xt(() => (Array.isArray(t) ? t.map((b) => fc(b)) : [fc(t)])),
    g = Re(
      d,
      (b) => {
        if ((h(), f.value && o)) {
          u = new ResizeObserver(e);
          for (const w of b) w && u.observe(w, s);
        }
      },
      { immediate: !0, flush: "post", deep: !0 },
    ),
    v = () => {
      h(), g();
    };
  return Vh(v), { isSupported: f, stop: v };
}
function zlt(t = "history", e = {}) {
  const {
    initialValue: r = {},
    removeNullishValues: o = !0,
    removeFalsyValues: s = !1,
    write: u = !0,
    window: f = Vr,
  } = e;
  if (!f) return Un(r);
  const h = Un({});
  function d() {
    if (t === "history") return f.location.search || "";
    if (t === "hash") {
      const T = f.location.hash || "",
        M = T.indexOf("?");
      return M > 0 ? T.slice(M) : "";
    } else return (f.location.hash || "").replace(/^#/, "");
  }
  function g(T) {
    const M = T.toString();
    if (t === "history") return `${M ? `?${M}` : ""}${f.location.hash || ""}`;
    if (t === "hash-params") return `${f.location.search || ""}${M ? `#${M}` : ""}`;
    const R = f.location.hash || "#",
      E = R.indexOf("?");
    return E > 0 ? `${R.slice(0, E)}${M ? `?${M}` : ""}` : `${R}${M ? `?${M}` : ""}`;
  }
  function v() {
    return new URLSearchParams(d());
  }
  function b(T) {
    const M = new Set(Object.keys(h));
    for (const R of T.keys()) {
      const E = T.getAll(R);
      (h[R] = E.length > 1 ? E : T.get(R) || ""), M.delete(R);
    }
    Array.from(M).forEach((R) => delete h[R]);
  }
  const { pause: w, resume: S } = Uy(
    h,
    () => {
      const T = new URLSearchParams("");
      Object.keys(h).forEach((M) => {
        const R = h[M];
        Array.isArray(R)
          ? R.forEach((E) => T.append(M, E))
          : (o && R == null) || (s && !R)
          ? T.delete(M)
          : T.set(M, R);
      }),
        P(T);
    },
    { deep: !0 },
  );
  function P(T, M) {
    w(),
      M && b(T),
      f.history.replaceState(f.history.state, f.document.title, f.location.pathname + g(T)),
      S();
  }
  function A() {
    u && P(v(), !0);
  }
  fs(f, "popstate", A, !1), t !== "history" && fs(f, "hashchange", A, !1);
  const L = v();
  return L.keys().next().value ? b(L) : Object.assign(h, r), h;
}
function Ilt(t = {}) {
  const {
      window: e = Vr,
      initialWidth: r = Number.POSITIVE_INFINITY,
      initialHeight: o = Number.POSITIVE_INFINITY,
      listenOrientation: s = !0,
      includeScrollbar: u = !0,
    } = t,
    f = Zt(r),
    h = Zt(o),
    d = () => {
      e &&
        (u
          ? ((f.value = e.innerWidth), (h.value = e.innerHeight))
          : ((f.value = e.document.documentElement.clientWidth),
            (h.value = e.document.documentElement.clientHeight)));
    };
  if ((d(), Kh(d), fs("resize", d, { passive: !0 }), s)) {
    const g = Vy("(orientation: portrait)");
    Re(g, () => d());
  }
  return { width: f, height: h };
}
const Ky = zlt("hash", { initialValue: { file: "", view: null } }),
  yr = xh(Ky, "file"),
  er = xh(Ky, "view");
var In = Uint8Array,
  Uo = Uint16Array,
  Flt = Int32Array,
  Xy = new In([
    0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0, 0, 0, 0,
  ]),
  Yy = new In([
    0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13,
    13, 0, 0,
  ]),
  qlt = new In([16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15]),
  Zy = function (t, e) {
    for (var r = new Uo(31), o = 0; o < 31; ++o) r[o] = e += 1 << t[o - 1];
    for (var s = new Flt(r[30]), o = 1; o < 30; ++o)
      for (var u = r[o]; u < r[o + 1]; ++u) s[u] = ((u - r[o]) << 5) | o;
    return { b: r, r: s };
  },
  Jy = Zy(Xy, 2),
  Qy = Jy.b,
  Hlt = Jy.r;
(Qy[28] = 258), (Hlt[258] = 28);
var Blt = Zy(Yy, 0),
  Wlt = Blt.b,
  Kf = new Uo(32768);
for (var xe = 0; xe < 32768; ++xe) {
  var pi = ((xe & 43690) >> 1) | ((xe & 21845) << 1);
  (pi = ((pi & 52428) >> 2) | ((pi & 13107) << 2)),
    (pi = ((pi & 61680) >> 4) | ((pi & 3855) << 4)),
    (Kf[xe] = (((pi & 65280) >> 8) | ((pi & 255) << 8)) >> 1);
}
var dl = function (t, e, r) {
    for (var o = t.length, s = 0, u = new Uo(e); s < o; ++s) t[s] && ++u[t[s] - 1];
    var f = new Uo(e);
    for (s = 1; s < e; ++s) f[s] = (f[s - 1] + u[s - 1]) << 1;
    var h;
    if (r) {
      h = new Uo(1 << e);
      var d = 15 - e;
      for (s = 0; s < o; ++s)
        if (t[s])
          for (
            var g = (s << 4) | t[s], v = e - t[s], b = f[t[s] - 1]++ << v, w = b | ((1 << v) - 1);
            b <= w;
            ++b
          )
            h[Kf[b] >> d] = g;
    } else for (h = new Uo(o), s = 0; s < o; ++s) t[s] && (h[s] = Kf[f[t[s] - 1]++] >> (15 - t[s]));
    return h;
  },
  Rl = new In(288);
for (var xe = 0; xe < 144; ++xe) Rl[xe] = 8;
for (var xe = 144; xe < 256; ++xe) Rl[xe] = 9;
for (var xe = 256; xe < 280; ++xe) Rl[xe] = 7;
for (var xe = 280; xe < 288; ++xe) Rl[xe] = 8;
var tb = new In(32);
for (var xe = 0; xe < 32; ++xe) tb[xe] = 5;
var Ult = dl(Rl, 9, 1),
  jlt = dl(tb, 5, 1),
  sf = function (t) {
    for (var e = t[0], r = 1; r < t.length; ++r) t[r] > e && (e = t[r]);
    return e;
  },
  nr = function (t, e, r) {
    var o = (e / 8) | 0;
    return ((t[o] | (t[o + 1] << 8)) >> (e & 7)) & r;
  },
  lf = function (t, e) {
    var r = (e / 8) | 0;
    return (t[r] | (t[r + 1] << 8) | (t[r + 2] << 16)) >> (e & 7);
  },
  Vlt = function (t) {
    return ((t + 7) / 8) | 0;
  },
  eb = function (t, e, r) {
    return (
      (e == null || e < 0) && (e = 0),
      (r == null || r > t.length) && (r = t.length),
      new In(t.subarray(e, r))
    );
  },
  Glt = [
    "unexpected EOF",
    "invalid block type",
    "invalid length/literal",
    "invalid distance",
    "stream finished",
    "no stream handler",
    ,
    "no callback",
    "invalid UTF-8 data",
    "extra field too long",
    "date not in range 1980-2099",
    "filename too long",
    "stream finishing",
    "invalid zip data",
  ],
  Tn = function (t, e, r) {
    var o = new Error(e || Glt[t]);
    if (((o.code = t), Error.captureStackTrace && Error.captureStackTrace(o, Tn), !r)) throw o;
    return o;
  },
  Xh = function (t, e, r, o) {
    var s = t.length,
      u = o ? o.length : 0;
    if (!s || (e.f && !e.l)) return r || new In(0);
    var f = !r,
      h = f || e.i != 2,
      d = e.i;
    f && (r = new In(s * 3));
    var g = function (Et) {
        var $ = r.length;
        if (Et > $) {
          var I = new In(Math.max($ * 2, Et));
          I.set(r), (r = I);
        }
      },
      v = e.f || 0,
      b = e.p || 0,
      w = e.b || 0,
      S = e.l,
      P = e.d,
      A = e.m,
      L = e.n,
      T = s * 8;
    do {
      if (!S) {
        v = nr(t, b, 1);
        var M = nr(t, b + 1, 3);
        if (((b += 3), M))
          if (M == 1) (S = Ult), (P = jlt), (A = 9), (L = 5);
          else if (M == 2) {
            var K = nr(t, b, 31) + 257,
              ht = nr(t, b + 10, 15) + 4,
              Y = K + nr(t, b + 5, 31) + 1;
            b += 14;
            for (var nt = new In(Y), at = new In(19), pt = 0; pt < ht; ++pt)
              at[qlt[pt]] = nr(t, b + pt * 3, 7);
            b += ht * 3;
            for (var gt = sf(at), G = (1 << gt) - 1, z = dl(at, gt, 1), pt = 0; pt < Y; ) {
              var k = z[nr(t, b, G)];
              b += k & 15;
              var R = k >> 4;
              if (R < 16) nt[pt++] = R;
              else {
                var F = 0,
                  H = 0;
                for (
                  R == 16
                    ? ((H = 3 + nr(t, b, 3)), (b += 2), (F = nt[pt - 1]))
                    : R == 17
                    ? ((H = 3 + nr(t, b, 7)), (b += 3))
                    : R == 18 && ((H = 11 + nr(t, b, 127)), (b += 7));
                  H--;
                )
                  nt[pt++] = F;
              }
            }
            var J = nt.subarray(0, K),
              yt = nt.subarray(K);
            (A = sf(J)), (L = sf(yt)), (S = dl(J, A, 1)), (P = dl(yt, L, 1));
          } else Tn(1);
        else {
          var R = Vlt(b) + 4,
            E = t[R - 4] | (t[R - 3] << 8),
            B = R + E;
          if (B > s) {
            d && Tn(0);
            break;
          }
          h && g(w + E), r.set(t.subarray(R, B), w), (e.b = w += E), (e.p = b = B * 8), (e.f = v);
          continue;
        }
        if (b > T) {
          d && Tn(0);
          break;
        }
      }
      h && g(w + 131072);
      for (var At = (1 << A) - 1, qt = (1 << L) - 1, Ht = b; ; Ht = b) {
        var F = S[lf(t, b) & At],
          Qt = F >> 4;
        if (((b += F & 15), b > T)) {
          d && Tn(0);
          break;
        }
        if ((F || Tn(2), Qt < 256)) r[w++] = Qt;
        else if (Qt == 256) {
          (Ht = b), (S = null);
          break;
        } else {
          var Jt = Qt - 254;
          if (Qt > 264) {
            var pt = Qt - 257,
              Gt = Xy[pt];
            (Jt = nr(t, b, (1 << Gt) - 1) + Qy[pt]), (b += Gt);
          }
          var Tt = P[lf(t, b) & qt],
            j = Tt >> 4;
          Tt || Tn(3), (b += Tt & 15);
          var yt = Wlt[j];
          if (j > 3) {
            var Gt = Yy[j];
            (yt += lf(t, b) & ((1 << Gt) - 1)), (b += Gt);
          }
          if (b > T) {
            d && Tn(0);
            break;
          }
          h && g(w + 131072);
          var rt = w + Jt;
          if (w < yt) {
            var lt = u - yt,
              Mt = Math.min(yt, rt);
            for (lt + w < 0 && Tn(3); w < Mt; ++w) r[w] = o[lt + w];
          }
          for (; w < rt; ++w) r[w] = r[w - yt];
        }
      }
      (e.l = S), (e.p = Ht), (e.b = w), (e.f = v), S && ((v = 1), (e.m = A), (e.d = P), (e.n = L));
    } while (!v);
    return w != r.length && f ? eb(r, 0, w) : r.subarray(0, w);
  },
  Klt = new In(0),
  Xlt = function (t) {
    (t[0] != 31 || t[1] != 139 || t[2] != 8) && Tn(6, "invalid gzip data");
    var e = t[3],
      r = 10;
    e & 4 && (r += (t[10] | (t[11] << 8)) + 2);
    for (var o = ((e >> 3) & 1) + ((e >> 4) & 1); o > 0; o -= !t[r++]);
    return r + (e & 2);
  },
  Ylt = function (t) {
    var e = t.length;
    return (t[e - 4] | (t[e - 3] << 8) | (t[e - 2] << 16) | (t[e - 1] << 24)) >>> 0;
  },
  Zlt = function (t, e) {
    return (
      ((t[0] & 15) != 8 || t[0] >> 4 > 7 || ((t[0] << 8) | t[1]) % 31) &&
        Tn(6, "invalid zlib data"),
      ((t[1] >> 5) & 1) == +!e &&
        Tn(6, "invalid zlib data: " + (t[1] & 32 ? "need" : "unexpected") + " dictionary"),
      ((t[1] >> 3) & 4) + 2
    );
  };
function Jlt(t, e) {
  return Xh(t, { i: 2 }, e && e.out, e && e.dictionary);
}
function Qlt(t, e) {
  var r = Xlt(t);
  return (
    r + 8 > t.length && Tn(6, "invalid gzip data"),
    Xh(t.subarray(r, -8), { i: 2 }, (e && e.out) || new In(Ylt(t)), e && e.dictionary)
  );
}
function tat(t, e) {
  return Xh(t.subarray(Zlt(t, e && e.dictionary), -4), { i: 2 }, e && e.out, e && e.dictionary);
}
function eat(t, e) {
  return t[0] == 31 && t[1] == 139 && t[2] == 8
    ? Qlt(t, e)
    : (t[0] & 15) != 8 || t[0] >> 4 > 7 || ((t[0] << 8) | t[1]) % 31
    ? Jlt(t, e)
    : tat(t, e);
}
var Xf = typeof TextDecoder < "u" && new TextDecoder(),
  nat = 0;
try {
  Xf.decode(Klt, { stream: !0 }), (nat = 1);
} catch {}
var rat = function (t) {
  for (var e = "", r = 0; ; ) {
    var o = t[r++],
      s = (o > 127) + (o > 223) + (o > 239);
    if (r + s > t.length) return { s: e, r: eb(t, r - 1) };
    s
      ? s == 3
        ? ((o =
            (((o & 15) << 18) | ((t[r++] & 63) << 12) | ((t[r++] & 63) << 6) | (t[r++] & 63)) -
            65536),
          (e += String.fromCharCode(55296 | (o >> 10), 56320 | (o & 1023))))
        : s & 1
        ? (e += String.fromCharCode(((o & 31) << 6) | (t[r++] & 63)))
        : (e += String.fromCharCode(((o & 15) << 12) | ((t[r++] & 63) << 6) | (t[r++] & 63)))
      : (e += String.fromCharCode(o));
  }
};
function iat(t, e) {
  if (e) {
    for (var r = "", o = 0; o < t.length; o += 16384)
      r += String.fromCharCode.apply(null, t.subarray(o, o + 16384));
    return r;
  } else {
    if (Xf) return Xf.decode(t);
    var s = rat(t),
      u = s.s,
      r = s.r;
    return r.length && Tn(8), u;
  }
}
const af = () => {},
  hn = () => Promise.resolve();
function oat() {
  const t = Un({ state: new fy(), waitForConnection: f, reconnect: s, ws: new EventTarget() });
  (t.state.filesMap = Un(t.state.filesMap)), (t.state.idMap = Un(t.state.idMap));
  let e;
  const r = {
    getFiles: () => e.files,
    getPaths: () => e.paths,
    getConfig: () => e.config,
    getModuleGraph: async (h) => e.moduleGraph[h],
    getUnhandledErrors: () => e.unhandledErrors,
    getTransformResult: async (h) => ({ code: h, source: "", map: null }),
    onDone: af,
    onCollected: hn,
    onTaskUpdate: af,
    writeFile: hn,
    rerun: hn,
    updateSnapshot: hn,
    resolveSnapshotPath: hn,
    snapshotSaved: hn,
    onAfterSuiteRun: hn,
    onCancel: hn,
    getCountOfFailedTests: () => 0,
    sendLog: hn,
    resolveSnapshotRawPath: hn,
    readSnapshotFile: hn,
    saveSnapshotFile: hn,
    readTestFile: hn,
    removeSnapshotFile: hn,
    onUnhandledError: af,
    saveTestFile: hn,
    getProvidedContext: () => ({}),
  };
  t.rpc = r;
  let o;
  function s() {
    u();
  }
  async function u() {
    var v;
    const h = await fetch(window.METADATA_PATH),
      d = ((v = h.headers.get("content-type")) == null ? void 0 : v.toLowerCase()) || "";
    if (d.includes("application/gzip") || d.includes("application/x-gzip")) {
      const b = new Uint8Array(await h.arrayBuffer()),
        w = iat(eat(b));
      e = jf(w);
    } else e = jf(await h.text());
    const g = new Event("open");
    t.ws.dispatchEvent(g);
  }
  u();
  function f() {
    return o;
  }
  return t;
}
const kl = Zt("idle"),
  yi = Zt([]),
  je = (function () {
    return jr
      ? oat()
      : gT(yT, {
          reactive: Un,
          handlers: {
            onTaskUpdate() {
              kl.value = "running";
            },
            onFinished(e, r) {
              (kl.value = "idle"), (yi.value = (r || []).map(qy));
            },
          },
        });
  })(),
  Yh = vs({}),
  Zi = Zt("CONNECTING"),
  mn = xt(() => je.state.getFiles()),
  Se = xt(() => mn.value.find((t) => t.id === yr.value)),
  nb = xt(
    () =>
      Bh(Se.value)
        .map((t) => (t == null ? void 0 : t.logs) || [])
        .flat() || [],
  );
function hc(t) {
  return mn.value.find((e) => e.id === t);
}
const sat = xt(() => Zi.value === "OPEN"),
  cf = xt(() => Zi.value === "CONNECTING");
xt(() => Zi.value === "CLOSED");
function lat(t = je.state.getFiles()) {
  return rb(t);
}
function rb(t) {
  return (
    t.forEach((e) => {
      delete e.result, Bh(e).forEach((r) => delete r.result);
    }),
    je.rpc.rerun(t.map((e) => e.filepath))
  );
}
function aat() {
  if (Se.value) return rb([Se.value]);
}
Re(
  () => je.ws,
  (t) => {
    (Zi.value = jr ? "OPEN" : "CONNECTING"),
      t.addEventListener("open", async () => {
        (Zi.value = "OPEN"), je.state.filesMap.clear();
        const [e, r, o] = await Promise.all([
          je.rpc.getFiles(),
          je.rpc.getConfig(),
          je.rpc.getUnhandledErrors(),
        ]);
        je.state.collectFiles(e), (yi.value = (o || []).map(qy)), (Yh.value = r);
      }),
      t.addEventListener("close", () => {
        setTimeout(() => {
          Zi.value === "CONNECTING" && (Zi.value = "CLOSED");
        }, 1e3);
      });
  },
  { immediate: !0 },
);
const cat = { "text-2xl": "" },
  uat = tt(
    "div",
    { "text-lg": "", op50: "" },
    " Check your terminal or start a new server with `vitest --ui` ",
    -1,
  ),
  fat = ie({
    __name: "ConnectionOverlay",
    setup(t) {
      return (e, r) =>
        U(sat)
          ? Vt("", !0)
          : (st(),
            kt(
              "div",
              {
                key: 0,
                fixed: "",
                "inset-0": "",
                p2: "",
                "z-10": "",
                "select-none": "",
                text: "center sm",
                bg: "overlay",
                "backdrop-blur-sm": "",
                "backdrop-saturate-0": "",
                onClick: r[0] || (r[0] = (...o) => U(je).reconnect && U(je).reconnect(...o)),
              },
              [
                tt(
                  "div",
                  {
                    "h-full": "",
                    flex: "~ col gap-2",
                    "items-center": "",
                    "justify-center": "",
                    class: ve(U(cf) ? "animate-pulse" : ""),
                  },
                  [
                    tt(
                      "div",
                      {
                        text: "5xl",
                        class: ve(
                          U(cf)
                            ? "i-carbon:renew animate-spin animate-reverse"
                            : "i-carbon-wifi-off",
                        ),
                      },
                      null,
                      2,
                    ),
                    tt("div", cat, Ut(U(cf) ? "Connecting..." : "Disconnected"), 1),
                    uat,
                  ],
                  2,
                ),
              ],
            ));
    },
  }),
  zl = $lt(),
  hat = xlt(zl),
  dat = { class: "scrolls scrolls-rounded task-error" },
  pat = ["onClickPassive"],
  gat = ["innerHTML"],
  vat = ie({
    __name: "ViewReportError",
    props: { root: {}, filename: {}, error: {} },
    setup(t) {
      const e = t;
      function r(f) {
        return f.startsWith(e.root) ? f.slice(e.root.length) : f;
      }
      const o = xt(() => jh(zl.value)),
        s = xt(() => {
          var f;
          return !!((f = e.error) != null && f.diff);
        }),
        u = xt(() => (e.error.diff ? o.value.toHtml(e.error.diff) : void 0));
      return (f, h) => {
        const d = uo("tooltip");
        return (
          st(),
          kt("div", dat, [
            tt("pre", null, [
              tt("b", null, Ut(f.error.name || f.error.nameStr), 1),
              me(": " + Ut(f.error.message), 1),
            ]),
            (st(!0),
            kt(
              ne,
              null,
              Rn(
                f.error.stacks,
                (g, v) => (
                  st(),
                  kt(
                    "div",
                    { key: v, class: "op80 flex gap-x-2 items-center", "data-testid": "stack" },
                    [
                      tt(
                        "pre",
                        null,
                        " - " + Ut(r(g.file)) + ":" + Ut(g.line) + ":" + Ut(g.column),
                        1,
                      ),
                      U(flt)(g.file, f.filename)
                        ? nn(
                            (st(),
                            kt(
                              "div",
                              {
                                key: 0,
                                class:
                                  "i-carbon-launch c-red-600 dark:c-red-400 hover:cursor-pointer min-w-1em min-h-1em",
                                tabindex: "0",
                                "aria-label": "Open in Editor",
                                onClickPassive: (b) => U(Fy)(g.file, g.line, g.column),
                              },
                              null,
                              40,
                              pat,
                            )),
                            [[d, "Open in Editor", void 0, { bottom: !0 }]],
                          )
                        : Vt("", !0),
                    ],
                  )
                ),
              ),
              128,
            )),
            U(s)
              ? (st(), kt("pre", { key: 0, "data-testid": "diff", innerHTML: U(u) }, null, 8, gat))
              : Vt("", !0),
          ])
        );
      };
    },
  }),
  mat = fo(vat, [["__scopeId", "data-v-93ed29fc"]]);
function Va(t) {
  return t
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
const yat = { "h-full": "", class: "scrolls" },
  bat = { key: 0, class: "scrolls scrolls-rounded task-error" },
  wat = ["innerHTML"],
  xat = { key: 1, bg: "green-500/10", text: "green-500 sm", p: "x4 y2", "m-2": "", rounded: "" },
  _at = ie({
    __name: "ViewReport",
    props: { file: {} },
    setup(t) {
      const e = t;
      function r(f, h) {
        var d;
        return ((d = f.result) == null ? void 0 : d.state) !== "fail"
          ? []
          : f.type === "test" || f.type === "custom"
          ? [{ ...f, level: h }]
          : [{ ...f, level: h }, ...f.tasks.flatMap((g) => r(g, h + 1))];
      }
      function o(f, h) {
        var v, b, w;
        let d = "";
        (v = h.message) != null &&
          v.includes("\x1B") &&
          (d = `<b>${h.nameStr || h.name}</b>: ${f.toHtml(Va(h.message))}`);
        const g = (b = h.stackStr) == null ? void 0 : b.includes("\x1B");
        return (
          (g || ((w = h.stack) != null && w.includes("\x1B"))) &&
            (d.length > 0
              ? (d += f.toHtml(Va(g ? h.stackStr : h.stack)))
              : (d = `<b>${h.nameStr || h.name}</b>: ${h.message}${f.toHtml(
                  Va(g ? h.stackStr : h.stack),
                )}`)),
          d.length > 0 ? d : null
        );
      }
      function s(f, h) {
        const d = jh(f);
        return h.map((g) => {
          var w;
          const v = g.result;
          if (!v) return g;
          const b =
            (w = v.errors) == null
              ? void 0
              : w
                  .map((S) => o(d, S))
                  .filter((S) => S != null)
                  .join("<br><br>");
          return b != null && b.length && (v.htmlError = b), g;
        });
      }
      const u = xt(() => {
        var v, b;
        const f = e.file,
          h =
            ((v = f == null ? void 0 : f.tasks) == null ? void 0 : v.flatMap((w) => r(w, 0))) ?? [],
          d = f == null ? void 0 : f.result;
        if ((b = d == null ? void 0 : d.errors) == null ? void 0 : b[0]) {
          const w = {
            id: f.id,
            name: f.name,
            level: 0,
            type: "suite",
            mode: "run",
            meta: {},
            tasks: [],
            result: d,
          };
          h.unshift(w);
        }
        return h.length > 0 ? s(zl.value, h) : h;
      });
      return (f, h) => (
        st(),
        kt("div", yat, [
          U(u).length
            ? (st(!0),
              kt(
                ne,
                { key: 0 },
                Rn(U(u), (d) => {
                  var g, v, b;
                  return (
                    st(),
                    kt("div", { key: d.id }, [
                      tt(
                        "div",
                        {
                          bg: "red-500/10",
                          text: "red-500 sm",
                          p: "x3 y2",
                          "m-2": "",
                          rounded: "",
                          style: An({
                            "margin-left": `${
                              (g = d.result) != null && g.htmlError ? 0.5 : 2 * d.level + 0.5
                            }rem`,
                          }),
                        },
                        [
                          me(Ut(d.name) + " ", 1),
                          (v = d.result) != null && v.htmlError
                            ? (st(),
                              kt("div", bat, [
                                tt("pre", { innerHTML: d.result.htmlError }, null, 8, wat),
                              ]))
                            : (b = d.result) != null && b.errors
                            ? (st(!0),
                              kt(
                                ne,
                                { key: 1 },
                                Rn(d.result.errors, (w, S) => {
                                  var P;
                                  return (
                                    st(),
                                    te(
                                      mat,
                                      {
                                        key: S,
                                        error: w,
                                        filename: (P = f.file) == null ? void 0 : P.name,
                                        root: U(Yh).root,
                                      },
                                      null,
                                      8,
                                      ["error", "filename", "root"],
                                    )
                                  );
                                }),
                                128,
                              ))
                            : Vt("", !0),
                        ],
                        4,
                      ),
                    ])
                  );
                }),
                128,
              ))
            : (st(), kt("div", xat, " All tests passed in this file ")),
        ])
      );
    },
  }),
  Sat = fo(_at, [["__scopeId", "data-v-5e7bb715"]]),
  kat = { border: "b base", "p-4": "" },
  Cat = ["innerHTML"],
  Tat = ie({
    __name: "ViewConsoleOutputEntry",
    props: { taskName: {}, type: {}, time: {}, content: {} },
    setup(t) {
      function e(r) {
        return new Date(r).toLocaleTimeString();
      }
      return (r, o) => (
        st(),
        kt("div", kat, [
          tt(
            "div",
            {
              "text-xs": "",
              "mb-1": "",
              class: ve(r.type === "stderr" ? "text-red-600 dark:text-red-300" : "op30"),
            },
            Ut(e(r.time)) + " | " + Ut(r.taskName) + " | " + Ut(r.type),
            3,
          ),
          tt("pre", { "data-type": "html", innerHTML: r.content }, null, 8, Cat),
        ])
      );
    },
  }),
  dc = xt(() =>
    mn.value.filter((t) => {
      var e;
      return ((e = t.result) == null ? void 0 : e.state) === "fail";
    }),
  ),
  pc = xt(() =>
    mn.value.filter((t) => {
      var e;
      return ((e = t.result) == null ? void 0 : e.state) === "pass";
    }),
  ),
  Zh = xt(() => mn.value.filter((t) => t.mode === "skip" || t.mode === "todo"));
xt(() =>
  mn.value.filter((t) => !dc.value.includes(t) && !pc.value.includes(t) && !Zh.value.includes(t)),
);
xt(() => Zh.value.filter((t) => t.mode === "skip"));
const Xv = xt(() => mn.value.filter(Wc));
xt(() => Zh.value.filter((t) => t.mode === "todo"));
const Eat = xt(() => kl.value === "idle"),
  Vc = xt(() => lb(mn.value)),
  ib = xt(() =>
    Vc.value.filter((t) => {
      var e;
      return ((e = t.result) == null ? void 0 : e.state) === "fail";
    }),
  ),
  ob = xt(() =>
    Vc.value.filter((t) => {
      var e;
      return ((e = t.result) == null ? void 0 : e.state) === "pass";
    }),
  ),
  sb = xt(() => Vc.value.filter((t) => t.mode === "skip" || t.mode === "todo")),
  Lat = xt(() => sb.value.filter((t) => t.mode === "skip")),
  Aat = xt(() => sb.value.filter((t) => t.mode === "todo"));
xt(() => ib.value.length + ob.value.length);
const Mat = xt(() => {
  const t = mn.value.reduce((e, r) => {
    var o;
    return (
      (e += Math.max(0, r.collectDuration || 0)),
      (e += Math.max(0, r.setupDuration || 0)),
      (e += Math.max(0, ((o = r.result) == null ? void 0 : o.duration) || 0)),
      e
    );
  }, 0);
  return t > 1e3 ? `${(t / 1e3).toFixed(2)}s` : `${Math.round(t)}ms`;
});
function Nat(t) {
  return (t = t || []), Array.isArray(t) ? t : [t];
}
function Yv(t) {
  return t.type === "test" || t.type === "custom";
}
function lb(t) {
  const e = [],
    r = Nat(t);
  for (const o of r)
    if (Yv(o)) e.push(o);
    else for (const s of o.tasks) Yv(s) ? e.push(s) : e.push(...lb(s));
  return e;
}
const Pat = {
    key: 0,
    "h-full": "",
    class: "scrolls",
    flex: "",
    "flex-col": "",
    "data-testid": "logs",
  },
  Oat = { key: 1, p6: "" },
  Dat = tt("pre", { inline: "" }, "console.log(foo)", -1),
  $at = ie({
    __name: "ViewConsoleOutput",
    setup(t) {
      const e = xt(() => {
        const o = nb.value;
        if (o) {
          const s = jh(zl.value);
          return o.map(({ taskId: u, type: f, time: h, content: d }) => ({
            taskId: u,
            type: f,
            time: h,
            content: s.toHtml(Va(d)),
          }));
        }
      });
      function r(o) {
        const s = o && je.state.idMap.get(o);
        return (s ? pT(s).slice(1).join(" > ") : "-") || "-";
      }
      return (o, s) => {
        var f;
        const u = Tat;
        return (f = U(e)) != null && f.length
          ? (st(),
            kt("div", Pat, [
              (st(!0),
              kt(
                ne,
                null,
                Rn(
                  U(e),
                  ({ taskId: h, type: d, time: g, content: v }) => (
                    st(),
                    kt("div", { key: h, "font-mono": "" }, [
                      Ft(u, { "task-name": r(h), type: d, time: g, content: v }, null, 8, [
                        "task-name",
                        "type",
                        "time",
                        "content",
                      ]),
                    ])
                  ),
                ),
                128,
              )),
            ]))
          : (st(),
            kt("p", Oat, [
              me(" Log something in your test and it would print here. (e.g. "),
              Dat,
              me(") "),
            ]));
      };
    },
  });
var uf = { exports: {} },
  Zv;
function ys() {
  return (
    Zv ||
      ((Zv = 1),
      (function (t, e) {
        (function (r, o) {
          t.exports = o();
        })(oo, function () {
          var r = navigator.userAgent,
            o = navigator.platform,
            s = /gecko\/\d/i.test(r),
            u = /MSIE \d/.test(r),
            f = /Trident\/(?:[7-9]|\d{2,})\..*rv:(\d+)/.exec(r),
            h = /Edge\/(\d+)/.exec(r),
            d = u || f || h,
            g = d && (u ? document.documentMode || 6 : +(h || f)[1]),
            v = !h && /WebKit\//.test(r),
            b = v && /Qt\/\d+\.\d+/.test(r),
            w = !h && /Chrome\/(\d+)/.exec(r),
            S = w && +w[1],
            P = /Opera\//.test(r),
            A = /Apple Computer/.test(navigator.vendor),
            L = /Mac OS X 1\d\D([8-9]|\d\d)\D/.test(r),
            T = /PhantomJS/.test(r),
            M = A && (/Mobile\/\w+/.test(r) || navigator.maxTouchPoints > 2),
            R = /Android/.test(r),
            E = M || R || /webOS|BlackBerry|Opera Mini|Opera Mobi|IEMobile/i.test(r),
            B = M || /Mac/.test(o),
            K = /\bCrOS\b/.test(r),
            ht = /win/i.test(o),
            Y = P && r.match(/Version\/(\d*\.\d*)/);
          Y && (Y = Number(Y[1])), Y && Y >= 15 && ((P = !1), (v = !0));
          var nt = B && (b || (P && (Y == null || Y < 12.11))),
            at = s || (d && g >= 9);
          function pt(n) {
            return new RegExp("(^|\\s)" + n + "(?:$|\\s)\\s*");
          }
          var gt = function (n, i) {
            var a = n.className,
              l = pt(i).exec(a);
            if (l) {
              var c = a.slice(l.index + l[0].length);
              n.className = a.slice(0, l.index) + (c ? l[1] + c : "");
            }
          };
          function G(n) {
            for (var i = n.childNodes.length; i > 0; --i) n.removeChild(n.firstChild);
            return n;
          }
          function z(n, i) {
            return G(n).appendChild(i);
          }
          function k(n, i, a, l) {
            var c = document.createElement(n);
            if ((a && (c.className = a), l && (c.style.cssText = l), typeof i == "string"))
              c.appendChild(document.createTextNode(i));
            else if (i) for (var p = 0; p < i.length; ++p) c.appendChild(i[p]);
            return c;
          }
          function F(n, i, a, l) {
            var c = k(n, i, a, l);
            return c.setAttribute("role", "presentation"), c;
          }
          var H;
          document.createRange
            ? (H = function (n, i, a, l) {
                var c = document.createRange();
                return c.setEnd(l || n, a), c.setStart(n, i), c;
              })
            : (H = function (n, i, a) {
                var l = document.body.createTextRange();
                try {
                  l.moveToElementText(n.parentNode);
                } catch {
                  return l;
                }
                return l.collapse(!0), l.moveEnd("character", a), l.moveStart("character", i), l;
              });
          function J(n, i) {
            if ((i.nodeType == 3 && (i = i.parentNode), n.contains)) return n.contains(i);
            do if ((i.nodeType == 11 && (i = i.host), i == n)) return !0;
            while ((i = i.parentNode));
          }
          function yt(n) {
            var i = n.ownerDocument || n,
              a;
            try {
              a = n.activeElement;
            } catch {
              a = i.body || null;
            }
            for (; a && a.shadowRoot && a.shadowRoot.activeElement; )
              a = a.shadowRoot.activeElement;
            return a;
          }
          function At(n, i) {
            var a = n.className;
            pt(i).test(a) || (n.className += (a ? " " : "") + i);
          }
          function qt(n, i) {
            for (var a = n.split(" "), l = 0; l < a.length; l++)
              a[l] && !pt(a[l]).test(i) && (i += " " + a[l]);
            return i;
          }
          var Ht = function (n) {
            n.select();
          };
          M
            ? (Ht = function (n) {
                (n.selectionStart = 0), (n.selectionEnd = n.value.length);
              })
            : d &&
              (Ht = function (n) {
                try {
                  n.select();
                } catch {}
              });
          function Qt(n) {
            return n.display.wrapper.ownerDocument;
          }
          function Jt(n) {
            return Gt(n.display.wrapper);
          }
          function Gt(n) {
            return n.getRootNode ? n.getRootNode() : n.ownerDocument;
          }
          function Tt(n) {
            return Qt(n).defaultView;
          }
          function j(n) {
            var i = Array.prototype.slice.call(arguments, 1);
            return function () {
              return n.apply(null, i);
            };
          }
          function rt(n, i, a) {
            i || (i = {});
            for (var l in n)
              n.hasOwnProperty(l) && (a !== !1 || !i.hasOwnProperty(l)) && (i[l] = n[l]);
            return i;
          }
          function lt(n, i, a, l, c) {
            i == null && ((i = n.search(/[^\s\u00a0]/)), i == -1 && (i = n.length));
            for (var p = l || 0, m = c || 0; ; ) {
              var y = n.indexOf("	", p);
              if (y < 0 || y >= i) return m + (i - p);
              (m += y - p), (m += a - (m % a)), (p = y + 1);
            }
          }
          var Mt = function () {
            (this.id = null),
              (this.f = null),
              (this.time = 0),
              (this.handler = j(this.onTimeout, this));
          };
          (Mt.prototype.onTimeout = function (n) {
            (n.id = 0), n.time <= +new Date() ? n.f() : setTimeout(n.handler, n.time - +new Date());
          }),
            (Mt.prototype.set = function (n, i) {
              this.f = i;
              var a = +new Date() + n;
              (!this.id || a < this.time) &&
                (clearTimeout(this.id), (this.id = setTimeout(this.handler, n)), (this.time = a));
            });
          function Et(n, i) {
            for (var a = 0; a < n.length; ++a) if (n[a] == i) return a;
            return -1;
          }
          var $ = 50,
            I = {
              toString: function () {
                return "CodeMirror.Pass";
              },
            },
            V = { scroll: !1 },
            Q = { origin: "*mouse" },
            ot = { origin: "+move" };
          function ut(n, i, a) {
            for (var l = 0, c = 0; ; ) {
              var p = n.indexOf("	", l);
              p == -1 && (p = n.length);
              var m = p - l;
              if (p == n.length || c + m >= i) return l + Math.min(m, i - c);
              if (((c += p - l), (c += a - (c % a)), (l = p + 1), c >= i)) return l;
            }
          }
          var St = [""];
          function mt(n) {
            for (; St.length <= n; ) St.push(ct(St) + " ");
            return St[n];
          }
          function ct(n) {
            return n[n.length - 1];
          }
          function ft(n, i) {
            for (var a = [], l = 0; l < n.length; l++) a[l] = i(n[l], l);
            return a;
          }
          function $t(n, i, a) {
            for (var l = 0, c = a(i); l < n.length && a(n[l]) <= c; ) l++;
            n.splice(l, 0, i);
          }
          function Nt() {}
          function Dt(n, i) {
            var a;
            return (
              Object.create ? (a = Object.create(n)) : ((Nt.prototype = n), (a = new Nt())),
              i && rt(i, a),
              a
            );
          }
          var Bt =
            /[\u00df\u0587\u0590-\u05f4\u0600-\u06ff\u3040-\u309f\u30a0-\u30ff\u3400-\u4db5\u4e00-\u9fcc\uac00-\ud7af]/;
          function Kt(n) {
            return /\w/.test(n) || (n > "" && (n.toUpperCase() != n.toLowerCase() || Bt.test(n)));
          }
          function re(n, i) {
            return i ? (i.source.indexOf("\\w") > -1 && Kt(n) ? !0 : i.test(n)) : Kt(n);
          }
          function oe(n) {
            for (var i in n) if (n.hasOwnProperty(i) && n[i]) return !1;
            return !0;
          }
          var fe =
            /[\u0300-\u036f\u0483-\u0489\u0591-\u05bd\u05bf\u05c1\u05c2\u05c4\u05c5\u05c7\u0610-\u061a\u064b-\u065e\u0670\u06d6-\u06dc\u06de-\u06e4\u06e7\u06e8\u06ea-\u06ed\u0711\u0730-\u074a\u07a6-\u07b0\u07eb-\u07f3\u0816-\u0819\u081b-\u0823\u0825-\u0827\u0829-\u082d\u0900-\u0902\u093c\u0941-\u0948\u094d\u0951-\u0955\u0962\u0963\u0981\u09bc\u09be\u09c1-\u09c4\u09cd\u09d7\u09e2\u09e3\u0a01\u0a02\u0a3c\u0a41\u0a42\u0a47\u0a48\u0a4b-\u0a4d\u0a51\u0a70\u0a71\u0a75\u0a81\u0a82\u0abc\u0ac1-\u0ac5\u0ac7\u0ac8\u0acd\u0ae2\u0ae3\u0b01\u0b3c\u0b3e\u0b3f\u0b41-\u0b44\u0b4d\u0b56\u0b57\u0b62\u0b63\u0b82\u0bbe\u0bc0\u0bcd\u0bd7\u0c3e-\u0c40\u0c46-\u0c48\u0c4a-\u0c4d\u0c55\u0c56\u0c62\u0c63\u0cbc\u0cbf\u0cc2\u0cc6\u0ccc\u0ccd\u0cd5\u0cd6\u0ce2\u0ce3\u0d3e\u0d41-\u0d44\u0d4d\u0d57\u0d62\u0d63\u0dca\u0dcf\u0dd2-\u0dd4\u0dd6\u0ddf\u0e31\u0e34-\u0e3a\u0e47-\u0e4e\u0eb1\u0eb4-\u0eb9\u0ebb\u0ebc\u0ec8-\u0ecd\u0f18\u0f19\u0f35\u0f37\u0f39\u0f71-\u0f7e\u0f80-\u0f84\u0f86\u0f87\u0f90-\u0f97\u0f99-\u0fbc\u0fc6\u102d-\u1030\u1032-\u1037\u1039\u103a\u103d\u103e\u1058\u1059\u105e-\u1060\u1071-\u1074\u1082\u1085\u1086\u108d\u109d\u135f\u1712-\u1714\u1732-\u1734\u1752\u1753\u1772\u1773\u17b7-\u17bd\u17c6\u17c9-\u17d3\u17dd\u180b-\u180d\u18a9\u1920-\u1922\u1927\u1928\u1932\u1939-\u193b\u1a17\u1a18\u1a56\u1a58-\u1a5e\u1a60\u1a62\u1a65-\u1a6c\u1a73-\u1a7c\u1a7f\u1b00-\u1b03\u1b34\u1b36-\u1b3a\u1b3c\u1b42\u1b6b-\u1b73\u1b80\u1b81\u1ba2-\u1ba5\u1ba8\u1ba9\u1c2c-\u1c33\u1c36\u1c37\u1cd0-\u1cd2\u1cd4-\u1ce0\u1ce2-\u1ce8\u1ced\u1dc0-\u1de6\u1dfd-\u1dff\u200c\u200d\u20d0-\u20f0\u2cef-\u2cf1\u2de0-\u2dff\u302a-\u302f\u3099\u309a\ua66f-\ua672\ua67c\ua67d\ua6f0\ua6f1\ua802\ua806\ua80b\ua825\ua826\ua8c4\ua8e0-\ua8f1\ua926-\ua92d\ua947-\ua951\ua980-\ua982\ua9b3\ua9b6-\ua9b9\ua9bc\uaa29-\uaa2e\uaa31\uaa32\uaa35\uaa36\uaa43\uaa4c\uaab0\uaab2-\uaab4\uaab7\uaab8\uaabe\uaabf\uaac1\uabe5\uabe8\uabed\udc00-\udfff\ufb1e\ufe00-\ufe0f\ufe20-\ufe26\uff9e\uff9f]/;
          function se(n) {
            return n.charCodeAt(0) >= 768 && fe.test(n);
          }
          function rn(n, i, a) {
            for (; (a < 0 ? i > 0 : i < n.length) && se(n.charAt(i)); ) i += a;
            return i;
          }
          function Pn(n, i, a) {
            for (var l = i > a ? -1 : 1; ; ) {
              if (i == a) return i;
              var c = (i + a) / 2,
                p = l < 0 ? Math.ceil(c) : Math.floor(c);
              if (p == i) return n(p) ? i : a;
              n(p) ? (a = p) : (i = p + l);
            }
          }
          function wn(n, i, a, l) {
            if (!n) return l(i, a, "ltr", 0);
            for (var c = !1, p = 0; p < n.length; ++p) {
              var m = n[p];
              ((m.from < a && m.to > i) || (i == a && m.to == i)) &&
                (l(Math.max(m.from, i), Math.min(m.to, a), m.level == 1 ? "rtl" : "ltr", p),
                (c = !0));
            }
            c || l(i, a, "ltr");
          }
          var cr = null;
          function Ae(n, i, a) {
            var l;
            cr = null;
            for (var c = 0; c < n.length; ++c) {
              var p = n[c];
              if (p.from < i && p.to > i) return c;
              p.to == i && (p.from != p.to && a == "before" ? (l = c) : (cr = c)),
                p.from == i && (p.from != p.to && a != "before" ? (l = c) : (cr = c));
            }
            return l ?? cr;
          }
          var xn = (function () {
            var n =
                "bbbbbbbbbtstwsbbbbbbbbbbbbbbssstwNN%%%NNNNNN,N,N1111111111NNNNNNNLLLLLLLLLLLLLLLLLLLLLLLLLLNNNNNNLLLLLLLLLLLLLLLLLLLLLLLLLLNNNNbbbbbbsbbbbbbbbbbbbbbbbbbbbbbbbbb,N%%%%NNNNLNNNNN%%11NLNNN1LNNNNNLLLLLLLLLLLLLLLLLLLLLLLNLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLLN",
              i =
                "nnnnnnNNr%%r,rNNmmmmmmmmmmmrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrmmmmmmmmmmmmmmmmmmmmmnnnnnnnnnn%nnrrrmrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrmmmmmmmnNmmmmmmrrmmNmmmmrr1111111111";
            function a(_) {
              return _ <= 247
                ? n.charAt(_)
                : 1424 <= _ && _ <= 1524
                ? "R"
                : 1536 <= _ && _ <= 1785
                ? i.charAt(_ - 1536)
                : 1774 <= _ && _ <= 2220
                ? "r"
                : 8192 <= _ && _ <= 8203
                ? "w"
                : _ == 8204
                ? "b"
                : "L";
            }
            var l = /[\u0590-\u05f4\u0600-\u06ff\u0700-\u08ac]/,
              c = /[stwN]/,
              p = /[LRr]/,
              m = /[Lb1n]/,
              y = /[1n]/;
            function x(_, N, D) {
              (this.level = _), (this.from = N), (this.to = D);
            }
            return function (_, N) {
              var D = N == "ltr" ? "L" : "R";
              if (_.length == 0 || (N == "ltr" && !l.test(_))) return !1;
              for (var W = _.length, q = [], Z = 0; Z < W; ++Z) q.push(a(_.charCodeAt(Z)));
              for (var it = 0, vt = D; it < W; ++it) {
                var bt = q[it];
                bt == "m" ? (q[it] = vt) : (vt = bt);
              }
              for (var Ct = 0, wt = D; Ct < W; ++Ct) {
                var Lt = q[Ct];
                Lt == "1" && wt == "r"
                  ? (q[Ct] = "n")
                  : p.test(Lt) && ((wt = Lt), Lt == "r" && (q[Ct] = "R"));
              }
              for (var zt = 1, Ot = q[0]; zt < W - 1; ++zt) {
                var Xt = q[zt];
                Xt == "+" && Ot == "1" && q[zt + 1] == "1"
                  ? (q[zt] = "1")
                  : Xt == "," && Ot == q[zt + 1] && (Ot == "1" || Ot == "n") && (q[zt] = Ot),
                  (Ot = Xt);
              }
              for (var ge = 0; ge < W; ++ge) {
                var We = q[ge];
                if (We == ",") q[ge] = "N";
                else if (We == "%") {
                  var _e = void 0;
                  for (_e = ge + 1; _e < W && q[_e] == "%"; ++_e);
                  for (
                    var kn = (ge && q[ge - 1] == "!") || (_e < W && q[_e] == "1") ? "1" : "N",
                      cn = ge;
                    cn < _e;
                    ++cn
                  )
                    q[cn] = kn;
                  ge = _e - 1;
                }
              }
              for (var Ne = 0, un = D; Ne < W; ++Ne) {
                var Ve = q[Ne];
                un == "L" && Ve == "1" ? (q[Ne] = "L") : p.test(Ve) && (un = Ve);
              }
              for (var $e = 0; $e < W; ++$e)
                if (c.test(q[$e])) {
                  var Pe = void 0;
                  for (Pe = $e + 1; Pe < W && c.test(q[Pe]); ++Pe);
                  for (
                    var Ee = ($e ? q[$e - 1] : D) == "L",
                      fn = (Pe < W ? q[Pe] : D) == "L",
                      Io = Ee == fn ? (Ee ? "L" : "R") : D,
                      ai = $e;
                    ai < Pe;
                    ++ai
                  )
                    q[ai] = Io;
                  $e = Pe - 1;
                }
              for (var Ze = [], pr, Ue = 0; Ue < W; )
                if (m.test(q[Ue])) {
                  var Wu = Ue;
                  for (++Ue; Ue < W && m.test(q[Ue]); ++Ue);
                  Ze.push(new x(0, Wu, Ue));
                } else {
                  var Pr = Ue,
                    Hi = Ze.length,
                    Bi = N == "rtl" ? 1 : 0;
                  for (++Ue; Ue < W && q[Ue] != "L"; ++Ue);
                  for (var tn = Pr; tn < Ue; )
                    if (y.test(q[tn])) {
                      Pr < tn && (Ze.splice(Hi, 0, new x(1, Pr, tn)), (Hi += Bi));
                      var Fo = tn;
                      for (++tn; tn < Ue && y.test(q[tn]); ++tn);
                      Ze.splice(Hi, 0, new x(2, Fo, tn)), (Hi += Bi), (Pr = tn);
                    } else ++tn;
                  Pr < Ue && Ze.splice(Hi, 0, new x(1, Pr, Ue));
                }
              return (
                N == "ltr" &&
                  (Ze[0].level == 1 &&
                    (pr = _.match(/^\s+/)) &&
                    ((Ze[0].from = pr[0].length), Ze.unshift(new x(0, 0, pr[0].length))),
                  ct(Ze).level == 1 &&
                    (pr = _.match(/\s+$/)) &&
                    ((ct(Ze).to -= pr[0].length), Ze.push(new x(0, W - pr[0].length, W)))),
                N == "rtl" ? Ze.reverse() : Ze
              );
            };
          })();
          function Yt(n, i) {
            var a = n.order;
            return a == null && (a = n.order = xn(n.text, i)), a;
          }
          var Hl = [],
            Rt = function (n, i, a) {
              if (n.addEventListener) n.addEventListener(i, a, !1);
              else if (n.attachEvent) n.attachEvent("on" + i, a);
              else {
                var l = n._handlers || (n._handlers = {});
                l[i] = (l[i] || Hl).concat(a);
              }
            };
          function Cr(n, i) {
            return (n._handlers && n._handlers[i]) || Hl;
          }
          function Ke(n, i, a) {
            if (n.removeEventListener) n.removeEventListener(i, a, !1);
            else if (n.detachEvent) n.detachEvent("on" + i, a);
            else {
              var l = n._handlers,
                c = l && l[i];
              if (c) {
                var p = Et(c, a);
                p > -1 && (l[i] = c.slice(0, p).concat(c.slice(p + 1)));
              }
            }
          }
          function ke(n, i) {
            var a = Cr(n, i);
            if (a.length)
              for (var l = Array.prototype.slice.call(arguments, 2), c = 0; c < a.length; ++c)
                a[c].apply(null, l);
          }
          function Ce(n, i, a) {
            return (
              typeof i == "string" &&
                (i = {
                  type: i,
                  preventDefault: function () {
                    this.defaultPrevented = !0;
                  },
                }),
              ke(n, a || i.type, n, i),
              on(i) || i.codemirrorIgnore
            );
          }
          function qn(n) {
            var i = n._handlers && n._handlers.cursorActivity;
            if (i)
              for (
                var a = n.curOp.cursorActivityHandlers || (n.curOp.cursorActivityHandlers = []),
                  l = 0;
                l < i.length;
                ++l
              )
                Et(a, i[l]) == -1 && a.push(i[l]);
          }
          function _n(n, i) {
            return Cr(n, i).length > 0;
          }
          function Vn(n) {
            (n.prototype.on = function (i, a) {
              Rt(this, i, a);
            }),
              (n.prototype.off = function (i, a) {
                Ke(this, i, a);
              });
          }
          function Xe(n) {
            n.preventDefault ? n.preventDefault() : (n.returnValue = !1);
          }
          function ho(n) {
            n.stopPropagation ? n.stopPropagation() : (n.cancelBubble = !0);
          }
          function on(n) {
            return n.defaultPrevented != null ? n.defaultPrevented : n.returnValue == !1;
          }
          function Yr(n) {
            Xe(n), ho(n);
          }
          function ws(n) {
            return n.target || n.srcElement;
          }
          function Gn(n) {
            var i = n.which;
            return (
              i == null &&
                (n.button & 1 ? (i = 1) : n.button & 2 ? (i = 3) : n.button & 4 && (i = 2)),
              B && n.ctrlKey && i == 1 && (i = 3),
              i
            );
          }
          var Yc = (function () {
              if (d && g < 9) return !1;
              var n = k("div");
              return "draggable" in n || "dragDrop" in n;
            })(),
            po;
          function Bl(n) {
            if (po == null) {
              var i = k("span", "​");
              z(n, k("span", [i, document.createTextNode("x")])),
                n.firstChild.offsetHeight != 0 &&
                  (po = i.offsetWidth <= 1 && i.offsetHeight > 2 && !(d && g < 8));
            }
            var a = po
              ? k("span", "​")
              : k("span", " ", null, "display: inline-block; width: 1px; margin-right: -1px");
            return a.setAttribute("cm-text", ""), a;
          }
          var xs;
          function Zr(n) {
            if (xs != null) return xs;
            var i = z(n, document.createTextNode("AخA")),
              a = H(i, 0, 1).getBoundingClientRect(),
              l = H(i, 1, 2).getBoundingClientRect();
            return G(n), !a || a.left == a.right ? !1 : (xs = l.right - a.right < 3);
          }
          var Hn =
              `

b`.split(/\n/).length != 3
                ? function (n) {
                    for (var i = 0, a = [], l = n.length; i <= l; ) {
                      var c = n.indexOf(
                        `
`,
                        i,
                      );
                      c == -1 && (c = n.length);
                      var p = n.slice(i, n.charAt(c - 1) == "\r" ? c - 1 : c),
                        m = p.indexOf("\r");
                      m != -1 ? (a.push(p.slice(0, m)), (i += m + 1)) : (a.push(p), (i = c + 1));
                    }
                    return a;
                  }
                : function (n) {
                    return n.split(/\r\n?|\n/);
                  },
            Jr = window.getSelection
              ? function (n) {
                  try {
                    return n.selectionStart != n.selectionEnd;
                  } catch {
                    return !1;
                  }
                }
              : function (n) {
                  var i;
                  try {
                    i = n.ownerDocument.selection.createRange();
                  } catch {}
                  return !i || i.parentElement() != n
                    ? !1
                    : i.compareEndPoints("StartToEnd", i) != 0;
                },
            Wl = (function () {
              var n = k("div");
              return "oncopy" in n
                ? !0
                : (n.setAttribute("oncopy", "return;"), typeof n.oncopy == "function");
            })(),
            Kn = null;
          function Zc(n) {
            if (Kn != null) return Kn;
            var i = z(n, k("span", "x")),
              a = i.getBoundingClientRect(),
              l = H(i, 0, 1).getBoundingClientRect();
            return (Kn = Math.abs(a.left - l.left) > 1);
          }
          var go = {},
            Xn = {};
          function Yn(n, i) {
            arguments.length > 2 && (i.dependencies = Array.prototype.slice.call(arguments, 2)),
              (go[n] = i);
          }
          function Pi(n, i) {
            Xn[n] = i;
          }
          function vo(n) {
            if (typeof n == "string" && Xn.hasOwnProperty(n)) n = Xn[n];
            else if (n && typeof n.name == "string" && Xn.hasOwnProperty(n.name)) {
              var i = Xn[n.name];
              typeof i == "string" && (i = { name: i }), (n = Dt(i, n)), (n.name = i.name);
            } else {
              if (typeof n == "string" && /^[\w\-]+\/[\w\-]+\+xml$/.test(n))
                return vo("application/xml");
              if (typeof n == "string" && /^[\w\-]+\/[\w\-]+\+json$/.test(n))
                return vo("application/json");
            }
            return typeof n == "string" ? { name: n } : n || { name: "null" };
          }
          function mo(n, i) {
            i = vo(i);
            var a = go[i.name];
            if (!a) return mo(n, "text/plain");
            var l = a(n, i);
            if (Qr.hasOwnProperty(i.name)) {
              var c = Qr[i.name];
              for (var p in c)
                c.hasOwnProperty(p) && (l.hasOwnProperty(p) && (l["_" + p] = l[p]), (l[p] = c[p]));
            }
            if (((l.name = i.name), i.helperType && (l.helperType = i.helperType), i.modeProps))
              for (var m in i.modeProps) l[m] = i.modeProps[m];
            return l;
          }
          var Qr = {};
          function yo(n, i) {
            var a = Qr.hasOwnProperty(n) ? Qr[n] : (Qr[n] = {});
            rt(i, a);
          }
          function ur(n, i) {
            if (i === !0) return i;
            if (n.copyState) return n.copyState(i);
            var a = {};
            for (var l in i) {
              var c = i[l];
              c instanceof Array && (c = c.concat([])), (a[l] = c);
            }
            return a;
          }
          function _s(n, i) {
            for (var a; n.innerMode && ((a = n.innerMode(i)), !(!a || a.mode == n)); )
              (i = a.state), (n = a.mode);
            return a || { mode: n, state: i };
          }
          function bo(n, i, a) {
            return n.startState ? n.startState(i, a) : !0;
          }
          var Te = function (n, i, a) {
            (this.pos = this.start = 0),
              (this.string = n),
              (this.tabSize = i || 8),
              (this.lastColumnPos = this.lastColumnValue = 0),
              (this.lineStart = 0),
              (this.lineOracle = a);
          };
          (Te.prototype.eol = function () {
            return this.pos >= this.string.length;
          }),
            (Te.prototype.sol = function () {
              return this.pos == this.lineStart;
            }),
            (Te.prototype.peek = function () {
              return this.string.charAt(this.pos) || void 0;
            }),
            (Te.prototype.next = function () {
              if (this.pos < this.string.length) return this.string.charAt(this.pos++);
            }),
            (Te.prototype.eat = function (n) {
              var i = this.string.charAt(this.pos),
                a;
              if ((typeof n == "string" ? (a = i == n) : (a = i && (n.test ? n.test(i) : n(i))), a))
                return ++this.pos, i;
            }),
            (Te.prototype.eatWhile = function (n) {
              for (var i = this.pos; this.eat(n); );
              return this.pos > i;
            }),
            (Te.prototype.eatSpace = function () {
              for (var n = this.pos; /[\s\u00a0]/.test(this.string.charAt(this.pos)); ) ++this.pos;
              return this.pos > n;
            }),
            (Te.prototype.skipToEnd = function () {
              this.pos = this.string.length;
            }),
            (Te.prototype.skipTo = function (n) {
              var i = this.string.indexOf(n, this.pos);
              if (i > -1) return (this.pos = i), !0;
            }),
            (Te.prototype.backUp = function (n) {
              this.pos -= n;
            }),
            (Te.prototype.column = function () {
              return (
                this.lastColumnPos < this.start &&
                  ((this.lastColumnValue = lt(
                    this.string,
                    this.start,
                    this.tabSize,
                    this.lastColumnPos,
                    this.lastColumnValue,
                  )),
                  (this.lastColumnPos = this.start)),
                this.lastColumnValue -
                  (this.lineStart ? lt(this.string, this.lineStart, this.tabSize) : 0)
              );
            }),
            (Te.prototype.indentation = function () {
              return (
                lt(this.string, null, this.tabSize) -
                (this.lineStart ? lt(this.string, this.lineStart, this.tabSize) : 0)
              );
            }),
            (Te.prototype.match = function (n, i, a) {
              if (typeof n == "string") {
                var l = function (m) {
                    return a ? m.toLowerCase() : m;
                  },
                  c = this.string.substr(this.pos, n.length);
                if (l(c) == l(n)) return i !== !1 && (this.pos += n.length), !0;
              } else {
                var p = this.string.slice(this.pos).match(n);
                return p && p.index > 0 ? null : (p && i !== !1 && (this.pos += p[0].length), p);
              }
            }),
            (Te.prototype.current = function () {
              return this.string.slice(this.start, this.pos);
            }),
            (Te.prototype.hideFirstChars = function (n, i) {
              this.lineStart += n;
              try {
                return i();
              } finally {
                this.lineStart -= n;
              }
            }),
            (Te.prototype.lookAhead = function (n) {
              var i = this.lineOracle;
              return i && i.lookAhead(n);
            }),
            (Te.prototype.baseToken = function () {
              var n = this.lineOracle;
              return n && n.baseToken(this.pos);
            });
          function Pt(n, i) {
            if (((i -= n.first), i < 0 || i >= n.size))
              throw new Error("There is no line " + (i + n.first) + " in the document.");
            for (var a = n; !a.lines; )
              for (var l = 0; ; ++l) {
                var c = a.children[l],
                  p = c.chunkSize();
                if (i < p) {
                  a = c;
                  break;
                }
                i -= p;
              }
            return a.lines[i];
          }
          function Tr(n, i, a) {
            var l = [],
              c = i.line;
            return (
              n.iter(i.line, a.line + 1, function (p) {
                var m = p.text;
                c == a.line && (m = m.slice(0, a.ch)),
                  c == i.line && (m = m.slice(i.ch)),
                  l.push(m),
                  ++c;
              }),
              l
            );
          }
          function Ss(n, i, a) {
            var l = [];
            return (
              n.iter(i, a, function (c) {
                l.push(c.text);
              }),
              l
            );
          }
          function On(n, i) {
            var a = i - n.height;
            if (a) for (var l = n; l; l = l.parent) l.height += a;
          }
          function C(n) {
            if (n.parent == null) return null;
            for (var i = n.parent, a = Et(i.lines, n), l = i.parent; l; i = l, l = l.parent)
              for (var c = 0; l.children[c] != i; ++c) a += l.children[c].chunkSize();
            return a + i.first;
          }
          function O(n, i) {
            var a = n.first;
            t: do {
              for (var l = 0; l < n.children.length; ++l) {
                var c = n.children[l],
                  p = c.height;
                if (i < p) {
                  n = c;
                  continue t;
                }
                (i -= p), (a += c.chunkSize());
              }
              return a;
            } while (!n.lines);
            for (var m = 0; m < n.lines.length; ++m) {
              var y = n.lines[m],
                x = y.height;
              if (i < x) break;
              i -= x;
            }
            return a + m;
          }
          function et(n, i) {
            return i >= n.first && i < n.first + n.size;
          }
          function dt(n, i) {
            return String(n.lineNumberFormatter(i + n.firstLineNumber));
          }
          function X(n, i, a) {
            if ((a === void 0 && (a = null), !(this instanceof X))) return new X(n, i, a);
            (this.line = n), (this.ch = i), (this.sticky = a);
          }
          function _t(n, i) {
            return n.line - i.line || n.ch - i.ch;
          }
          function ce(n, i) {
            return n.sticky == i.sticky && _t(n, i) == 0;
          }
          function Fe(n) {
            return X(n.line, n.ch);
          }
          function sn(n, i) {
            return _t(n, i) < 0 ? i : n;
          }
          function wo(n, i) {
            return _t(n, i) < 0 ? n : i;
          }
          function fd(n, i) {
            return Math.max(n.first, Math.min(i, n.first + n.size - 1));
          }
          function Wt(n, i) {
            if (i.line < n.first) return X(n.first, 0);
            var a = n.first + n.size - 1;
            return i.line > a ? X(a, Pt(n, a).text.length) : Yb(i, Pt(n, i.line).text.length);
          }
          function Yb(n, i) {
            var a = n.ch;
            return a == null || a > i ? X(n.line, i) : a < 0 ? X(n.line, 0) : n;
          }
          function hd(n, i) {
            for (var a = [], l = 0; l < i.length; l++) a[l] = Wt(n, i[l]);
            return a;
          }
          var Ul = function (n, i) {
              (this.state = n), (this.lookAhead = i);
            },
            fr = function (n, i, a, l) {
              (this.state = i),
                (this.doc = n),
                (this.line = a),
                (this.maxLookAhead = l || 0),
                (this.baseTokens = null),
                (this.baseTokenPos = 1);
            };
          (fr.prototype.lookAhead = function (n) {
            var i = this.doc.getLine(this.line + n);
            return i != null && n > this.maxLookAhead && (this.maxLookAhead = n), i;
          }),
            (fr.prototype.baseToken = function (n) {
              if (!this.baseTokens) return null;
              for (; this.baseTokens[this.baseTokenPos] <= n; ) this.baseTokenPos += 2;
              var i = this.baseTokens[this.baseTokenPos + 1];
              return {
                type: i && i.replace(/( |^)overlay .*/, ""),
                size: this.baseTokens[this.baseTokenPos] - n,
              };
            }),
            (fr.prototype.nextLine = function () {
              this.line++, this.maxLookAhead > 0 && this.maxLookAhead--;
            }),
            (fr.fromSaved = function (n, i, a) {
              return i instanceof Ul
                ? new fr(n, ur(n.mode, i.state), a, i.lookAhead)
                : new fr(n, ur(n.mode, i), a);
            }),
            (fr.prototype.save = function (n) {
              var i = n !== !1 ? ur(this.doc.mode, this.state) : this.state;
              return this.maxLookAhead > 0 ? new Ul(i, this.maxLookAhead) : i;
            });
          function dd(n, i, a, l) {
            var c = [n.state.modeGen],
              p = {};
            bd(
              n,
              i.text,
              n.doc.mode,
              a,
              function (_, N) {
                return c.push(_, N);
              },
              p,
              l,
            );
            for (
              var m = a.state,
                y = function (_) {
                  a.baseTokens = c;
                  var N = n.state.overlays[_],
                    D = 1,
                    W = 0;
                  (a.state = !0),
                    bd(
                      n,
                      i.text,
                      N.mode,
                      a,
                      function (q, Z) {
                        for (var it = D; W < q; ) {
                          var vt = c[D];
                          vt > q && c.splice(D, 1, q, c[D + 1], vt),
                            (D += 2),
                            (W = Math.min(q, vt));
                        }
                        if (Z)
                          if (N.opaque) c.splice(it, D - it, q, "overlay " + Z), (D = it + 2);
                          else
                            for (; it < D; it += 2) {
                              var bt = c[it + 1];
                              c[it + 1] = (bt ? bt + " " : "") + "overlay " + Z;
                            }
                      },
                      p,
                    ),
                    (a.state = m),
                    (a.baseTokens = null),
                    (a.baseTokenPos = 1);
                },
                x = 0;
              x < n.state.overlays.length;
              ++x
            )
              y(x);
            return { styles: c, classes: p.bgClass || p.textClass ? p : null };
          }
          function pd(n, i, a) {
            if (!i.styles || i.styles[0] != n.state.modeGen) {
              var l = ks(n, C(i)),
                c = i.text.length > n.options.maxHighlightLength && ur(n.doc.mode, l.state),
                p = dd(n, i, l);
              c && (l.state = c),
                (i.stateAfter = l.save(!c)),
                (i.styles = p.styles),
                p.classes
                  ? (i.styleClasses = p.classes)
                  : i.styleClasses && (i.styleClasses = null),
                a === n.doc.highlightFrontier &&
                  (n.doc.modeFrontier = Math.max(n.doc.modeFrontier, ++n.doc.highlightFrontier));
            }
            return i.styles;
          }
          function ks(n, i, a) {
            var l = n.doc,
              c = n.display;
            if (!l.mode.startState) return new fr(l, !0, i);
            var p = Zb(n, i, a),
              m = p > l.first && Pt(l, p - 1).stateAfter,
              y = m ? fr.fromSaved(l, m, p) : new fr(l, bo(l.mode), p);
            return (
              l.iter(p, i, function (x) {
                Jc(n, x.text, y);
                var _ = y.line;
                (x.stateAfter =
                  _ == i - 1 || _ % 5 == 0 || (_ >= c.viewFrom && _ < c.viewTo) ? y.save() : null),
                  y.nextLine();
              }),
              a && (l.modeFrontier = y.line),
              y
            );
          }
          function Jc(n, i, a, l) {
            var c = n.doc.mode,
              p = new Te(i, n.options.tabSize, a);
            for (p.start = p.pos = l || 0, i == "" && gd(c, a.state); !p.eol(); )
              Qc(c, p, a.state), (p.start = p.pos);
          }
          function gd(n, i) {
            if (n.blankLine) return n.blankLine(i);
            if (n.innerMode) {
              var a = _s(n, i);
              if (a.mode.blankLine) return a.mode.blankLine(a.state);
            }
          }
          function Qc(n, i, a, l) {
            for (var c = 0; c < 10; c++) {
              l && (l[0] = _s(n, a).mode);
              var p = n.token(i, a);
              if (i.pos > i.start) return p;
            }
            throw new Error("Mode " + n.name + " failed to advance stream.");
          }
          var vd = function (n, i, a) {
            (this.start = n.start),
              (this.end = n.pos),
              (this.string = n.current()),
              (this.type = i || null),
              (this.state = a);
          };
          function md(n, i, a, l) {
            var c = n.doc,
              p = c.mode,
              m;
            i = Wt(c, i);
            var y = Pt(c, i.line),
              x = ks(n, i.line, a),
              _ = new Te(y.text, n.options.tabSize, x),
              N;
            for (l && (N = []); (l || _.pos < i.ch) && !_.eol(); )
              (_.start = _.pos),
                (m = Qc(p, _, x.state)),
                l && N.push(new vd(_, m, ur(c.mode, x.state)));
            return l ? N : new vd(_, m, x.state);
          }
          function yd(n, i) {
            if (n)
              for (;;) {
                var a = n.match(/(?:^|\s+)line-(background-)?(\S+)/);
                if (!a) break;
                n = n.slice(0, a.index) + n.slice(a.index + a[0].length);
                var l = a[1] ? "bgClass" : "textClass";
                i[l] == null
                  ? (i[l] = a[2])
                  : new RegExp("(?:^|\\s)" + a[2] + "(?:$|\\s)").test(i[l]) || (i[l] += " " + a[2]);
              }
            return n;
          }
          function bd(n, i, a, l, c, p, m) {
            var y = a.flattenSpans;
            y == null && (y = n.options.flattenSpans);
            var x = 0,
              _ = null,
              N = new Te(i, n.options.tabSize, l),
              D,
              W = n.options.addModeClass && [null];
            for (i == "" && yd(gd(a, l.state), p); !N.eol(); ) {
              if (
                (N.pos > n.options.maxHighlightLength
                  ? ((y = !1), m && Jc(n, i, l, N.pos), (N.pos = i.length), (D = null))
                  : (D = yd(Qc(a, N, l.state, W), p)),
                W)
              ) {
                var q = W[0].name;
                q && (D = "m-" + (D ? q + " " + D : q));
              }
              if (!y || _ != D) {
                for (; x < N.start; ) (x = Math.min(N.start, x + 5e3)), c(x, _);
                _ = D;
              }
              N.start = N.pos;
            }
            for (; x < N.pos; ) {
              var Z = Math.min(N.pos, x + 5e3);
              c(Z, _), (x = Z);
            }
          }
          function Zb(n, i, a) {
            for (
              var l, c, p = n.doc, m = a ? -1 : i - (n.doc.mode.innerMode ? 1e3 : 100), y = i;
              y > m;
              --y
            ) {
              if (y <= p.first) return p.first;
              var x = Pt(p, y - 1),
                _ = x.stateAfter;
              if (_ && (!a || y + (_ instanceof Ul ? _.lookAhead : 0) <= p.modeFrontier)) return y;
              var N = lt(x.text, null, n.options.tabSize);
              (c == null || l > N) && ((c = y - 1), (l = N));
            }
            return c;
          }
          function Jb(n, i) {
            if (((n.modeFrontier = Math.min(n.modeFrontier, i)), !(n.highlightFrontier < i - 10))) {
              for (var a = n.first, l = i - 1; l > a; l--) {
                var c = Pt(n, l).stateAfter;
                if (c && (!(c instanceof Ul) || l + c.lookAhead < i)) {
                  a = l + 1;
                  break;
                }
              }
              n.highlightFrontier = Math.min(n.highlightFrontier, a);
            }
          }
          var wd = !1,
            Er = !1;
          function Qb() {
            wd = !0;
          }
          function tw() {
            Er = !0;
          }
          function jl(n, i, a) {
            (this.marker = n), (this.from = i), (this.to = a);
          }
          function Cs(n, i) {
            if (n)
              for (var a = 0; a < n.length; ++a) {
                var l = n[a];
                if (l.marker == i) return l;
              }
          }
          function ew(n, i) {
            for (var a, l = 0; l < n.length; ++l) n[l] != i && (a || (a = [])).push(n[l]);
            return a;
          }
          function nw(n, i, a) {
            var l = a && window.WeakSet && (a.markedSpans || (a.markedSpans = new WeakSet()));
            l && n.markedSpans && l.has(n.markedSpans)
              ? n.markedSpans.push(i)
              : ((n.markedSpans = n.markedSpans ? n.markedSpans.concat([i]) : [i]),
                l && l.add(n.markedSpans)),
              i.marker.attachLine(n);
          }
          function rw(n, i, a) {
            var l;
            if (n)
              for (var c = 0; c < n.length; ++c) {
                var p = n[c],
                  m = p.marker,
                  y = p.from == null || (m.inclusiveLeft ? p.from <= i : p.from < i);
                if (y || (p.from == i && m.type == "bookmark" && (!a || !p.marker.insertLeft))) {
                  var x = p.to == null || (m.inclusiveRight ? p.to >= i : p.to > i);
                  (l || (l = [])).push(new jl(m, p.from, x ? null : p.to));
                }
              }
            return l;
          }
          function iw(n, i, a) {
            var l;
            if (n)
              for (var c = 0; c < n.length; ++c) {
                var p = n[c],
                  m = p.marker,
                  y = p.to == null || (m.inclusiveRight ? p.to >= i : p.to > i);
                if (y || (p.from == i && m.type == "bookmark" && (!a || p.marker.insertLeft))) {
                  var x = p.from == null || (m.inclusiveLeft ? p.from <= i : p.from < i);
                  (l || (l = [])).push(
                    new jl(m, x ? null : p.from - i, p.to == null ? null : p.to - i),
                  );
                }
              }
            return l;
          }
          function tu(n, i) {
            if (i.full) return null;
            var a = et(n, i.from.line) && Pt(n, i.from.line).markedSpans,
              l = et(n, i.to.line) && Pt(n, i.to.line).markedSpans;
            if (!a && !l) return null;
            var c = i.from.ch,
              p = i.to.ch,
              m = _t(i.from, i.to) == 0,
              y = rw(a, c, m),
              x = iw(l, p, m),
              _ = i.text.length == 1,
              N = ct(i.text).length + (_ ? c : 0);
            if (y)
              for (var D = 0; D < y.length; ++D) {
                var W = y[D];
                if (W.to == null) {
                  var q = Cs(x, W.marker);
                  q ? _ && (W.to = q.to == null ? null : q.to + N) : (W.to = c);
                }
              }
            if (x)
              for (var Z = 0; Z < x.length; ++Z) {
                var it = x[Z];
                if ((it.to != null && (it.to += N), it.from == null)) {
                  var vt = Cs(y, it.marker);
                  vt || ((it.from = N), _ && (y || (y = [])).push(it));
                } else (it.from += N), _ && (y || (y = [])).push(it);
              }
            y && (y = xd(y)), x && x != y && (x = xd(x));
            var bt = [y];
            if (!_) {
              var Ct = i.text.length - 2,
                wt;
              if (Ct > 0 && y)
                for (var Lt = 0; Lt < y.length; ++Lt)
                  y[Lt].to == null && (wt || (wt = [])).push(new jl(y[Lt].marker, null, null));
              for (var zt = 0; zt < Ct; ++zt) bt.push(wt);
              bt.push(x);
            }
            return bt;
          }
          function xd(n) {
            for (var i = 0; i < n.length; ++i) {
              var a = n[i];
              a.from != null &&
                a.from == a.to &&
                a.marker.clearWhenEmpty !== !1 &&
                n.splice(i--, 1);
            }
            return n.length ? n : null;
          }
          function ow(n, i, a) {
            var l = null;
            if (
              (n.iter(i.line, a.line + 1, function (q) {
                if (q.markedSpans)
                  for (var Z = 0; Z < q.markedSpans.length; ++Z) {
                    var it = q.markedSpans[Z].marker;
                    it.readOnly && (!l || Et(l, it) == -1) && (l || (l = [])).push(it);
                  }
              }),
              !l)
            )
              return null;
            for (var c = [{ from: i, to: a }], p = 0; p < l.length; ++p)
              for (var m = l[p], y = m.find(0), x = 0; x < c.length; ++x) {
                var _ = c[x];
                if (!(_t(_.to, y.from) < 0 || _t(_.from, y.to) > 0)) {
                  var N = [x, 1],
                    D = _t(_.from, y.from),
                    W = _t(_.to, y.to);
                  (D < 0 || (!m.inclusiveLeft && !D)) && N.push({ from: _.from, to: y.from }),
                    (W > 0 || (!m.inclusiveRight && !W)) && N.push({ from: y.to, to: _.to }),
                    c.splice.apply(c, N),
                    (x += N.length - 3);
                }
              }
            return c;
          }
          function _d(n) {
            var i = n.markedSpans;
            if (i) {
              for (var a = 0; a < i.length; ++a) i[a].marker.detachLine(n);
              n.markedSpans = null;
            }
          }
          function Sd(n, i) {
            if (i) {
              for (var a = 0; a < i.length; ++a) i[a].marker.attachLine(n);
              n.markedSpans = i;
            }
          }
          function Vl(n) {
            return n.inclusiveLeft ? -1 : 0;
          }
          function Gl(n) {
            return n.inclusiveRight ? 1 : 0;
          }
          function eu(n, i) {
            var a = n.lines.length - i.lines.length;
            if (a != 0) return a;
            var l = n.find(),
              c = i.find(),
              p = _t(l.from, c.from) || Vl(n) - Vl(i);
            if (p) return -p;
            var m = _t(l.to, c.to) || Gl(n) - Gl(i);
            return m || i.id - n.id;
          }
          function kd(n, i) {
            var a = Er && n.markedSpans,
              l;
            if (a)
              for (var c = void 0, p = 0; p < a.length; ++p)
                (c = a[p]),
                  c.marker.collapsed &&
                    (i ? c.from : c.to) == null &&
                    (!l || eu(l, c.marker) < 0) &&
                    (l = c.marker);
            return l;
          }
          function Cd(n) {
            return kd(n, !0);
          }
          function Kl(n) {
            return kd(n, !1);
          }
          function sw(n, i) {
            var a = Er && n.markedSpans,
              l;
            if (a)
              for (var c = 0; c < a.length; ++c) {
                var p = a[c];
                p.marker.collapsed &&
                  (p.from == null || p.from < i) &&
                  (p.to == null || p.to > i) &&
                  (!l || eu(l, p.marker) < 0) &&
                  (l = p.marker);
              }
            return l;
          }
          function Td(n, i, a, l, c) {
            var p = Pt(n, i),
              m = Er && p.markedSpans;
            if (m)
              for (var y = 0; y < m.length; ++y) {
                var x = m[y];
                if (x.marker.collapsed) {
                  var _ = x.marker.find(0),
                    N = _t(_.from, a) || Vl(x.marker) - Vl(c),
                    D = _t(_.to, l) || Gl(x.marker) - Gl(c);
                  if (
                    !((N >= 0 && D <= 0) || (N <= 0 && D >= 0)) &&
                    ((N <= 0 &&
                      (x.marker.inclusiveRight && c.inclusiveLeft
                        ? _t(_.to, a) >= 0
                        : _t(_.to, a) > 0)) ||
                      (N >= 0 &&
                        (x.marker.inclusiveRight && c.inclusiveLeft
                          ? _t(_.from, l) <= 0
                          : _t(_.from, l) < 0)))
                  )
                    return !0;
                }
              }
          }
          function Zn(n) {
            for (var i; (i = Cd(n)); ) n = i.find(-1, !0).line;
            return n;
          }
          function lw(n) {
            for (var i; (i = Kl(n)); ) n = i.find(1, !0).line;
            return n;
          }
          function aw(n) {
            for (var i, a; (i = Kl(n)); ) (n = i.find(1, !0).line), (a || (a = [])).push(n);
            return a;
          }
          function nu(n, i) {
            var a = Pt(n, i),
              l = Zn(a);
            return a == l ? i : C(l);
          }
          function Ed(n, i) {
            if (i > n.lastLine()) return i;
            var a = Pt(n, i),
              l;
            if (!ti(n, a)) return i;
            for (; (l = Kl(a)); ) a = l.find(1, !0).line;
            return C(a) + 1;
          }
          function ti(n, i) {
            var a = Er && i.markedSpans;
            if (a) {
              for (var l = void 0, c = 0; c < a.length; ++c)
                if (((l = a[c]), !!l.marker.collapsed)) {
                  if (l.from == null) return !0;
                  if (!l.marker.widgetNode && l.from == 0 && l.marker.inclusiveLeft && ru(n, i, l))
                    return !0;
                }
            }
          }
          function ru(n, i, a) {
            if (a.to == null) {
              var l = a.marker.find(1, !0);
              return ru(n, l.line, Cs(l.line.markedSpans, a.marker));
            }
            if (a.marker.inclusiveRight && a.to == i.text.length) return !0;
            for (var c = void 0, p = 0; p < i.markedSpans.length; ++p)
              if (
                ((c = i.markedSpans[p]),
                c.marker.collapsed &&
                  !c.marker.widgetNode &&
                  c.from == a.to &&
                  (c.to == null || c.to != a.from) &&
                  (c.marker.inclusiveLeft || a.marker.inclusiveRight) &&
                  ru(n, i, c))
              )
                return !0;
          }
          function Lr(n) {
            n = Zn(n);
            for (var i = 0, a = n.parent, l = 0; l < a.lines.length; ++l) {
              var c = a.lines[l];
              if (c == n) break;
              i += c.height;
            }
            for (var p = a.parent; p; a = p, p = a.parent)
              for (var m = 0; m < p.children.length; ++m) {
                var y = p.children[m];
                if (y == a) break;
                i += y.height;
              }
            return i;
          }
          function Xl(n) {
            if (n.height == 0) return 0;
            for (var i = n.text.length, a, l = n; (a = Cd(l)); ) {
              var c = a.find(0, !0);
              (l = c.from.line), (i += c.from.ch - c.to.ch);
            }
            for (l = n; (a = Kl(l)); ) {
              var p = a.find(0, !0);
              (i -= l.text.length - p.from.ch), (l = p.to.line), (i += l.text.length - p.to.ch);
            }
            return i;
          }
          function iu(n) {
            var i = n.display,
              a = n.doc;
            (i.maxLine = Pt(a, a.first)),
              (i.maxLineLength = Xl(i.maxLine)),
              (i.maxLineChanged = !0),
              a.iter(function (l) {
                var c = Xl(l);
                c > i.maxLineLength && ((i.maxLineLength = c), (i.maxLine = l));
              });
          }
          var xo = function (n, i, a) {
            (this.text = n), Sd(this, i), (this.height = a ? a(this) : 1);
          };
          (xo.prototype.lineNo = function () {
            return C(this);
          }),
            Vn(xo);
          function cw(n, i, a, l) {
            (n.text = i),
              n.stateAfter && (n.stateAfter = null),
              n.styles && (n.styles = null),
              n.order != null && (n.order = null),
              _d(n),
              Sd(n, a);
            var c = l ? l(n) : 1;
            c != n.height && On(n, c);
          }
          function uw(n) {
            (n.parent = null), _d(n);
          }
          var fw = {},
            hw = {};
          function Ld(n, i) {
            if (!n || /^\s*$/.test(n)) return null;
            var a = i.addModeClass ? hw : fw;
            return a[n] || (a[n] = n.replace(/\S+/g, "cm-$&"));
          }
          function Ad(n, i) {
            var a = F("span", null, null, v ? "padding-right: .1px" : null),
              l = {
                pre: F("pre", [a], "CodeMirror-line"),
                content: a,
                col: 0,
                pos: 0,
                cm: n,
                trailingSpace: !1,
                splitSpaces: n.getOption("lineWrapping"),
              };
            i.measure = {};
            for (var c = 0; c <= (i.rest ? i.rest.length : 0); c++) {
              var p = c ? i.rest[c - 1] : i.line,
                m = void 0;
              (l.pos = 0),
                (l.addToken = pw),
                Zr(n.display.measure) &&
                  (m = Yt(p, n.doc.direction)) &&
                  (l.addToken = vw(l.addToken, m)),
                (l.map = []);
              var y = i != n.display.externalMeasured && C(p);
              mw(p, l, pd(n, p, y)),
                p.styleClasses &&
                  (p.styleClasses.bgClass &&
                    (l.bgClass = qt(p.styleClasses.bgClass, l.bgClass || "")),
                  p.styleClasses.textClass &&
                    (l.textClass = qt(p.styleClasses.textClass, l.textClass || ""))),
                l.map.length == 0 && l.map.push(0, 0, l.content.appendChild(Bl(n.display.measure))),
                c == 0
                  ? ((i.measure.map = l.map), (i.measure.cache = {}))
                  : ((i.measure.maps || (i.measure.maps = [])).push(l.map),
                    (i.measure.caches || (i.measure.caches = [])).push({}));
            }
            if (v) {
              var x = l.content.lastChild;
              (/\bcm-tab\b/.test(x.className) || (x.querySelector && x.querySelector(".cm-tab"))) &&
                (l.content.className = "cm-tab-wrap-hack");
            }
            return (
              ke(n, "renderLine", n, i.line, l.pre),
              l.pre.className && (l.textClass = qt(l.pre.className, l.textClass || "")),
              l
            );
          }
          function dw(n) {
            var i = k("span", "•", "cm-invalidchar");
            return (
              (i.title = "\\u" + n.charCodeAt(0).toString(16)),
              i.setAttribute("aria-label", i.title),
              i
            );
          }
          function pw(n, i, a, l, c, p, m) {
            if (i) {
              var y = n.splitSpaces ? gw(i, n.trailingSpace) : i,
                x = n.cm.state.specialChars,
                _ = !1,
                N;
              if (!x.test(i))
                (n.col += i.length),
                  (N = document.createTextNode(y)),
                  n.map.push(n.pos, n.pos + i.length, N),
                  d && g < 9 && (_ = !0),
                  (n.pos += i.length);
              else {
                N = document.createDocumentFragment();
                for (var D = 0; ; ) {
                  x.lastIndex = D;
                  var W = x.exec(i),
                    q = W ? W.index - D : i.length - D;
                  if (q) {
                    var Z = document.createTextNode(y.slice(D, D + q));
                    d && g < 9 ? N.appendChild(k("span", [Z])) : N.appendChild(Z),
                      n.map.push(n.pos, n.pos + q, Z),
                      (n.col += q),
                      (n.pos += q);
                  }
                  if (!W) break;
                  D += q + 1;
                  var it = void 0;
                  if (W[0] == "	") {
                    var vt = n.cm.options.tabSize,
                      bt = vt - (n.col % vt);
                    (it = N.appendChild(k("span", mt(bt), "cm-tab"))),
                      it.setAttribute("role", "presentation"),
                      it.setAttribute("cm-text", "	"),
                      (n.col += bt);
                  } else
                    W[0] == "\r" ||
                    W[0] ==
                      `
`
                      ? ((it = N.appendChild(
                          k("span", W[0] == "\r" ? "␍" : "␤", "cm-invalidchar"),
                        )),
                        it.setAttribute("cm-text", W[0]),
                        (n.col += 1))
                      : ((it = n.cm.options.specialCharPlaceholder(W[0])),
                        it.setAttribute("cm-text", W[0]),
                        d && g < 9 ? N.appendChild(k("span", [it])) : N.appendChild(it),
                        (n.col += 1));
                  n.map.push(n.pos, n.pos + 1, it), n.pos++;
                }
              }
              if (
                ((n.trailingSpace = y.charCodeAt(i.length - 1) == 32), a || l || c || _ || p || m)
              ) {
                var Ct = a || "";
                l && (Ct += l), c && (Ct += c);
                var wt = k("span", [N], Ct, p);
                if (m)
                  for (var Lt in m)
                    m.hasOwnProperty(Lt) &&
                      Lt != "style" &&
                      Lt != "class" &&
                      wt.setAttribute(Lt, m[Lt]);
                return n.content.appendChild(wt);
              }
              n.content.appendChild(N);
            }
          }
          function gw(n, i) {
            if (n.length > 1 && !/  /.test(n)) return n;
            for (var a = i, l = "", c = 0; c < n.length; c++) {
              var p = n.charAt(c);
              p == " " && a && (c == n.length - 1 || n.charCodeAt(c + 1) == 32) && (p = " "),
                (l += p),
                (a = p == " ");
            }
            return l;
          }
          function vw(n, i) {
            return function (a, l, c, p, m, y, x) {
              c = c ? c + " cm-force-border" : "cm-force-border";
              for (var _ = a.pos, N = _ + l.length; ; ) {
                for (
                  var D = void 0, W = 0;
                  W < i.length && ((D = i[W]), !(D.to > _ && D.from <= _));
                  W++
                );
                if (D.to >= N) return n(a, l, c, p, m, y, x);
                n(a, l.slice(0, D.to - _), c, p, null, y, x),
                  (p = null),
                  (l = l.slice(D.to - _)),
                  (_ = D.to);
              }
            };
          }
          function Md(n, i, a, l) {
            var c = !l && a.widgetNode;
            c && n.map.push(n.pos, n.pos + i, c),
              !l &&
                n.cm.display.input.needsContentAttribute &&
                (c || (c = n.content.appendChild(document.createElement("span"))),
                c.setAttribute("cm-marker", a.id)),
              c && (n.cm.display.input.setUneditable(c), n.content.appendChild(c)),
              (n.pos += i),
              (n.trailingSpace = !1);
          }
          function mw(n, i, a) {
            var l = n.markedSpans,
              c = n.text,
              p = 0;
            if (!l) {
              for (var m = 1; m < a.length; m += 2)
                i.addToken(i, c.slice(p, (p = a[m])), Ld(a[m + 1], i.cm.options));
              return;
            }
            for (var y = c.length, x = 0, _ = 1, N = "", D, W, q = 0, Z, it, vt, bt, Ct; ; ) {
              if (q == x) {
                (Z = it = vt = W = ""), (Ct = null), (bt = null), (q = 1 / 0);
                for (var wt = [], Lt = void 0, zt = 0; zt < l.length; ++zt) {
                  var Ot = l[zt],
                    Xt = Ot.marker;
                  if (Xt.type == "bookmark" && Ot.from == x && Xt.widgetNode) wt.push(Xt);
                  else if (
                    Ot.from <= x &&
                    (Ot.to == null || Ot.to > x || (Xt.collapsed && Ot.to == x && Ot.from == x))
                  ) {
                    if (
                      (Ot.to != null && Ot.to != x && q > Ot.to && ((q = Ot.to), (it = "")),
                      Xt.className && (Z += " " + Xt.className),
                      Xt.css && (W = (W ? W + ";" : "") + Xt.css),
                      Xt.startStyle && Ot.from == x && (vt += " " + Xt.startStyle),
                      Xt.endStyle && Ot.to == q && (Lt || (Lt = [])).push(Xt.endStyle, Ot.to),
                      Xt.title && ((Ct || (Ct = {})).title = Xt.title),
                      Xt.attributes)
                    )
                      for (var ge in Xt.attributes) (Ct || (Ct = {}))[ge] = Xt.attributes[ge];
                    Xt.collapsed && (!bt || eu(bt.marker, Xt) < 0) && (bt = Ot);
                  } else Ot.from > x && q > Ot.from && (q = Ot.from);
                }
                if (Lt)
                  for (var We = 0; We < Lt.length; We += 2) Lt[We + 1] == q && (it += " " + Lt[We]);
                if (!bt || bt.from == x) for (var _e = 0; _e < wt.length; ++_e) Md(i, 0, wt[_e]);
                if (bt && (bt.from || 0) == x) {
                  if (
                    (Md(i, (bt.to == null ? y + 1 : bt.to) - x, bt.marker, bt.from == null),
                    bt.to == null)
                  )
                    return;
                  bt.to == x && (bt = !1);
                }
              }
              if (x >= y) break;
              for (var kn = Math.min(y, q); ; ) {
                if (N) {
                  var cn = x + N.length;
                  if (!bt) {
                    var Ne = cn > kn ? N.slice(0, kn - x) : N;
                    i.addToken(i, Ne, D ? D + Z : Z, vt, x + Ne.length == q ? it : "", W, Ct);
                  }
                  if (cn >= kn) {
                    (N = N.slice(kn - x)), (x = kn);
                    break;
                  }
                  (x = cn), (vt = "");
                }
                (N = c.slice(p, (p = a[_++]))), (D = Ld(a[_++], i.cm.options));
              }
            }
          }
          function Nd(n, i, a) {
            (this.line = i),
              (this.rest = aw(i)),
              (this.size = this.rest ? C(ct(this.rest)) - a + 1 : 1),
              (this.node = this.text = null),
              (this.hidden = ti(n, i));
          }
          function Yl(n, i, a) {
            for (var l = [], c, p = i; p < a; p = c) {
              var m = new Nd(n.doc, Pt(n.doc, p), p);
              (c = p + m.size), l.push(m);
            }
            return l;
          }
          var _o = null;
          function yw(n) {
            _o ? _o.ops.push(n) : (n.ownsGroup = _o = { ops: [n], delayedCallbacks: [] });
          }
          function bw(n) {
            var i = n.delayedCallbacks,
              a = 0;
            do {
              for (; a < i.length; a++) i[a].call(null);
              for (var l = 0; l < n.ops.length; l++) {
                var c = n.ops[l];
                if (c.cursorActivityHandlers)
                  for (; c.cursorActivityCalled < c.cursorActivityHandlers.length; )
                    c.cursorActivityHandlers[c.cursorActivityCalled++].call(null, c.cm);
              }
            } while (a < i.length);
          }
          function ww(n, i) {
            var a = n.ownsGroup;
            if (a)
              try {
                bw(a);
              } finally {
                (_o = null), i(a);
              }
          }
          var Ts = null;
          function qe(n, i) {
            var a = Cr(n, i);
            if (a.length) {
              var l = Array.prototype.slice.call(arguments, 2),
                c;
              _o ? (c = _o.delayedCallbacks) : Ts ? (c = Ts) : ((c = Ts = []), setTimeout(xw, 0));
              for (
                var p = function (y) {
                    c.push(function () {
                      return a[y].apply(null, l);
                    });
                  },
                  m = 0;
                m < a.length;
                ++m
              )
                p(m);
            }
          }
          function xw() {
            var n = Ts;
            Ts = null;
            for (var i = 0; i < n.length; ++i) n[i]();
          }
          function Pd(n, i, a, l) {
            for (var c = 0; c < i.changes.length; c++) {
              var p = i.changes[c];
              p == "text"
                ? Sw(n, i)
                : p == "gutter"
                ? Dd(n, i, a, l)
                : p == "class"
                ? ou(n, i)
                : p == "widget" && kw(n, i, l);
            }
            i.changes = null;
          }
          function Es(n) {
            return (
              n.node == n.text &&
                ((n.node = k("div", null, null, "position: relative")),
                n.text.parentNode && n.text.parentNode.replaceChild(n.node, n.text),
                n.node.appendChild(n.text),
                d && g < 8 && (n.node.style.zIndex = 2)),
              n.node
            );
          }
          function _w(n, i) {
            var a = i.bgClass ? i.bgClass + " " + (i.line.bgClass || "") : i.line.bgClass;
            if ((a && (a += " CodeMirror-linebackground"), i.background))
              a
                ? (i.background.className = a)
                : (i.background.parentNode.removeChild(i.background), (i.background = null));
            else if (a) {
              var l = Es(i);
              (i.background = l.insertBefore(k("div", null, a), l.firstChild)),
                n.display.input.setUneditable(i.background);
            }
          }
          function Od(n, i) {
            var a = n.display.externalMeasured;
            return a && a.line == i.line
              ? ((n.display.externalMeasured = null), (i.measure = a.measure), a.built)
              : Ad(n, i);
          }
          function Sw(n, i) {
            var a = i.text.className,
              l = Od(n, i);
            i.text == i.node && (i.node = l.pre),
              i.text.parentNode.replaceChild(l.pre, i.text),
              (i.text = l.pre),
              l.bgClass != i.bgClass || l.textClass != i.textClass
                ? ((i.bgClass = l.bgClass), (i.textClass = l.textClass), ou(n, i))
                : a && (i.text.className = a);
          }
          function ou(n, i) {
            _w(n, i),
              i.line.wrapClass
                ? (Es(i).className = i.line.wrapClass)
                : i.node != i.text && (i.node.className = "");
            var a = i.textClass ? i.textClass + " " + (i.line.textClass || "") : i.line.textClass;
            i.text.className = a || "";
          }
          function Dd(n, i, a, l) {
            if (
              (i.gutter && (i.node.removeChild(i.gutter), (i.gutter = null)),
              i.gutterBackground &&
                (i.node.removeChild(i.gutterBackground), (i.gutterBackground = null)),
              i.line.gutterClass)
            ) {
              var c = Es(i);
              (i.gutterBackground = k(
                "div",
                null,
                "CodeMirror-gutter-background " + i.line.gutterClass,
                "left: " +
                  (n.options.fixedGutter ? l.fixedPos : -l.gutterTotalWidth) +
                  "px; width: " +
                  l.gutterTotalWidth +
                  "px",
              )),
                n.display.input.setUneditable(i.gutterBackground),
                c.insertBefore(i.gutterBackground, i.text);
            }
            var p = i.line.gutterMarkers;
            if (n.options.lineNumbers || p) {
              var m = Es(i),
                y = (i.gutter = k(
                  "div",
                  null,
                  "CodeMirror-gutter-wrapper",
                  "left: " + (n.options.fixedGutter ? l.fixedPos : -l.gutterTotalWidth) + "px",
                ));
              if (
                (y.setAttribute("aria-hidden", "true"),
                n.display.input.setUneditable(y),
                m.insertBefore(y, i.text),
                i.line.gutterClass && (y.className += " " + i.line.gutterClass),
                n.options.lineNumbers &&
                  (!p || !p["CodeMirror-linenumbers"]) &&
                  (i.lineNumber = y.appendChild(
                    k(
                      "div",
                      dt(n.options, a),
                      "CodeMirror-linenumber CodeMirror-gutter-elt",
                      "left: " +
                        l.gutterLeft["CodeMirror-linenumbers"] +
                        "px; width: " +
                        n.display.lineNumInnerWidth +
                        "px",
                    ),
                  )),
                p)
              )
                for (var x = 0; x < n.display.gutterSpecs.length; ++x) {
                  var _ = n.display.gutterSpecs[x].className,
                    N = p.hasOwnProperty(_) && p[_];
                  N &&
                    y.appendChild(
                      k(
                        "div",
                        [N],
                        "CodeMirror-gutter-elt",
                        "left: " + l.gutterLeft[_] + "px; width: " + l.gutterWidth[_] + "px",
                      ),
                    );
                }
            }
          }
          function kw(n, i, a) {
            i.alignable && (i.alignable = null);
            for (var l = pt("CodeMirror-linewidget"), c = i.node.firstChild, p = void 0; c; c = p)
              (p = c.nextSibling), l.test(c.className) && i.node.removeChild(c);
            $d(n, i, a);
          }
          function Cw(n, i, a, l) {
            var c = Od(n, i);
            return (
              (i.text = i.node = c.pre),
              c.bgClass && (i.bgClass = c.bgClass),
              c.textClass && (i.textClass = c.textClass),
              ou(n, i),
              Dd(n, i, a, l),
              $d(n, i, l),
              i.node
            );
          }
          function $d(n, i, a) {
            if ((Rd(n, i.line, i, a, !0), i.rest))
              for (var l = 0; l < i.rest.length; l++) Rd(n, i.rest[l], i, a, !1);
          }
          function Rd(n, i, a, l, c) {
            if (i.widgets)
              for (var p = Es(a), m = 0, y = i.widgets; m < y.length; ++m) {
                var x = y[m],
                  _ = k(
                    "div",
                    [x.node],
                    "CodeMirror-linewidget" + (x.className ? " " + x.className : ""),
                  );
                x.handleMouseEvents || _.setAttribute("cm-ignore-events", "true"),
                  Tw(x, _, a, l),
                  n.display.input.setUneditable(_),
                  c && x.above ? p.insertBefore(_, a.gutter || a.text) : p.appendChild(_),
                  qe(x, "redraw");
              }
          }
          function Tw(n, i, a, l) {
            if (n.noHScroll) {
              (a.alignable || (a.alignable = [])).push(i);
              var c = l.wrapperWidth;
              (i.style.left = l.fixedPos + "px"),
                n.coverGutter ||
                  ((c -= l.gutterTotalWidth), (i.style.paddingLeft = l.gutterTotalWidth + "px")),
                (i.style.width = c + "px");
            }
            n.coverGutter &&
              ((i.style.zIndex = 5),
              (i.style.position = "relative"),
              n.noHScroll || (i.style.marginLeft = -l.gutterTotalWidth + "px"));
          }
          function Ls(n) {
            if (n.height != null) return n.height;
            var i = n.doc.cm;
            if (!i) return 0;
            if (!J(document.body, n.node)) {
              var a = "position: relative;";
              n.coverGutter && (a += "margin-left: -" + i.display.gutters.offsetWidth + "px;"),
                n.noHScroll && (a += "width: " + i.display.wrapper.clientWidth + "px;"),
                z(i.display.measure, k("div", [n.node], null, a));
            }
            return (n.height = n.node.parentNode.offsetHeight);
          }
          function Ar(n, i) {
            for (var a = ws(i); a != n.wrapper; a = a.parentNode)
              if (
                !a ||
                (a.nodeType == 1 && a.getAttribute("cm-ignore-events") == "true") ||
                (a.parentNode == n.sizer && a != n.mover)
              )
                return !0;
          }
          function Zl(n) {
            return n.lineSpace.offsetTop;
          }
          function su(n) {
            return n.mover.offsetHeight - n.lineSpace.offsetHeight;
          }
          function zd(n) {
            if (n.cachedPaddingH) return n.cachedPaddingH;
            var i = z(n.measure, k("pre", "x", "CodeMirror-line-like")),
              a = window.getComputedStyle ? window.getComputedStyle(i) : i.currentStyle,
              l = { left: parseInt(a.paddingLeft), right: parseInt(a.paddingRight) };
            return !isNaN(l.left) && !isNaN(l.right) && (n.cachedPaddingH = l), l;
          }
          function hr(n) {
            return $ - n.display.nativeBarWidth;
          }
          function Oi(n) {
            return n.display.scroller.clientWidth - hr(n) - n.display.barWidth;
          }
          function lu(n) {
            return n.display.scroller.clientHeight - hr(n) - n.display.barHeight;
          }
          function Ew(n, i, a) {
            var l = n.options.lineWrapping,
              c = l && Oi(n);
            if (!i.measure.heights || (l && i.measure.width != c)) {
              var p = (i.measure.heights = []);
              if (l) {
                i.measure.width = c;
                for (var m = i.text.firstChild.getClientRects(), y = 0; y < m.length - 1; y++) {
                  var x = m[y],
                    _ = m[y + 1];
                  Math.abs(x.bottom - _.bottom) > 2 && p.push((x.bottom + _.top) / 2 - a.top);
                }
              }
              p.push(a.bottom - a.top);
            }
          }
          function Id(n, i, a) {
            if (n.line == i) return { map: n.measure.map, cache: n.measure.cache };
            if (n.rest) {
              for (var l = 0; l < n.rest.length; l++)
                if (n.rest[l] == i) return { map: n.measure.maps[l], cache: n.measure.caches[l] };
              for (var c = 0; c < n.rest.length; c++)
                if (C(n.rest[c]) > a)
                  return { map: n.measure.maps[c], cache: n.measure.caches[c], before: !0 };
            }
          }
          function Lw(n, i) {
            i = Zn(i);
            var a = C(i),
              l = (n.display.externalMeasured = new Nd(n.doc, i, a));
            l.lineN = a;
            var c = (l.built = Ad(n, l));
            return (l.text = c.pre), z(n.display.lineMeasure, c.pre), l;
          }
          function Fd(n, i, a, l) {
            return dr(n, So(n, i), a, l);
          }
          function au(n, i) {
            if (i >= n.display.viewFrom && i < n.display.viewTo) return n.display.view[Ri(n, i)];
            var a = n.display.externalMeasured;
            if (a && i >= a.lineN && i < a.lineN + a.size) return a;
          }
          function So(n, i) {
            var a = C(i),
              l = au(n, a);
            l && !l.text
              ? (l = null)
              : l && l.changes && (Pd(n, l, a, du(n)), (n.curOp.forceUpdate = !0)),
              l || (l = Lw(n, i));
            var c = Id(l, i, a);
            return {
              line: i,
              view: l,
              rect: null,
              map: c.map,
              cache: c.cache,
              before: c.before,
              hasHeights: !1,
            };
          }
          function dr(n, i, a, l, c) {
            i.before && (a = -1);
            var p = a + (l || ""),
              m;
            return (
              i.cache.hasOwnProperty(p)
                ? (m = i.cache[p])
                : (i.rect || (i.rect = i.view.text.getBoundingClientRect()),
                  i.hasHeights || (Ew(n, i.view, i.rect), (i.hasHeights = !0)),
                  (m = Mw(n, i, a, l)),
                  m.bogus || (i.cache[p] = m)),
              {
                left: m.left,
                right: m.right,
                top: c ? m.rtop : m.top,
                bottom: c ? m.rbottom : m.bottom,
              }
            );
          }
          var qd = { left: 0, right: 0, top: 0, bottom: 0 };
          function Hd(n, i, a) {
            for (var l, c, p, m, y, x, _ = 0; _ < n.length; _ += 3)
              if (
                ((y = n[_]),
                (x = n[_ + 1]),
                i < y
                  ? ((c = 0), (p = 1), (m = "left"))
                  : i < x
                  ? ((c = i - y), (p = c + 1))
                  : (_ == n.length - 3 || (i == x && n[_ + 3] > i)) &&
                    ((p = x - y), (c = p - 1), i >= x && (m = "right")),
                c != null)
              ) {
                if (
                  ((l = n[_ + 2]),
                  y == x && a == (l.insertLeft ? "left" : "right") && (m = a),
                  a == "left" && c == 0)
                )
                  for (; _ && n[_ - 2] == n[_ - 3] && n[_ - 1].insertLeft; )
                    (l = n[(_ -= 3) + 2]), (m = "left");
                if (a == "right" && c == x - y)
                  for (; _ < n.length - 3 && n[_ + 3] == n[_ + 4] && !n[_ + 5].insertLeft; )
                    (l = n[(_ += 3) + 2]), (m = "right");
                break;
              }
            return { node: l, start: c, end: p, collapse: m, coverStart: y, coverEnd: x };
          }
          function Aw(n, i) {
            var a = qd;
            if (i == "left") for (var l = 0; l < n.length && (a = n[l]).left == a.right; l++);
            else for (var c = n.length - 1; c >= 0 && (a = n[c]).left == a.right; c--);
            return a;
          }
          function Mw(n, i, a, l) {
            var c = Hd(i.map, a, l),
              p = c.node,
              m = c.start,
              y = c.end,
              x = c.collapse,
              _;
            if (p.nodeType == 3) {
              for (var N = 0; N < 4; N++) {
                for (; m && se(i.line.text.charAt(c.coverStart + m)); ) --m;
                for (; c.coverStart + y < c.coverEnd && se(i.line.text.charAt(c.coverStart + y)); )
                  ++y;
                if (
                  (d && g < 9 && m == 0 && y == c.coverEnd - c.coverStart
                    ? (_ = p.parentNode.getBoundingClientRect())
                    : (_ = Aw(H(p, m, y).getClientRects(), l)),
                  _.left || _.right || m == 0)
                )
                  break;
                (y = m), (m = m - 1), (x = "right");
              }
              d && g < 11 && (_ = Nw(n.display.measure, _));
            } else {
              m > 0 && (x = l = "right");
              var D;
              n.options.lineWrapping && (D = p.getClientRects()).length > 1
                ? (_ = D[l == "right" ? D.length - 1 : 0])
                : (_ = p.getBoundingClientRect());
            }
            if (d && g < 9 && !m && (!_ || (!_.left && !_.right))) {
              var W = p.parentNode.getClientRects()[0];
              W
                ? (_ = {
                    left: W.left,
                    right: W.left + Co(n.display),
                    top: W.top,
                    bottom: W.bottom,
                  })
                : (_ = qd);
            }
            for (
              var q = _.top - i.rect.top,
                Z = _.bottom - i.rect.top,
                it = (q + Z) / 2,
                vt = i.view.measure.heights,
                bt = 0;
              bt < vt.length - 1 && !(it < vt[bt]);
              bt++
            );
            var Ct = bt ? vt[bt - 1] : 0,
              wt = vt[bt],
              Lt = {
                left: (x == "right" ? _.right : _.left) - i.rect.left,
                right: (x == "left" ? _.left : _.right) - i.rect.left,
                top: Ct,
                bottom: wt,
              };
            return (
              !_.left && !_.right && (Lt.bogus = !0),
              n.options.singleCursorHeightPerLine || ((Lt.rtop = q), (Lt.rbottom = Z)),
              Lt
            );
          }
          function Nw(n, i) {
            if (
              !window.screen ||
              screen.logicalXDPI == null ||
              screen.logicalXDPI == screen.deviceXDPI ||
              !Zc(n)
            )
              return i;
            var a = screen.logicalXDPI / screen.deviceXDPI,
              l = screen.logicalYDPI / screen.deviceYDPI;
            return { left: i.left * a, right: i.right * a, top: i.top * l, bottom: i.bottom * l };
          }
          function Bd(n) {
            if (n.measure && ((n.measure.cache = {}), (n.measure.heights = null), n.rest))
              for (var i = 0; i < n.rest.length; i++) n.measure.caches[i] = {};
          }
          function Wd(n) {
            (n.display.externalMeasure = null), G(n.display.lineMeasure);
            for (var i = 0; i < n.display.view.length; i++) Bd(n.display.view[i]);
          }
          function As(n) {
            Wd(n),
              (n.display.cachedCharWidth =
                n.display.cachedTextHeight =
                n.display.cachedPaddingH =
                  null),
              n.options.lineWrapping || (n.display.maxLineChanged = !0),
              (n.display.lineNumChars = null);
          }
          function Ud(n) {
            return w && R
              ? -(
                  n.body.getBoundingClientRect().left -
                  parseInt(getComputedStyle(n.body).marginLeft)
                )
              : n.defaultView.pageXOffset || (n.documentElement || n.body).scrollLeft;
          }
          function jd(n) {
            return w && R
              ? -(n.body.getBoundingClientRect().top - parseInt(getComputedStyle(n.body).marginTop))
              : n.defaultView.pageYOffset || (n.documentElement || n.body).scrollTop;
          }
          function cu(n) {
            var i = Zn(n),
              a = i.widgets,
              l = 0;
            if (a) for (var c = 0; c < a.length; ++c) a[c].above && (l += Ls(a[c]));
            return l;
          }
          function Jl(n, i, a, l, c) {
            if (!c) {
              var p = cu(i);
              (a.top += p), (a.bottom += p);
            }
            if (l == "line") return a;
            l || (l = "local");
            var m = Lr(i);
            if (
              (l == "local" ? (m += Zl(n.display)) : (m -= n.display.viewOffset),
              l == "page" || l == "window")
            ) {
              var y = n.display.lineSpace.getBoundingClientRect();
              m += y.top + (l == "window" ? 0 : jd(Qt(n)));
              var x = y.left + (l == "window" ? 0 : Ud(Qt(n)));
              (a.left += x), (a.right += x);
            }
            return (a.top += m), (a.bottom += m), a;
          }
          function Vd(n, i, a) {
            if (a == "div") return i;
            var l = i.left,
              c = i.top;
            if (a == "page") (l -= Ud(Qt(n))), (c -= jd(Qt(n)));
            else if (a == "local" || !a) {
              var p = n.display.sizer.getBoundingClientRect();
              (l += p.left), (c += p.top);
            }
            var m = n.display.lineSpace.getBoundingClientRect();
            return { left: l - m.left, top: c - m.top };
          }
          function Ql(n, i, a, l, c) {
            return l || (l = Pt(n.doc, i.line)), Jl(n, l, Fd(n, l, i.ch, c), a);
          }
          function Jn(n, i, a, l, c, p) {
            (l = l || Pt(n.doc, i.line)), c || (c = So(n, l));
            function m(Z, it) {
              var vt = dr(n, c, Z, it ? "right" : "left", p);
              return it ? (vt.left = vt.right) : (vt.right = vt.left), Jl(n, l, vt, a);
            }
            var y = Yt(l, n.doc.direction),
              x = i.ch,
              _ = i.sticky;
            if (
              (x >= l.text.length
                ? ((x = l.text.length), (_ = "before"))
                : x <= 0 && ((x = 0), (_ = "after")),
              !y)
            )
              return m(_ == "before" ? x - 1 : x, _ == "before");
            function N(Z, it, vt) {
              var bt = y[it],
                Ct = bt.level == 1;
              return m(vt ? Z - 1 : Z, Ct != vt);
            }
            var D = Ae(y, x, _),
              W = cr,
              q = N(x, D, _ == "before");
            return W != null && (q.other = N(x, W, _ != "before")), q;
          }
          function Gd(n, i) {
            var a = 0;
            (i = Wt(n.doc, i)), n.options.lineWrapping || (a = Co(n.display) * i.ch);
            var l = Pt(n.doc, i.line),
              c = Lr(l) + Zl(n.display);
            return { left: a, right: a, top: c, bottom: c + l.height };
          }
          function uu(n, i, a, l, c) {
            var p = X(n, i, a);
            return (p.xRel = c), l && (p.outside = l), p;
          }
          function fu(n, i, a) {
            var l = n.doc;
            if (((a += n.display.viewOffset), a < 0)) return uu(l.first, 0, null, -1, -1);
            var c = O(l, a),
              p = l.first + l.size - 1;
            if (c > p) return uu(l.first + l.size - 1, Pt(l, p).text.length, null, 1, 1);
            i < 0 && (i = 0);
            for (var m = Pt(l, c); ; ) {
              var y = Pw(n, m, c, i, a),
                x = sw(m, y.ch + (y.xRel > 0 || y.outside > 0 ? 1 : 0));
              if (!x) return y;
              var _ = x.find(1);
              if (_.line == c) return _;
              m = Pt(l, (c = _.line));
            }
          }
          function Kd(n, i, a, l) {
            l -= cu(i);
            var c = i.text.length,
              p = Pn(
                function (m) {
                  return dr(n, a, m - 1).bottom <= l;
                },
                c,
                0,
              );
            return (
              (c = Pn(
                function (m) {
                  return dr(n, a, m).top > l;
                },
                p,
                c,
              )),
              { begin: p, end: c }
            );
          }
          function Xd(n, i, a, l) {
            a || (a = So(n, i));
            var c = Jl(n, i, dr(n, a, l), "line").top;
            return Kd(n, i, a, c);
          }
          function hu(n, i, a, l) {
            return n.bottom <= a ? !1 : n.top > a ? !0 : (l ? n.left : n.right) > i;
          }
          function Pw(n, i, a, l, c) {
            c -= Lr(i);
            var p = So(n, i),
              m = cu(i),
              y = 0,
              x = i.text.length,
              _ = !0,
              N = Yt(i, n.doc.direction);
            if (N) {
              var D = (n.options.lineWrapping ? Dw : Ow)(n, i, a, p, N, l, c);
              (_ = D.level != 1), (y = _ ? D.from : D.to - 1), (x = _ ? D.to : D.from - 1);
            }
            var W = null,
              q = null,
              Z = Pn(
                function (zt) {
                  var Ot = dr(n, p, zt);
                  return (
                    (Ot.top += m),
                    (Ot.bottom += m),
                    hu(Ot, l, c, !1)
                      ? (Ot.top <= c && Ot.left <= l && ((W = zt), (q = Ot)), !0)
                      : !1
                  );
                },
                y,
                x,
              ),
              it,
              vt,
              bt = !1;
            if (q) {
              var Ct = l - q.left < q.right - l,
                wt = Ct == _;
              (Z = W + (wt ? 0 : 1)), (vt = wt ? "after" : "before"), (it = Ct ? q.left : q.right);
            } else {
              !_ && (Z == x || Z == y) && Z++,
                (vt =
                  Z == 0
                    ? "after"
                    : Z == i.text.length
                    ? "before"
                    : dr(n, p, Z - (_ ? 1 : 0)).bottom + m <= c == _
                    ? "after"
                    : "before");
              var Lt = Jn(n, X(a, Z, vt), "line", i, p);
              (it = Lt.left), (bt = c < Lt.top ? -1 : c >= Lt.bottom ? 1 : 0);
            }
            return (Z = rn(i.text, Z, 1)), uu(a, Z, vt, bt, l - it);
          }
          function Ow(n, i, a, l, c, p, m) {
            var y = Pn(
                function (D) {
                  var W = c[D],
                    q = W.level != 1;
                  return hu(
                    Jn(n, X(a, q ? W.to : W.from, q ? "before" : "after"), "line", i, l),
                    p,
                    m,
                    !0,
                  );
                },
                0,
                c.length - 1,
              ),
              x = c[y];
            if (y > 0) {
              var _ = x.level != 1,
                N = Jn(n, X(a, _ ? x.from : x.to, _ ? "after" : "before"), "line", i, l);
              hu(N, p, m, !0) && N.top > m && (x = c[y - 1]);
            }
            return x;
          }
          function Dw(n, i, a, l, c, p, m) {
            var y = Kd(n, i, l, m),
              x = y.begin,
              _ = y.end;
            /\s/.test(i.text.charAt(_ - 1)) && _--;
            for (var N = null, D = null, W = 0; W < c.length; W++) {
              var q = c[W];
              if (!(q.from >= _ || q.to <= x)) {
                var Z = q.level != 1,
                  it = dr(n, l, Z ? Math.min(_, q.to) - 1 : Math.max(x, q.from)).right,
                  vt = it < p ? p - it + 1e9 : it - p;
                (!N || D > vt) && ((N = q), (D = vt));
              }
            }
            return (
              N || (N = c[c.length - 1]),
              N.from < x && (N = { from: x, to: N.to, level: N.level }),
              N.to > _ && (N = { from: N.from, to: _, level: N.level }),
              N
            );
          }
          var Di;
          function ko(n) {
            if (n.cachedTextHeight != null) return n.cachedTextHeight;
            if (Di == null) {
              Di = k("pre", null, "CodeMirror-line-like");
              for (var i = 0; i < 49; ++i)
                Di.appendChild(document.createTextNode("x")), Di.appendChild(k("br"));
              Di.appendChild(document.createTextNode("x"));
            }
            z(n.measure, Di);
            var a = Di.offsetHeight / 50;
            return a > 3 && (n.cachedTextHeight = a), G(n.measure), a || 1;
          }
          function Co(n) {
            if (n.cachedCharWidth != null) return n.cachedCharWidth;
            var i = k("span", "xxxxxxxxxx"),
              a = k("pre", [i], "CodeMirror-line-like");
            z(n.measure, a);
            var l = i.getBoundingClientRect(),
              c = (l.right - l.left) / 10;
            return c > 2 && (n.cachedCharWidth = c), c || 10;
          }
          function du(n) {
            for (
              var i = n.display,
                a = {},
                l = {},
                c = i.gutters.clientLeft,
                p = i.gutters.firstChild,
                m = 0;
              p;
              p = p.nextSibling, ++m
            ) {
              var y = n.display.gutterSpecs[m].className;
              (a[y] = p.offsetLeft + p.clientLeft + c), (l[y] = p.clientWidth);
            }
            return {
              fixedPos: pu(i),
              gutterTotalWidth: i.gutters.offsetWidth,
              gutterLeft: a,
              gutterWidth: l,
              wrapperWidth: i.wrapper.clientWidth,
            };
          }
          function pu(n) {
            return n.scroller.getBoundingClientRect().left - n.sizer.getBoundingClientRect().left;
          }
          function Yd(n) {
            var i = ko(n.display),
              a = n.options.lineWrapping,
              l = a && Math.max(5, n.display.scroller.clientWidth / Co(n.display) - 3);
            return function (c) {
              if (ti(n.doc, c)) return 0;
              var p = 0;
              if (c.widgets)
                for (var m = 0; m < c.widgets.length; m++)
                  c.widgets[m].height && (p += c.widgets[m].height);
              return a ? p + (Math.ceil(c.text.length / l) || 1) * i : p + i;
            };
          }
          function gu(n) {
            var i = n.doc,
              a = Yd(n);
            i.iter(function (l) {
              var c = a(l);
              c != l.height && On(l, c);
            });
          }
          function $i(n, i, a, l) {
            var c = n.display;
            if (!a && ws(i).getAttribute("cm-not-content") == "true") return null;
            var p,
              m,
              y = c.lineSpace.getBoundingClientRect();
            try {
              (p = i.clientX - y.left), (m = i.clientY - y.top);
            } catch {
              return null;
            }
            var x = fu(n, p, m),
              _;
            if (l && x.xRel > 0 && (_ = Pt(n.doc, x.line).text).length == x.ch) {
              var N = lt(_, _.length, n.options.tabSize) - _.length;
              x = X(x.line, Math.max(0, Math.round((p - zd(n.display).left) / Co(n.display)) - N));
            }
            return x;
          }
          function Ri(n, i) {
            if (i >= n.display.viewTo || ((i -= n.display.viewFrom), i < 0)) return null;
            for (var a = n.display.view, l = 0; l < a.length; l++)
              if (((i -= a[l].size), i < 0)) return l;
          }
          function ln(n, i, a, l) {
            i == null && (i = n.doc.first),
              a == null && (a = n.doc.first + n.doc.size),
              l || (l = 0);
            var c = n.display;
            if (
              (l &&
                a < c.viewTo &&
                (c.updateLineNumbers == null || c.updateLineNumbers > i) &&
                (c.updateLineNumbers = i),
              (n.curOp.viewChanged = !0),
              i >= c.viewTo)
            )
              Er && nu(n.doc, i) < c.viewTo && ni(n);
            else if (a <= c.viewFrom)
              Er && Ed(n.doc, a + l) > c.viewFrom ? ni(n) : ((c.viewFrom += l), (c.viewTo += l));
            else if (i <= c.viewFrom && a >= c.viewTo) ni(n);
            else if (i <= c.viewFrom) {
              var p = ta(n, a, a + l, 1);
              p
                ? ((c.view = c.view.slice(p.index)), (c.viewFrom = p.lineN), (c.viewTo += l))
                : ni(n);
            } else if (a >= c.viewTo) {
              var m = ta(n, i, i, -1);
              m ? ((c.view = c.view.slice(0, m.index)), (c.viewTo = m.lineN)) : ni(n);
            } else {
              var y = ta(n, i, i, -1),
                x = ta(n, a, a + l, 1);
              y && x
                ? ((c.view = c.view
                    .slice(0, y.index)
                    .concat(Yl(n, y.lineN, x.lineN))
                    .concat(c.view.slice(x.index))),
                  (c.viewTo += l))
                : ni(n);
            }
            var _ = c.externalMeasured;
            _ &&
              (a < _.lineN ? (_.lineN += l) : i < _.lineN + _.size && (c.externalMeasured = null));
          }
          function ei(n, i, a) {
            n.curOp.viewChanged = !0;
            var l = n.display,
              c = n.display.externalMeasured;
            if (
              (c && i >= c.lineN && i < c.lineN + c.size && (l.externalMeasured = null),
              !(i < l.viewFrom || i >= l.viewTo))
            ) {
              var p = l.view[Ri(n, i)];
              if (p.node != null) {
                var m = p.changes || (p.changes = []);
                Et(m, a) == -1 && m.push(a);
              }
            }
          }
          function ni(n) {
            (n.display.viewFrom = n.display.viewTo = n.doc.first),
              (n.display.view = []),
              (n.display.viewOffset = 0);
          }
          function ta(n, i, a, l) {
            var c = Ri(n, i),
              p,
              m = n.display.view;
            if (!Er || a == n.doc.first + n.doc.size) return { index: c, lineN: a };
            for (var y = n.display.viewFrom, x = 0; x < c; x++) y += m[x].size;
            if (y != i) {
              if (l > 0) {
                if (c == m.length - 1) return null;
                (p = y + m[c].size - i), c++;
              } else p = y - i;
              (i += p), (a += p);
            }
            for (; nu(n.doc, a) != a; ) {
              if (c == (l < 0 ? 0 : m.length - 1)) return null;
              (a += l * m[c - (l < 0 ? 1 : 0)].size), (c += l);
            }
            return { index: c, lineN: a };
          }
          function $w(n, i, a) {
            var l = n.display,
              c = l.view;
            c.length == 0 || i >= l.viewTo || a <= l.viewFrom
              ? ((l.view = Yl(n, i, a)), (l.viewFrom = i))
              : (l.viewFrom > i
                  ? (l.view = Yl(n, i, l.viewFrom).concat(l.view))
                  : l.viewFrom < i && (l.view = l.view.slice(Ri(n, i))),
                (l.viewFrom = i),
                l.viewTo < a
                  ? (l.view = l.view.concat(Yl(n, l.viewTo, a)))
                  : l.viewTo > a && (l.view = l.view.slice(0, Ri(n, a)))),
              (l.viewTo = a);
          }
          function Zd(n) {
            for (var i = n.display.view, a = 0, l = 0; l < i.length; l++) {
              var c = i[l];
              !c.hidden && (!c.node || c.changes) && ++a;
            }
            return a;
          }
          function Ms(n) {
            n.display.input.showSelection(n.display.input.prepareSelection());
          }
          function Jd(n, i) {
            i === void 0 && (i = !0);
            var a = n.doc,
              l = {},
              c = (l.cursors = document.createDocumentFragment()),
              p = (l.selection = document.createDocumentFragment()),
              m = n.options.$customCursor;
            m && (i = !0);
            for (var y = 0; y < a.sel.ranges.length; y++)
              if (!(!i && y == a.sel.primIndex)) {
                var x = a.sel.ranges[y];
                if (!(x.from().line >= n.display.viewTo || x.to().line < n.display.viewFrom)) {
                  var _ = x.empty();
                  if (m) {
                    var N = m(n, x);
                    N && vu(n, N, c);
                  } else (_ || n.options.showCursorWhenSelecting) && vu(n, x.head, c);
                  _ || Rw(n, x, p);
                }
              }
            return l;
          }
          function vu(n, i, a) {
            var l = Jn(n, i, "div", null, null, !n.options.singleCursorHeightPerLine),
              c = a.appendChild(k("div", " ", "CodeMirror-cursor"));
            if (
              ((c.style.left = l.left + "px"),
              (c.style.top = l.top + "px"),
              (c.style.height = Math.max(0, l.bottom - l.top) * n.options.cursorHeight + "px"),
              /\bcm-fat-cursor\b/.test(n.getWrapperElement().className))
            ) {
              var p = Ql(n, i, "div", null, null),
                m = p.right - p.left;
              c.style.width = (m > 0 ? m : n.defaultCharWidth()) + "px";
            }
            if (l.other) {
              var y = a.appendChild(k("div", " ", "CodeMirror-cursor CodeMirror-secondarycursor"));
              (y.style.display = ""),
                (y.style.left = l.other.left + "px"),
                (y.style.top = l.other.top + "px"),
                (y.style.height = (l.other.bottom - l.other.top) * 0.85 + "px");
            }
          }
          function ea(n, i) {
            return n.top - i.top || n.left - i.left;
          }
          function Rw(n, i, a) {
            var l = n.display,
              c = n.doc,
              p = document.createDocumentFragment(),
              m = zd(n.display),
              y = m.left,
              x = Math.max(l.sizerWidth, Oi(n) - l.sizer.offsetLeft) - m.right,
              _ = c.direction == "ltr";
            function N(wt, Lt, zt, Ot) {
              Lt < 0 && (Lt = 0),
                (Lt = Math.round(Lt)),
                (Ot = Math.round(Ot)),
                p.appendChild(
                  k(
                    "div",
                    null,
                    "CodeMirror-selected",
                    "position: absolute; left: " +
                      wt +
                      `px;
                             top: ` +
                      Lt +
                      "px; width: " +
                      (zt ?? x - wt) +
                      `px;
                             height: ` +
                      (Ot - Lt) +
                      "px",
                  ),
                );
            }
            function D(wt, Lt, zt) {
              var Ot = Pt(c, wt),
                Xt = Ot.text.length,
                ge,
                We;
              function _e(Ne, un) {
                return Ql(n, X(wt, Ne), "div", Ot, un);
              }
              function kn(Ne, un, Ve) {
                var $e = Xd(n, Ot, null, Ne),
                  Pe = (un == "ltr") == (Ve == "after") ? "left" : "right",
                  Ee =
                    Ve == "after"
                      ? $e.begin
                      : $e.end - (/\s/.test(Ot.text.charAt($e.end - 1)) ? 2 : 1);
                return _e(Ee, Pe)[Pe];
              }
              var cn = Yt(Ot, c.direction);
              return (
                wn(cn, Lt || 0, zt ?? Xt, function (Ne, un, Ve, $e) {
                  var Pe = Ve == "ltr",
                    Ee = _e(Ne, Pe ? "left" : "right"),
                    fn = _e(un - 1, Pe ? "right" : "left"),
                    Io = Lt == null && Ne == 0,
                    ai = zt == null && un == Xt,
                    Ze = $e == 0,
                    pr = !cn || $e == cn.length - 1;
                  if (fn.top - Ee.top <= 3) {
                    var Ue = (_ ? Io : ai) && Ze,
                      Wu = (_ ? ai : Io) && pr,
                      Pr = Ue ? y : (Pe ? Ee : fn).left,
                      Hi = Wu ? x : (Pe ? fn : Ee).right;
                    N(Pr, Ee.top, Hi - Pr, Ee.bottom);
                  } else {
                    var Bi, tn, Fo, Uu;
                    Pe
                      ? ((Bi = _ && Io && Ze ? y : Ee.left),
                        (tn = _ ? x : kn(Ne, Ve, "before")),
                        (Fo = _ ? y : kn(un, Ve, "after")),
                        (Uu = _ && ai && pr ? x : fn.right))
                      : ((Bi = _ ? kn(Ne, Ve, "before") : y),
                        (tn = !_ && Io && Ze ? x : Ee.right),
                        (Fo = !_ && ai && pr ? y : fn.left),
                        (Uu = _ ? kn(un, Ve, "after") : x)),
                      N(Bi, Ee.top, tn - Bi, Ee.bottom),
                      Ee.bottom < fn.top && N(y, Ee.bottom, null, fn.top),
                      N(Fo, fn.top, Uu - Fo, fn.bottom);
                  }
                  (!ge || ea(Ee, ge) < 0) && (ge = Ee),
                    ea(fn, ge) < 0 && (ge = fn),
                    (!We || ea(Ee, We) < 0) && (We = Ee),
                    ea(fn, We) < 0 && (We = fn);
                }),
                { start: ge, end: We }
              );
            }
            var W = i.from(),
              q = i.to();
            if (W.line == q.line) D(W.line, W.ch, q.ch);
            else {
              var Z = Pt(c, W.line),
                it = Pt(c, q.line),
                vt = Zn(Z) == Zn(it),
                bt = D(W.line, W.ch, vt ? Z.text.length + 1 : null).end,
                Ct = D(q.line, vt ? 0 : null, q.ch).start;
              vt &&
                (bt.top < Ct.top - 2
                  ? (N(bt.right, bt.top, null, bt.bottom), N(y, Ct.top, Ct.left, Ct.bottom))
                  : N(bt.right, bt.top, Ct.left - bt.right, bt.bottom)),
                bt.bottom < Ct.top && N(y, bt.bottom, null, Ct.top);
            }
            a.appendChild(p);
          }
          function mu(n) {
            if (n.state.focused) {
              var i = n.display;
              clearInterval(i.blinker);
              var a = !0;
              (i.cursorDiv.style.visibility = ""),
                n.options.cursorBlinkRate > 0
                  ? (i.blinker = setInterval(function () {
                      n.hasFocus() || To(n),
                        (i.cursorDiv.style.visibility = (a = !a) ? "" : "hidden");
                    }, n.options.cursorBlinkRate))
                  : n.options.cursorBlinkRate < 0 && (i.cursorDiv.style.visibility = "hidden");
            }
          }
          function Qd(n) {
            n.hasFocus() || (n.display.input.focus(), n.state.focused || bu(n));
          }
          function yu(n) {
            (n.state.delayingBlurEvent = !0),
              setTimeout(function () {
                n.state.delayingBlurEvent &&
                  ((n.state.delayingBlurEvent = !1), n.state.focused && To(n));
              }, 100);
          }
          function bu(n, i) {
            n.state.delayingBlurEvent && !n.state.draggingText && (n.state.delayingBlurEvent = !1),
              n.options.readOnly != "nocursor" &&
                (n.state.focused ||
                  (ke(n, "focus", n, i),
                  (n.state.focused = !0),
                  At(n.display.wrapper, "CodeMirror-focused"),
                  !n.curOp &&
                    n.display.selForContextMenu != n.doc.sel &&
                    (n.display.input.reset(),
                    v &&
                      setTimeout(function () {
                        return n.display.input.reset(!0);
                      }, 20)),
                  n.display.input.receivedFocus()),
                mu(n));
          }
          function To(n, i) {
            n.state.delayingBlurEvent ||
              (n.state.focused &&
                (ke(n, "blur", n, i),
                (n.state.focused = !1),
                gt(n.display.wrapper, "CodeMirror-focused")),
              clearInterval(n.display.blinker),
              setTimeout(function () {
                n.state.focused || (n.display.shift = !1);
              }, 150));
          }
          function na(n) {
            for (
              var i = n.display,
                a = i.lineDiv.offsetTop,
                l = Math.max(0, i.scroller.getBoundingClientRect().top),
                c = i.lineDiv.getBoundingClientRect().top,
                p = 0,
                m = 0;
              m < i.view.length;
              m++
            ) {
              var y = i.view[m],
                x = n.options.lineWrapping,
                _ = void 0,
                N = 0;
              if (!y.hidden) {
                if (((c += y.line.height), d && g < 8)) {
                  var D = y.node.offsetTop + y.node.offsetHeight;
                  (_ = D - a), (a = D);
                } else {
                  var W = y.node.getBoundingClientRect();
                  (_ = W.bottom - W.top),
                    !x &&
                      y.text.firstChild &&
                      (N = y.text.firstChild.getBoundingClientRect().right - W.left - 1);
                }
                var q = y.line.height - _;
                if (
                  (q > 0.005 || q < -0.005) &&
                  (c < l && (p -= q), On(y.line, _), tp(y.line), y.rest)
                )
                  for (var Z = 0; Z < y.rest.length; Z++) tp(y.rest[Z]);
                if (N > n.display.sizerWidth) {
                  var it = Math.ceil(N / Co(n.display));
                  it > n.display.maxLineLength &&
                    ((n.display.maxLineLength = it),
                    (n.display.maxLine = y.line),
                    (n.display.maxLineChanged = !0));
                }
              }
            }
            Math.abs(p) > 2 && (i.scroller.scrollTop += p);
          }
          function tp(n) {
            if (n.widgets)
              for (var i = 0; i < n.widgets.length; ++i) {
                var a = n.widgets[i],
                  l = a.node.parentNode;
                l && (a.height = l.offsetHeight);
              }
          }
          function ra(n, i, a) {
            var l = a && a.top != null ? Math.max(0, a.top) : n.scroller.scrollTop;
            l = Math.floor(l - Zl(n));
            var c = a && a.bottom != null ? a.bottom : l + n.wrapper.clientHeight,
              p = O(i, l),
              m = O(i, c);
            if (a && a.ensure) {
              var y = a.ensure.from.line,
                x = a.ensure.to.line;
              y < p
                ? ((p = y), (m = O(i, Lr(Pt(i, y)) + n.wrapper.clientHeight)))
                : Math.min(x, i.lastLine()) >= m &&
                  ((p = O(i, Lr(Pt(i, x)) - n.wrapper.clientHeight)), (m = x));
            }
            return { from: p, to: Math.max(m, p + 1) };
          }
          function zw(n, i) {
            if (!Ce(n, "scrollCursorIntoView")) {
              var a = n.display,
                l = a.sizer.getBoundingClientRect(),
                c = null,
                p = a.wrapper.ownerDocument;
              if (
                (i.top + l.top < 0
                  ? (c = !0)
                  : i.bottom + l.top >
                      (p.defaultView.innerHeight || p.documentElement.clientHeight) && (c = !1),
                c != null && !T)
              ) {
                var m = k(
                  "div",
                  "​",
                  null,
                  `position: absolute;
                         top: ` +
                    (i.top - a.viewOffset - Zl(n.display)) +
                    `px;
                         height: ` +
                    (i.bottom - i.top + hr(n) + a.barHeight) +
                    `px;
                         left: ` +
                    i.left +
                    "px; width: " +
                    Math.max(2, i.right - i.left) +
                    "px;",
                );
                n.display.lineSpace.appendChild(m),
                  m.scrollIntoView(c),
                  n.display.lineSpace.removeChild(m);
              }
            }
          }
          function Iw(n, i, a, l) {
            l == null && (l = 0);
            var c;
            !n.options.lineWrapping &&
              i == a &&
              ((a = i.sticky == "before" ? X(i.line, i.ch + 1, "before") : i),
              (i = i.ch ? X(i.line, i.sticky == "before" ? i.ch - 1 : i.ch, "after") : i));
            for (var p = 0; p < 5; p++) {
              var m = !1,
                y = Jn(n, i),
                x = !a || a == i ? y : Jn(n, a);
              c = {
                left: Math.min(y.left, x.left),
                top: Math.min(y.top, x.top) - l,
                right: Math.max(y.left, x.left),
                bottom: Math.max(y.bottom, x.bottom) + l,
              };
              var _ = wu(n, c),
                N = n.doc.scrollTop,
                D = n.doc.scrollLeft;
              if (
                (_.scrollTop != null &&
                  (Ps(n, _.scrollTop), Math.abs(n.doc.scrollTop - N) > 1 && (m = !0)),
                _.scrollLeft != null &&
                  (zi(n, _.scrollLeft), Math.abs(n.doc.scrollLeft - D) > 1 && (m = !0)),
                !m)
              )
                break;
            }
            return c;
          }
          function Fw(n, i) {
            var a = wu(n, i);
            a.scrollTop != null && Ps(n, a.scrollTop), a.scrollLeft != null && zi(n, a.scrollLeft);
          }
          function wu(n, i) {
            var a = n.display,
              l = ko(n.display);
            i.top < 0 && (i.top = 0);
            var c = n.curOp && n.curOp.scrollTop != null ? n.curOp.scrollTop : a.scroller.scrollTop,
              p = lu(n),
              m = {};
            i.bottom - i.top > p && (i.bottom = i.top + p);
            var y = n.doc.height + su(a),
              x = i.top < l,
              _ = i.bottom > y - l;
            if (i.top < c) m.scrollTop = x ? 0 : i.top;
            else if (i.bottom > c + p) {
              var N = Math.min(i.top, (_ ? y : i.bottom) - p);
              N != c && (m.scrollTop = N);
            }
            var D = n.options.fixedGutter ? 0 : a.gutters.offsetWidth,
              W =
                n.curOp && n.curOp.scrollLeft != null
                  ? n.curOp.scrollLeft
                  : a.scroller.scrollLeft - D,
              q = Oi(n) - a.gutters.offsetWidth,
              Z = i.right - i.left > q;
            return (
              Z && (i.right = i.left + q),
              i.left < 10
                ? (m.scrollLeft = 0)
                : i.left < W
                ? (m.scrollLeft = Math.max(0, i.left + D - (Z ? 0 : 10)))
                : i.right > q + W - 3 && (m.scrollLeft = i.right + (Z ? 0 : 10) - q),
              m
            );
          }
          function xu(n, i) {
            i != null &&
              (ia(n),
              (n.curOp.scrollTop =
                (n.curOp.scrollTop == null ? n.doc.scrollTop : n.curOp.scrollTop) + i));
          }
          function Eo(n) {
            ia(n);
            var i = n.getCursor();
            n.curOp.scrollToPos = { from: i, to: i, margin: n.options.cursorScrollMargin };
          }
          function Ns(n, i, a) {
            (i != null || a != null) && ia(n),
              i != null && (n.curOp.scrollLeft = i),
              a != null && (n.curOp.scrollTop = a);
          }
          function qw(n, i) {
            ia(n), (n.curOp.scrollToPos = i);
          }
          function ia(n) {
            var i = n.curOp.scrollToPos;
            if (i) {
              n.curOp.scrollToPos = null;
              var a = Gd(n, i.from),
                l = Gd(n, i.to);
              ep(n, a, l, i.margin);
            }
          }
          function ep(n, i, a, l) {
            var c = wu(n, {
              left: Math.min(i.left, a.left),
              top: Math.min(i.top, a.top) - l,
              right: Math.max(i.right, a.right),
              bottom: Math.max(i.bottom, a.bottom) + l,
            });
            Ns(n, c.scrollLeft, c.scrollTop);
          }
          function Ps(n, i) {
            Math.abs(n.doc.scrollTop - i) < 2 ||
              (s || Su(n, { top: i }), np(n, i, !0), s && Su(n), $s(n, 100));
          }
          function np(n, i, a) {
            (i = Math.max(
              0,
              Math.min(n.display.scroller.scrollHeight - n.display.scroller.clientHeight, i),
            )),
              !(n.display.scroller.scrollTop == i && !a) &&
                ((n.doc.scrollTop = i),
                n.display.scrollbars.setScrollTop(i),
                n.display.scroller.scrollTop != i && (n.display.scroller.scrollTop = i));
          }
          function zi(n, i, a, l) {
            (i = Math.max(
              0,
              Math.min(i, n.display.scroller.scrollWidth - n.display.scroller.clientWidth),
            )),
              !((a ? i == n.doc.scrollLeft : Math.abs(n.doc.scrollLeft - i) < 2) && !l) &&
                ((n.doc.scrollLeft = i),
                lp(n),
                n.display.scroller.scrollLeft != i && (n.display.scroller.scrollLeft = i),
                n.display.scrollbars.setScrollLeft(i));
          }
          function Os(n) {
            var i = n.display,
              a = i.gutters.offsetWidth,
              l = Math.round(n.doc.height + su(n.display));
            return {
              clientHeight: i.scroller.clientHeight,
              viewHeight: i.wrapper.clientHeight,
              scrollWidth: i.scroller.scrollWidth,
              clientWidth: i.scroller.clientWidth,
              viewWidth: i.wrapper.clientWidth,
              barLeft: n.options.fixedGutter ? a : 0,
              docHeight: l,
              scrollHeight: l + hr(n) + i.barHeight,
              nativeBarWidth: i.nativeBarWidth,
              gutterWidth: a,
            };
          }
          var Ii = function (n, i, a) {
            this.cm = a;
            var l = (this.vert = k(
                "div",
                [k("div", null, null, "min-width: 1px")],
                "CodeMirror-vscrollbar",
              )),
              c = (this.horiz = k(
                "div",
                [k("div", null, null, "height: 100%; min-height: 1px")],
                "CodeMirror-hscrollbar",
              ));
            (l.tabIndex = c.tabIndex = -1),
              n(l),
              n(c),
              Rt(l, "scroll", function () {
                l.clientHeight && i(l.scrollTop, "vertical");
              }),
              Rt(c, "scroll", function () {
                c.clientWidth && i(c.scrollLeft, "horizontal");
              }),
              (this.checkedZeroWidth = !1),
              d && g < 8 && (this.horiz.style.minHeight = this.vert.style.minWidth = "18px");
          };
          (Ii.prototype.update = function (n) {
            var i = n.scrollWidth > n.clientWidth + 1,
              a = n.scrollHeight > n.clientHeight + 1,
              l = n.nativeBarWidth;
            if (a) {
              (this.vert.style.display = "block"), (this.vert.style.bottom = i ? l + "px" : "0");
              var c = n.viewHeight - (i ? l : 0);
              this.vert.firstChild.style.height =
                Math.max(0, n.scrollHeight - n.clientHeight + c) + "px";
            } else
              (this.vert.scrollTop = 0),
                (this.vert.style.display = ""),
                (this.vert.firstChild.style.height = "0");
            if (i) {
              (this.horiz.style.display = "block"),
                (this.horiz.style.right = a ? l + "px" : "0"),
                (this.horiz.style.left = n.barLeft + "px");
              var p = n.viewWidth - n.barLeft - (a ? l : 0);
              this.horiz.firstChild.style.width =
                Math.max(0, n.scrollWidth - n.clientWidth + p) + "px";
            } else (this.horiz.style.display = ""), (this.horiz.firstChild.style.width = "0");
            return (
              !this.checkedZeroWidth &&
                n.clientHeight > 0 &&
                (l == 0 && this.zeroWidthHack(), (this.checkedZeroWidth = !0)),
              { right: a ? l : 0, bottom: i ? l : 0 }
            );
          }),
            (Ii.prototype.setScrollLeft = function (n) {
              this.horiz.scrollLeft != n && (this.horiz.scrollLeft = n),
                this.disableHoriz &&
                  this.enableZeroWidthBar(this.horiz, this.disableHoriz, "horiz");
            }),
            (Ii.prototype.setScrollTop = function (n) {
              this.vert.scrollTop != n && (this.vert.scrollTop = n),
                this.disableVert && this.enableZeroWidthBar(this.vert, this.disableVert, "vert");
            }),
            (Ii.prototype.zeroWidthHack = function () {
              var n = B && !L ? "12px" : "18px";
              (this.horiz.style.height = this.vert.style.width = n),
                (this.horiz.style.visibility = this.vert.style.visibility = "hidden"),
                (this.disableHoriz = new Mt()),
                (this.disableVert = new Mt());
            }),
            (Ii.prototype.enableZeroWidthBar = function (n, i, a) {
              n.style.visibility = "";
              function l() {
                var c = n.getBoundingClientRect(),
                  p =
                    a == "vert"
                      ? document.elementFromPoint(c.right - 1, (c.top + c.bottom) / 2)
                      : document.elementFromPoint((c.right + c.left) / 2, c.bottom - 1);
                p != n ? (n.style.visibility = "hidden") : i.set(1e3, l);
              }
              i.set(1e3, l);
            }),
            (Ii.prototype.clear = function () {
              var n = this.horiz.parentNode;
              n.removeChild(this.horiz), n.removeChild(this.vert);
            });
          var Ds = function () {};
          (Ds.prototype.update = function () {
            return { bottom: 0, right: 0 };
          }),
            (Ds.prototype.setScrollLeft = function () {}),
            (Ds.prototype.setScrollTop = function () {}),
            (Ds.prototype.clear = function () {});
          function Lo(n, i) {
            i || (i = Os(n));
            var a = n.display.barWidth,
              l = n.display.barHeight;
            rp(n, i);
            for (var c = 0; (c < 4 && a != n.display.barWidth) || l != n.display.barHeight; c++)
              a != n.display.barWidth && n.options.lineWrapping && na(n),
                rp(n, Os(n)),
                (a = n.display.barWidth),
                (l = n.display.barHeight);
          }
          function rp(n, i) {
            var a = n.display,
              l = a.scrollbars.update(i);
            (a.sizer.style.paddingRight = (a.barWidth = l.right) + "px"),
              (a.sizer.style.paddingBottom = (a.barHeight = l.bottom) + "px"),
              (a.heightForcer.style.borderBottom = l.bottom + "px solid transparent"),
              l.right && l.bottom
                ? ((a.scrollbarFiller.style.display = "block"),
                  (a.scrollbarFiller.style.height = l.bottom + "px"),
                  (a.scrollbarFiller.style.width = l.right + "px"))
                : (a.scrollbarFiller.style.display = ""),
              l.bottom && n.options.coverGutterNextToScrollbar && n.options.fixedGutter
                ? ((a.gutterFiller.style.display = "block"),
                  (a.gutterFiller.style.height = l.bottom + "px"),
                  (a.gutterFiller.style.width = i.gutterWidth + "px"))
                : (a.gutterFiller.style.display = "");
          }
          var ip = { native: Ii, null: Ds };
          function op(n) {
            n.display.scrollbars &&
              (n.display.scrollbars.clear(),
              n.display.scrollbars.addClass &&
                gt(n.display.wrapper, n.display.scrollbars.addClass)),
              (n.display.scrollbars = new ip[n.options.scrollbarStyle](
                function (i) {
                  n.display.wrapper.insertBefore(i, n.display.scrollbarFiller),
                    Rt(i, "mousedown", function () {
                      n.state.focused &&
                        setTimeout(function () {
                          return n.display.input.focus();
                        }, 0);
                    }),
                    i.setAttribute("cm-not-content", "true");
                },
                function (i, a) {
                  a == "horizontal" ? zi(n, i) : Ps(n, i);
                },
                n,
              )),
              n.display.scrollbars.addClass && At(n.display.wrapper, n.display.scrollbars.addClass);
          }
          var Hw = 0;
          function Fi(n) {
            (n.curOp = {
              cm: n,
              viewChanged: !1,
              startHeight: n.doc.height,
              forceUpdate: !1,
              updateInput: 0,
              typing: !1,
              changeObjs: null,
              cursorActivityHandlers: null,
              cursorActivityCalled: 0,
              selectionChanged: !1,
              updateMaxLine: !1,
              scrollLeft: null,
              scrollTop: null,
              scrollToPos: null,
              focus: !1,
              id: ++Hw,
              markArrays: null,
            }),
              yw(n.curOp);
          }
          function qi(n) {
            var i = n.curOp;
            i &&
              ww(i, function (a) {
                for (var l = 0; l < a.ops.length; l++) a.ops[l].cm.curOp = null;
                Bw(a);
              });
          }
          function Bw(n) {
            for (var i = n.ops, a = 0; a < i.length; a++) Ww(i[a]);
            for (var l = 0; l < i.length; l++) Uw(i[l]);
            for (var c = 0; c < i.length; c++) jw(i[c]);
            for (var p = 0; p < i.length; p++) Vw(i[p]);
            for (var m = 0; m < i.length; m++) Gw(i[m]);
          }
          function Ww(n) {
            var i = n.cm,
              a = i.display;
            Xw(i),
              n.updateMaxLine && iu(i),
              (n.mustUpdate =
                n.viewChanged ||
                n.forceUpdate ||
                n.scrollTop != null ||
                (n.scrollToPos &&
                  (n.scrollToPos.from.line < a.viewFrom || n.scrollToPos.to.line >= a.viewTo)) ||
                (a.maxLineChanged && i.options.lineWrapping)),
              (n.update =
                n.mustUpdate &&
                new oa(
                  i,
                  n.mustUpdate && { top: n.scrollTop, ensure: n.scrollToPos },
                  n.forceUpdate,
                ));
          }
          function Uw(n) {
            n.updatedDisplay = n.mustUpdate && _u(n.cm, n.update);
          }
          function jw(n) {
            var i = n.cm,
              a = i.display;
            n.updatedDisplay && na(i),
              (n.barMeasure = Os(i)),
              a.maxLineChanged &&
                !i.options.lineWrapping &&
                ((n.adjustWidthTo = Fd(i, a.maxLine, a.maxLine.text.length).left + 3),
                (i.display.sizerWidth = n.adjustWidthTo),
                (n.barMeasure.scrollWidth = Math.max(
                  a.scroller.clientWidth,
                  a.sizer.offsetLeft + n.adjustWidthTo + hr(i) + i.display.barWidth,
                )),
                (n.maxScrollLeft = Math.max(0, a.sizer.offsetLeft + n.adjustWidthTo - Oi(i)))),
              (n.updatedDisplay || n.selectionChanged) &&
                (n.preparedSelection = a.input.prepareSelection());
          }
          function Vw(n) {
            var i = n.cm;
            n.adjustWidthTo != null &&
              ((i.display.sizer.style.minWidth = n.adjustWidthTo + "px"),
              n.maxScrollLeft < i.doc.scrollLeft &&
                zi(i, Math.min(i.display.scroller.scrollLeft, n.maxScrollLeft), !0),
              (i.display.maxLineChanged = !1));
            var a = n.focus && n.focus == yt(Jt(i));
            n.preparedSelection && i.display.input.showSelection(n.preparedSelection, a),
              (n.updatedDisplay || n.startHeight != i.doc.height) && Lo(i, n.barMeasure),
              n.updatedDisplay && Cu(i, n.barMeasure),
              n.selectionChanged && mu(i),
              i.state.focused && n.updateInput && i.display.input.reset(n.typing),
              a && Qd(n.cm);
          }
          function Gw(n) {
            var i = n.cm,
              a = i.display,
              l = i.doc;
            if (
              (n.updatedDisplay && sp(i, n.update),
              a.wheelStartX != null &&
                (n.scrollTop != null || n.scrollLeft != null || n.scrollToPos) &&
                (a.wheelStartX = a.wheelStartY = null),
              n.scrollTop != null && np(i, n.scrollTop, n.forceScroll),
              n.scrollLeft != null && zi(i, n.scrollLeft, !0, !0),
              n.scrollToPos)
            ) {
              var c = Iw(
                i,
                Wt(l, n.scrollToPos.from),
                Wt(l, n.scrollToPos.to),
                n.scrollToPos.margin,
              );
              zw(i, c);
            }
            var p = n.maybeHiddenMarkers,
              m = n.maybeUnhiddenMarkers;
            if (p) for (var y = 0; y < p.length; ++y) p[y].lines.length || ke(p[y], "hide");
            if (m) for (var x = 0; x < m.length; ++x) m[x].lines.length && ke(m[x], "unhide");
            a.wrapper.offsetHeight && (l.scrollTop = i.display.scroller.scrollTop),
              n.changeObjs && ke(i, "changes", i, n.changeObjs),
              n.update && n.update.finish();
          }
          function Sn(n, i) {
            if (n.curOp) return i();
            Fi(n);
            try {
              return i();
            } finally {
              qi(n);
            }
          }
          function He(n, i) {
            return function () {
              if (n.curOp) return i.apply(n, arguments);
              Fi(n);
              try {
                return i.apply(n, arguments);
              } finally {
                qi(n);
              }
            };
          }
          function Qe(n) {
            return function () {
              if (this.curOp) return n.apply(this, arguments);
              Fi(this);
              try {
                return n.apply(this, arguments);
              } finally {
                qi(this);
              }
            };
          }
          function Be(n) {
            return function () {
              var i = this.cm;
              if (!i || i.curOp) return n.apply(this, arguments);
              Fi(i);
              try {
                return n.apply(this, arguments);
              } finally {
                qi(i);
              }
            };
          }
          function $s(n, i) {
            n.doc.highlightFrontier < n.display.viewTo && n.state.highlight.set(i, j(Kw, n));
          }
          function Kw(n) {
            var i = n.doc;
            if (!(i.highlightFrontier >= n.display.viewTo)) {
              var a = +new Date() + n.options.workTime,
                l = ks(n, i.highlightFrontier),
                c = [];
              i.iter(l.line, Math.min(i.first + i.size, n.display.viewTo + 500), function (p) {
                if (l.line >= n.display.viewFrom) {
                  var m = p.styles,
                    y = p.text.length > n.options.maxHighlightLength ? ur(i.mode, l.state) : null,
                    x = dd(n, p, l, !0);
                  y && (l.state = y), (p.styles = x.styles);
                  var _ = p.styleClasses,
                    N = x.classes;
                  N ? (p.styleClasses = N) : _ && (p.styleClasses = null);
                  for (
                    var D =
                        !m ||
                        m.length != p.styles.length ||
                        (_ != N &&
                          (!_ || !N || _.bgClass != N.bgClass || _.textClass != N.textClass)),
                      W = 0;
                    !D && W < m.length;
                    ++W
                  )
                    D = m[W] != p.styles[W];
                  D && c.push(l.line), (p.stateAfter = l.save()), l.nextLine();
                } else
                  p.text.length <= n.options.maxHighlightLength && Jc(n, p.text, l),
                    (p.stateAfter = l.line % 5 == 0 ? l.save() : null),
                    l.nextLine();
                if (+new Date() > a) return $s(n, n.options.workDelay), !0;
              }),
                (i.highlightFrontier = l.line),
                (i.modeFrontier = Math.max(i.modeFrontier, l.line)),
                c.length &&
                  Sn(n, function () {
                    for (var p = 0; p < c.length; p++) ei(n, c[p], "text");
                  });
            }
          }
          var oa = function (n, i, a) {
            var l = n.display;
            (this.viewport = i),
              (this.visible = ra(l, n.doc, i)),
              (this.editorIsHidden = !l.wrapper.offsetWidth),
              (this.wrapperHeight = l.wrapper.clientHeight),
              (this.wrapperWidth = l.wrapper.clientWidth),
              (this.oldDisplayWidth = Oi(n)),
              (this.force = a),
              (this.dims = du(n)),
              (this.events = []);
          };
          (oa.prototype.signal = function (n, i) {
            _n(n, i) && this.events.push(arguments);
          }),
            (oa.prototype.finish = function () {
              for (var n = 0; n < this.events.length; n++) ke.apply(null, this.events[n]);
            });
          function Xw(n) {
            var i = n.display;
            !i.scrollbarsClipped &&
              i.scroller.offsetWidth &&
              ((i.nativeBarWidth = i.scroller.offsetWidth - i.scroller.clientWidth),
              (i.heightForcer.style.height = hr(n) + "px"),
              (i.sizer.style.marginBottom = -i.nativeBarWidth + "px"),
              (i.sizer.style.borderRightWidth = hr(n) + "px"),
              (i.scrollbarsClipped = !0));
          }
          function Yw(n) {
            if (n.hasFocus()) return null;
            var i = yt(Jt(n));
            if (!i || !J(n.display.lineDiv, i)) return null;
            var a = { activeElt: i };
            if (window.getSelection) {
              var l = Tt(n).getSelection();
              l.anchorNode &&
                l.extend &&
                J(n.display.lineDiv, l.anchorNode) &&
                ((a.anchorNode = l.anchorNode),
                (a.anchorOffset = l.anchorOffset),
                (a.focusNode = l.focusNode),
                (a.focusOffset = l.focusOffset));
            }
            return a;
          }
          function Zw(n) {
            if (
              !(!n || !n.activeElt || n.activeElt == yt(Gt(n.activeElt))) &&
              (n.activeElt.focus(),
              !/^(INPUT|TEXTAREA)$/.test(n.activeElt.nodeName) &&
                n.anchorNode &&
                J(document.body, n.anchorNode) &&
                J(document.body, n.focusNode))
            ) {
              var i = n.activeElt.ownerDocument,
                a = i.defaultView.getSelection(),
                l = i.createRange();
              l.setEnd(n.anchorNode, n.anchorOffset),
                l.collapse(!1),
                a.removeAllRanges(),
                a.addRange(l),
                a.extend(n.focusNode, n.focusOffset);
            }
          }
          function _u(n, i) {
            var a = n.display,
              l = n.doc;
            if (i.editorIsHidden) return ni(n), !1;
            if (
              !i.force &&
              i.visible.from >= a.viewFrom &&
              i.visible.to <= a.viewTo &&
              (a.updateLineNumbers == null || a.updateLineNumbers >= a.viewTo) &&
              a.renderedView == a.view &&
              Zd(n) == 0
            )
              return !1;
            ap(n) && (ni(n), (i.dims = du(n)));
            var c = l.first + l.size,
              p = Math.max(i.visible.from - n.options.viewportMargin, l.first),
              m = Math.min(c, i.visible.to + n.options.viewportMargin);
            a.viewFrom < p && p - a.viewFrom < 20 && (p = Math.max(l.first, a.viewFrom)),
              a.viewTo > m && a.viewTo - m < 20 && (m = Math.min(c, a.viewTo)),
              Er && ((p = nu(n.doc, p)), (m = Ed(n.doc, m)));
            var y =
              p != a.viewFrom ||
              m != a.viewTo ||
              a.lastWrapHeight != i.wrapperHeight ||
              a.lastWrapWidth != i.wrapperWidth;
            $w(n, p, m),
              (a.viewOffset = Lr(Pt(n.doc, a.viewFrom))),
              (n.display.mover.style.top = a.viewOffset + "px");
            var x = Zd(n);
            if (
              !y &&
              x == 0 &&
              !i.force &&
              a.renderedView == a.view &&
              (a.updateLineNumbers == null || a.updateLineNumbers >= a.viewTo)
            )
              return !1;
            var _ = Yw(n);
            return (
              x > 4 && (a.lineDiv.style.display = "none"),
              Jw(n, a.updateLineNumbers, i.dims),
              x > 4 && (a.lineDiv.style.display = ""),
              (a.renderedView = a.view),
              Zw(_),
              G(a.cursorDiv),
              G(a.selectionDiv),
              (a.gutters.style.height = a.sizer.style.minHeight = 0),
              y &&
                ((a.lastWrapHeight = i.wrapperHeight),
                (a.lastWrapWidth = i.wrapperWidth),
                $s(n, 400)),
              (a.updateLineNumbers = null),
              !0
            );
          }
          function sp(n, i) {
            for (var a = i.viewport, l = !0; ; l = !1) {
              if (!l || !n.options.lineWrapping || i.oldDisplayWidth == Oi(n)) {
                if (
                  (a &&
                    a.top != null &&
                    (a = { top: Math.min(n.doc.height + su(n.display) - lu(n), a.top) }),
                  (i.visible = ra(n.display, n.doc, a)),
                  i.visible.from >= n.display.viewFrom && i.visible.to <= n.display.viewTo)
                )
                  break;
              } else l && (i.visible = ra(n.display, n.doc, a));
              if (!_u(n, i)) break;
              na(n);
              var c = Os(n);
              Ms(n), Lo(n, c), Cu(n, c), (i.force = !1);
            }
            i.signal(n, "update", n),
              (n.display.viewFrom != n.display.reportedViewFrom ||
                n.display.viewTo != n.display.reportedViewTo) &&
                (i.signal(n, "viewportChange", n, n.display.viewFrom, n.display.viewTo),
                (n.display.reportedViewFrom = n.display.viewFrom),
                (n.display.reportedViewTo = n.display.viewTo));
          }
          function Su(n, i) {
            var a = new oa(n, i);
            if (_u(n, a)) {
              na(n), sp(n, a);
              var l = Os(n);
              Ms(n), Lo(n, l), Cu(n, l), a.finish();
            }
          }
          function Jw(n, i, a) {
            var l = n.display,
              c = n.options.lineNumbers,
              p = l.lineDiv,
              m = p.firstChild;
            function y(Z) {
              var it = Z.nextSibling;
              return (
                v && B && n.display.currentWheelTarget == Z
                  ? (Z.style.display = "none")
                  : Z.parentNode.removeChild(Z),
                it
              );
            }
            for (var x = l.view, _ = l.viewFrom, N = 0; N < x.length; N++) {
              var D = x[N];
              if (!D.hidden)
                if (!D.node || D.node.parentNode != p) {
                  var W = Cw(n, D, _, a);
                  p.insertBefore(W, m);
                } else {
                  for (; m != D.node; ) m = y(m);
                  var q = c && i != null && i <= _ && D.lineNumber;
                  D.changes && (Et(D.changes, "gutter") > -1 && (q = !1), Pd(n, D, _, a)),
                    q &&
                      (G(D.lineNumber),
                      D.lineNumber.appendChild(document.createTextNode(dt(n.options, _)))),
                    (m = D.node.nextSibling);
                }
              _ += D.size;
            }
            for (; m; ) m = y(m);
          }
          function ku(n) {
            var i = n.gutters.offsetWidth;
            (n.sizer.style.marginLeft = i + "px"), qe(n, "gutterChanged", n);
          }
          function Cu(n, i) {
            (n.display.sizer.style.minHeight = i.docHeight + "px"),
              (n.display.heightForcer.style.top = i.docHeight + "px"),
              (n.display.gutters.style.height = i.docHeight + n.display.barHeight + hr(n) + "px");
          }
          function lp(n) {
            var i = n.display,
              a = i.view;
            if (!(!i.alignWidgets && (!i.gutters.firstChild || !n.options.fixedGutter))) {
              for (
                var l = pu(i) - i.scroller.scrollLeft + n.doc.scrollLeft,
                  c = i.gutters.offsetWidth,
                  p = l + "px",
                  m = 0;
                m < a.length;
                m++
              )
                if (!a[m].hidden) {
                  n.options.fixedGutter &&
                    (a[m].gutter && (a[m].gutter.style.left = p),
                    a[m].gutterBackground && (a[m].gutterBackground.style.left = p));
                  var y = a[m].alignable;
                  if (y) for (var x = 0; x < y.length; x++) y[x].style.left = p;
                }
              n.options.fixedGutter && (i.gutters.style.left = l + c + "px");
            }
          }
          function ap(n) {
            if (!n.options.lineNumbers) return !1;
            var i = n.doc,
              a = dt(n.options, i.first + i.size - 1),
              l = n.display;
            if (a.length != l.lineNumChars) {
              var c = l.measure.appendChild(
                  k("div", [k("div", a)], "CodeMirror-linenumber CodeMirror-gutter-elt"),
                ),
                p = c.firstChild.offsetWidth,
                m = c.offsetWidth - p;
              return (
                (l.lineGutter.style.width = ""),
                (l.lineNumInnerWidth = Math.max(p, l.lineGutter.offsetWidth - m) + 1),
                (l.lineNumWidth = l.lineNumInnerWidth + m),
                (l.lineNumChars = l.lineNumInnerWidth ? a.length : -1),
                (l.lineGutter.style.width = l.lineNumWidth + "px"),
                ku(n.display),
                !0
              );
            }
            return !1;
          }
          function Tu(n, i) {
            for (var a = [], l = !1, c = 0; c < n.length; c++) {
              var p = n[c],
                m = null;
              if (
                (typeof p != "string" && ((m = p.style), (p = p.className)),
                p == "CodeMirror-linenumbers")
              )
                if (i) l = !0;
                else continue;
              a.push({ className: p, style: m });
            }
            return i && !l && a.push({ className: "CodeMirror-linenumbers", style: null }), a;
          }
          function cp(n) {
            var i = n.gutters,
              a = n.gutterSpecs;
            G(i), (n.lineGutter = null);
            for (var l = 0; l < a.length; ++l) {
              var c = a[l],
                p = c.className,
                m = c.style,
                y = i.appendChild(k("div", null, "CodeMirror-gutter " + p));
              m && (y.style.cssText = m),
                p == "CodeMirror-linenumbers" &&
                  ((n.lineGutter = y), (y.style.width = (n.lineNumWidth || 1) + "px"));
            }
            (i.style.display = a.length ? "" : "none"), ku(n);
          }
          function Rs(n) {
            cp(n.display), ln(n), lp(n);
          }
          function Qw(n, i, a, l) {
            var c = this;
            (this.input = a),
              (c.scrollbarFiller = k("div", null, "CodeMirror-scrollbar-filler")),
              c.scrollbarFiller.setAttribute("cm-not-content", "true"),
              (c.gutterFiller = k("div", null, "CodeMirror-gutter-filler")),
              c.gutterFiller.setAttribute("cm-not-content", "true"),
              (c.lineDiv = F("div", null, "CodeMirror-code")),
              (c.selectionDiv = k("div", null, null, "position: relative; z-index: 1")),
              (c.cursorDiv = k("div", null, "CodeMirror-cursors")),
              (c.measure = k("div", null, "CodeMirror-measure")),
              (c.lineMeasure = k("div", null, "CodeMirror-measure")),
              (c.lineSpace = F(
                "div",
                [c.measure, c.lineMeasure, c.selectionDiv, c.cursorDiv, c.lineDiv],
                null,
                "position: relative; outline: none",
              ));
            var p = F("div", [c.lineSpace], "CodeMirror-lines");
            (c.mover = k("div", [p], null, "position: relative")),
              (c.sizer = k("div", [c.mover], "CodeMirror-sizer")),
              (c.sizerWidth = null),
              (c.heightForcer = k(
                "div",
                null,
                null,
                "position: absolute; height: " + $ + "px; width: 1px;",
              )),
              (c.gutters = k("div", null, "CodeMirror-gutters")),
              (c.lineGutter = null),
              (c.scroller = k("div", [c.sizer, c.heightForcer, c.gutters], "CodeMirror-scroll")),
              c.scroller.setAttribute("tabIndex", "-1"),
              (c.wrapper = k("div", [c.scrollbarFiller, c.gutterFiller, c.scroller], "CodeMirror")),
              w && S >= 105 && (c.wrapper.style.clipPath = "inset(0px)"),
              c.wrapper.setAttribute("translate", "no"),
              d && g < 8 && ((c.gutters.style.zIndex = -1), (c.scroller.style.paddingRight = 0)),
              !v && !(s && E) && (c.scroller.draggable = !0),
              n && (n.appendChild ? n.appendChild(c.wrapper) : n(c.wrapper)),
              (c.viewFrom = c.viewTo = i.first),
              (c.reportedViewFrom = c.reportedViewTo = i.first),
              (c.view = []),
              (c.renderedView = null),
              (c.externalMeasured = null),
              (c.viewOffset = 0),
              (c.lastWrapHeight = c.lastWrapWidth = 0),
              (c.updateLineNumbers = null),
              (c.nativeBarWidth = c.barHeight = c.barWidth = 0),
              (c.scrollbarsClipped = !1),
              (c.lineNumWidth = c.lineNumInnerWidth = c.lineNumChars = null),
              (c.alignWidgets = !1),
              (c.cachedCharWidth = c.cachedTextHeight = c.cachedPaddingH = null),
              (c.maxLine = null),
              (c.maxLineLength = 0),
              (c.maxLineChanged = !1),
              (c.wheelDX = c.wheelDY = c.wheelStartX = c.wheelStartY = null),
              (c.shift = !1),
              (c.selForContextMenu = null),
              (c.activeTouch = null),
              (c.gutterSpecs = Tu(l.gutters, l.lineNumbers)),
              cp(c),
              a.init(c);
          }
          var sa = 0,
            Mr = null;
          d ? (Mr = -0.53) : s ? (Mr = 15) : w ? (Mr = -0.7) : A && (Mr = -1 / 3);
          function up(n) {
            var i = n.wheelDeltaX,
              a = n.wheelDeltaY;
            return (
              i == null && n.detail && n.axis == n.HORIZONTAL_AXIS && (i = n.detail),
              a == null && n.detail && n.axis == n.VERTICAL_AXIS
                ? (a = n.detail)
                : a == null && (a = n.wheelDelta),
              { x: i, y: a }
            );
          }
          function tx(n) {
            var i = up(n);
            return (i.x *= Mr), (i.y *= Mr), i;
          }
          function fp(n, i) {
            w &&
              S == 102 &&
              (n.display.chromeScrollHack == null
                ? (n.display.sizer.style.pointerEvents = "none")
                : clearTimeout(n.display.chromeScrollHack),
              (n.display.chromeScrollHack = setTimeout(function () {
                (n.display.chromeScrollHack = null), (n.display.sizer.style.pointerEvents = "");
              }, 100)));
            var a = up(i),
              l = a.x,
              c = a.y,
              p = Mr;
            i.deltaMode === 0 && ((l = i.deltaX), (c = i.deltaY), (p = 1));
            var m = n.display,
              y = m.scroller,
              x = y.scrollWidth > y.clientWidth,
              _ = y.scrollHeight > y.clientHeight;
            if ((l && x) || (c && _)) {
              if (c && B && v) {
                t: for (var N = i.target, D = m.view; N != y; N = N.parentNode)
                  for (var W = 0; W < D.length; W++)
                    if (D[W].node == N) {
                      n.display.currentWheelTarget = N;
                      break t;
                    }
              }
              if (l && !s && !P && p != null) {
                c && _ && Ps(n, Math.max(0, y.scrollTop + c * p)),
                  zi(n, Math.max(0, y.scrollLeft + l * p)),
                  (!c || (c && _)) && Xe(i),
                  (m.wheelStartX = null);
                return;
              }
              if (c && p != null) {
                var q = c * p,
                  Z = n.doc.scrollTop,
                  it = Z + m.wrapper.clientHeight;
                q < 0 ? (Z = Math.max(0, Z + q - 50)) : (it = Math.min(n.doc.height, it + q + 50)),
                  Su(n, { top: Z, bottom: it });
              }
              sa < 20 &&
                i.deltaMode !== 0 &&
                (m.wheelStartX == null
                  ? ((m.wheelStartX = y.scrollLeft),
                    (m.wheelStartY = y.scrollTop),
                    (m.wheelDX = l),
                    (m.wheelDY = c),
                    setTimeout(function () {
                      if (m.wheelStartX != null) {
                        var vt = y.scrollLeft - m.wheelStartX,
                          bt = y.scrollTop - m.wheelStartY,
                          Ct =
                            (bt && m.wheelDY && bt / m.wheelDY) ||
                            (vt && m.wheelDX && vt / m.wheelDX);
                        (m.wheelStartX = m.wheelStartY = null),
                          Ct && ((Mr = (Mr * sa + Ct) / (sa + 1)), ++sa);
                      }
                    }, 200))
                  : ((m.wheelDX += l), (m.wheelDY += c)));
            }
          }
          var Dn = function (n, i) {
            (this.ranges = n), (this.primIndex = i);
          };
          (Dn.prototype.primary = function () {
            return this.ranges[this.primIndex];
          }),
            (Dn.prototype.equals = function (n) {
              if (n == this) return !0;
              if (n.primIndex != this.primIndex || n.ranges.length != this.ranges.length) return !1;
              for (var i = 0; i < this.ranges.length; i++) {
                var a = this.ranges[i],
                  l = n.ranges[i];
                if (!ce(a.anchor, l.anchor) || !ce(a.head, l.head)) return !1;
              }
              return !0;
            }),
            (Dn.prototype.deepCopy = function () {
              for (var n = [], i = 0; i < this.ranges.length; i++)
                n[i] = new ue(Fe(this.ranges[i].anchor), Fe(this.ranges[i].head));
              return new Dn(n, this.primIndex);
            }),
            (Dn.prototype.somethingSelected = function () {
              for (var n = 0; n < this.ranges.length; n++) if (!this.ranges[n].empty()) return !0;
              return !1;
            }),
            (Dn.prototype.contains = function (n, i) {
              i || (i = n);
              for (var a = 0; a < this.ranges.length; a++) {
                var l = this.ranges[a];
                if (_t(i, l.from()) >= 0 && _t(n, l.to()) <= 0) return a;
              }
              return -1;
            });
          var ue = function (n, i) {
            (this.anchor = n), (this.head = i);
          };
          (ue.prototype.from = function () {
            return wo(this.anchor, this.head);
          }),
            (ue.prototype.to = function () {
              return sn(this.anchor, this.head);
            }),
            (ue.prototype.empty = function () {
              return this.head.line == this.anchor.line && this.head.ch == this.anchor.ch;
            });
          function Qn(n, i, a) {
            var l = n && n.options.selectionsMayTouch,
              c = i[a];
            i.sort(function (W, q) {
              return _t(W.from(), q.from());
            }),
              (a = Et(i, c));
            for (var p = 1; p < i.length; p++) {
              var m = i[p],
                y = i[p - 1],
                x = _t(y.to(), m.from());
              if (l && !m.empty() ? x > 0 : x >= 0) {
                var _ = wo(y.from(), m.from()),
                  N = sn(y.to(), m.to()),
                  D = y.empty() ? m.from() == m.head : y.from() == y.head;
                p <= a && --a, i.splice(--p, 2, new ue(D ? N : _, D ? _ : N));
              }
            }
            return new Dn(i, a);
          }
          function ri(n, i) {
            return new Dn([new ue(n, i || n)], 0);
          }
          function ii(n) {
            return n.text
              ? X(
                  n.from.line + n.text.length - 1,
                  ct(n.text).length + (n.text.length == 1 ? n.from.ch : 0),
                )
              : n.to;
          }
          function hp(n, i) {
            if (_t(n, i.from) < 0) return n;
            if (_t(n, i.to) <= 0) return ii(i);
            var a = n.line + i.text.length - (i.to.line - i.from.line) - 1,
              l = n.ch;
            return n.line == i.to.line && (l += ii(i).ch - i.to.ch), X(a, l);
          }
          function Eu(n, i) {
            for (var a = [], l = 0; l < n.sel.ranges.length; l++) {
              var c = n.sel.ranges[l];
              a.push(new ue(hp(c.anchor, i), hp(c.head, i)));
            }
            return Qn(n.cm, a, n.sel.primIndex);
          }
          function dp(n, i, a) {
            return n.line == i.line
              ? X(a.line, n.ch - i.ch + a.ch)
              : X(a.line + (n.line - i.line), n.ch);
          }
          function ex(n, i, a) {
            for (var l = [], c = X(n.first, 0), p = c, m = 0; m < i.length; m++) {
              var y = i[m],
                x = dp(y.from, c, p),
                _ = dp(ii(y), c, p);
              if (((c = y.to), (p = _), a == "around")) {
                var N = n.sel.ranges[m],
                  D = _t(N.head, N.anchor) < 0;
                l[m] = new ue(D ? _ : x, D ? x : _);
              } else l[m] = new ue(x, x);
            }
            return new Dn(l, n.sel.primIndex);
          }
          function Lu(n) {
            (n.doc.mode = mo(n.options, n.doc.modeOption)), zs(n);
          }
          function zs(n) {
            n.doc.iter(function (i) {
              i.stateAfter && (i.stateAfter = null), i.styles && (i.styles = null);
            }),
              (n.doc.modeFrontier = n.doc.highlightFrontier = n.doc.first),
              $s(n, 100),
              n.state.modeGen++,
              n.curOp && ln(n);
          }
          function pp(n, i) {
            return (
              i.from.ch == 0 &&
              i.to.ch == 0 &&
              ct(i.text) == "" &&
              (!n.cm || n.cm.options.wholeLineUpdateBefore)
            );
          }
          function Au(n, i, a, l) {
            function c(Ct) {
              return a ? a[Ct] : null;
            }
            function p(Ct, wt, Lt) {
              cw(Ct, wt, Lt, l), qe(Ct, "change", Ct, i);
            }
            function m(Ct, wt) {
              for (var Lt = [], zt = Ct; zt < wt; ++zt) Lt.push(new xo(_[zt], c(zt), l));
              return Lt;
            }
            var y = i.from,
              x = i.to,
              _ = i.text,
              N = Pt(n, y.line),
              D = Pt(n, x.line),
              W = ct(_),
              q = c(_.length - 1),
              Z = x.line - y.line;
            if (i.full) n.insert(0, m(0, _.length)), n.remove(_.length, n.size - _.length);
            else if (pp(n, i)) {
              var it = m(0, _.length - 1);
              p(D, D.text, q), Z && n.remove(y.line, Z), it.length && n.insert(y.line, it);
            } else if (N == D)
              if (_.length == 1) p(N, N.text.slice(0, y.ch) + W + N.text.slice(x.ch), q);
              else {
                var vt = m(1, _.length - 1);
                vt.push(new xo(W + N.text.slice(x.ch), q, l)),
                  p(N, N.text.slice(0, y.ch) + _[0], c(0)),
                  n.insert(y.line + 1, vt);
              }
            else if (_.length == 1)
              p(N, N.text.slice(0, y.ch) + _[0] + D.text.slice(x.ch), c(0)),
                n.remove(y.line + 1, Z);
            else {
              p(N, N.text.slice(0, y.ch) + _[0], c(0)), p(D, W + D.text.slice(x.ch), q);
              var bt = m(1, _.length - 1);
              Z > 1 && n.remove(y.line + 1, Z - 1), n.insert(y.line + 1, bt);
            }
            qe(n, "change", n, i);
          }
          function oi(n, i, a) {
            function l(c, p, m) {
              if (c.linked)
                for (var y = 0; y < c.linked.length; ++y) {
                  var x = c.linked[y];
                  if (x.doc != p) {
                    var _ = m && x.sharedHist;
                    (a && !_) || (i(x.doc, _), l(x.doc, c, _));
                  }
                }
            }
            l(n, null, !0);
          }
          function gp(n, i) {
            if (i.cm) throw new Error("This document is already in use.");
            (n.doc = i),
              (i.cm = n),
              gu(n),
              Lu(n),
              vp(n),
              (n.options.direction = i.direction),
              n.options.lineWrapping || iu(n),
              (n.options.mode = i.modeOption),
              ln(n);
          }
          function vp(n) {
            (n.doc.direction == "rtl" ? At : gt)(n.display.lineDiv, "CodeMirror-rtl");
          }
          function nx(n) {
            Sn(n, function () {
              vp(n), ln(n);
            });
          }
          function la(n) {
            (this.done = []),
              (this.undone = []),
              (this.undoDepth = n ? n.undoDepth : 1 / 0),
              (this.lastModTime = this.lastSelTime = 0),
              (this.lastOp = this.lastSelOp = null),
              (this.lastOrigin = this.lastSelOrigin = null),
              (this.generation = this.maxGeneration = n ? n.maxGeneration : 1);
          }
          function Mu(n, i) {
            var a = { from: Fe(i.from), to: ii(i), text: Tr(n, i.from, i.to) };
            return (
              bp(n, a, i.from.line, i.to.line + 1),
              oi(
                n,
                function (l) {
                  return bp(l, a, i.from.line, i.to.line + 1);
                },
                !0,
              ),
              a
            );
          }
          function mp(n) {
            for (; n.length; ) {
              var i = ct(n);
              if (i.ranges) n.pop();
              else break;
            }
          }
          function rx(n, i) {
            if (i) return mp(n.done), ct(n.done);
            if (n.done.length && !ct(n.done).ranges) return ct(n.done);
            if (n.done.length > 1 && !n.done[n.done.length - 2].ranges)
              return n.done.pop(), ct(n.done);
          }
          function yp(n, i, a, l) {
            var c = n.history;
            c.undone.length = 0;
            var p = +new Date(),
              m,
              y;
            if (
              (c.lastOp == l ||
                (c.lastOrigin == i.origin &&
                  i.origin &&
                  ((i.origin.charAt(0) == "+" &&
                    c.lastModTime > p - (n.cm ? n.cm.options.historyEventDelay : 500)) ||
                    i.origin.charAt(0) == "*"))) &&
              (m = rx(c, c.lastOp == l))
            )
              (y = ct(m.changes)),
                _t(i.from, i.to) == 0 && _t(i.from, y.to) == 0
                  ? (y.to = ii(i))
                  : m.changes.push(Mu(n, i));
            else {
              var x = ct(c.done);
              for (
                (!x || !x.ranges) && aa(n.sel, c.done),
                  m = { changes: [Mu(n, i)], generation: c.generation },
                  c.done.push(m);
                c.done.length > c.undoDepth;
              )
                c.done.shift(), c.done[0].ranges || c.done.shift();
            }
            c.done.push(a),
              (c.generation = ++c.maxGeneration),
              (c.lastModTime = c.lastSelTime = p),
              (c.lastOp = c.lastSelOp = l),
              (c.lastOrigin = c.lastSelOrigin = i.origin),
              y || ke(n, "historyAdded");
          }
          function ix(n, i, a, l) {
            var c = i.charAt(0);
            return (
              c == "*" ||
              (c == "+" &&
                a.ranges.length == l.ranges.length &&
                a.somethingSelected() == l.somethingSelected() &&
                new Date() - n.history.lastSelTime <= (n.cm ? n.cm.options.historyEventDelay : 500))
            );
          }
          function ox(n, i, a, l) {
            var c = n.history,
              p = l && l.origin;
            a == c.lastSelOp ||
            (p &&
              c.lastSelOrigin == p &&
              ((c.lastModTime == c.lastSelTime && c.lastOrigin == p) || ix(n, p, ct(c.done), i)))
              ? (c.done[c.done.length - 1] = i)
              : aa(i, c.done),
              (c.lastSelTime = +new Date()),
              (c.lastSelOrigin = p),
              (c.lastSelOp = a),
              l && l.clearRedo !== !1 && mp(c.undone);
          }
          function aa(n, i) {
            var a = ct(i);
            (a && a.ranges && a.equals(n)) || i.push(n);
          }
          function bp(n, i, a, l) {
            var c = i["spans_" + n.id],
              p = 0;
            n.iter(Math.max(n.first, a), Math.min(n.first + n.size, l), function (m) {
              m.markedSpans && ((c || (c = i["spans_" + n.id] = {}))[p] = m.markedSpans), ++p;
            });
          }
          function sx(n) {
            if (!n) return null;
            for (var i, a = 0; a < n.length; ++a)
              n[a].marker.explicitlyCleared ? i || (i = n.slice(0, a)) : i && i.push(n[a]);
            return i ? (i.length ? i : null) : n;
          }
          function lx(n, i) {
            var a = i["spans_" + n.id];
            if (!a) return null;
            for (var l = [], c = 0; c < i.text.length; ++c) l.push(sx(a[c]));
            return l;
          }
          function wp(n, i) {
            var a = lx(n, i),
              l = tu(n, i);
            if (!a) return l;
            if (!l) return a;
            for (var c = 0; c < a.length; ++c) {
              var p = a[c],
                m = l[c];
              if (p && m)
                t: for (var y = 0; y < m.length; ++y) {
                  for (var x = m[y], _ = 0; _ < p.length; ++_)
                    if (p[_].marker == x.marker) continue t;
                  p.push(x);
                }
              else m && (a[c] = m);
            }
            return a;
          }
          function Ao(n, i, a) {
            for (var l = [], c = 0; c < n.length; ++c) {
              var p = n[c];
              if (p.ranges) {
                l.push(a ? Dn.prototype.deepCopy.call(p) : p);
                continue;
              }
              var m = p.changes,
                y = [];
              l.push({ changes: y });
              for (var x = 0; x < m.length; ++x) {
                var _ = m[x],
                  N = void 0;
                if ((y.push({ from: _.from, to: _.to, text: _.text }), i))
                  for (var D in _)
                    (N = D.match(/^spans_(\d+)$/)) &&
                      Et(i, Number(N[1])) > -1 &&
                      ((ct(y)[D] = _[D]), delete _[D]);
              }
            }
            return l;
          }
          function Nu(n, i, a, l) {
            if (l) {
              var c = n.anchor;
              if (a) {
                var p = _t(i, c) < 0;
                p != _t(a, c) < 0 ? ((c = i), (i = a)) : p != _t(i, a) < 0 && (i = a);
              }
              return new ue(c, i);
            } else return new ue(a || i, i);
          }
          function ca(n, i, a, l, c) {
            c == null && (c = n.cm && (n.cm.display.shift || n.extend)),
              Ye(n, new Dn([Nu(n.sel.primary(), i, a, c)], 0), l);
          }
          function xp(n, i, a) {
            for (
              var l = [], c = n.cm && (n.cm.display.shift || n.extend), p = 0;
              p < n.sel.ranges.length;
              p++
            )
              l[p] = Nu(n.sel.ranges[p], i[p], null, c);
            var m = Qn(n.cm, l, n.sel.primIndex);
            Ye(n, m, a);
          }
          function Pu(n, i, a, l) {
            var c = n.sel.ranges.slice(0);
            (c[i] = a), Ye(n, Qn(n.cm, c, n.sel.primIndex), l);
          }
          function _p(n, i, a, l) {
            Ye(n, ri(i, a), l);
          }
          function ax(n, i, a) {
            var l = {
              ranges: i.ranges,
              update: function (c) {
                this.ranges = [];
                for (var p = 0; p < c.length; p++)
                  this.ranges[p] = new ue(Wt(n, c[p].anchor), Wt(n, c[p].head));
              },
              origin: a && a.origin,
            };
            return (
              ke(n, "beforeSelectionChange", n, l),
              n.cm && ke(n.cm, "beforeSelectionChange", n.cm, l),
              l.ranges != i.ranges ? Qn(n.cm, l.ranges, l.ranges.length - 1) : i
            );
          }
          function Sp(n, i, a) {
            var l = n.history.done,
              c = ct(l);
            c && c.ranges ? ((l[l.length - 1] = i), ua(n, i, a)) : Ye(n, i, a);
          }
          function Ye(n, i, a) {
            ua(n, i, a), ox(n, n.sel, n.cm ? n.cm.curOp.id : NaN, a);
          }
          function ua(n, i, a) {
            (_n(n, "beforeSelectionChange") || (n.cm && _n(n.cm, "beforeSelectionChange"))) &&
              (i = ax(n, i, a));
            var l = (a && a.bias) || (_t(i.primary().head, n.sel.primary().head) < 0 ? -1 : 1);
            kp(n, Tp(n, i, l, !0)),
              !(a && a.scroll === !1) &&
                n.cm &&
                n.cm.getOption("readOnly") != "nocursor" &&
                Eo(n.cm);
          }
          function kp(n, i) {
            i.equals(n.sel) ||
              ((n.sel = i),
              n.cm && ((n.cm.curOp.updateInput = 1), (n.cm.curOp.selectionChanged = !0), qn(n.cm)),
              qe(n, "cursorActivity", n));
          }
          function Cp(n) {
            kp(n, Tp(n, n.sel, null, !1));
          }
          function Tp(n, i, a, l) {
            for (var c, p = 0; p < i.ranges.length; p++) {
              var m = i.ranges[p],
                y = i.ranges.length == n.sel.ranges.length && n.sel.ranges[p],
                x = fa(n, m.anchor, y && y.anchor, a, l),
                _ = m.head == m.anchor ? x : fa(n, m.head, y && y.head, a, l);
              (c || x != m.anchor || _ != m.head) &&
                (c || (c = i.ranges.slice(0, p)), (c[p] = new ue(x, _)));
            }
            return c ? Qn(n.cm, c, i.primIndex) : i;
          }
          function Mo(n, i, a, l, c) {
            var p = Pt(n, i.line);
            if (p.markedSpans)
              for (var m = 0; m < p.markedSpans.length; ++m) {
                var y = p.markedSpans[m],
                  x = y.marker,
                  _ = "selectLeft" in x ? !x.selectLeft : x.inclusiveLeft,
                  N = "selectRight" in x ? !x.selectRight : x.inclusiveRight;
                if (
                  (y.from == null || (_ ? y.from <= i.ch : y.from < i.ch)) &&
                  (y.to == null || (N ? y.to >= i.ch : y.to > i.ch))
                ) {
                  if (c && (ke(x, "beforeCursorEnter"), x.explicitlyCleared))
                    if (p.markedSpans) {
                      --m;
                      continue;
                    } else break;
                  if (!x.atomic) continue;
                  if (a) {
                    var D = x.find(l < 0 ? 1 : -1),
                      W = void 0;
                    if (
                      ((l < 0 ? N : _) && (D = Ep(n, D, -l, D && D.line == i.line ? p : null)),
                      D && D.line == i.line && (W = _t(D, a)) && (l < 0 ? W < 0 : W > 0))
                    )
                      return Mo(n, D, i, l, c);
                  }
                  var q = x.find(l < 0 ? -1 : 1);
                  return (
                    (l < 0 ? _ : N) && (q = Ep(n, q, l, q.line == i.line ? p : null)),
                    q ? Mo(n, q, i, l, c) : null
                  );
                }
              }
            return i;
          }
          function fa(n, i, a, l, c) {
            var p = l || 1,
              m =
                Mo(n, i, a, p, c) ||
                (!c && Mo(n, i, a, p, !0)) ||
                Mo(n, i, a, -p, c) ||
                (!c && Mo(n, i, a, -p, !0));
            return m || ((n.cantEdit = !0), X(n.first, 0));
          }
          function Ep(n, i, a, l) {
            return a < 0 && i.ch == 0
              ? i.line > n.first
                ? Wt(n, X(i.line - 1))
                : null
              : a > 0 && i.ch == (l || Pt(n, i.line)).text.length
              ? i.line < n.first + n.size - 1
                ? X(i.line + 1, 0)
                : null
              : new X(i.line, i.ch + a);
          }
          function Lp(n) {
            n.setSelection(X(n.firstLine(), 0), X(n.lastLine()), V);
          }
          function Ap(n, i, a) {
            var l = {
              canceled: !1,
              from: i.from,
              to: i.to,
              text: i.text,
              origin: i.origin,
              cancel: function () {
                return (l.canceled = !0);
              },
            };
            return (
              a &&
                (l.update = function (c, p, m, y) {
                  c && (l.from = Wt(n, c)),
                    p && (l.to = Wt(n, p)),
                    m && (l.text = m),
                    y !== void 0 && (l.origin = y);
                }),
              ke(n, "beforeChange", n, l),
              n.cm && ke(n.cm, "beforeChange", n.cm, l),
              l.canceled
                ? (n.cm && (n.cm.curOp.updateInput = 2), null)
                : { from: l.from, to: l.to, text: l.text, origin: l.origin }
            );
          }
          function No(n, i, a) {
            if (n.cm) {
              if (!n.cm.curOp) return He(n.cm, No)(n, i, a);
              if (n.cm.state.suppressEdits) return;
            }
            if (
              !(
                (_n(n, "beforeChange") || (n.cm && _n(n.cm, "beforeChange"))) &&
                ((i = Ap(n, i, !0)), !i)
              )
            ) {
              var l = wd && !a && ow(n, i.from, i.to);
              if (l)
                for (var c = l.length - 1; c >= 0; --c)
                  Mp(n, {
                    from: l[c].from,
                    to: l[c].to,
                    text: c ? [""] : i.text,
                    origin: i.origin,
                  });
              else Mp(n, i);
            }
          }
          function Mp(n, i) {
            if (!(i.text.length == 1 && i.text[0] == "" && _t(i.from, i.to) == 0)) {
              var a = Eu(n, i);
              yp(n, i, a, n.cm ? n.cm.curOp.id : NaN), Is(n, i, a, tu(n, i));
              var l = [];
              oi(n, function (c, p) {
                !p && Et(l, c.history) == -1 && (Dp(c.history, i), l.push(c.history)),
                  Is(c, i, null, tu(c, i));
              });
            }
          }
          function ha(n, i, a) {
            var l = n.cm && n.cm.state.suppressEdits;
            if (!(l && !a)) {
              for (
                var c = n.history,
                  p,
                  m = n.sel,
                  y = i == "undo" ? c.done : c.undone,
                  x = i == "undo" ? c.undone : c.done,
                  _ = 0;
                _ < y.length && ((p = y[_]), !(a ? p.ranges && !p.equals(n.sel) : !p.ranges));
                _++
              );
              if (_ != y.length) {
                for (c.lastOrigin = c.lastSelOrigin = null; ; )
                  if (((p = y.pop()), p.ranges)) {
                    if ((aa(p, x), a && !p.equals(n.sel))) {
                      Ye(n, p, { clearRedo: !1 });
                      return;
                    }
                    m = p;
                  } else if (l) {
                    y.push(p);
                    return;
                  } else break;
                var N = [];
                aa(m, x),
                  x.push({ changes: N, generation: c.generation }),
                  (c.generation = p.generation || ++c.maxGeneration);
                for (
                  var D = _n(n, "beforeChange") || (n.cm && _n(n.cm, "beforeChange")),
                    W = function (it) {
                      var vt = p.changes[it];
                      if (((vt.origin = i), D && !Ap(n, vt, !1))) return (y.length = 0), {};
                      N.push(Mu(n, vt));
                      var bt = it ? Eu(n, vt) : ct(y);
                      Is(n, vt, bt, wp(n, vt)),
                        !it && n.cm && n.cm.scrollIntoView({ from: vt.from, to: ii(vt) });
                      var Ct = [];
                      oi(n, function (wt, Lt) {
                        !Lt &&
                          Et(Ct, wt.history) == -1 &&
                          (Dp(wt.history, vt), Ct.push(wt.history)),
                          Is(wt, vt, null, wp(wt, vt));
                      });
                    },
                    q = p.changes.length - 1;
                  q >= 0;
                  --q
                ) {
                  var Z = W(q);
                  if (Z) return Z.v;
                }
              }
            }
          }
          function Np(n, i) {
            if (
              i != 0 &&
              ((n.first += i),
              (n.sel = new Dn(
                ft(n.sel.ranges, function (c) {
                  return new ue(X(c.anchor.line + i, c.anchor.ch), X(c.head.line + i, c.head.ch));
                }),
                n.sel.primIndex,
              )),
              n.cm)
            ) {
              ln(n.cm, n.first, n.first - i, i);
              for (var a = n.cm.display, l = a.viewFrom; l < a.viewTo; l++) ei(n.cm, l, "gutter");
            }
          }
          function Is(n, i, a, l) {
            if (n.cm && !n.cm.curOp) return He(n.cm, Is)(n, i, a, l);
            if (i.to.line < n.first) {
              Np(n, i.text.length - 1 - (i.to.line - i.from.line));
              return;
            }
            if (!(i.from.line > n.lastLine())) {
              if (i.from.line < n.first) {
                var c = i.text.length - 1 - (n.first - i.from.line);
                Np(n, c),
                  (i = {
                    from: X(n.first, 0),
                    to: X(i.to.line + c, i.to.ch),
                    text: [ct(i.text)],
                    origin: i.origin,
                  });
              }
              var p = n.lastLine();
              i.to.line > p &&
                (i = {
                  from: i.from,
                  to: X(p, Pt(n, p).text.length),
                  text: [i.text[0]],
                  origin: i.origin,
                }),
                (i.removed = Tr(n, i.from, i.to)),
                a || (a = Eu(n, i)),
                n.cm ? cx(n.cm, i, l) : Au(n, i, l),
                ua(n, a, V),
                n.cantEdit && fa(n, X(n.firstLine(), 0)) && (n.cantEdit = !1);
            }
          }
          function cx(n, i, a) {
            var l = n.doc,
              c = n.display,
              p = i.from,
              m = i.to,
              y = !1,
              x = p.line;
            n.options.lineWrapping ||
              ((x = C(Zn(Pt(l, p.line)))),
              l.iter(x, m.line + 1, function (q) {
                if (q == c.maxLine) return (y = !0), !0;
              })),
              l.sel.contains(i.from, i.to) > -1 && qn(n),
              Au(l, i, a, Yd(n)),
              n.options.lineWrapping ||
                (l.iter(x, p.line + i.text.length, function (q) {
                  var Z = Xl(q);
                  Z > c.maxLineLength &&
                    ((c.maxLine = q), (c.maxLineLength = Z), (c.maxLineChanged = !0), (y = !1));
                }),
                y && (n.curOp.updateMaxLine = !0)),
              Jb(l, p.line),
              $s(n, 400);
            var _ = i.text.length - (m.line - p.line) - 1;
            i.full
              ? ln(n)
              : p.line == m.line && i.text.length == 1 && !pp(n.doc, i)
              ? ei(n, p.line, "text")
              : ln(n, p.line, m.line + 1, _);
            var N = _n(n, "changes"),
              D = _n(n, "change");
            if (D || N) {
              var W = { from: p, to: m, text: i.text, removed: i.removed, origin: i.origin };
              D && qe(n, "change", n, W),
                N && (n.curOp.changeObjs || (n.curOp.changeObjs = [])).push(W);
            }
            n.display.selForContextMenu = null;
          }
          function Po(n, i, a, l, c) {
            var p;
            l || (l = a),
              _t(l, a) < 0 && ((p = [l, a]), (a = p[0]), (l = p[1])),
              typeof i == "string" && (i = n.splitLines(i)),
              No(n, { from: a, to: l, text: i, origin: c });
          }
          function Pp(n, i, a, l) {
            a < n.line ? (n.line += l) : i < n.line && ((n.line = i), (n.ch = 0));
          }
          function Op(n, i, a, l) {
            for (var c = 0; c < n.length; ++c) {
              var p = n[c],
                m = !0;
              if (p.ranges) {
                p.copied || ((p = n[c] = p.deepCopy()), (p.copied = !0));
                for (var y = 0; y < p.ranges.length; y++)
                  Pp(p.ranges[y].anchor, i, a, l), Pp(p.ranges[y].head, i, a, l);
                continue;
              }
              for (var x = 0; x < p.changes.length; ++x) {
                var _ = p.changes[x];
                if (a < _.from.line)
                  (_.from = X(_.from.line + l, _.from.ch)), (_.to = X(_.to.line + l, _.to.ch));
                else if (i <= _.to.line) {
                  m = !1;
                  break;
                }
              }
              m || (n.splice(0, c + 1), (c = 0));
            }
          }
          function Dp(n, i) {
            var a = i.from.line,
              l = i.to.line,
              c = i.text.length - (l - a) - 1;
            Op(n.done, a, l, c), Op(n.undone, a, l, c);
          }
          function Fs(n, i, a, l) {
            var c = i,
              p = i;
            return (
              typeof i == "number" ? (p = Pt(n, fd(n, i))) : (c = C(i)),
              c == null ? null : (l(p, c) && n.cm && ei(n.cm, c, a), p)
            );
          }
          function qs(n) {
            (this.lines = n), (this.parent = null);
            for (var i = 0, a = 0; a < n.length; ++a) (n[a].parent = this), (i += n[a].height);
            this.height = i;
          }
          qs.prototype = {
            chunkSize: function () {
              return this.lines.length;
            },
            removeInner: function (n, i) {
              for (var a = n, l = n + i; a < l; ++a) {
                var c = this.lines[a];
                (this.height -= c.height), uw(c), qe(c, "delete");
              }
              this.lines.splice(n, i);
            },
            collapse: function (n) {
              n.push.apply(n, this.lines);
            },
            insertInner: function (n, i, a) {
              (this.height += a),
                (this.lines = this.lines.slice(0, n).concat(i).concat(this.lines.slice(n)));
              for (var l = 0; l < i.length; ++l) i[l].parent = this;
            },
            iterN: function (n, i, a) {
              for (var l = n + i; n < l; ++n) if (a(this.lines[n])) return !0;
            },
          };
          function Hs(n) {
            this.children = n;
            for (var i = 0, a = 0, l = 0; l < n.length; ++l) {
              var c = n[l];
              (i += c.chunkSize()), (a += c.height), (c.parent = this);
            }
            (this.size = i), (this.height = a), (this.parent = null);
          }
          Hs.prototype = {
            chunkSize: function () {
              return this.size;
            },
            removeInner: function (n, i) {
              this.size -= i;
              for (var a = 0; a < this.children.length; ++a) {
                var l = this.children[a],
                  c = l.chunkSize();
                if (n < c) {
                  var p = Math.min(i, c - n),
                    m = l.height;
                  if (
                    (l.removeInner(n, p),
                    (this.height -= m - l.height),
                    c == p && (this.children.splice(a--, 1), (l.parent = null)),
                    (i -= p) == 0)
                  )
                    break;
                  n = 0;
                } else n -= c;
              }
              if (
                this.size - i < 25 &&
                (this.children.length > 1 || !(this.children[0] instanceof qs))
              ) {
                var y = [];
                this.collapse(y), (this.children = [new qs(y)]), (this.children[0].parent = this);
              }
            },
            collapse: function (n) {
              for (var i = 0; i < this.children.length; ++i) this.children[i].collapse(n);
            },
            insertInner: function (n, i, a) {
              (this.size += i.length), (this.height += a);
              for (var l = 0; l < this.children.length; ++l) {
                var c = this.children[l],
                  p = c.chunkSize();
                if (n <= p) {
                  if ((c.insertInner(n, i, a), c.lines && c.lines.length > 50)) {
                    for (var m = (c.lines.length % 25) + 25, y = m; y < c.lines.length; ) {
                      var x = new qs(c.lines.slice(y, (y += 25)));
                      (c.height -= x.height), this.children.splice(++l, 0, x), (x.parent = this);
                    }
                    (c.lines = c.lines.slice(0, m)), this.maybeSpill();
                  }
                  break;
                }
                n -= p;
              }
            },
            maybeSpill: function () {
              if (!(this.children.length <= 10)) {
                var n = this;
                do {
                  var i = n.children.splice(n.children.length - 5, 5),
                    a = new Hs(i);
                  if (n.parent) {
                    (n.size -= a.size), (n.height -= a.height);
                    var c = Et(n.parent.children, n);
                    n.parent.children.splice(c + 1, 0, a);
                  } else {
                    var l = new Hs(n.children);
                    (l.parent = n), (n.children = [l, a]), (n = l);
                  }
                  a.parent = n.parent;
                } while (n.children.length > 10);
                n.parent.maybeSpill();
              }
            },
            iterN: function (n, i, a) {
              for (var l = 0; l < this.children.length; ++l) {
                var c = this.children[l],
                  p = c.chunkSize();
                if (n < p) {
                  var m = Math.min(i, p - n);
                  if (c.iterN(n, m, a)) return !0;
                  if ((i -= m) == 0) break;
                  n = 0;
                } else n -= p;
              }
            },
          };
          var Bs = function (n, i, a) {
            if (a) for (var l in a) a.hasOwnProperty(l) && (this[l] = a[l]);
            (this.doc = n), (this.node = i);
          };
          (Bs.prototype.clear = function () {
            var n = this.doc.cm,
              i = this.line.widgets,
              a = this.line,
              l = C(a);
            if (!(l == null || !i)) {
              for (var c = 0; c < i.length; ++c) i[c] == this && i.splice(c--, 1);
              i.length || (a.widgets = null);
              var p = Ls(this);
              On(a, Math.max(0, a.height - p)),
                n &&
                  (Sn(n, function () {
                    $p(n, a, -p), ei(n, l, "widget");
                  }),
                  qe(n, "lineWidgetCleared", n, this, l));
            }
          }),
            (Bs.prototype.changed = function () {
              var n = this,
                i = this.height,
                a = this.doc.cm,
                l = this.line;
              this.height = null;
              var c = Ls(this) - i;
              c &&
                (ti(this.doc, l) || On(l, l.height + c),
                a &&
                  Sn(a, function () {
                    (a.curOp.forceUpdate = !0), $p(a, l, c), qe(a, "lineWidgetChanged", a, n, C(l));
                  }));
            }),
            Vn(Bs);
          function $p(n, i, a) {
            Lr(i) < ((n.curOp && n.curOp.scrollTop) || n.doc.scrollTop) && xu(n, a);
          }
          function ux(n, i, a, l) {
            var c = new Bs(n, a, l),
              p = n.cm;
            return (
              p && c.noHScroll && (p.display.alignWidgets = !0),
              Fs(n, i, "widget", function (m) {
                var y = m.widgets || (m.widgets = []);
                if (
                  (c.insertAt == null
                    ? y.push(c)
                    : y.splice(Math.min(y.length, Math.max(0, c.insertAt)), 0, c),
                  (c.line = m),
                  p && !ti(n, m))
                ) {
                  var x = Lr(m) < n.scrollTop;
                  On(m, m.height + Ls(c)), x && xu(p, c.height), (p.curOp.forceUpdate = !0);
                }
                return !0;
              }),
              p && qe(p, "lineWidgetAdded", p, c, typeof i == "number" ? i : C(i)),
              c
            );
          }
          var Rp = 0,
            si = function (n, i) {
              (this.lines = []), (this.type = i), (this.doc = n), (this.id = ++Rp);
            };
          (si.prototype.clear = function () {
            if (!this.explicitlyCleared) {
              var n = this.doc.cm,
                i = n && !n.curOp;
              if ((i && Fi(n), _n(this, "clear"))) {
                var a = this.find();
                a && qe(this, "clear", a.from, a.to);
              }
              for (var l = null, c = null, p = 0; p < this.lines.length; ++p) {
                var m = this.lines[p],
                  y = Cs(m.markedSpans, this);
                n && !this.collapsed
                  ? ei(n, C(m), "text")
                  : n && (y.to != null && (c = C(m)), y.from != null && (l = C(m))),
                  (m.markedSpans = ew(m.markedSpans, y)),
                  y.from == null && this.collapsed && !ti(this.doc, m) && n && On(m, ko(n.display));
              }
              if (n && this.collapsed && !n.options.lineWrapping)
                for (var x = 0; x < this.lines.length; ++x) {
                  var _ = Zn(this.lines[x]),
                    N = Xl(_);
                  N > n.display.maxLineLength &&
                    ((n.display.maxLine = _),
                    (n.display.maxLineLength = N),
                    (n.display.maxLineChanged = !0));
                }
              l != null && n && this.collapsed && ln(n, l, c + 1),
                (this.lines.length = 0),
                (this.explicitlyCleared = !0),
                this.atomic && this.doc.cantEdit && ((this.doc.cantEdit = !1), n && Cp(n.doc)),
                n && qe(n, "markerCleared", n, this, l, c),
                i && qi(n),
                this.parent && this.parent.clear();
            }
          }),
            (si.prototype.find = function (n, i) {
              n == null && this.type == "bookmark" && (n = 1);
              for (var a, l, c = 0; c < this.lines.length; ++c) {
                var p = this.lines[c],
                  m = Cs(p.markedSpans, this);
                if (m.from != null && ((a = X(i ? p : C(p), m.from)), n == -1)) return a;
                if (m.to != null && ((l = X(i ? p : C(p), m.to)), n == 1)) return l;
              }
              return a && { from: a, to: l };
            }),
            (si.prototype.changed = function () {
              var n = this,
                i = this.find(-1, !0),
                a = this,
                l = this.doc.cm;
              !i ||
                !l ||
                Sn(l, function () {
                  var c = i.line,
                    p = C(i.line),
                    m = au(l, p);
                  if (
                    (m && (Bd(m), (l.curOp.selectionChanged = l.curOp.forceUpdate = !0)),
                    (l.curOp.updateMaxLine = !0),
                    !ti(a.doc, c) && a.height != null)
                  ) {
                    var y = a.height;
                    a.height = null;
                    var x = Ls(a) - y;
                    x && On(c, c.height + x);
                  }
                  qe(l, "markerChanged", l, n);
                });
            }),
            (si.prototype.attachLine = function (n) {
              if (!this.lines.length && this.doc.cm) {
                var i = this.doc.cm.curOp;
                (!i.maybeHiddenMarkers || Et(i.maybeHiddenMarkers, this) == -1) &&
                  (i.maybeUnhiddenMarkers || (i.maybeUnhiddenMarkers = [])).push(this);
              }
              this.lines.push(n);
            }),
            (si.prototype.detachLine = function (n) {
              if ((this.lines.splice(Et(this.lines, n), 1), !this.lines.length && this.doc.cm)) {
                var i = this.doc.cm.curOp;
                (i.maybeHiddenMarkers || (i.maybeHiddenMarkers = [])).push(this);
              }
            }),
            Vn(si);
          function Oo(n, i, a, l, c) {
            if (l && l.shared) return fx(n, i, a, l, c);
            if (n.cm && !n.cm.curOp) return He(n.cm, Oo)(n, i, a, l, c);
            var p = new si(n, c),
              m = _t(i, a);
            if ((l && rt(l, p, !1), m > 0 || (m == 0 && p.clearWhenEmpty !== !1))) return p;
            if (
              (p.replacedWith &&
                ((p.collapsed = !0),
                (p.widgetNode = F("span", [p.replacedWith], "CodeMirror-widget")),
                l.handleMouseEvents || p.widgetNode.setAttribute("cm-ignore-events", "true"),
                l.insertLeft && (p.widgetNode.insertLeft = !0)),
              p.collapsed)
            ) {
              if (Td(n, i.line, i, a, p) || (i.line != a.line && Td(n, a.line, i, a, p)))
                throw new Error("Inserting collapsed marker partially overlapping an existing one");
              tw();
            }
            p.addToHistory && yp(n, { from: i, to: a, origin: "markText" }, n.sel, NaN);
            var y = i.line,
              x = n.cm,
              _;
            if (
              (n.iter(y, a.line + 1, function (D) {
                x &&
                  p.collapsed &&
                  !x.options.lineWrapping &&
                  Zn(D) == x.display.maxLine &&
                  (_ = !0),
                  p.collapsed && y != i.line && On(D, 0),
                  nw(
                    D,
                    new jl(p, y == i.line ? i.ch : null, y == a.line ? a.ch : null),
                    n.cm && n.cm.curOp,
                  ),
                  ++y;
              }),
              p.collapsed &&
                n.iter(i.line, a.line + 1, function (D) {
                  ti(n, D) && On(D, 0);
                }),
              p.clearOnEnter &&
                Rt(p, "beforeCursorEnter", function () {
                  return p.clear();
                }),
              p.readOnly &&
                (Qb(), (n.history.done.length || n.history.undone.length) && n.clearHistory()),
              p.collapsed && ((p.id = ++Rp), (p.atomic = !0)),
              x)
            ) {
              if ((_ && (x.curOp.updateMaxLine = !0), p.collapsed)) ln(x, i.line, a.line + 1);
              else if (
                p.className ||
                p.startStyle ||
                p.endStyle ||
                p.css ||
                p.attributes ||
                p.title
              )
                for (var N = i.line; N <= a.line; N++) ei(x, N, "text");
              p.atomic && Cp(x.doc), qe(x, "markerAdded", x, p);
            }
            return p;
          }
          var Ws = function (n, i) {
            (this.markers = n), (this.primary = i);
            for (var a = 0; a < n.length; ++a) n[a].parent = this;
          };
          (Ws.prototype.clear = function () {
            if (!this.explicitlyCleared) {
              this.explicitlyCleared = !0;
              for (var n = 0; n < this.markers.length; ++n) this.markers[n].clear();
              qe(this, "clear");
            }
          }),
            (Ws.prototype.find = function (n, i) {
              return this.primary.find(n, i);
            }),
            Vn(Ws);
          function fx(n, i, a, l, c) {
            (l = rt(l)), (l.shared = !1);
            var p = [Oo(n, i, a, l, c)],
              m = p[0],
              y = l.widgetNode;
            return (
              oi(n, function (x) {
                y && (l.widgetNode = y.cloneNode(!0)), p.push(Oo(x, Wt(x, i), Wt(x, a), l, c));
                for (var _ = 0; _ < x.linked.length; ++_) if (x.linked[_].isParent) return;
                m = ct(p);
              }),
              new Ws(p, m)
            );
          }
          function zp(n) {
            return n.findMarks(X(n.first, 0), n.clipPos(X(n.lastLine())), function (i) {
              return i.parent;
            });
          }
          function hx(n, i) {
            for (var a = 0; a < i.length; a++) {
              var l = i[a],
                c = l.find(),
                p = n.clipPos(c.from),
                m = n.clipPos(c.to);
              if (_t(p, m)) {
                var y = Oo(n, p, m, l.primary, l.primary.type);
                l.markers.push(y), (y.parent = l);
              }
            }
          }
          function dx(n) {
            for (
              var i = function (l) {
                  var c = n[l],
                    p = [c.primary.doc];
                  oi(c.primary.doc, function (x) {
                    return p.push(x);
                  });
                  for (var m = 0; m < c.markers.length; m++) {
                    var y = c.markers[m];
                    Et(p, y.doc) == -1 && ((y.parent = null), c.markers.splice(m--, 1));
                  }
                },
                a = 0;
              a < n.length;
              a++
            )
              i(a);
          }
          var px = 0,
            an = function (n, i, a, l, c) {
              if (!(this instanceof an)) return new an(n, i, a, l, c);
              a == null && (a = 0),
                Hs.call(this, [new qs([new xo("", null)])]),
                (this.first = a),
                (this.scrollTop = this.scrollLeft = 0),
                (this.cantEdit = !1),
                (this.cleanGeneration = 1),
                (this.modeFrontier = this.highlightFrontier = a);
              var p = X(a, 0);
              (this.sel = ri(p)),
                (this.history = new la(null)),
                (this.id = ++px),
                (this.modeOption = i),
                (this.lineSep = l),
                (this.direction = c == "rtl" ? "rtl" : "ltr"),
                (this.extend = !1),
                typeof n == "string" && (n = this.splitLines(n)),
                Au(this, { from: p, to: p, text: n }),
                Ye(this, ri(p), V);
            };
          (an.prototype = Dt(Hs.prototype, {
            constructor: an,
            iter: function (n, i, a) {
              a
                ? this.iterN(n - this.first, i - n, a)
                : this.iterN(this.first, this.first + this.size, n);
            },
            insert: function (n, i) {
              for (var a = 0, l = 0; l < i.length; ++l) a += i[l].height;
              this.insertInner(n - this.first, i, a);
            },
            remove: function (n, i) {
              this.removeInner(n - this.first, i);
            },
            getValue: function (n) {
              var i = Ss(this, this.first, this.first + this.size);
              return n === !1 ? i : i.join(n || this.lineSeparator());
            },
            setValue: Be(function (n) {
              var i = X(this.first, 0),
                a = this.first + this.size - 1;
              No(
                this,
                {
                  from: i,
                  to: X(a, Pt(this, a).text.length),
                  text: this.splitLines(n),
                  origin: "setValue",
                  full: !0,
                },
                !0,
              ),
                this.cm && Ns(this.cm, 0, 0),
                Ye(this, ri(i), V);
            }),
            replaceRange: function (n, i, a, l) {
              (i = Wt(this, i)), (a = a ? Wt(this, a) : i), Po(this, n, i, a, l);
            },
            getRange: function (n, i, a) {
              var l = Tr(this, Wt(this, n), Wt(this, i));
              return a === !1 ? l : a === "" ? l.join("") : l.join(a || this.lineSeparator());
            },
            getLine: function (n) {
              var i = this.getLineHandle(n);
              return i && i.text;
            },
            getLineHandle: function (n) {
              if (et(this, n)) return Pt(this, n);
            },
            getLineNumber: function (n) {
              return C(n);
            },
            getLineHandleVisualStart: function (n) {
              return typeof n == "number" && (n = Pt(this, n)), Zn(n);
            },
            lineCount: function () {
              return this.size;
            },
            firstLine: function () {
              return this.first;
            },
            lastLine: function () {
              return this.first + this.size - 1;
            },
            clipPos: function (n) {
              return Wt(this, n);
            },
            getCursor: function (n) {
              var i = this.sel.primary(),
                a;
              return (
                n == null || n == "head"
                  ? (a = i.head)
                  : n == "anchor"
                  ? (a = i.anchor)
                  : n == "end" || n == "to" || n === !1
                  ? (a = i.to())
                  : (a = i.from()),
                a
              );
            },
            listSelections: function () {
              return this.sel.ranges;
            },
            somethingSelected: function () {
              return this.sel.somethingSelected();
            },
            setCursor: Be(function (n, i, a) {
              _p(this, Wt(this, typeof n == "number" ? X(n, i || 0) : n), null, a);
            }),
            setSelection: Be(function (n, i, a) {
              _p(this, Wt(this, n), Wt(this, i || n), a);
            }),
            extendSelection: Be(function (n, i, a) {
              ca(this, Wt(this, n), i && Wt(this, i), a);
            }),
            extendSelections: Be(function (n, i) {
              xp(this, hd(this, n), i);
            }),
            extendSelectionsBy: Be(function (n, i) {
              var a = ft(this.sel.ranges, n);
              xp(this, hd(this, a), i);
            }),
            setSelections: Be(function (n, i, a) {
              if (n.length) {
                for (var l = [], c = 0; c < n.length; c++)
                  l[c] = new ue(Wt(this, n[c].anchor), Wt(this, n[c].head || n[c].anchor));
                i == null && (i = Math.min(n.length - 1, this.sel.primIndex)),
                  Ye(this, Qn(this.cm, l, i), a);
              }
            }),
            addSelection: Be(function (n, i, a) {
              var l = this.sel.ranges.slice(0);
              l.push(new ue(Wt(this, n), Wt(this, i || n))),
                Ye(this, Qn(this.cm, l, l.length - 1), a);
            }),
            getSelection: function (n) {
              for (var i = this.sel.ranges, a, l = 0; l < i.length; l++) {
                var c = Tr(this, i[l].from(), i[l].to());
                a = a ? a.concat(c) : c;
              }
              return n === !1 ? a : a.join(n || this.lineSeparator());
            },
            getSelections: function (n) {
              for (var i = [], a = this.sel.ranges, l = 0; l < a.length; l++) {
                var c = Tr(this, a[l].from(), a[l].to());
                n !== !1 && (c = c.join(n || this.lineSeparator())), (i[l] = c);
              }
              return i;
            },
            replaceSelection: function (n, i, a) {
              for (var l = [], c = 0; c < this.sel.ranges.length; c++) l[c] = n;
              this.replaceSelections(l, i, a || "+input");
            },
            replaceSelections: Be(function (n, i, a) {
              for (var l = [], c = this.sel, p = 0; p < c.ranges.length; p++) {
                var m = c.ranges[p];
                l[p] = { from: m.from(), to: m.to(), text: this.splitLines(n[p]), origin: a };
              }
              for (var y = i && i != "end" && ex(this, l, i), x = l.length - 1; x >= 0; x--)
                No(this, l[x]);
              y ? Sp(this, y) : this.cm && Eo(this.cm);
            }),
            undo: Be(function () {
              ha(this, "undo");
            }),
            redo: Be(function () {
              ha(this, "redo");
            }),
            undoSelection: Be(function () {
              ha(this, "undo", !0);
            }),
            redoSelection: Be(function () {
              ha(this, "redo", !0);
            }),
            setExtending: function (n) {
              this.extend = n;
            },
            getExtending: function () {
              return this.extend;
            },
            historySize: function () {
              for (var n = this.history, i = 0, a = 0, l = 0; l < n.done.length; l++)
                n.done[l].ranges || ++i;
              for (var c = 0; c < n.undone.length; c++) n.undone[c].ranges || ++a;
              return { undo: i, redo: a };
            },
            clearHistory: function () {
              var n = this;
              (this.history = new la(this.history)),
                oi(
                  this,
                  function (i) {
                    return (i.history = n.history);
                  },
                  !0,
                );
            },
            markClean: function () {
              this.cleanGeneration = this.changeGeneration(!0);
            },
            changeGeneration: function (n) {
              return (
                n &&
                  (this.history.lastOp = this.history.lastSelOp = this.history.lastOrigin = null),
                this.history.generation
              );
            },
            isClean: function (n) {
              return this.history.generation == (n || this.cleanGeneration);
            },
            getHistory: function () {
              return { done: Ao(this.history.done), undone: Ao(this.history.undone) };
            },
            setHistory: function (n) {
              var i = (this.history = new la(this.history));
              (i.done = Ao(n.done.slice(0), null, !0)),
                (i.undone = Ao(n.undone.slice(0), null, !0));
            },
            setGutterMarker: Be(function (n, i, a) {
              return Fs(this, n, "gutter", function (l) {
                var c = l.gutterMarkers || (l.gutterMarkers = {});
                return (c[i] = a), !a && oe(c) && (l.gutterMarkers = null), !0;
              });
            }),
            clearGutter: Be(function (n) {
              var i = this;
              this.iter(function (a) {
                a.gutterMarkers &&
                  a.gutterMarkers[n] &&
                  Fs(i, a, "gutter", function () {
                    return (
                      (a.gutterMarkers[n] = null),
                      oe(a.gutterMarkers) && (a.gutterMarkers = null),
                      !0
                    );
                  });
              });
            }),
            lineInfo: function (n) {
              var i;
              if (typeof n == "number") {
                if (!et(this, n) || ((i = n), (n = Pt(this, n)), !n)) return null;
              } else if (((i = C(n)), i == null)) return null;
              return {
                line: i,
                handle: n,
                text: n.text,
                gutterMarkers: n.gutterMarkers,
                textClass: n.textClass,
                bgClass: n.bgClass,
                wrapClass: n.wrapClass,
                widgets: n.widgets,
              };
            },
            addLineClass: Be(function (n, i, a) {
              return Fs(this, n, i == "gutter" ? "gutter" : "class", function (l) {
                var c =
                  i == "text"
                    ? "textClass"
                    : i == "background"
                    ? "bgClass"
                    : i == "gutter"
                    ? "gutterClass"
                    : "wrapClass";
                if (!l[c]) l[c] = a;
                else {
                  if (pt(a).test(l[c])) return !1;
                  l[c] += " " + a;
                }
                return !0;
              });
            }),
            removeLineClass: Be(function (n, i, a) {
              return Fs(this, n, i == "gutter" ? "gutter" : "class", function (l) {
                var c =
                    i == "text"
                      ? "textClass"
                      : i == "background"
                      ? "bgClass"
                      : i == "gutter"
                      ? "gutterClass"
                      : "wrapClass",
                  p = l[c];
                if (p)
                  if (a == null) l[c] = null;
                  else {
                    var m = p.match(pt(a));
                    if (!m) return !1;
                    var y = m.index + m[0].length;
                    l[c] =
                      p.slice(0, m.index) + (!m.index || y == p.length ? "" : " ") + p.slice(y) ||
                      null;
                  }
                else return !1;
                return !0;
              });
            }),
            addLineWidget: Be(function (n, i, a) {
              return ux(this, n, i, a);
            }),
            removeLineWidget: function (n) {
              n.clear();
            },
            markText: function (n, i, a) {
              return Oo(this, Wt(this, n), Wt(this, i), a, (a && a.type) || "range");
            },
            setBookmark: function (n, i) {
              var a = {
                replacedWith: i && (i.nodeType == null ? i.widget : i),
                insertLeft: i && i.insertLeft,
                clearWhenEmpty: !1,
                shared: i && i.shared,
                handleMouseEvents: i && i.handleMouseEvents,
              };
              return (n = Wt(this, n)), Oo(this, n, n, a, "bookmark");
            },
            findMarksAt: function (n) {
              n = Wt(this, n);
              var i = [],
                a = Pt(this, n.line).markedSpans;
              if (a)
                for (var l = 0; l < a.length; ++l) {
                  var c = a[l];
                  (c.from == null || c.from <= n.ch) &&
                    (c.to == null || c.to >= n.ch) &&
                    i.push(c.marker.parent || c.marker);
                }
              return i;
            },
            findMarks: function (n, i, a) {
              (n = Wt(this, n)), (i = Wt(this, i));
              var l = [],
                c = n.line;
              return (
                this.iter(n.line, i.line + 1, function (p) {
                  var m = p.markedSpans;
                  if (m)
                    for (var y = 0; y < m.length; y++) {
                      var x = m[y];
                      !(
                        (x.to != null && c == n.line && n.ch >= x.to) ||
                        (x.from == null && c != n.line) ||
                        (x.from != null && c == i.line && x.from >= i.ch)
                      ) &&
                        (!a || a(x.marker)) &&
                        l.push(x.marker.parent || x.marker);
                    }
                  ++c;
                }),
                l
              );
            },
            getAllMarks: function () {
              var n = [];
              return (
                this.iter(function (i) {
                  var a = i.markedSpans;
                  if (a)
                    for (var l = 0; l < a.length; ++l) a[l].from != null && n.push(a[l].marker);
                }),
                n
              );
            },
            posFromIndex: function (n) {
              var i,
                a = this.first,
                l = this.lineSeparator().length;
              return (
                this.iter(function (c) {
                  var p = c.text.length + l;
                  if (p > n) return (i = n), !0;
                  (n -= p), ++a;
                }),
                Wt(this, X(a, i))
              );
            },
            indexFromPos: function (n) {
              n = Wt(this, n);
              var i = n.ch;
              if (n.line < this.first || n.ch < 0) return 0;
              var a = this.lineSeparator().length;
              return (
                this.iter(this.first, n.line, function (l) {
                  i += l.text.length + a;
                }),
                i
              );
            },
            copy: function (n) {
              var i = new an(
                Ss(this, this.first, this.first + this.size),
                this.modeOption,
                this.first,
                this.lineSep,
                this.direction,
              );
              return (
                (i.scrollTop = this.scrollTop),
                (i.scrollLeft = this.scrollLeft),
                (i.sel = this.sel),
                (i.extend = !1),
                n &&
                  ((i.history.undoDepth = this.history.undoDepth), i.setHistory(this.getHistory())),
                i
              );
            },
            linkedDoc: function (n) {
              n || (n = {});
              var i = this.first,
                a = this.first + this.size;
              n.from != null && n.from > i && (i = n.from), n.to != null && n.to < a && (a = n.to);
              var l = new an(
                Ss(this, i, a),
                n.mode || this.modeOption,
                i,
                this.lineSep,
                this.direction,
              );
              return (
                n.sharedHist && (l.history = this.history),
                (this.linked || (this.linked = [])).push({ doc: l, sharedHist: n.sharedHist }),
                (l.linked = [{ doc: this, isParent: !0, sharedHist: n.sharedHist }]),
                hx(l, zp(this)),
                l
              );
            },
            unlinkDoc: function (n) {
              if ((n instanceof be && (n = n.doc), this.linked))
                for (var i = 0; i < this.linked.length; ++i) {
                  var a = this.linked[i];
                  if (a.doc == n) {
                    this.linked.splice(i, 1), n.unlinkDoc(this), dx(zp(this));
                    break;
                  }
                }
              if (n.history == this.history) {
                var l = [n.id];
                oi(
                  n,
                  function (c) {
                    return l.push(c.id);
                  },
                  !0,
                ),
                  (n.history = new la(null)),
                  (n.history.done = Ao(this.history.done, l)),
                  (n.history.undone = Ao(this.history.undone, l));
              }
            },
            iterLinkedDocs: function (n) {
              oi(this, n);
            },
            getMode: function () {
              return this.mode;
            },
            getEditor: function () {
              return this.cm;
            },
            splitLines: function (n) {
              return this.lineSep ? n.split(this.lineSep) : Hn(n);
            },
            lineSeparator: function () {
              return (
                this.lineSep ||
                `
`
              );
            },
            setDirection: Be(function (n) {
              n != "rtl" && (n = "ltr"),
                n != this.direction &&
                  ((this.direction = n),
                  this.iter(function (i) {
                    return (i.order = null);
                  }),
                  this.cm && nx(this.cm));
            }),
          })),
            (an.prototype.eachLine = an.prototype.iter);
          var Ip = 0;
          function gx(n) {
            var i = this;
            if ((Fp(i), !(Ce(i, n) || Ar(i.display, n)))) {
              Xe(n), d && (Ip = +new Date());
              var a = $i(i, n, !0),
                l = n.dataTransfer.files;
              if (!(!a || i.isReadOnly()))
                if (l && l.length && window.FileReader && window.File)
                  for (
                    var c = l.length,
                      p = Array(c),
                      m = 0,
                      y = function () {
                        ++m == c &&
                          He(i, function () {
                            a = Wt(i.doc, a);
                            var q = {
                              from: a,
                              to: a,
                              text: i.doc.splitLines(
                                p
                                  .filter(function (Z) {
                                    return Z != null;
                                  })
                                  .join(i.doc.lineSeparator()),
                              ),
                              origin: "paste",
                            };
                            No(i.doc, q), Sp(i.doc, ri(Wt(i.doc, a), Wt(i.doc, ii(q))));
                          })();
                      },
                      x = function (q, Z) {
                        if (
                          i.options.allowDropFileTypes &&
                          Et(i.options.allowDropFileTypes, q.type) == -1
                        ) {
                          y();
                          return;
                        }
                        var it = new FileReader();
                        (it.onerror = function () {
                          return y();
                        }),
                          (it.onload = function () {
                            var vt = it.result;
                            if (/[\x00-\x08\x0e-\x1f]{2}/.test(vt)) {
                              y();
                              return;
                            }
                            (p[Z] = vt), y();
                          }),
                          it.readAsText(q);
                      },
                      _ = 0;
                    _ < l.length;
                    _++
                  )
                    x(l[_], _);
                else {
                  if (i.state.draggingText && i.doc.sel.contains(a) > -1) {
                    i.state.draggingText(n),
                      setTimeout(function () {
                        return i.display.input.focus();
                      }, 20);
                    return;
                  }
                  try {
                    var N = n.dataTransfer.getData("Text");
                    if (N) {
                      var D;
                      if (
                        (i.state.draggingText &&
                          !i.state.draggingText.copy &&
                          (D = i.listSelections()),
                        ua(i.doc, ri(a, a)),
                        D)
                      )
                        for (var W = 0; W < D.length; ++W)
                          Po(i.doc, "", D[W].anchor, D[W].head, "drag");
                      i.replaceSelection(N, "around", "paste"), i.display.input.focus();
                    }
                  } catch {}
                }
            }
          }
          function vx(n, i) {
            if (d && (!n.state.draggingText || +new Date() - Ip < 100)) {
              Yr(i);
              return;
            }
            if (
              !(Ce(n, i) || Ar(n.display, i)) &&
              (i.dataTransfer.setData("Text", n.getSelection()),
              (i.dataTransfer.effectAllowed = "copyMove"),
              i.dataTransfer.setDragImage && !A)
            ) {
              var a = k("img", null, null, "position: fixed; left: 0; top: 0;");
              (a.src =
                "data:image/gif;base64,R0lGODlhAQABAAAAACH5BAEKAAEALAAAAAABAAEAAAICTAEAOw=="),
                P &&
                  ((a.width = a.height = 1),
                  n.display.wrapper.appendChild(a),
                  (a._top = a.offsetTop)),
                i.dataTransfer.setDragImage(a, 0, 0),
                P && a.parentNode.removeChild(a);
            }
          }
          function mx(n, i) {
            var a = $i(n, i);
            if (a) {
              var l = document.createDocumentFragment();
              vu(n, a, l),
                n.display.dragCursor ||
                  ((n.display.dragCursor = k(
                    "div",
                    null,
                    "CodeMirror-cursors CodeMirror-dragcursors",
                  )),
                  n.display.lineSpace.insertBefore(n.display.dragCursor, n.display.cursorDiv)),
                z(n.display.dragCursor, l);
            }
          }
          function Fp(n) {
            n.display.dragCursor &&
              (n.display.lineSpace.removeChild(n.display.dragCursor),
              (n.display.dragCursor = null));
          }
          function qp(n) {
            if (document.getElementsByClassName) {
              for (
                var i = document.getElementsByClassName("CodeMirror"), a = [], l = 0;
                l < i.length;
                l++
              ) {
                var c = i[l].CodeMirror;
                c && a.push(c);
              }
              a.length &&
                a[0].operation(function () {
                  for (var p = 0; p < a.length; p++) n(a[p]);
                });
            }
          }
          var Hp = !1;
          function yx() {
            Hp || (bx(), (Hp = !0));
          }
          function bx() {
            var n;
            Rt(window, "resize", function () {
              n == null &&
                (n = setTimeout(function () {
                  (n = null), qp(wx);
                }, 100));
            }),
              Rt(window, "blur", function () {
                return qp(To);
              });
          }
          function wx(n) {
            var i = n.display;
            (i.cachedCharWidth = i.cachedTextHeight = i.cachedPaddingH = null),
              (i.scrollbarsClipped = !1),
              n.setSize();
          }
          for (
            var li = {
                3: "Pause",
                8: "Backspace",
                9: "Tab",
                13: "Enter",
                16: "Shift",
                17: "Ctrl",
                18: "Alt",
                19: "Pause",
                20: "CapsLock",
                27: "Esc",
                32: "Space",
                33: "PageUp",
                34: "PageDown",
                35: "End",
                36: "Home",
                37: "Left",
                38: "Up",
                39: "Right",
                40: "Down",
                44: "PrintScrn",
                45: "Insert",
                46: "Delete",
                59: ";",
                61: "=",
                91: "Mod",
                92: "Mod",
                93: "Mod",
                106: "*",
                107: "=",
                109: "-",
                110: ".",
                111: "/",
                145: "ScrollLock",
                173: "-",
                186: ";",
                187: "=",
                188: ",",
                189: "-",
                190: ".",
                191: "/",
                192: "`",
                219: "[",
                220: "\\",
                221: "]",
                222: "'",
                224: "Mod",
                63232: "Up",
                63233: "Down",
                63234: "Left",
                63235: "Right",
                63272: "Delete",
                63273: "Home",
                63275: "End",
                63276: "PageUp",
                63277: "PageDown",
                63302: "Insert",
              },
              Us = 0;
            Us < 10;
            Us++
          )
            li[Us + 48] = li[Us + 96] = String(Us);
          for (var da = 65; da <= 90; da++) li[da] = String.fromCharCode(da);
          for (var js = 1; js <= 12; js++) li[js + 111] = li[js + 63235] = "F" + js;
          var Nr = {};
          (Nr.basic = {
            Left: "goCharLeft",
            Right: "goCharRight",
            Up: "goLineUp",
            Down: "goLineDown",
            End: "goLineEnd",
            Home: "goLineStartSmart",
            PageUp: "goPageUp",
            PageDown: "goPageDown",
            Delete: "delCharAfter",
            Backspace: "delCharBefore",
            "Shift-Backspace": "delCharBefore",
            Tab: "defaultTab",
            "Shift-Tab": "indentAuto",
            Enter: "newlineAndIndent",
            Insert: "toggleOverwrite",
            Esc: "singleSelection",
          }),
            (Nr.pcDefault = {
              "Ctrl-A": "selectAll",
              "Ctrl-D": "deleteLine",
              "Ctrl-Z": "undo",
              "Shift-Ctrl-Z": "redo",
              "Ctrl-Y": "redo",
              "Ctrl-Home": "goDocStart",
              "Ctrl-End": "goDocEnd",
              "Ctrl-Up": "goLineUp",
              "Ctrl-Down": "goLineDown",
              "Ctrl-Left": "goGroupLeft",
              "Ctrl-Right": "goGroupRight",
              "Alt-Left": "goLineStart",
              "Alt-Right": "goLineEnd",
              "Ctrl-Backspace": "delGroupBefore",
              "Ctrl-Delete": "delGroupAfter",
              "Ctrl-S": "save",
              "Ctrl-F": "find",
              "Ctrl-G": "findNext",
              "Shift-Ctrl-G": "findPrev",
              "Shift-Ctrl-F": "replace",
              "Shift-Ctrl-R": "replaceAll",
              "Ctrl-[": "indentLess",
              "Ctrl-]": "indentMore",
              "Ctrl-U": "undoSelection",
              "Shift-Ctrl-U": "redoSelection",
              "Alt-U": "redoSelection",
              fallthrough: "basic",
            }),
            (Nr.emacsy = {
              "Ctrl-F": "goCharRight",
              "Ctrl-B": "goCharLeft",
              "Ctrl-P": "goLineUp",
              "Ctrl-N": "goLineDown",
              "Ctrl-A": "goLineStart",
              "Ctrl-E": "goLineEnd",
              "Ctrl-V": "goPageDown",
              "Shift-Ctrl-V": "goPageUp",
              "Ctrl-D": "delCharAfter",
              "Ctrl-H": "delCharBefore",
              "Alt-Backspace": "delWordBefore",
              "Ctrl-K": "killLine",
              "Ctrl-T": "transposeChars",
              "Ctrl-O": "openLine",
            }),
            (Nr.macDefault = {
              "Cmd-A": "selectAll",
              "Cmd-D": "deleteLine",
              "Cmd-Z": "undo",
              "Shift-Cmd-Z": "redo",
              "Cmd-Y": "redo",
              "Cmd-Home": "goDocStart",
              "Cmd-Up": "goDocStart",
              "Cmd-End": "goDocEnd",
              "Cmd-Down": "goDocEnd",
              "Alt-Left": "goGroupLeft",
              "Alt-Right": "goGroupRight",
              "Cmd-Left": "goLineLeft",
              "Cmd-Right": "goLineRight",
              "Alt-Backspace": "delGroupBefore",
              "Ctrl-Alt-Backspace": "delGroupAfter",
              "Alt-Delete": "delGroupAfter",
              "Cmd-S": "save",
              "Cmd-F": "find",
              "Cmd-G": "findNext",
              "Shift-Cmd-G": "findPrev",
              "Cmd-Alt-F": "replace",
              "Shift-Cmd-Alt-F": "replaceAll",
              "Cmd-[": "indentLess",
              "Cmd-]": "indentMore",
              "Cmd-Backspace": "delWrappedLineLeft",
              "Cmd-Delete": "delWrappedLineRight",
              "Cmd-U": "undoSelection",
              "Shift-Cmd-U": "redoSelection",
              "Ctrl-Up": "goDocStart",
              "Ctrl-Down": "goDocEnd",
              fallthrough: ["basic", "emacsy"],
            }),
            (Nr.default = B ? Nr.macDefault : Nr.pcDefault);
          function xx(n) {
            var i = n.split(/-(?!$)/);
            n = i[i.length - 1];
            for (var a, l, c, p, m = 0; m < i.length - 1; m++) {
              var y = i[m];
              if (/^(cmd|meta|m)$/i.test(y)) p = !0;
              else if (/^a(lt)?$/i.test(y)) a = !0;
              else if (/^(c|ctrl|control)$/i.test(y)) l = !0;
              else if (/^s(hift)?$/i.test(y)) c = !0;
              else throw new Error("Unrecognized modifier name: " + y);
            }
            return (
              a && (n = "Alt-" + n),
              l && (n = "Ctrl-" + n),
              p && (n = "Cmd-" + n),
              c && (n = "Shift-" + n),
              n
            );
          }
          function _x(n) {
            var i = {};
            for (var a in n)
              if (n.hasOwnProperty(a)) {
                var l = n[a];
                if (/^(name|fallthrough|(de|at)tach)$/.test(a)) continue;
                if (l == "...") {
                  delete n[a];
                  continue;
                }
                for (var c = ft(a.split(" "), xx), p = 0; p < c.length; p++) {
                  var m = void 0,
                    y = void 0;
                  p == c.length - 1
                    ? ((y = c.join(" ")), (m = l))
                    : ((y = c.slice(0, p + 1).join(" ")), (m = "..."));
                  var x = i[y];
                  if (!x) i[y] = m;
                  else if (x != m) throw new Error("Inconsistent bindings for " + y);
                }
                delete n[a];
              }
            for (var _ in i) n[_] = i[_];
            return n;
          }
          function Do(n, i, a, l) {
            i = pa(i);
            var c = i.call ? i.call(n, l) : i[n];
            if (c === !1) return "nothing";
            if (c === "...") return "multi";
            if (c != null && a(c)) return "handled";
            if (i.fallthrough) {
              if (Object.prototype.toString.call(i.fallthrough) != "[object Array]")
                return Do(n, i.fallthrough, a, l);
              for (var p = 0; p < i.fallthrough.length; p++) {
                var m = Do(n, i.fallthrough[p], a, l);
                if (m) return m;
              }
            }
          }
          function Bp(n) {
            var i = typeof n == "string" ? n : li[n.keyCode];
            return i == "Ctrl" || i == "Alt" || i == "Shift" || i == "Mod";
          }
          function Wp(n, i, a) {
            var l = n;
            return (
              i.altKey && l != "Alt" && (n = "Alt-" + n),
              (nt ? i.metaKey : i.ctrlKey) && l != "Ctrl" && (n = "Ctrl-" + n),
              (nt ? i.ctrlKey : i.metaKey) && l != "Mod" && (n = "Cmd-" + n),
              !a && i.shiftKey && l != "Shift" && (n = "Shift-" + n),
              n
            );
          }
          function Up(n, i) {
            if (P && n.keyCode == 34 && n.char) return !1;
            var a = li[n.keyCode];
            return a == null || n.altGraphKey
              ? !1
              : (n.keyCode == 3 && n.code && (a = n.code), Wp(a, n, i));
          }
          function pa(n) {
            return typeof n == "string" ? Nr[n] : n;
          }
          function $o(n, i) {
            for (var a = n.doc.sel.ranges, l = [], c = 0; c < a.length; c++) {
              for (var p = i(a[c]); l.length && _t(p.from, ct(l).to) <= 0; ) {
                var m = l.pop();
                if (_t(m.from, p.from) < 0) {
                  p.from = m.from;
                  break;
                }
              }
              l.push(p);
            }
            Sn(n, function () {
              for (var y = l.length - 1; y >= 0; y--) Po(n.doc, "", l[y].from, l[y].to, "+delete");
              Eo(n);
            });
          }
          function Ou(n, i, a) {
            var l = rn(n.text, i + a, a);
            return l < 0 || l > n.text.length ? null : l;
          }
          function Du(n, i, a) {
            var l = Ou(n, i.ch, a);
            return l == null ? null : new X(i.line, l, a < 0 ? "after" : "before");
          }
          function $u(n, i, a, l, c) {
            if (n) {
              i.doc.direction == "rtl" && (c = -c);
              var p = Yt(a, i.doc.direction);
              if (p) {
                var m = c < 0 ? ct(p) : p[0],
                  y = c < 0 == (m.level == 1),
                  x = y ? "after" : "before",
                  _;
                if (m.level > 0 || i.doc.direction == "rtl") {
                  var N = So(i, a);
                  _ = c < 0 ? a.text.length - 1 : 0;
                  var D = dr(i, N, _).top;
                  (_ = Pn(
                    function (W) {
                      return dr(i, N, W).top == D;
                    },
                    c < 0 == (m.level == 1) ? m.from : m.to - 1,
                    _,
                  )),
                    x == "before" && (_ = Ou(a, _, 1));
                } else _ = c < 0 ? m.to : m.from;
                return new X(l, _, x);
              }
            }
            return new X(l, c < 0 ? a.text.length : 0, c < 0 ? "before" : "after");
          }
          function Sx(n, i, a, l) {
            var c = Yt(i, n.doc.direction);
            if (!c) return Du(i, a, l);
            a.ch >= i.text.length
              ? ((a.ch = i.text.length), (a.sticky = "before"))
              : a.ch <= 0 && ((a.ch = 0), (a.sticky = "after"));
            var p = Ae(c, a.ch, a.sticky),
              m = c[p];
            if (
              n.doc.direction == "ltr" &&
              m.level % 2 == 0 &&
              (l > 0 ? m.to > a.ch : m.from < a.ch)
            )
              return Du(i, a, l);
            var y = function (bt, Ct) {
                return Ou(i, bt instanceof X ? bt.ch : bt, Ct);
              },
              x,
              _ = function (bt) {
                return n.options.lineWrapping
                  ? ((x = x || So(n, i)), Xd(n, i, x, bt))
                  : { begin: 0, end: i.text.length };
              },
              N = _(a.sticky == "before" ? y(a, -1) : a.ch);
            if (n.doc.direction == "rtl" || m.level == 1) {
              var D = (m.level == 1) == l < 0,
                W = y(a, D ? 1 : -1);
              if (W != null && (D ? W <= m.to && W <= N.end : W >= m.from && W >= N.begin)) {
                var q = D ? "before" : "after";
                return new X(a.line, W, q);
              }
            }
            var Z = function (bt, Ct, wt) {
                for (
                  var Lt = function (ge, We) {
                    return We ? new X(a.line, y(ge, 1), "before") : new X(a.line, ge, "after");
                  };
                  bt >= 0 && bt < c.length;
                  bt += Ct
                ) {
                  var zt = c[bt],
                    Ot = Ct > 0 == (zt.level != 1),
                    Xt = Ot ? wt.begin : y(wt.end, -1);
                  if (
                    (zt.from <= Xt && Xt < zt.to) ||
                    ((Xt = Ot ? zt.from : y(zt.to, -1)), wt.begin <= Xt && Xt < wt.end)
                  )
                    return Lt(Xt, Ot);
                }
              },
              it = Z(p + l, l, N);
            if (it) return it;
            var vt = l > 0 ? N.end : y(N.begin, -1);
            return vt != null &&
              !(l > 0 && vt == i.text.length) &&
              ((it = Z(l > 0 ? 0 : c.length - 1, l, _(vt))), it)
              ? it
              : null;
          }
          var Vs = {
            selectAll: Lp,
            singleSelection: function (n) {
              return n.setSelection(n.getCursor("anchor"), n.getCursor("head"), V);
            },
            killLine: function (n) {
              return $o(n, function (i) {
                if (i.empty()) {
                  var a = Pt(n.doc, i.head.line).text.length;
                  return i.head.ch == a && i.head.line < n.lastLine()
                    ? { from: i.head, to: X(i.head.line + 1, 0) }
                    : { from: i.head, to: X(i.head.line, a) };
                } else return { from: i.from(), to: i.to() };
              });
            },
            deleteLine: function (n) {
              return $o(n, function (i) {
                return { from: X(i.from().line, 0), to: Wt(n.doc, X(i.to().line + 1, 0)) };
              });
            },
            delLineLeft: function (n) {
              return $o(n, function (i) {
                return { from: X(i.from().line, 0), to: i.from() };
              });
            },
            delWrappedLineLeft: function (n) {
              return $o(n, function (i) {
                var a = n.charCoords(i.head, "div").top + 5,
                  l = n.coordsChar({ left: 0, top: a }, "div");
                return { from: l, to: i.from() };
              });
            },
            delWrappedLineRight: function (n) {
              return $o(n, function (i) {
                var a = n.charCoords(i.head, "div").top + 5,
                  l = n.coordsChar({ left: n.display.lineDiv.offsetWidth + 100, top: a }, "div");
                return { from: i.from(), to: l };
              });
            },
            undo: function (n) {
              return n.undo();
            },
            redo: function (n) {
              return n.redo();
            },
            undoSelection: function (n) {
              return n.undoSelection();
            },
            redoSelection: function (n) {
              return n.redoSelection();
            },
            goDocStart: function (n) {
              return n.extendSelection(X(n.firstLine(), 0));
            },
            goDocEnd: function (n) {
              return n.extendSelection(X(n.lastLine()));
            },
            goLineStart: function (n) {
              return n.extendSelectionsBy(
                function (i) {
                  return jp(n, i.head.line);
                },
                { origin: "+move", bias: 1 },
              );
            },
            goLineStartSmart: function (n) {
              return n.extendSelectionsBy(
                function (i) {
                  return Vp(n, i.head);
                },
                { origin: "+move", bias: 1 },
              );
            },
            goLineEnd: function (n) {
              return n.extendSelectionsBy(
                function (i) {
                  return kx(n, i.head.line);
                },
                { origin: "+move", bias: -1 },
              );
            },
            goLineRight: function (n) {
              return n.extendSelectionsBy(function (i) {
                var a = n.cursorCoords(i.head, "div").top + 5;
                return n.coordsChar({ left: n.display.lineDiv.offsetWidth + 100, top: a }, "div");
              }, ot);
            },
            goLineLeft: function (n) {
              return n.extendSelectionsBy(function (i) {
                var a = n.cursorCoords(i.head, "div").top + 5;
                return n.coordsChar({ left: 0, top: a }, "div");
              }, ot);
            },
            goLineLeftSmart: function (n) {
              return n.extendSelectionsBy(function (i) {
                var a = n.cursorCoords(i.head, "div").top + 5,
                  l = n.coordsChar({ left: 0, top: a }, "div");
                return l.ch < n.getLine(l.line).search(/\S/) ? Vp(n, i.head) : l;
              }, ot);
            },
            goLineUp: function (n) {
              return n.moveV(-1, "line");
            },
            goLineDown: function (n) {
              return n.moveV(1, "line");
            },
            goPageUp: function (n) {
              return n.moveV(-1, "page");
            },
            goPageDown: function (n) {
              return n.moveV(1, "page");
            },
            goCharLeft: function (n) {
              return n.moveH(-1, "char");
            },
            goCharRight: function (n) {
              return n.moveH(1, "char");
            },
            goColumnLeft: function (n) {
              return n.moveH(-1, "column");
            },
            goColumnRight: function (n) {
              return n.moveH(1, "column");
            },
            goWordLeft: function (n) {
              return n.moveH(-1, "word");
            },
            goGroupRight: function (n) {
              return n.moveH(1, "group");
            },
            goGroupLeft: function (n) {
              return n.moveH(-1, "group");
            },
            goWordRight: function (n) {
              return n.moveH(1, "word");
            },
            delCharBefore: function (n) {
              return n.deleteH(-1, "codepoint");
            },
            delCharAfter: function (n) {
              return n.deleteH(1, "char");
            },
            delWordBefore: function (n) {
              return n.deleteH(-1, "word");
            },
            delWordAfter: function (n) {
              return n.deleteH(1, "word");
            },
            delGroupBefore: function (n) {
              return n.deleteH(-1, "group");
            },
            delGroupAfter: function (n) {
              return n.deleteH(1, "group");
            },
            indentAuto: function (n) {
              return n.indentSelection("smart");
            },
            indentMore: function (n) {
              return n.indentSelection("add");
            },
            indentLess: function (n) {
              return n.indentSelection("subtract");
            },
            insertTab: function (n) {
              return n.replaceSelection("	");
            },
            insertSoftTab: function (n) {
              for (
                var i = [], a = n.listSelections(), l = n.options.tabSize, c = 0;
                c < a.length;
                c++
              ) {
                var p = a[c].from(),
                  m = lt(n.getLine(p.line), p.ch, l);
                i.push(mt(l - (m % l)));
              }
              n.replaceSelections(i);
            },
            defaultTab: function (n) {
              n.somethingSelected() ? n.indentSelection("add") : n.execCommand("insertTab");
            },
            transposeChars: function (n) {
              return Sn(n, function () {
                for (var i = n.listSelections(), a = [], l = 0; l < i.length; l++)
                  if (i[l].empty()) {
                    var c = i[l].head,
                      p = Pt(n.doc, c.line).text;
                    if (p) {
                      if ((c.ch == p.length && (c = new X(c.line, c.ch - 1)), c.ch > 0))
                        (c = new X(c.line, c.ch + 1)),
                          n.replaceRange(
                            p.charAt(c.ch - 1) + p.charAt(c.ch - 2),
                            X(c.line, c.ch - 2),
                            c,
                            "+transpose",
                          );
                      else if (c.line > n.doc.first) {
                        var m = Pt(n.doc, c.line - 1).text;
                        m &&
                          ((c = new X(c.line, 1)),
                          n.replaceRange(
                            p.charAt(0) + n.doc.lineSeparator() + m.charAt(m.length - 1),
                            X(c.line - 1, m.length - 1),
                            c,
                            "+transpose",
                          ));
                      }
                    }
                    a.push(new ue(c, c));
                  }
                n.setSelections(a);
              });
            },
            newlineAndIndent: function (n) {
              return Sn(n, function () {
                for (var i = n.listSelections(), a = i.length - 1; a >= 0; a--)
                  n.replaceRange(n.doc.lineSeparator(), i[a].anchor, i[a].head, "+input");
                i = n.listSelections();
                for (var l = 0; l < i.length; l++) n.indentLine(i[l].from().line, null, !0);
                Eo(n);
              });
            },
            openLine: function (n) {
              return n.replaceSelection(
                `
`,
                "start",
              );
            },
            toggleOverwrite: function (n) {
              return n.toggleOverwrite();
            },
          };
          function jp(n, i) {
            var a = Pt(n.doc, i),
              l = Zn(a);
            return l != a && (i = C(l)), $u(!0, n, l, i, 1);
          }
          function kx(n, i) {
            var a = Pt(n.doc, i),
              l = lw(a);
            return l != a && (i = C(l)), $u(!0, n, a, i, -1);
          }
          function Vp(n, i) {
            var a = jp(n, i.line),
              l = Pt(n.doc, a.line),
              c = Yt(l, n.doc.direction);
            if (!c || c[0].level == 0) {
              var p = Math.max(a.ch, l.text.search(/\S/)),
                m = i.line == a.line && i.ch <= p && i.ch;
              return X(a.line, m ? 0 : p, a.sticky);
            }
            return a;
          }
          function ga(n, i, a) {
            if (typeof i == "string" && ((i = Vs[i]), !i)) return !1;
            n.display.input.ensurePolled();
            var l = n.display.shift,
              c = !1;
            try {
              n.isReadOnly() && (n.state.suppressEdits = !0),
                a && (n.display.shift = !1),
                (c = i(n) != I);
            } finally {
              (n.display.shift = l), (n.state.suppressEdits = !1);
            }
            return c;
          }
          function Cx(n, i, a) {
            for (var l = 0; l < n.state.keyMaps.length; l++) {
              var c = Do(i, n.state.keyMaps[l], a, n);
              if (c) return c;
            }
            return (
              (n.options.extraKeys && Do(i, n.options.extraKeys, a, n)) ||
              Do(i, n.options.keyMap, a, n)
            );
          }
          var Tx = new Mt();
          function Gs(n, i, a, l) {
            var c = n.state.keySeq;
            if (c) {
              if (Bp(i)) return "handled";
              if (
                (/\'$/.test(i)
                  ? (n.state.keySeq = null)
                  : Tx.set(50, function () {
                      n.state.keySeq == c && ((n.state.keySeq = null), n.display.input.reset());
                    }),
                Gp(n, c + " " + i, a, l))
              )
                return !0;
            }
            return Gp(n, i, a, l);
          }
          function Gp(n, i, a, l) {
            var c = Cx(n, i, l);
            return (
              c == "multi" && (n.state.keySeq = i),
              c == "handled" && qe(n, "keyHandled", n, i, a),
              (c == "handled" || c == "multi") && (Xe(a), mu(n)),
              !!c
            );
          }
          function Kp(n, i) {
            var a = Up(i, !0);
            return a
              ? i.shiftKey && !n.state.keySeq
                ? Gs(n, "Shift-" + a, i, function (l) {
                    return ga(n, l, !0);
                  }) ||
                  Gs(n, a, i, function (l) {
                    if (typeof l == "string" ? /^go[A-Z]/.test(l) : l.motion) return ga(n, l);
                  })
                : Gs(n, a, i, function (l) {
                    return ga(n, l);
                  })
              : !1;
          }
          function Ex(n, i, a) {
            return Gs(n, "'" + a + "'", i, function (l) {
              return ga(n, l, !0);
            });
          }
          var Ru = null;
          function Xp(n) {
            var i = this;
            if (
              !(n.target && n.target != i.display.input.getField()) &&
              ((i.curOp.focus = yt(Jt(i))), !Ce(i, n))
            ) {
              d && g < 11 && n.keyCode == 27 && (n.returnValue = !1);
              var a = n.keyCode;
              i.display.shift = a == 16 || n.shiftKey;
              var l = Kp(i, n);
              P &&
                ((Ru = l ? a : null),
                !l &&
                  a == 88 &&
                  !Wl &&
                  (B ? n.metaKey : n.ctrlKey) &&
                  i.replaceSelection("", null, "cut")),
                s &&
                  !B &&
                  !l &&
                  a == 46 &&
                  n.shiftKey &&
                  !n.ctrlKey &&
                  document.execCommand &&
                  document.execCommand("cut"),
                a == 18 && !/\bCodeMirror-crosshair\b/.test(i.display.lineDiv.className) && Lx(i);
            }
          }
          function Lx(n) {
            var i = n.display.lineDiv;
            At(i, "CodeMirror-crosshair");
            function a(l) {
              (l.keyCode == 18 || !l.altKey) &&
                (gt(i, "CodeMirror-crosshair"),
                Ke(document, "keyup", a),
                Ke(document, "mouseover", a));
            }
            Rt(document, "keyup", a), Rt(document, "mouseover", a);
          }
          function Yp(n) {
            n.keyCode == 16 && (this.doc.sel.shift = !1), Ce(this, n);
          }
          function Zp(n) {
            var i = this;
            if (
              !(n.target && n.target != i.display.input.getField()) &&
              !(Ar(i.display, n) || Ce(i, n) || (n.ctrlKey && !n.altKey) || (B && n.metaKey))
            ) {
              var a = n.keyCode,
                l = n.charCode;
              if (P && a == Ru) {
                (Ru = null), Xe(n);
                return;
              }
              if (!(P && (!n.which || n.which < 10) && Kp(i, n))) {
                var c = String.fromCharCode(l ?? a);
                c != "\b" && (Ex(i, n, c) || i.display.input.onKeyPress(n));
              }
            }
          }
          var Ax = 400,
            zu = function (n, i, a) {
              (this.time = n), (this.pos = i), (this.button = a);
            };
          zu.prototype.compare = function (n, i, a) {
            return this.time + Ax > n && _t(i, this.pos) == 0 && a == this.button;
          };
          var Ks, Xs;
          function Mx(n, i) {
            var a = +new Date();
            return Xs && Xs.compare(a, n, i)
              ? ((Ks = Xs = null), "triple")
              : Ks && Ks.compare(a, n, i)
              ? ((Xs = new zu(a, n, i)), (Ks = null), "double")
              : ((Ks = new zu(a, n, i)), (Xs = null), "single");
          }
          function Jp(n) {
            var i = this,
              a = i.display;
            if (!(Ce(i, n) || (a.activeTouch && a.input.supportsTouch()))) {
              if ((a.input.ensurePolled(), (a.shift = n.shiftKey), Ar(a, n))) {
                v ||
                  ((a.scroller.draggable = !1),
                  setTimeout(function () {
                    return (a.scroller.draggable = !0);
                  }, 100));
                return;
              }
              if (!Iu(i, n)) {
                var l = $i(i, n),
                  c = Gn(n),
                  p = l ? Mx(l, c) : "single";
                Tt(i).focus(),
                  c == 1 && i.state.selectingText && i.state.selectingText(n),
                  !(l && Nx(i, c, l, p, n)) &&
                    (c == 1
                      ? l
                        ? Ox(i, l, p, n)
                        : ws(n) == a.scroller && Xe(n)
                      : c == 2
                      ? (l && ca(i.doc, l),
                        setTimeout(function () {
                          return a.input.focus();
                        }, 20))
                      : c == 3 && (at ? i.display.input.onContextMenu(n) : yu(i)));
              }
            }
          }
          function Nx(n, i, a, l, c) {
            var p = "Click";
            return (
              l == "double" ? (p = "Double" + p) : l == "triple" && (p = "Triple" + p),
              (p = (i == 1 ? "Left" : i == 2 ? "Middle" : "Right") + p),
              Gs(n, Wp(p, c), c, function (m) {
                if ((typeof m == "string" && (m = Vs[m]), !m)) return !1;
                var y = !1;
                try {
                  n.isReadOnly() && (n.state.suppressEdits = !0), (y = m(n, a) != I);
                } finally {
                  n.state.suppressEdits = !1;
                }
                return y;
              })
            );
          }
          function Px(n, i, a) {
            var l = n.getOption("configureMouse"),
              c = l ? l(n, i, a) : {};
            if (c.unit == null) {
              var p = K ? a.shiftKey && a.metaKey : a.altKey;
              c.unit = p ? "rectangle" : i == "single" ? "char" : i == "double" ? "word" : "line";
            }
            return (
              (c.extend == null || n.doc.extend) && (c.extend = n.doc.extend || a.shiftKey),
              c.addNew == null && (c.addNew = B ? a.metaKey : a.ctrlKey),
              c.moveOnDrag == null && (c.moveOnDrag = !(B ? a.altKey : a.ctrlKey)),
              c
            );
          }
          function Ox(n, i, a, l) {
            d ? setTimeout(j(Qd, n), 0) : (n.curOp.focus = yt(Jt(n)));
            var c = Px(n, a, l),
              p = n.doc.sel,
              m;
            n.options.dragDrop &&
            Yc &&
            !n.isReadOnly() &&
            a == "single" &&
            (m = p.contains(i)) > -1 &&
            (_t((m = p.ranges[m]).from(), i) < 0 || i.xRel > 0) &&
            (_t(m.to(), i) > 0 || i.xRel < 0)
              ? Dx(n, l, i, c)
              : $x(n, l, i, c);
          }
          function Dx(n, i, a, l) {
            var c = n.display,
              p = !1,
              m = He(n, function (_) {
                v && (c.scroller.draggable = !1),
                  (n.state.draggingText = !1),
                  n.state.delayingBlurEvent &&
                    (n.hasFocus() ? (n.state.delayingBlurEvent = !1) : yu(n)),
                  Ke(c.wrapper.ownerDocument, "mouseup", m),
                  Ke(c.wrapper.ownerDocument, "mousemove", y),
                  Ke(c.scroller, "dragstart", x),
                  Ke(c.scroller, "drop", m),
                  p ||
                    (Xe(_),
                    l.addNew || ca(n.doc, a, null, null, l.extend),
                    (v && !A) || (d && g == 9)
                      ? setTimeout(function () {
                          c.wrapper.ownerDocument.body.focus({ preventScroll: !0 }),
                            c.input.focus();
                        }, 20)
                      : c.input.focus());
              }),
              y = function (_) {
                p = p || Math.abs(i.clientX - _.clientX) + Math.abs(i.clientY - _.clientY) >= 10;
              },
              x = function () {
                return (p = !0);
              };
            v && (c.scroller.draggable = !0),
              (n.state.draggingText = m),
              (m.copy = !l.moveOnDrag),
              Rt(c.wrapper.ownerDocument, "mouseup", m),
              Rt(c.wrapper.ownerDocument, "mousemove", y),
              Rt(c.scroller, "dragstart", x),
              Rt(c.scroller, "drop", m),
              (n.state.delayingBlurEvent = !0),
              setTimeout(function () {
                return c.input.focus();
              }, 20),
              c.scroller.dragDrop && c.scroller.dragDrop();
          }
          function Qp(n, i, a) {
            if (a == "char") return new ue(i, i);
            if (a == "word") return n.findWordAt(i);
            if (a == "line") return new ue(X(i.line, 0), Wt(n.doc, X(i.line + 1, 0)));
            var l = a(n, i);
            return new ue(l.from, l.to);
          }
          function $x(n, i, a, l) {
            d && yu(n);
            var c = n.display,
              p = n.doc;
            Xe(i);
            var m,
              y,
              x = p.sel,
              _ = x.ranges;
            if (
              (l.addNew && !l.extend
                ? ((y = p.sel.contains(a)), y > -1 ? (m = _[y]) : (m = new ue(a, a)))
                : ((m = p.sel.primary()), (y = p.sel.primIndex)),
              l.unit == "rectangle")
            )
              l.addNew || (m = new ue(a, a)), (a = $i(n, i, !0, !0)), (y = -1);
            else {
              var N = Qp(n, a, l.unit);
              l.extend ? (m = Nu(m, N.anchor, N.head, l.extend)) : (m = N);
            }
            l.addNew
              ? y == -1
                ? ((y = _.length), Ye(p, Qn(n, _.concat([m]), y), { scroll: !1, origin: "*mouse" }))
                : _.length > 1 && _[y].empty() && l.unit == "char" && !l.extend
                ? (Ye(p, Qn(n, _.slice(0, y).concat(_.slice(y + 1)), 0), {
                    scroll: !1,
                    origin: "*mouse",
                  }),
                  (x = p.sel))
                : Pu(p, y, m, Q)
              : ((y = 0), Ye(p, new Dn([m], 0), Q), (x = p.sel));
            var D = a;
            function W(wt) {
              if (_t(D, wt) != 0)
                if (((D = wt), l.unit == "rectangle")) {
                  for (
                    var Lt = [],
                      zt = n.options.tabSize,
                      Ot = lt(Pt(p, a.line).text, a.ch, zt),
                      Xt = lt(Pt(p, wt.line).text, wt.ch, zt),
                      ge = Math.min(Ot, Xt),
                      We = Math.max(Ot, Xt),
                      _e = Math.min(a.line, wt.line),
                      kn = Math.min(n.lastLine(), Math.max(a.line, wt.line));
                    _e <= kn;
                    _e++
                  ) {
                    var cn = Pt(p, _e).text,
                      Ne = ut(cn, ge, zt);
                    ge == We
                      ? Lt.push(new ue(X(_e, Ne), X(_e, Ne)))
                      : cn.length > Ne && Lt.push(new ue(X(_e, Ne), X(_e, ut(cn, We, zt))));
                  }
                  Lt.length || Lt.push(new ue(a, a)),
                    Ye(p, Qn(n, x.ranges.slice(0, y).concat(Lt), y), {
                      origin: "*mouse",
                      scroll: !1,
                    }),
                    n.scrollIntoView(wt);
                } else {
                  var un = m,
                    Ve = Qp(n, wt, l.unit),
                    $e = un.anchor,
                    Pe;
                  _t(Ve.anchor, $e) > 0
                    ? ((Pe = Ve.head), ($e = wo(un.from(), Ve.anchor)))
                    : ((Pe = Ve.anchor), ($e = sn(un.to(), Ve.head)));
                  var Ee = x.ranges.slice(0);
                  (Ee[y] = Rx(n, new ue(Wt(p, $e), Pe))), Ye(p, Qn(n, Ee, y), Q);
                }
            }
            var q = c.wrapper.getBoundingClientRect(),
              Z = 0;
            function it(wt) {
              var Lt = ++Z,
                zt = $i(n, wt, !0, l.unit == "rectangle");
              if (zt)
                if (_t(zt, D) != 0) {
                  (n.curOp.focus = yt(Jt(n))), W(zt);
                  var Ot = ra(c, p);
                  (zt.line >= Ot.to || zt.line < Ot.from) &&
                    setTimeout(
                      He(n, function () {
                        Z == Lt && it(wt);
                      }),
                      150,
                    );
                } else {
                  var Xt = wt.clientY < q.top ? -20 : wt.clientY > q.bottom ? 20 : 0;
                  Xt &&
                    setTimeout(
                      He(n, function () {
                        Z == Lt && ((c.scroller.scrollTop += Xt), it(wt));
                      }),
                      50,
                    );
                }
            }
            function vt(wt) {
              (n.state.selectingText = !1),
                (Z = 1 / 0),
                wt && (Xe(wt), c.input.focus()),
                Ke(c.wrapper.ownerDocument, "mousemove", bt),
                Ke(c.wrapper.ownerDocument, "mouseup", Ct),
                (p.history.lastSelOrigin = null);
            }
            var bt = He(n, function (wt) {
                wt.buttons === 0 || !Gn(wt) ? vt(wt) : it(wt);
              }),
              Ct = He(n, vt);
            (n.state.selectingText = Ct),
              Rt(c.wrapper.ownerDocument, "mousemove", bt),
              Rt(c.wrapper.ownerDocument, "mouseup", Ct);
          }
          function Rx(n, i) {
            var a = i.anchor,
              l = i.head,
              c = Pt(n.doc, a.line);
            if (_t(a, l) == 0 && a.sticky == l.sticky) return i;
            var p = Yt(c);
            if (!p) return i;
            var m = Ae(p, a.ch, a.sticky),
              y = p[m];
            if (y.from != a.ch && y.to != a.ch) return i;
            var x = m + ((y.from == a.ch) == (y.level != 1) ? 0 : 1);
            if (x == 0 || x == p.length) return i;
            var _;
            if (l.line != a.line) _ = (l.line - a.line) * (n.doc.direction == "ltr" ? 1 : -1) > 0;
            else {
              var N = Ae(p, l.ch, l.sticky),
                D = N - m || (l.ch - a.ch) * (y.level == 1 ? -1 : 1);
              N == x - 1 || N == x ? (_ = D < 0) : (_ = D > 0);
            }
            var W = p[x + (_ ? -1 : 0)],
              q = _ == (W.level == 1),
              Z = q ? W.from : W.to,
              it = q ? "after" : "before";
            return a.ch == Z && a.sticky == it ? i : new ue(new X(a.line, Z, it), l);
          }
          function tg(n, i, a, l) {
            var c, p;
            if (i.touches) (c = i.touches[0].clientX), (p = i.touches[0].clientY);
            else
              try {
                (c = i.clientX), (p = i.clientY);
              } catch {
                return !1;
              }
            if (c >= Math.floor(n.display.gutters.getBoundingClientRect().right)) return !1;
            l && Xe(i);
            var m = n.display,
              y = m.lineDiv.getBoundingClientRect();
            if (p > y.bottom || !_n(n, a)) return on(i);
            p -= y.top - m.viewOffset;
            for (var x = 0; x < n.display.gutterSpecs.length; ++x) {
              var _ = m.gutters.childNodes[x];
              if (_ && _.getBoundingClientRect().right >= c) {
                var N = O(n.doc, p),
                  D = n.display.gutterSpecs[x];
                return ke(n, a, n, N, D.className, i), on(i);
              }
            }
          }
          function Iu(n, i) {
            return tg(n, i, "gutterClick", !0);
          }
          function eg(n, i) {
            Ar(n.display, i) ||
              zx(n, i) ||
              Ce(n, i, "contextmenu") ||
              at ||
              n.display.input.onContextMenu(i);
          }
          function zx(n, i) {
            return _n(n, "gutterContextMenu") ? tg(n, i, "gutterContextMenu", !1) : !1;
          }
          function ng(n) {
            (n.display.wrapper.className =
              n.display.wrapper.className.replace(/\s*cm-s-\S+/g, "") +
              n.options.theme.replace(/(^|\s)\s*/g, " cm-s-")),
              As(n);
          }
          var Ro = {
              toString: function () {
                return "CodeMirror.Init";
              },
            },
            rg = {},
            va = {};
          function Ix(n) {
            var i = n.optionHandlers;
            function a(l, c, p, m) {
              (n.defaults[l] = c),
                p &&
                  (i[l] = m
                    ? function (y, x, _) {
                        _ != Ro && p(y, x, _);
                      }
                    : p);
            }
            (n.defineOption = a),
              (n.Init = Ro),
              a(
                "value",
                "",
                function (l, c) {
                  return l.setValue(c);
                },
                !0,
              ),
              a(
                "mode",
                null,
                function (l, c) {
                  (l.doc.modeOption = c), Lu(l);
                },
                !0,
              ),
              a("indentUnit", 2, Lu, !0),
              a("indentWithTabs", !1),
              a("smartIndent", !0),
              a(
                "tabSize",
                4,
                function (l) {
                  zs(l), As(l), ln(l);
                },
                !0,
              ),
              a("lineSeparator", null, function (l, c) {
                if (((l.doc.lineSep = c), !!c)) {
                  var p = [],
                    m = l.doc.first;
                  l.doc.iter(function (x) {
                    for (var _ = 0; ; ) {
                      var N = x.text.indexOf(c, _);
                      if (N == -1) break;
                      (_ = N + c.length), p.push(X(m, N));
                    }
                    m++;
                  });
                  for (var y = p.length - 1; y >= 0; y--)
                    Po(l.doc, c, p[y], X(p[y].line, p[y].ch + c.length));
                }
              }),
              a(
                "specialChars",
                /[\u0000-\u001f\u007f-\u009f\u00ad\u061c\u200b\u200e\u200f\u2028\u2029\u202d\u202e\u2066\u2067\u2069\ufeff\ufff9-\ufffc]/g,
                function (l, c, p) {
                  (l.state.specialChars = new RegExp(c.source + (c.test("	") ? "" : "|	"), "g")),
                    p != Ro && l.refresh();
                },
              ),
              a(
                "specialCharPlaceholder",
                dw,
                function (l) {
                  return l.refresh();
                },
                !0,
              ),
              a("electricChars", !0),
              a(
                "inputStyle",
                E ? "contenteditable" : "textarea",
                function () {
                  throw new Error("inputStyle can not (yet) be changed in a running editor");
                },
                !0,
              ),
              a(
                "spellcheck",
                !1,
                function (l, c) {
                  return (l.getInputField().spellcheck = c);
                },
                !0,
              ),
              a(
                "autocorrect",
                !1,
                function (l, c) {
                  return (l.getInputField().autocorrect = c);
                },
                !0,
              ),
              a(
                "autocapitalize",
                !1,
                function (l, c) {
                  return (l.getInputField().autocapitalize = c);
                },
                !0,
              ),
              a("rtlMoveVisually", !ht),
              a("wholeLineUpdateBefore", !0),
              a(
                "theme",
                "default",
                function (l) {
                  ng(l), Rs(l);
                },
                !0,
              ),
              a("keyMap", "default", function (l, c, p) {
                var m = pa(c),
                  y = p != Ro && pa(p);
                y && y.detach && y.detach(l, m), m.attach && m.attach(l, y || null);
              }),
              a("extraKeys", null),
              a("configureMouse", null),
              a("lineWrapping", !1, qx, !0),
              a(
                "gutters",
                [],
                function (l, c) {
                  (l.display.gutterSpecs = Tu(c, l.options.lineNumbers)), Rs(l);
                },
                !0,
              ),
              a(
                "fixedGutter",
                !0,
                function (l, c) {
                  (l.display.gutters.style.left = c ? pu(l.display) + "px" : "0"), l.refresh();
                },
                !0,
              ),
              a(
                "coverGutterNextToScrollbar",
                !1,
                function (l) {
                  return Lo(l);
                },
                !0,
              ),
              a(
                "scrollbarStyle",
                "native",
                function (l) {
                  op(l),
                    Lo(l),
                    l.display.scrollbars.setScrollTop(l.doc.scrollTop),
                    l.display.scrollbars.setScrollLeft(l.doc.scrollLeft);
                },
                !0,
              ),
              a(
                "lineNumbers",
                !1,
                function (l, c) {
                  (l.display.gutterSpecs = Tu(l.options.gutters, c)), Rs(l);
                },
                !0,
              ),
              a("firstLineNumber", 1, Rs, !0),
              a(
                "lineNumberFormatter",
                function (l) {
                  return l;
                },
                Rs,
                !0,
              ),
              a("showCursorWhenSelecting", !1, Ms, !0),
              a("resetSelectionOnContextMenu", !0),
              a("lineWiseCopyCut", !0),
              a("pasteLinesPerSelection", !0),
              a("selectionsMayTouch", !1),
              a("readOnly", !1, function (l, c) {
                c == "nocursor" && (To(l), l.display.input.blur()),
                  l.display.input.readOnlyChanged(c);
              }),
              a("screenReaderLabel", null, function (l, c) {
                (c = c === "" ? null : c), l.display.input.screenReaderLabelChanged(c);
              }),
              a(
                "disableInput",
                !1,
                function (l, c) {
                  c || l.display.input.reset();
                },
                !0,
              ),
              a("dragDrop", !0, Fx),
              a("allowDropFileTypes", null),
              a("cursorBlinkRate", 530),
              a("cursorScrollMargin", 0),
              a("cursorHeight", 1, Ms, !0),
              a("singleCursorHeightPerLine", !0, Ms, !0),
              a("workTime", 100),
              a("workDelay", 100),
              a("flattenSpans", !0, zs, !0),
              a("addModeClass", !1, zs, !0),
              a("pollInterval", 100),
              a("undoDepth", 200, function (l, c) {
                return (l.doc.history.undoDepth = c);
              }),
              a("historyEventDelay", 1250),
              a(
                "viewportMargin",
                10,
                function (l) {
                  return l.refresh();
                },
                !0,
              ),
              a("maxHighlightLength", 1e4, zs, !0),
              a("moveInputWithCursor", !0, function (l, c) {
                c || l.display.input.resetPosition();
              }),
              a("tabindex", null, function (l, c) {
                return (l.display.input.getField().tabIndex = c || "");
              }),
              a("autofocus", null),
              a(
                "direction",
                "ltr",
                function (l, c) {
                  return l.doc.setDirection(c);
                },
                !0,
              ),
              a("phrases", null);
          }
          function Fx(n, i, a) {
            var l = a && a != Ro;
            if (!i != !l) {
              var c = n.display.dragFunctions,
                p = i ? Rt : Ke;
              p(n.display.scroller, "dragstart", c.start),
                p(n.display.scroller, "dragenter", c.enter),
                p(n.display.scroller, "dragover", c.over),
                p(n.display.scroller, "dragleave", c.leave),
                p(n.display.scroller, "drop", c.drop);
            }
          }
          function qx(n) {
            n.options.lineWrapping
              ? (At(n.display.wrapper, "CodeMirror-wrap"),
                (n.display.sizer.style.minWidth = ""),
                (n.display.sizerWidth = null))
              : (gt(n.display.wrapper, "CodeMirror-wrap"), iu(n)),
              gu(n),
              ln(n),
              As(n),
              setTimeout(function () {
                return Lo(n);
              }, 100);
          }
          function be(n, i) {
            var a = this;
            if (!(this instanceof be)) return new be(n, i);
            (this.options = i = i ? rt(i) : {}), rt(rg, i, !1);
            var l = i.value;
            typeof l == "string"
              ? (l = new an(l, i.mode, null, i.lineSeparator, i.direction))
              : i.mode && (l.modeOption = i.mode),
              (this.doc = l);
            var c = new be.inputStyles[i.inputStyle](this),
              p = (this.display = new Qw(n, l, c, i));
            (p.wrapper.CodeMirror = this),
              ng(this),
              i.lineWrapping && (this.display.wrapper.className += " CodeMirror-wrap"),
              op(this),
              (this.state = {
                keyMaps: [],
                overlays: [],
                modeGen: 0,
                overwrite: !1,
                delayingBlurEvent: !1,
                focused: !1,
                suppressEdits: !1,
                pasteIncoming: -1,
                cutIncoming: -1,
                selectingText: !1,
                draggingText: !1,
                highlight: new Mt(),
                keySeq: null,
                specialChars: null,
              }),
              i.autofocus && !E && p.input.focus(),
              d &&
                g < 11 &&
                setTimeout(function () {
                  return a.display.input.reset(!0);
                }, 20),
              Hx(this),
              yx(),
              Fi(this),
              (this.curOp.forceUpdate = !0),
              gp(this, l),
              (i.autofocus && !E) || this.hasFocus()
                ? setTimeout(function () {
                    a.hasFocus() && !a.state.focused && bu(a);
                  }, 20)
                : To(this);
            for (var m in va) va.hasOwnProperty(m) && va[m](this, i[m], Ro);
            ap(this), i.finishInit && i.finishInit(this);
            for (var y = 0; y < Fu.length; ++y) Fu[y](this);
            qi(this),
              v &&
                i.lineWrapping &&
                getComputedStyle(p.lineDiv).textRendering == "optimizelegibility" &&
                (p.lineDiv.style.textRendering = "auto");
          }
          (be.defaults = rg), (be.optionHandlers = va);
          function Hx(n) {
            var i = n.display;
            Rt(i.scroller, "mousedown", He(n, Jp)),
              d && g < 11
                ? Rt(
                    i.scroller,
                    "dblclick",
                    He(n, function (x) {
                      if (!Ce(n, x)) {
                        var _ = $i(n, x);
                        if (!(!_ || Iu(n, x) || Ar(n.display, x))) {
                          Xe(x);
                          var N = n.findWordAt(_);
                          ca(n.doc, N.anchor, N.head);
                        }
                      }
                    }),
                  )
                : Rt(i.scroller, "dblclick", function (x) {
                    return Ce(n, x) || Xe(x);
                  }),
              Rt(i.scroller, "contextmenu", function (x) {
                return eg(n, x);
              }),
              Rt(i.input.getField(), "contextmenu", function (x) {
                i.scroller.contains(x.target) || eg(n, x);
              });
            var a,
              l = { end: 0 };
            function c() {
              i.activeTouch &&
                ((a = setTimeout(function () {
                  return (i.activeTouch = null);
                }, 1e3)),
                (l = i.activeTouch),
                (l.end = +new Date()));
            }
            function p(x) {
              if (x.touches.length != 1) return !1;
              var _ = x.touches[0];
              return _.radiusX <= 1 && _.radiusY <= 1;
            }
            function m(x, _) {
              if (_.left == null) return !0;
              var N = _.left - x.left,
                D = _.top - x.top;
              return N * N + D * D > 20 * 20;
            }
            Rt(i.scroller, "touchstart", function (x) {
              if (!Ce(n, x) && !p(x) && !Iu(n, x)) {
                i.input.ensurePolled(), clearTimeout(a);
                var _ = +new Date();
                (i.activeTouch = { start: _, moved: !1, prev: _ - l.end <= 300 ? l : null }),
                  x.touches.length == 1 &&
                    ((i.activeTouch.left = x.touches[0].pageX),
                    (i.activeTouch.top = x.touches[0].pageY));
              }
            }),
              Rt(i.scroller, "touchmove", function () {
                i.activeTouch && (i.activeTouch.moved = !0);
              }),
              Rt(i.scroller, "touchend", function (x) {
                var _ = i.activeTouch;
                if (_ && !Ar(i, x) && _.left != null && !_.moved && new Date() - _.start < 300) {
                  var N = n.coordsChar(i.activeTouch, "page"),
                    D;
                  !_.prev || m(_, _.prev)
                    ? (D = new ue(N, N))
                    : !_.prev.prev || m(_, _.prev.prev)
                    ? (D = n.findWordAt(N))
                    : (D = new ue(X(N.line, 0), Wt(n.doc, X(N.line + 1, 0)))),
                    n.setSelection(D.anchor, D.head),
                    n.focus(),
                    Xe(x);
                }
                c();
              }),
              Rt(i.scroller, "touchcancel", c),
              Rt(i.scroller, "scroll", function () {
                i.scroller.clientHeight &&
                  (Ps(n, i.scroller.scrollTop),
                  zi(n, i.scroller.scrollLeft, !0),
                  ke(n, "scroll", n));
              }),
              Rt(i.scroller, "mousewheel", function (x) {
                return fp(n, x);
              }),
              Rt(i.scroller, "DOMMouseScroll", function (x) {
                return fp(n, x);
              }),
              Rt(i.wrapper, "scroll", function () {
                return (i.wrapper.scrollTop = i.wrapper.scrollLeft = 0);
              }),
              (i.dragFunctions = {
                enter: function (x) {
                  Ce(n, x) || Yr(x);
                },
                over: function (x) {
                  Ce(n, x) || (mx(n, x), Yr(x));
                },
                start: function (x) {
                  return vx(n, x);
                },
                drop: He(n, gx),
                leave: function (x) {
                  Ce(n, x) || Fp(n);
                },
              });
            var y = i.input.getField();
            Rt(y, "keyup", function (x) {
              return Yp.call(n, x);
            }),
              Rt(y, "keydown", He(n, Xp)),
              Rt(y, "keypress", He(n, Zp)),
              Rt(y, "focus", function (x) {
                return bu(n, x);
              }),
              Rt(y, "blur", function (x) {
                return To(n, x);
              });
          }
          var Fu = [];
          be.defineInitHook = function (n) {
            return Fu.push(n);
          };
          function Ys(n, i, a, l) {
            var c = n.doc,
              p;
            a == null && (a = "add"),
              a == "smart" && (c.mode.indent ? (p = ks(n, i).state) : (a = "prev"));
            var m = n.options.tabSize,
              y = Pt(c, i),
              x = lt(y.text, null, m);
            y.stateAfter && (y.stateAfter = null);
            var _ = y.text.match(/^\s*/)[0],
              N;
            if (!l && !/\S/.test(y.text)) (N = 0), (a = "not");
            else if (
              a == "smart" &&
              ((N = c.mode.indent(p, y.text.slice(_.length), y.text)), N == I || N > 150)
            ) {
              if (!l) return;
              a = "prev";
            }
            a == "prev"
              ? i > c.first
                ? (N = lt(Pt(c, i - 1).text, null, m))
                : (N = 0)
              : a == "add"
              ? (N = x + n.options.indentUnit)
              : a == "subtract"
              ? (N = x - n.options.indentUnit)
              : typeof a == "number" && (N = x + a),
              (N = Math.max(0, N));
            var D = "",
              W = 0;
            if (n.options.indentWithTabs)
              for (var q = Math.floor(N / m); q; --q) (W += m), (D += "	");
            if ((W < N && (D += mt(N - W)), D != _))
              return Po(c, D, X(i, 0), X(i, _.length), "+input"), (y.stateAfter = null), !0;
            for (var Z = 0; Z < c.sel.ranges.length; Z++) {
              var it = c.sel.ranges[Z];
              if (it.head.line == i && it.head.ch < _.length) {
                var vt = X(i, _.length);
                Pu(c, Z, new ue(vt, vt));
                break;
              }
            }
          }
          var tr = null;
          function ma(n) {
            tr = n;
          }
          function qu(n, i, a, l, c) {
            var p = n.doc;
            (n.display.shift = !1), l || (l = p.sel);
            var m = +new Date() - 200,
              y = c == "paste" || n.state.pasteIncoming > m,
              x = Hn(i),
              _ = null;
            if (y && l.ranges.length > 1)
              if (
                tr &&
                tr.text.join(`
`) == i
              ) {
                if (l.ranges.length % tr.text.length == 0) {
                  _ = [];
                  for (var N = 0; N < tr.text.length; N++) _.push(p.splitLines(tr.text[N]));
                }
              } else
                x.length == l.ranges.length &&
                  n.options.pasteLinesPerSelection &&
                  (_ = ft(x, function (bt) {
                    return [bt];
                  }));
            for (var D = n.curOp.updateInput, W = l.ranges.length - 1; W >= 0; W--) {
              var q = l.ranges[W],
                Z = q.from(),
                it = q.to();
              q.empty() &&
                (a && a > 0
                  ? (Z = X(Z.line, Z.ch - a))
                  : n.state.overwrite && !y
                  ? (it = X(it.line, Math.min(Pt(p, it.line).text.length, it.ch + ct(x).length)))
                  : y &&
                    tr &&
                    tr.lineWise &&
                    tr.text.join(`
`) ==
                      x.join(`
`) &&
                    (Z = it = X(Z.line, 0)));
              var vt = {
                from: Z,
                to: it,
                text: _ ? _[W % _.length] : x,
                origin: c || (y ? "paste" : n.state.cutIncoming > m ? "cut" : "+input"),
              };
              No(n.doc, vt), qe(n, "inputRead", n, vt);
            }
            i && !y && og(n, i),
              Eo(n),
              n.curOp.updateInput < 2 && (n.curOp.updateInput = D),
              (n.curOp.typing = !0),
              (n.state.pasteIncoming = n.state.cutIncoming = -1);
          }
          function ig(n, i) {
            var a = n.clipboardData && n.clipboardData.getData("Text");
            if (a)
              return (
                n.preventDefault(),
                !i.isReadOnly() &&
                  !i.options.disableInput &&
                  i.hasFocus() &&
                  Sn(i, function () {
                    return qu(i, a, 0, null, "paste");
                  }),
                !0
              );
          }
          function og(n, i) {
            if (!(!n.options.electricChars || !n.options.smartIndent))
              for (var a = n.doc.sel, l = a.ranges.length - 1; l >= 0; l--) {
                var c = a.ranges[l];
                if (!(c.head.ch > 100 || (l && a.ranges[l - 1].head.line == c.head.line))) {
                  var p = n.getModeAt(c.head),
                    m = !1;
                  if (p.electricChars) {
                    for (var y = 0; y < p.electricChars.length; y++)
                      if (i.indexOf(p.electricChars.charAt(y)) > -1) {
                        m = Ys(n, c.head.line, "smart");
                        break;
                      }
                  } else
                    p.electricInput &&
                      p.electricInput.test(Pt(n.doc, c.head.line).text.slice(0, c.head.ch)) &&
                      (m = Ys(n, c.head.line, "smart"));
                  m && qe(n, "electricInput", n, c.head.line);
                }
              }
          }
          function sg(n) {
            for (var i = [], a = [], l = 0; l < n.doc.sel.ranges.length; l++) {
              var c = n.doc.sel.ranges[l].head.line,
                p = { anchor: X(c, 0), head: X(c + 1, 0) };
              a.push(p), i.push(n.getRange(p.anchor, p.head));
            }
            return { text: i, ranges: a };
          }
          function Hu(n, i, a, l) {
            n.setAttribute("autocorrect", a ? "on" : "off"),
              n.setAttribute("autocapitalize", l ? "on" : "off"),
              n.setAttribute("spellcheck", !!i);
          }
          function lg() {
            var n = k(
                "textarea",
                null,
                null,
                "position: absolute; bottom: -1em; padding: 0; width: 1px; height: 1em; min-height: 1em; outline: none",
              ),
              i = k(
                "div",
                [n],
                null,
                "overflow: hidden; position: relative; width: 3px; height: 0px;",
              );
            return (
              v ? (n.style.width = "1000px") : n.setAttribute("wrap", "off"),
              M && (n.style.border = "1px solid black"),
              i
            );
          }
          function Bx(n) {
            var i = n.optionHandlers,
              a = (n.helpers = {});
            (n.prototype = {
              constructor: n,
              focus: function () {
                Tt(this).focus(), this.display.input.focus();
              },
              setOption: function (l, c) {
                var p = this.options,
                  m = p[l];
                (p[l] == c && l != "mode") ||
                  ((p[l] = c),
                  i.hasOwnProperty(l) && He(this, i[l])(this, c, m),
                  ke(this, "optionChange", this, l));
              },
              getOption: function (l) {
                return this.options[l];
              },
              getDoc: function () {
                return this.doc;
              },
              addKeyMap: function (l, c) {
                this.state.keyMaps[c ? "push" : "unshift"](pa(l));
              },
              removeKeyMap: function (l) {
                for (var c = this.state.keyMaps, p = 0; p < c.length; ++p)
                  if (c[p] == l || c[p].name == l) return c.splice(p, 1), !0;
              },
              addOverlay: Qe(function (l, c) {
                var p = l.token ? l : n.getMode(this.options, l);
                if (p.startState) throw new Error("Overlays may not be stateful.");
                $t(
                  this.state.overlays,
                  { mode: p, modeSpec: l, opaque: c && c.opaque, priority: (c && c.priority) || 0 },
                  function (m) {
                    return m.priority;
                  },
                ),
                  this.state.modeGen++,
                  ln(this);
              }),
              removeOverlay: Qe(function (l) {
                for (var c = this.state.overlays, p = 0; p < c.length; ++p) {
                  var m = c[p].modeSpec;
                  if (m == l || (typeof l == "string" && m.name == l)) {
                    c.splice(p, 1), this.state.modeGen++, ln(this);
                    return;
                  }
                }
              }),
              indentLine: Qe(function (l, c, p) {
                typeof c != "string" &&
                  typeof c != "number" &&
                  (c == null
                    ? (c = this.options.smartIndent ? "smart" : "prev")
                    : (c = c ? "add" : "subtract")),
                  et(this.doc, l) && Ys(this, l, c, p);
              }),
              indentSelection: Qe(function (l) {
                for (var c = this.doc.sel.ranges, p = -1, m = 0; m < c.length; m++) {
                  var y = c[m];
                  if (y.empty())
                    y.head.line > p &&
                      (Ys(this, y.head.line, l, !0),
                      (p = y.head.line),
                      m == this.doc.sel.primIndex && Eo(this));
                  else {
                    var x = y.from(),
                      _ = y.to(),
                      N = Math.max(p, x.line);
                    p = Math.min(this.lastLine(), _.line - (_.ch ? 0 : 1)) + 1;
                    for (var D = N; D < p; ++D) Ys(this, D, l);
                    var W = this.doc.sel.ranges;
                    x.ch == 0 &&
                      c.length == W.length &&
                      W[m].from().ch > 0 &&
                      Pu(this.doc, m, new ue(x, W[m].to()), V);
                  }
                }
              }),
              getTokenAt: function (l, c) {
                return md(this, l, c);
              },
              getLineTokens: function (l, c) {
                return md(this, X(l), c, !0);
              },
              getTokenTypeAt: function (l) {
                l = Wt(this.doc, l);
                var c = pd(this, Pt(this.doc, l.line)),
                  p = 0,
                  m = (c.length - 1) / 2,
                  y = l.ch,
                  x;
                if (y == 0) x = c[2];
                else
                  for (;;) {
                    var _ = (p + m) >> 1;
                    if ((_ ? c[_ * 2 - 1] : 0) >= y) m = _;
                    else if (c[_ * 2 + 1] < y) p = _ + 1;
                    else {
                      x = c[_ * 2 + 2];
                      break;
                    }
                  }
                var N = x ? x.indexOf("overlay ") : -1;
                return N < 0 ? x : N == 0 ? null : x.slice(0, N - 1);
              },
              getModeAt: function (l) {
                var c = this.doc.mode;
                return c.innerMode ? n.innerMode(c, this.getTokenAt(l).state).mode : c;
              },
              getHelper: function (l, c) {
                return this.getHelpers(l, c)[0];
              },
              getHelpers: function (l, c) {
                var p = [];
                if (!a.hasOwnProperty(c)) return p;
                var m = a[c],
                  y = this.getModeAt(l);
                if (typeof y[c] == "string") m[y[c]] && p.push(m[y[c]]);
                else if (y[c])
                  for (var x = 0; x < y[c].length; x++) {
                    var _ = m[y[c][x]];
                    _ && p.push(_);
                  }
                else
                  y.helperType && m[y.helperType]
                    ? p.push(m[y.helperType])
                    : m[y.name] && p.push(m[y.name]);
                for (var N = 0; N < m._global.length; N++) {
                  var D = m._global[N];
                  D.pred(y, this) && Et(p, D.val) == -1 && p.push(D.val);
                }
                return p;
              },
              getStateAfter: function (l, c) {
                var p = this.doc;
                return (l = fd(p, l ?? p.first + p.size - 1)), ks(this, l + 1, c).state;
              },
              cursorCoords: function (l, c) {
                var p,
                  m = this.doc.sel.primary();
                return (
                  l == null
                    ? (p = m.head)
                    : typeof l == "object"
                    ? (p = Wt(this.doc, l))
                    : (p = l ? m.from() : m.to()),
                  Jn(this, p, c || "page")
                );
              },
              charCoords: function (l, c) {
                return Ql(this, Wt(this.doc, l), c || "page");
              },
              coordsChar: function (l, c) {
                return (l = Vd(this, l, c || "page")), fu(this, l.left, l.top);
              },
              lineAtHeight: function (l, c) {
                return (
                  (l = Vd(this, { top: l, left: 0 }, c || "page").top),
                  O(this.doc, l + this.display.viewOffset)
                );
              },
              heightAtLine: function (l, c, p) {
                var m = !1,
                  y;
                if (typeof l == "number") {
                  var x = this.doc.first + this.doc.size - 1;
                  l < this.doc.first ? (l = this.doc.first) : l > x && ((l = x), (m = !0)),
                    (y = Pt(this.doc, l));
                } else y = l;
                return (
                  Jl(this, y, { top: 0, left: 0 }, c || "page", p || m).top +
                  (m ? this.doc.height - Lr(y) : 0)
                );
              },
              defaultTextHeight: function () {
                return ko(this.display);
              },
              defaultCharWidth: function () {
                return Co(this.display);
              },
              getViewport: function () {
                return { from: this.display.viewFrom, to: this.display.viewTo };
              },
              addWidget: function (l, c, p, m, y) {
                var x = this.display;
                l = Jn(this, Wt(this.doc, l));
                var _ = l.bottom,
                  N = l.left;
                if (
                  ((c.style.position = "absolute"),
                  c.setAttribute("cm-ignore-events", "true"),
                  this.display.input.setUneditable(c),
                  x.sizer.appendChild(c),
                  m == "over")
                )
                  _ = l.top;
                else if (m == "above" || m == "near") {
                  var D = Math.max(x.wrapper.clientHeight, this.doc.height),
                    W = Math.max(x.sizer.clientWidth, x.lineSpace.clientWidth);
                  (m == "above" || l.bottom + c.offsetHeight > D) && l.top > c.offsetHeight
                    ? (_ = l.top - c.offsetHeight)
                    : l.bottom + c.offsetHeight <= D && (_ = l.bottom),
                    N + c.offsetWidth > W && (N = W - c.offsetWidth);
                }
                (c.style.top = _ + "px"),
                  (c.style.left = c.style.right = ""),
                  y == "right"
                    ? ((N = x.sizer.clientWidth - c.offsetWidth), (c.style.right = "0px"))
                    : (y == "left"
                        ? (N = 0)
                        : y == "middle" && (N = (x.sizer.clientWidth - c.offsetWidth) / 2),
                      (c.style.left = N + "px")),
                  p &&
                    Fw(this, {
                      left: N,
                      top: _,
                      right: N + c.offsetWidth,
                      bottom: _ + c.offsetHeight,
                    });
              },
              triggerOnKeyDown: Qe(Xp),
              triggerOnKeyPress: Qe(Zp),
              triggerOnKeyUp: Yp,
              triggerOnMouseDown: Qe(Jp),
              execCommand: function (l) {
                if (Vs.hasOwnProperty(l)) return Vs[l].call(null, this);
              },
              triggerElectric: Qe(function (l) {
                og(this, l);
              }),
              findPosH: function (l, c, p, m) {
                var y = 1;
                c < 0 && ((y = -1), (c = -c));
                for (
                  var x = Wt(this.doc, l), _ = 0;
                  _ < c && ((x = Bu(this.doc, x, y, p, m)), !x.hitSide);
                  ++_
                );
                return x;
              },
              moveH: Qe(function (l, c) {
                var p = this;
                this.extendSelectionsBy(function (m) {
                  return p.display.shift || p.doc.extend || m.empty()
                    ? Bu(p.doc, m.head, l, c, p.options.rtlMoveVisually)
                    : l < 0
                    ? m.from()
                    : m.to();
                }, ot);
              }),
              deleteH: Qe(function (l, c) {
                var p = this.doc.sel,
                  m = this.doc;
                p.somethingSelected()
                  ? m.replaceSelection("", null, "+delete")
                  : $o(this, function (y) {
                      var x = Bu(m, y.head, l, c, !1);
                      return l < 0 ? { from: x, to: y.head } : { from: y.head, to: x };
                    });
              }),
              findPosV: function (l, c, p, m) {
                var y = 1,
                  x = m;
                c < 0 && ((y = -1), (c = -c));
                for (var _ = Wt(this.doc, l), N = 0; N < c; ++N) {
                  var D = Jn(this, _, "div");
                  if ((x == null ? (x = D.left) : (D.left = x), (_ = ag(this, D, y, p)), _.hitSide))
                    break;
                }
                return _;
              },
              moveV: Qe(function (l, c) {
                var p = this,
                  m = this.doc,
                  y = [],
                  x = !this.display.shift && !m.extend && m.sel.somethingSelected();
                if (
                  (m.extendSelectionsBy(function (N) {
                    if (x) return l < 0 ? N.from() : N.to();
                    var D = Jn(p, N.head, "div");
                    N.goalColumn != null && (D.left = N.goalColumn), y.push(D.left);
                    var W = ag(p, D, l, c);
                    return (
                      c == "page" && N == m.sel.primary() && xu(p, Ql(p, W, "div").top - D.top), W
                    );
                  }, ot),
                  y.length)
                )
                  for (var _ = 0; _ < m.sel.ranges.length; _++) m.sel.ranges[_].goalColumn = y[_];
              }),
              findWordAt: function (l) {
                var c = this.doc,
                  p = Pt(c, l.line).text,
                  m = l.ch,
                  y = l.ch;
                if (p) {
                  var x = this.getHelper(l, "wordChars");
                  (l.sticky == "before" || y == p.length) && m ? --m : ++y;
                  for (
                    var _ = p.charAt(m),
                      N = re(_, x)
                        ? function (D) {
                            return re(D, x);
                          }
                        : /\s/.test(_)
                        ? function (D) {
                            return /\s/.test(D);
                          }
                        : function (D) {
                            return !/\s/.test(D) && !re(D);
                          };
                    m > 0 && N(p.charAt(m - 1));
                  )
                    --m;
                  for (; y < p.length && N(p.charAt(y)); ) ++y;
                }
                return new ue(X(l.line, m), X(l.line, y));
              },
              toggleOverwrite: function (l) {
                (l != null && l == this.state.overwrite) ||
                  ((this.state.overwrite = !this.state.overwrite)
                    ? At(this.display.cursorDiv, "CodeMirror-overwrite")
                    : gt(this.display.cursorDiv, "CodeMirror-overwrite"),
                  ke(this, "overwriteToggle", this, this.state.overwrite));
              },
              hasFocus: function () {
                return this.display.input.getField() == yt(Jt(this));
              },
              isReadOnly: function () {
                return !!(this.options.readOnly || this.doc.cantEdit);
              },
              scrollTo: Qe(function (l, c) {
                Ns(this, l, c);
              }),
              getScrollInfo: function () {
                var l = this.display.scroller;
                return {
                  left: l.scrollLeft,
                  top: l.scrollTop,
                  height: l.scrollHeight - hr(this) - this.display.barHeight,
                  width: l.scrollWidth - hr(this) - this.display.barWidth,
                  clientHeight: lu(this),
                  clientWidth: Oi(this),
                };
              },
              scrollIntoView: Qe(function (l, c) {
                l == null
                  ? ((l = { from: this.doc.sel.primary().head, to: null }),
                    c == null && (c = this.options.cursorScrollMargin))
                  : typeof l == "number"
                  ? (l = { from: X(l, 0), to: null })
                  : l.from == null && (l = { from: l, to: null }),
                  l.to || (l.to = l.from),
                  (l.margin = c || 0),
                  l.from.line != null ? qw(this, l) : ep(this, l.from, l.to, l.margin);
              }),
              setSize: Qe(function (l, c) {
                var p = this,
                  m = function (x) {
                    return typeof x == "number" || /^\d+$/.test(String(x)) ? x + "px" : x;
                  };
                l != null && (this.display.wrapper.style.width = m(l)),
                  c != null && (this.display.wrapper.style.height = m(c)),
                  this.options.lineWrapping && Wd(this);
                var y = this.display.viewFrom;
                this.doc.iter(y, this.display.viewTo, function (x) {
                  if (x.widgets) {
                    for (var _ = 0; _ < x.widgets.length; _++)
                      if (x.widgets[_].noHScroll) {
                        ei(p, y, "widget");
                        break;
                      }
                  }
                  ++y;
                }),
                  (this.curOp.forceUpdate = !0),
                  ke(this, "refresh", this);
              }),
              operation: function (l) {
                return Sn(this, l);
              },
              startOperation: function () {
                return Fi(this);
              },
              endOperation: function () {
                return qi(this);
              },
              refresh: Qe(function () {
                var l = this.display.cachedTextHeight;
                ln(this),
                  (this.curOp.forceUpdate = !0),
                  As(this),
                  Ns(this, this.doc.scrollLeft, this.doc.scrollTop),
                  ku(this.display),
                  (l == null ||
                    Math.abs(l - ko(this.display)) > 0.5 ||
                    this.options.lineWrapping) &&
                    gu(this),
                  ke(this, "refresh", this);
              }),
              swapDoc: Qe(function (l) {
                var c = this.doc;
                return (
                  (c.cm = null),
                  this.state.selectingText && this.state.selectingText(),
                  gp(this, l),
                  As(this),
                  this.display.input.reset(),
                  Ns(this, l.scrollLeft, l.scrollTop),
                  (this.curOp.forceScroll = !0),
                  qe(this, "swapDoc", this, c),
                  c
                );
              }),
              phrase: function (l) {
                var c = this.options.phrases;
                return c && Object.prototype.hasOwnProperty.call(c, l) ? c[l] : l;
              },
              getInputField: function () {
                return this.display.input.getField();
              },
              getWrapperElement: function () {
                return this.display.wrapper;
              },
              getScrollerElement: function () {
                return this.display.scroller;
              },
              getGutterElement: function () {
                return this.display.gutters;
              },
            }),
              Vn(n),
              (n.registerHelper = function (l, c, p) {
                a.hasOwnProperty(l) || (a[l] = n[l] = { _global: [] }), (a[l][c] = p);
              }),
              (n.registerGlobalHelper = function (l, c, p, m) {
                n.registerHelper(l, c, m), a[l]._global.push({ pred: p, val: m });
              });
          }
          function Bu(n, i, a, l, c) {
            var p = i,
              m = a,
              y = Pt(n, i.line),
              x = c && n.direction == "rtl" ? -a : a;
            function _() {
              var Ct = i.line + x;
              return Ct < n.first || Ct >= n.first + n.size
                ? !1
                : ((i = new X(Ct, i.ch, i.sticky)), (y = Pt(n, Ct)));
            }
            function N(Ct) {
              var wt;
              if (l == "codepoint") {
                var Lt = y.text.charCodeAt(i.ch + (a > 0 ? 0 : -1));
                if (isNaN(Lt)) wt = null;
                else {
                  var zt = a > 0 ? Lt >= 55296 && Lt < 56320 : Lt >= 56320 && Lt < 57343;
                  wt = new X(
                    i.line,
                    Math.max(0, Math.min(y.text.length, i.ch + a * (zt ? 2 : 1))),
                    -a,
                  );
                }
              } else c ? (wt = Sx(n.cm, y, i, a)) : (wt = Du(y, i, a));
              if (wt == null)
                if (!Ct && _()) i = $u(c, n.cm, y, i.line, x);
                else return !1;
              else i = wt;
              return !0;
            }
            if (l == "char" || l == "codepoint") N();
            else if (l == "column") N(!0);
            else if (l == "word" || l == "group")
              for (
                var D = null, W = l == "group", q = n.cm && n.cm.getHelper(i, "wordChars"), Z = !0;
                !(a < 0 && !N(!Z));
                Z = !1
              ) {
                var it =
                    y.text.charAt(i.ch) ||
                    `
`,
                  vt = re(it, q)
                    ? "w"
                    : W &&
                      it ==
                        `
`
                    ? "n"
                    : !W || /\s/.test(it)
                    ? null
                    : "p";
                if ((W && !Z && !vt && (vt = "s"), D && D != vt)) {
                  a < 0 && ((a = 1), N(), (i.sticky = "after"));
                  break;
                }
                if ((vt && (D = vt), a > 0 && !N(!Z))) break;
              }
            var bt = fa(n, i, p, m, !0);
            return ce(p, bt) && (bt.hitSide = !0), bt;
          }
          function ag(n, i, a, l) {
            var c = n.doc,
              p = i.left,
              m;
            if (l == "page") {
              var y = Math.min(
                  n.display.wrapper.clientHeight,
                  Tt(n).innerHeight || c(n).documentElement.clientHeight,
                ),
                x = Math.max(y - 0.5 * ko(n.display), 3);
              m = (a > 0 ? i.bottom : i.top) + a * x;
            } else l == "line" && (m = a > 0 ? i.bottom + 3 : i.top - 3);
            for (var _; (_ = fu(n, p, m)), !!_.outside; ) {
              if (a < 0 ? m <= 0 : m >= c.height) {
                _.hitSide = !0;
                break;
              }
              m += a * 5;
            }
            return _;
          }
          var he = function (n) {
            (this.cm = n),
              (this.lastAnchorNode =
                this.lastAnchorOffset =
                this.lastFocusNode =
                this.lastFocusOffset =
                  null),
              (this.polling = new Mt()),
              (this.composing = null),
              (this.gracePeriod = !1),
              (this.readDOMTimeout = null);
          };
          (he.prototype.init = function (n) {
            var i = this,
              a = this,
              l = a.cm,
              c = (a.div = n.lineDiv);
            (c.contentEditable = !0),
              Hu(c, l.options.spellcheck, l.options.autocorrect, l.options.autocapitalize);
            function p(y) {
              for (var x = y.target; x; x = x.parentNode) {
                if (x == c) return !0;
                if (/\bCodeMirror-(?:line)?widget\b/.test(x.className)) break;
              }
              return !1;
            }
            Rt(c, "paste", function (y) {
              !p(y) ||
                Ce(l, y) ||
                ig(y, l) ||
                (g <= 11 &&
                  setTimeout(
                    He(l, function () {
                      return i.updateFromDOM();
                    }),
                    20,
                  ));
            }),
              Rt(c, "compositionstart", function (y) {
                i.composing = { data: y.data, done: !1 };
              }),
              Rt(c, "compositionupdate", function (y) {
                i.composing || (i.composing = { data: y.data, done: !1 });
              }),
              Rt(c, "compositionend", function (y) {
                i.composing &&
                  (y.data != i.composing.data && i.readFromDOMSoon(), (i.composing.done = !0));
              }),
              Rt(c, "touchstart", function () {
                return a.forceCompositionEnd();
              }),
              Rt(c, "input", function () {
                i.composing || i.readFromDOMSoon();
              });
            function m(y) {
              if (!(!p(y) || Ce(l, y))) {
                if (l.somethingSelected())
                  ma({ lineWise: !1, text: l.getSelections() }),
                    y.type == "cut" && l.replaceSelection("", null, "cut");
                else if (l.options.lineWiseCopyCut) {
                  var x = sg(l);
                  ma({ lineWise: !0, text: x.text }),
                    y.type == "cut" &&
                      l.operation(function () {
                        l.setSelections(x.ranges, 0, V), l.replaceSelection("", null, "cut");
                      });
                } else return;
                if (y.clipboardData) {
                  y.clipboardData.clearData();
                  var _ = tr.text.join(`
`);
                  if ((y.clipboardData.setData("Text", _), y.clipboardData.getData("Text") == _)) {
                    y.preventDefault();
                    return;
                  }
                }
                var N = lg(),
                  D = N.firstChild;
                Hu(D),
                  l.display.lineSpace.insertBefore(N, l.display.lineSpace.firstChild),
                  (D.value = tr.text.join(`
`));
                var W = yt(Gt(c));
                Ht(D),
                  setTimeout(function () {
                    l.display.lineSpace.removeChild(N),
                      W.focus(),
                      W == c && a.showPrimarySelection();
                  }, 50);
              }
            }
            Rt(c, "copy", m), Rt(c, "cut", m);
          }),
            (he.prototype.screenReaderLabelChanged = function (n) {
              n ? this.div.setAttribute("aria-label", n) : this.div.removeAttribute("aria-label");
            }),
            (he.prototype.prepareSelection = function () {
              var n = Jd(this.cm, !1);
              return (n.focus = yt(Gt(this.div)) == this.div), n;
            }),
            (he.prototype.showSelection = function (n, i) {
              !n ||
                !this.cm.display.view.length ||
                ((n.focus || i) && this.showPrimarySelection(), this.showMultipleSelections(n));
            }),
            (he.prototype.getSelection = function () {
              return this.cm.display.wrapper.ownerDocument.getSelection();
            }),
            (he.prototype.showPrimarySelection = function () {
              var n = this.getSelection(),
                i = this.cm,
                a = i.doc.sel.primary(),
                l = a.from(),
                c = a.to();
              if (
                i.display.viewTo == i.display.viewFrom ||
                l.line >= i.display.viewTo ||
                c.line < i.display.viewFrom
              ) {
                n.removeAllRanges();
                return;
              }
              var p = ya(i, n.anchorNode, n.anchorOffset),
                m = ya(i, n.focusNode, n.focusOffset);
              if (!(p && !p.bad && m && !m.bad && _t(wo(p, m), l) == 0 && _t(sn(p, m), c) == 0)) {
                var y = i.display.view,
                  x = (l.line >= i.display.viewFrom && cg(i, l)) || {
                    node: y[0].measure.map[2],
                    offset: 0,
                  },
                  _ = c.line < i.display.viewTo && cg(i, c);
                if (!_) {
                  var N = y[y.length - 1].measure,
                    D = N.maps ? N.maps[N.maps.length - 1] : N.map;
                  _ = { node: D[D.length - 1], offset: D[D.length - 2] - D[D.length - 3] };
                }
                if (!x || !_) {
                  n.removeAllRanges();
                  return;
                }
                var W = n.rangeCount && n.getRangeAt(0),
                  q;
                try {
                  q = H(x.node, x.offset, _.offset, _.node);
                } catch {}
                q &&
                  (!s && i.state.focused
                    ? (n.collapse(x.node, x.offset),
                      q.collapsed || (n.removeAllRanges(), n.addRange(q)))
                    : (n.removeAllRanges(), n.addRange(q)),
                  W && n.anchorNode == null ? n.addRange(W) : s && this.startGracePeriod()),
                  this.rememberSelection();
              }
            }),
            (he.prototype.startGracePeriod = function () {
              var n = this;
              clearTimeout(this.gracePeriod),
                (this.gracePeriod = setTimeout(function () {
                  (n.gracePeriod = !1),
                    n.selectionChanged() &&
                      n.cm.operation(function () {
                        return (n.cm.curOp.selectionChanged = !0);
                      });
                }, 20));
            }),
            (he.prototype.showMultipleSelections = function (n) {
              z(this.cm.display.cursorDiv, n.cursors), z(this.cm.display.selectionDiv, n.selection);
            }),
            (he.prototype.rememberSelection = function () {
              var n = this.getSelection();
              (this.lastAnchorNode = n.anchorNode),
                (this.lastAnchorOffset = n.anchorOffset),
                (this.lastFocusNode = n.focusNode),
                (this.lastFocusOffset = n.focusOffset);
            }),
            (he.prototype.selectionInEditor = function () {
              var n = this.getSelection();
              if (!n.rangeCount) return !1;
              var i = n.getRangeAt(0).commonAncestorContainer;
              return J(this.div, i);
            }),
            (he.prototype.focus = function () {
              this.cm.options.readOnly != "nocursor" &&
                ((!this.selectionInEditor() || yt(Gt(this.div)) != this.div) &&
                  this.showSelection(this.prepareSelection(), !0),
                this.div.focus());
            }),
            (he.prototype.blur = function () {
              this.div.blur();
            }),
            (he.prototype.getField = function () {
              return this.div;
            }),
            (he.prototype.supportsTouch = function () {
              return !0;
            }),
            (he.prototype.receivedFocus = function () {
              var n = this,
                i = this;
              this.selectionInEditor()
                ? setTimeout(function () {
                    return n.pollSelection();
                  }, 20)
                : Sn(this.cm, function () {
                    return (i.cm.curOp.selectionChanged = !0);
                  });
              function a() {
                i.cm.state.focused &&
                  (i.pollSelection(), i.polling.set(i.cm.options.pollInterval, a));
              }
              this.polling.set(this.cm.options.pollInterval, a);
            }),
            (he.prototype.selectionChanged = function () {
              var n = this.getSelection();
              return (
                n.anchorNode != this.lastAnchorNode ||
                n.anchorOffset != this.lastAnchorOffset ||
                n.focusNode != this.lastFocusNode ||
                n.focusOffset != this.lastFocusOffset
              );
            }),
            (he.prototype.pollSelection = function () {
              if (!(this.readDOMTimeout != null || this.gracePeriod || !this.selectionChanged())) {
                var n = this.getSelection(),
                  i = this.cm;
                if (R && w && this.cm.display.gutterSpecs.length && Wx(n.anchorNode)) {
                  this.cm.triggerOnKeyDown({
                    type: "keydown",
                    keyCode: 8,
                    preventDefault: Math.abs,
                  }),
                    this.blur(),
                    this.focus();
                  return;
                }
                if (!this.composing) {
                  this.rememberSelection();
                  var a = ya(i, n.anchorNode, n.anchorOffset),
                    l = ya(i, n.focusNode, n.focusOffset);
                  a &&
                    l &&
                    Sn(i, function () {
                      Ye(i.doc, ri(a, l), V), (a.bad || l.bad) && (i.curOp.selectionChanged = !0);
                    });
                }
              }
            }),
            (he.prototype.pollContent = function () {
              this.readDOMTimeout != null &&
                (clearTimeout(this.readDOMTimeout), (this.readDOMTimeout = null));
              var n = this.cm,
                i = n.display,
                a = n.doc.sel.primary(),
                l = a.from(),
                c = a.to();
              if (
                (l.ch == 0 &&
                  l.line > n.firstLine() &&
                  (l = X(l.line - 1, Pt(n.doc, l.line - 1).length)),
                c.ch == Pt(n.doc, c.line).text.length &&
                  c.line < n.lastLine() &&
                  (c = X(c.line + 1, 0)),
                l.line < i.viewFrom || c.line > i.viewTo - 1)
              )
                return !1;
              var p, m, y;
              l.line == i.viewFrom || (p = Ri(n, l.line)) == 0
                ? ((m = C(i.view[0].line)), (y = i.view[0].node))
                : ((m = C(i.view[p].line)), (y = i.view[p - 1].node.nextSibling));
              var x = Ri(n, c.line),
                _,
                N;
              if (
                (x == i.view.length - 1
                  ? ((_ = i.viewTo - 1), (N = i.lineDiv.lastChild))
                  : ((_ = C(i.view[x + 1].line) - 1), (N = i.view[x + 1].node.previousSibling)),
                !y)
              )
                return !1;
              for (
                var D = n.doc.splitLines(Ux(n, y, N, m, _)),
                  W = Tr(n.doc, X(m, 0), X(_, Pt(n.doc, _).text.length));
                D.length > 1 && W.length > 1;
              )
                if (ct(D) == ct(W)) D.pop(), W.pop(), _--;
                else if (D[0] == W[0]) D.shift(), W.shift(), m++;
                else break;
              for (
                var q = 0, Z = 0, it = D[0], vt = W[0], bt = Math.min(it.length, vt.length);
                q < bt && it.charCodeAt(q) == vt.charCodeAt(q);
              )
                ++q;
              for (
                var Ct = ct(D),
                  wt = ct(W),
                  Lt = Math.min(
                    Ct.length - (D.length == 1 ? q : 0),
                    wt.length - (W.length == 1 ? q : 0),
                  );
                Z < Lt && Ct.charCodeAt(Ct.length - Z - 1) == wt.charCodeAt(wt.length - Z - 1);
              )
                ++Z;
              if (D.length == 1 && W.length == 1 && m == l.line)
                for (
                  ;
                  q &&
                  q > l.ch &&
                  Ct.charCodeAt(Ct.length - Z - 1) == wt.charCodeAt(wt.length - Z - 1);
                )
                  q--, Z++;
              (D[D.length - 1] = Ct.slice(0, Ct.length - Z).replace(/^\u200b+/, "")),
                (D[0] = D[0].slice(q).replace(/\u200b+$/, ""));
              var zt = X(m, q),
                Ot = X(_, W.length ? ct(W).length - Z : 0);
              if (D.length > 1 || D[0] || _t(zt, Ot)) return Po(n.doc, D, zt, Ot, "+input"), !0;
            }),
            (he.prototype.ensurePolled = function () {
              this.forceCompositionEnd();
            }),
            (he.prototype.reset = function () {
              this.forceCompositionEnd();
            }),
            (he.prototype.forceCompositionEnd = function () {
              this.composing &&
                (clearTimeout(this.readDOMTimeout),
                (this.composing = null),
                this.updateFromDOM(),
                this.div.blur(),
                this.div.focus());
            }),
            (he.prototype.readFromDOMSoon = function () {
              var n = this;
              this.readDOMTimeout == null &&
                (this.readDOMTimeout = setTimeout(function () {
                  if (((n.readDOMTimeout = null), n.composing))
                    if (n.composing.done) n.composing = null;
                    else return;
                  n.updateFromDOM();
                }, 80));
            }),
            (he.prototype.updateFromDOM = function () {
              var n = this;
              (this.cm.isReadOnly() || !this.pollContent()) &&
                Sn(this.cm, function () {
                  return ln(n.cm);
                });
            }),
            (he.prototype.setUneditable = function (n) {
              n.contentEditable = "false";
            }),
            (he.prototype.onKeyPress = function (n) {
              n.charCode == 0 ||
                this.composing ||
                (n.preventDefault(),
                this.cm.isReadOnly() ||
                  He(this.cm, qu)(
                    this.cm,
                    String.fromCharCode(n.charCode == null ? n.keyCode : n.charCode),
                    0,
                  ));
            }),
            (he.prototype.readOnlyChanged = function (n) {
              this.div.contentEditable = String(n != "nocursor");
            }),
            (he.prototype.onContextMenu = function () {}),
            (he.prototype.resetPosition = function () {}),
            (he.prototype.needsContentAttribute = !0);
          function cg(n, i) {
            var a = au(n, i.line);
            if (!a || a.hidden) return null;
            var l = Pt(n.doc, i.line),
              c = Id(a, l, i.line),
              p = Yt(l, n.doc.direction),
              m = "left";
            if (p) {
              var y = Ae(p, i.ch);
              m = y % 2 ? "right" : "left";
            }
            var x = Hd(c.map, i.ch, m);
            return (x.offset = x.collapse == "right" ? x.end : x.start), x;
          }
          function Wx(n) {
            for (var i = n; i; i = i.parentNode)
              if (/CodeMirror-gutter-wrapper/.test(i.className)) return !0;
            return !1;
          }
          function zo(n, i) {
            return i && (n.bad = !0), n;
          }
          function Ux(n, i, a, l, c) {
            var p = "",
              m = !1,
              y = n.doc.lineSeparator(),
              x = !1;
            function _(q) {
              return function (Z) {
                return Z.id == q;
              };
            }
            function N() {
              m && ((p += y), x && (p += y), (m = x = !1));
            }
            function D(q) {
              q && (N(), (p += q));
            }
            function W(q) {
              if (q.nodeType == 1) {
                var Z = q.getAttribute("cm-text");
                if (Z) {
                  D(Z);
                  return;
                }
                var it = q.getAttribute("cm-marker"),
                  vt;
                if (it) {
                  var bt = n.findMarks(X(l, 0), X(c + 1, 0), _(+it));
                  bt.length && (vt = bt[0].find(0)) && D(Tr(n.doc, vt.from, vt.to).join(y));
                  return;
                }
                if (q.getAttribute("contenteditable") == "false") return;
                var Ct = /^(pre|div|p|li|table|br)$/i.test(q.nodeName);
                if (!/^br$/i.test(q.nodeName) && q.textContent.length == 0) return;
                Ct && N();
                for (var wt = 0; wt < q.childNodes.length; wt++) W(q.childNodes[wt]);
                /^(pre|p)$/i.test(q.nodeName) && (x = !0), Ct && (m = !0);
              } else
                q.nodeType == 3 && D(q.nodeValue.replace(/\u200b/g, "").replace(/\u00a0/g, " "));
            }
            for (; W(i), i != a; ) (i = i.nextSibling), (x = !1);
            return p;
          }
          function ya(n, i, a) {
            var l;
            if (i == n.display.lineDiv) {
              if (((l = n.display.lineDiv.childNodes[a]), !l))
                return zo(n.clipPos(X(n.display.viewTo - 1)), !0);
              (i = null), (a = 0);
            } else
              for (l = i; ; l = l.parentNode) {
                if (!l || l == n.display.lineDiv) return null;
                if (l.parentNode && l.parentNode == n.display.lineDiv) break;
              }
            for (var c = 0; c < n.display.view.length; c++) {
              var p = n.display.view[c];
              if (p.node == l) return jx(p, i, a);
            }
          }
          function jx(n, i, a) {
            var l = n.text.firstChild,
              c = !1;
            if (!i || !J(l, i)) return zo(X(C(n.line), 0), !0);
            if (i == l && ((c = !0), (i = l.childNodes[a]), (a = 0), !i)) {
              var p = n.rest ? ct(n.rest) : n.line;
              return zo(X(C(p), p.text.length), c);
            }
            var m = i.nodeType == 3 ? i : null,
              y = i;
            for (
              !m &&
              i.childNodes.length == 1 &&
              i.firstChild.nodeType == 3 &&
              ((m = i.firstChild), a && (a = m.nodeValue.length));
              y.parentNode != l;
            )
              y = y.parentNode;
            var x = n.measure,
              _ = x.maps;
            function N(vt, bt, Ct) {
              for (var wt = -1; wt < (_ ? _.length : 0); wt++)
                for (var Lt = wt < 0 ? x.map : _[wt], zt = 0; zt < Lt.length; zt += 3) {
                  var Ot = Lt[zt + 2];
                  if (Ot == vt || Ot == bt) {
                    var Xt = C(wt < 0 ? n.line : n.rest[wt]),
                      ge = Lt[zt] + Ct;
                    return (Ct < 0 || Ot != vt) && (ge = Lt[zt + (Ct ? 1 : 0)]), X(Xt, ge);
                  }
                }
            }
            var D = N(m, y, a);
            if (D) return zo(D, c);
            for (var W = y.nextSibling, q = m ? m.nodeValue.length - a : 0; W; W = W.nextSibling) {
              if (((D = N(W, W.firstChild, 0)), D)) return zo(X(D.line, D.ch - q), c);
              q += W.textContent.length;
            }
            for (var Z = y.previousSibling, it = a; Z; Z = Z.previousSibling) {
              if (((D = N(Z, Z.firstChild, -1)), D)) return zo(X(D.line, D.ch + it), c);
              it += Z.textContent.length;
            }
          }
          var Me = function (n) {
            (this.cm = n),
              (this.prevInput = ""),
              (this.pollingFast = !1),
              (this.polling = new Mt()),
              (this.hasSelection = !1),
              (this.composing = null),
              (this.resetting = !1);
          };
          (Me.prototype.init = function (n) {
            var i = this,
              a = this,
              l = this.cm;
            this.createField(n);
            var c = this.textarea;
            n.wrapper.insertBefore(this.wrapper, n.wrapper.firstChild),
              M && (c.style.width = "0px"),
              Rt(c, "input", function () {
                d && g >= 9 && i.hasSelection && (i.hasSelection = null), a.poll();
              }),
              Rt(c, "paste", function (m) {
                Ce(l, m) || ig(m, l) || ((l.state.pasteIncoming = +new Date()), a.fastPoll());
              });
            function p(m) {
              if (!Ce(l, m)) {
                if (l.somethingSelected()) ma({ lineWise: !1, text: l.getSelections() });
                else if (l.options.lineWiseCopyCut) {
                  var y = sg(l);
                  ma({ lineWise: !0, text: y.text }),
                    m.type == "cut"
                      ? l.setSelections(y.ranges, null, V)
                      : ((a.prevInput = ""),
                        (c.value = y.text.join(`
`)),
                        Ht(c));
                } else return;
                m.type == "cut" && (l.state.cutIncoming = +new Date());
              }
            }
            Rt(c, "cut", p),
              Rt(c, "copy", p),
              Rt(n.scroller, "paste", function (m) {
                if (!(Ar(n, m) || Ce(l, m))) {
                  if (!c.dispatchEvent) {
                    (l.state.pasteIncoming = +new Date()), a.focus();
                    return;
                  }
                  var y = new Event("paste");
                  (y.clipboardData = m.clipboardData), c.dispatchEvent(y);
                }
              }),
              Rt(n.lineSpace, "selectstart", function (m) {
                Ar(n, m) || Xe(m);
              }),
              Rt(c, "compositionstart", function () {
                var m = l.getCursor("from");
                a.composing && a.composing.range.clear(),
                  (a.composing = {
                    start: m,
                    range: l.markText(m, l.getCursor("to"), { className: "CodeMirror-composing" }),
                  });
              }),
              Rt(c, "compositionend", function () {
                a.composing && (a.poll(), a.composing.range.clear(), (a.composing = null));
              });
          }),
            (Me.prototype.createField = function (n) {
              (this.wrapper = lg()), (this.textarea = this.wrapper.firstChild);
              var i = this.cm.options;
              Hu(this.textarea, i.spellcheck, i.autocorrect, i.autocapitalize);
            }),
            (Me.prototype.screenReaderLabelChanged = function (n) {
              n
                ? this.textarea.setAttribute("aria-label", n)
                : this.textarea.removeAttribute("aria-label");
            }),
            (Me.prototype.prepareSelection = function () {
              var n = this.cm,
                i = n.display,
                a = n.doc,
                l = Jd(n);
              if (n.options.moveInputWithCursor) {
                var c = Jn(n, a.sel.primary().head, "div"),
                  p = i.wrapper.getBoundingClientRect(),
                  m = i.lineDiv.getBoundingClientRect();
                (l.teTop = Math.max(
                  0,
                  Math.min(i.wrapper.clientHeight - 10, c.top + m.top - p.top),
                )),
                  (l.teLeft = Math.max(
                    0,
                    Math.min(i.wrapper.clientWidth - 10, c.left + m.left - p.left),
                  ));
              }
              return l;
            }),
            (Me.prototype.showSelection = function (n) {
              var i = this.cm,
                a = i.display;
              z(a.cursorDiv, n.cursors),
                z(a.selectionDiv, n.selection),
                n.teTop != null &&
                  ((this.wrapper.style.top = n.teTop + "px"),
                  (this.wrapper.style.left = n.teLeft + "px"));
            }),
            (Me.prototype.reset = function (n) {
              if (!(this.contextMenuPending || (this.composing && n))) {
                var i = this.cm;
                if (((this.resetting = !0), i.somethingSelected())) {
                  this.prevInput = "";
                  var a = i.getSelection();
                  (this.textarea.value = a),
                    i.state.focused && Ht(this.textarea),
                    d && g >= 9 && (this.hasSelection = a);
                } else
                  n ||
                    ((this.prevInput = this.textarea.value = ""),
                    d && g >= 9 && (this.hasSelection = null));
                this.resetting = !1;
              }
            }),
            (Me.prototype.getField = function () {
              return this.textarea;
            }),
            (Me.prototype.supportsTouch = function () {
              return !1;
            }),
            (Me.prototype.focus = function () {
              if (
                this.cm.options.readOnly != "nocursor" &&
                (!E || yt(Gt(this.textarea)) != this.textarea)
              )
                try {
                  this.textarea.focus();
                } catch {}
            }),
            (Me.prototype.blur = function () {
              this.textarea.blur();
            }),
            (Me.prototype.resetPosition = function () {
              this.wrapper.style.top = this.wrapper.style.left = 0;
            }),
            (Me.prototype.receivedFocus = function () {
              this.slowPoll();
            }),
            (Me.prototype.slowPoll = function () {
              var n = this;
              this.pollingFast ||
                this.polling.set(this.cm.options.pollInterval, function () {
                  n.poll(), n.cm.state.focused && n.slowPoll();
                });
            }),
            (Me.prototype.fastPoll = function () {
              var n = !1,
                i = this;
              i.pollingFast = !0;
              function a() {
                var l = i.poll();
                !l && !n ? ((n = !0), i.polling.set(60, a)) : ((i.pollingFast = !1), i.slowPoll());
              }
              i.polling.set(20, a);
            }),
            (Me.prototype.poll = function () {
              var n = this,
                i = this.cm,
                a = this.textarea,
                l = this.prevInput;
              if (
                this.contextMenuPending ||
                this.resetting ||
                !i.state.focused ||
                (Jr(a) && !l && !this.composing) ||
                i.isReadOnly() ||
                i.options.disableInput ||
                i.state.keySeq
              )
                return !1;
              var c = a.value;
              if (c == l && !i.somethingSelected()) return !1;
              if ((d && g >= 9 && this.hasSelection === c) || (B && /[\uf700-\uf7ff]/.test(c)))
                return i.display.input.reset(), !1;
              if (i.doc.sel == i.display.selForContextMenu) {
                var p = c.charCodeAt(0);
                if ((p == 8203 && !l && (l = "​"), p == 8666))
                  return this.reset(), this.cm.execCommand("undo");
              }
              for (
                var m = 0, y = Math.min(l.length, c.length);
                m < y && l.charCodeAt(m) == c.charCodeAt(m);
              )
                ++m;
              return (
                Sn(i, function () {
                  qu(i, c.slice(m), l.length - m, null, n.composing ? "*compose" : null),
                    c.length > 1e3 ||
                    c.indexOf(`
`) > -1
                      ? (a.value = n.prevInput = "")
                      : (n.prevInput = c),
                    n.composing &&
                      (n.composing.range.clear(),
                      (n.composing.range = i.markText(n.composing.start, i.getCursor("to"), {
                        className: "CodeMirror-composing",
                      })));
                }),
                !0
              );
            }),
            (Me.prototype.ensurePolled = function () {
              this.pollingFast && this.poll() && (this.pollingFast = !1);
            }),
            (Me.prototype.onKeyPress = function () {
              d && g >= 9 && (this.hasSelection = null), this.fastPoll();
            }),
            (Me.prototype.onContextMenu = function (n) {
              var i = this,
                a = i.cm,
                l = a.display,
                c = i.textarea;
              i.contextMenuPending && i.contextMenuPending();
              var p = $i(a, n),
                m = l.scroller.scrollTop;
              if (!p || P) return;
              var y = a.options.resetSelectionOnContextMenu;
              y && a.doc.sel.contains(p) == -1 && He(a, Ye)(a.doc, ri(p), V);
              var x = c.style.cssText,
                _ = i.wrapper.style.cssText,
                N = i.wrapper.offsetParent.getBoundingClientRect();
              (i.wrapper.style.cssText = "position: static"),
                (c.style.cssText =
                  `position: absolute; width: 30px; height: 30px;
      top: ` +
                  (n.clientY - N.top - 5) +
                  "px; left: " +
                  (n.clientX - N.left - 5) +
                  `px;
      z-index: 1000; background: ` +
                  (d ? "rgba(255, 255, 255, .05)" : "transparent") +
                  `;
      outline: none; border-width: 0; outline: none; overflow: hidden; opacity: .05; filter: alpha(opacity=5);`);
              var D;
              v && (D = c.ownerDocument.defaultView.scrollY),
                l.input.focus(),
                v && c.ownerDocument.defaultView.scrollTo(null, D),
                l.input.reset(),
                a.somethingSelected() || (c.value = i.prevInput = " "),
                (i.contextMenuPending = q),
                (l.selForContextMenu = a.doc.sel),
                clearTimeout(l.detectingSelectAll);
              function W() {
                if (c.selectionStart != null) {
                  var it = a.somethingSelected(),
                    vt = "​" + (it ? c.value : "");
                  (c.value = "⇚"),
                    (c.value = vt),
                    (i.prevInput = it ? "" : "​"),
                    (c.selectionStart = 1),
                    (c.selectionEnd = vt.length),
                    (l.selForContextMenu = a.doc.sel);
                }
              }
              function q() {
                if (
                  i.contextMenuPending == q &&
                  ((i.contextMenuPending = !1),
                  (i.wrapper.style.cssText = _),
                  (c.style.cssText = x),
                  d && g < 9 && l.scrollbars.setScrollTop((l.scroller.scrollTop = m)),
                  c.selectionStart != null)
                ) {
                  (!d || (d && g < 9)) && W();
                  var it = 0,
                    vt = function () {
                      l.selForContextMenu == a.doc.sel &&
                      c.selectionStart == 0 &&
                      c.selectionEnd > 0 &&
                      i.prevInput == "​"
                        ? He(a, Lp)(a)
                        : it++ < 10
                        ? (l.detectingSelectAll = setTimeout(vt, 500))
                        : ((l.selForContextMenu = null), l.input.reset());
                    };
                  l.detectingSelectAll = setTimeout(vt, 200);
                }
              }
              if ((d && g >= 9 && W(), at)) {
                Yr(n);
                var Z = function () {
                  Ke(window, "mouseup", Z), setTimeout(q, 20);
                };
                Rt(window, "mouseup", Z);
              } else setTimeout(q, 50);
            }),
            (Me.prototype.readOnlyChanged = function (n) {
              n || this.reset(),
                (this.textarea.disabled = n == "nocursor"),
                (this.textarea.readOnly = !!n);
            }),
            (Me.prototype.setUneditable = function () {}),
            (Me.prototype.needsContentAttribute = !1);
          function Vx(n, i) {
            if (
              ((i = i ? rt(i) : {}),
              (i.value = n.value),
              !i.tabindex && n.tabIndex && (i.tabindex = n.tabIndex),
              !i.placeholder && n.placeholder && (i.placeholder = n.placeholder),
              i.autofocus == null)
            ) {
              var a = yt(Gt(n));
              i.autofocus = a == n || (n.getAttribute("autofocus") != null && a == document.body);
            }
            function l() {
              n.value = y.getValue();
            }
            var c;
            if (n.form && (Rt(n.form, "submit", l), !i.leaveSubmitMethodAlone)) {
              var p = n.form;
              c = p.submit;
              try {
                var m = (p.submit = function () {
                  l(), (p.submit = c), p.submit(), (p.submit = m);
                });
              } catch {}
            }
            (i.finishInit = function (x) {
              (x.save = l),
                (x.getTextArea = function () {
                  return n;
                }),
                (x.toTextArea = function () {
                  (x.toTextArea = isNaN),
                    l(),
                    n.parentNode.removeChild(x.getWrapperElement()),
                    (n.style.display = ""),
                    n.form &&
                      (Ke(n.form, "submit", l),
                      !i.leaveSubmitMethodAlone &&
                        typeof n.form.submit == "function" &&
                        (n.form.submit = c));
                });
            }),
              (n.style.display = "none");
            var y = be(function (x) {
              return n.parentNode.insertBefore(x, n.nextSibling);
            }, i);
            return y;
          }
          function Gx(n) {
            (n.off = Ke),
              (n.on = Rt),
              (n.wheelEventPixels = tx),
              (n.Doc = an),
              (n.splitLines = Hn),
              (n.countColumn = lt),
              (n.findColumn = ut),
              (n.isWordChar = Kt),
              (n.Pass = I),
              (n.signal = ke),
              (n.Line = xo),
              (n.changeEnd = ii),
              (n.scrollbarModel = ip),
              (n.Pos = X),
              (n.cmpPos = _t),
              (n.modes = go),
              (n.mimeModes = Xn),
              (n.resolveMode = vo),
              (n.getMode = mo),
              (n.modeExtensions = Qr),
              (n.extendMode = yo),
              (n.copyState = ur),
              (n.startState = bo),
              (n.innerMode = _s),
              (n.commands = Vs),
              (n.keyMap = Nr),
              (n.keyName = Up),
              (n.isModifierKey = Bp),
              (n.lookupKey = Do),
              (n.normalizeKeyMap = _x),
              (n.StringStream = Te),
              (n.SharedTextMarker = Ws),
              (n.TextMarker = si),
              (n.LineWidget = Bs),
              (n.e_preventDefault = Xe),
              (n.e_stopPropagation = ho),
              (n.e_stop = Yr),
              (n.addClass = At),
              (n.contains = J),
              (n.rmClass = gt),
              (n.keyNames = li);
          }
          Ix(be), Bx(be);
          var Kx = "iter insert remove copy getEditor constructor".split(" ");
          for (var ba in an.prototype)
            an.prototype.hasOwnProperty(ba) &&
              Et(Kx, ba) < 0 &&
              (be.prototype[ba] = (function (n) {
                return function () {
                  return n.apply(this.doc, arguments);
                };
              })(an.prototype[ba]));
          return (
            Vn(an),
            (be.inputStyles = { textarea: Me, contenteditable: he }),
            (be.defineMode = function (n) {
              !be.defaults.mode && n != "null" && (be.defaults.mode = n), Yn.apply(this, arguments);
            }),
            (be.defineMIME = Pi),
            be.defineMode("null", function () {
              return {
                token: function (n) {
                  return n.skipToEnd();
                },
              };
            }),
            be.defineMIME("text/plain", "null"),
            (be.defineExtension = function (n, i) {
              be.prototype[n] = i;
            }),
            (be.defineDocExtension = function (n, i) {
              an.prototype[n] = i;
            }),
            (be.fromTextArea = Vx),
            Gx(be),
            (be.version = "5.65.16"),
            be
          );
        });
      })(uf)),
    uf.exports
  );
}
var Rat = ys();
const zat = hy(Rat);
var Jv = { exports: {} },
  Qv;
function ab() {
  return (
    Qv ||
      ((Qv = 1),
      (function (t, e) {
        (function (r) {
          r(ys());
        })(function (r) {
          r.defineMode("javascript", function (o, s) {
            var u = o.indentUnit,
              f = s.statementIndent,
              h = s.jsonld,
              d = s.json || h,
              g = s.trackScope !== !1,
              v = s.typescript,
              b = s.wordCharacters || /[\w$\xa1-\uffff]/,
              w = (function () {
                function C(Fe) {
                  return { type: Fe, style: "keyword" };
                }
                var O = C("keyword a"),
                  et = C("keyword b"),
                  dt = C("keyword c"),
                  X = C("keyword d"),
                  _t = C("operator"),
                  ce = { type: "atom", style: "atom" };
                return {
                  if: C("if"),
                  while: O,
                  with: O,
                  else: et,
                  do: et,
                  try: et,
                  finally: et,
                  return: X,
                  break: X,
                  continue: X,
                  new: C("new"),
                  delete: dt,
                  void: dt,
                  throw: dt,
                  debugger: C("debugger"),
                  var: C("var"),
                  const: C("var"),
                  let: C("var"),
                  function: C("function"),
                  catch: C("catch"),
                  for: C("for"),
                  switch: C("switch"),
                  case: C("case"),
                  default: C("default"),
                  in: _t,
                  typeof: _t,
                  instanceof: _t,
                  true: ce,
                  false: ce,
                  null: ce,
                  undefined: ce,
                  NaN: ce,
                  Infinity: ce,
                  this: C("this"),
                  class: C("class"),
                  super: C("atom"),
                  yield: dt,
                  export: C("export"),
                  import: C("import"),
                  extends: dt,
                  await: dt,
                };
              })(),
              S = /[+\-*&%=<>!?|~^@]/,
              P =
                /^@(context|id|value|language|type|container|list|set|reverse|index|base|vocab|graph)"/;
            function A(C) {
              for (var O = !1, et, dt = !1; (et = C.next()) != null; ) {
                if (!O) {
                  if (et == "/" && !dt) return;
                  et == "[" ? (dt = !0) : dt && et == "]" && (dt = !1);
                }
                O = !O && et == "\\";
              }
            }
            var L, T;
            function M(C, O, et) {
              return (L = C), (T = et), O;
            }
            function R(C, O) {
              var et = C.next();
              if (et == '"' || et == "'") return (O.tokenize = E(et)), O.tokenize(C, O);
              if (et == "." && C.match(/^\d[\d_]*(?:[eE][+\-]?[\d_]+)?/))
                return M("number", "number");
              if (et == "." && C.match("..")) return M("spread", "meta");
              if (/[\[\]{}\(\),;\:\.]/.test(et)) return M(et);
              if (et == "=" && C.eat(">")) return M("=>", "operator");
              if (et == "0" && C.match(/^(?:x[\dA-Fa-f_]+|o[0-7_]+|b[01_]+)n?/))
                return M("number", "number");
              if (/\d/.test(et))
                return (
                  C.match(/^[\d_]*(?:n|(?:\.[\d_]*)?(?:[eE][+\-]?[\d_]+)?)?/), M("number", "number")
                );
              if (et == "/")
                return C.eat("*")
                  ? ((O.tokenize = B), B(C, O))
                  : C.eat("/")
                  ? (C.skipToEnd(), M("comment", "comment"))
                  : On(C, O, 1)
                  ? (A(C), C.match(/^\b(([gimyus])(?![gimyus]*\2))+\b/), M("regexp", "string-2"))
                  : (C.eat("="), M("operator", "operator", C.current()));
              if (et == "`") return (O.tokenize = K), K(C, O);
              if (et == "#" && C.peek() == "!") return C.skipToEnd(), M("meta", "meta");
              if (et == "#" && C.eatWhile(b)) return M("variable", "property");
              if (
                (et == "<" && C.match("!--")) ||
                (et == "-" && C.match("->") && !/\S/.test(C.string.slice(0, C.start)))
              )
                return C.skipToEnd(), M("comment", "comment");
              if (S.test(et))
                return (
                  (et != ">" || !O.lexical || O.lexical.type != ">") &&
                    (C.eat("=")
                      ? (et == "!" || et == "=") && C.eat("=")
                      : /[<>*+\-|&?]/.test(et) && (C.eat(et), et == ">" && C.eat(et))),
                  et == "?" && C.eat(".") ? M(".") : M("operator", "operator", C.current())
                );
              if (b.test(et)) {
                C.eatWhile(b);
                var dt = C.current();
                if (O.lastType != ".") {
                  if (w.propertyIsEnumerable(dt)) {
                    var X = w[dt];
                    return M(X.type, X.style, dt);
                  }
                  if (dt == "async" && C.match(/^(\s|\/\*([^*]|\*(?!\/))*?\*\/)*[\[\(\w]/, !1))
                    return M("async", "keyword", dt);
                }
                return M("variable", "variable", dt);
              }
            }
            function E(C) {
              return function (O, et) {
                var dt = !1,
                  X;
                if (h && O.peek() == "@" && O.match(P))
                  return (et.tokenize = R), M("jsonld-keyword", "meta");
                for (; (X = O.next()) != null && !(X == C && !dt); ) dt = !dt && X == "\\";
                return dt || (et.tokenize = R), M("string", "string");
              };
            }
            function B(C, O) {
              for (var et = !1, dt; (dt = C.next()); ) {
                if (dt == "/" && et) {
                  O.tokenize = R;
                  break;
                }
                et = dt == "*";
              }
              return M("comment", "comment");
            }
            function K(C, O) {
              for (var et = !1, dt; (dt = C.next()) != null; ) {
                if (!et && (dt == "`" || (dt == "$" && C.eat("{")))) {
                  O.tokenize = R;
                  break;
                }
                et = !et && dt == "\\";
              }
              return M("quasi", "string-2", C.current());
            }
            var ht = "([{}])";
            function Y(C, O) {
              O.fatArrowAt && (O.fatArrowAt = null);
              var et = C.string.indexOf("=>", C.start);
              if (!(et < 0)) {
                if (v) {
                  var dt = /:\s*(?:\w+(?:<[^>]*>|\[\])?|\{[^}]*\})\s*$/.exec(
                    C.string.slice(C.start, et),
                  );
                  dt && (et = dt.index);
                }
                for (var X = 0, _t = !1, ce = et - 1; ce >= 0; --ce) {
                  var Fe = C.string.charAt(ce),
                    sn = ht.indexOf(Fe);
                  if (sn >= 0 && sn < 3) {
                    if (!X) {
                      ++ce;
                      break;
                    }
                    if (--X == 0) {
                      Fe == "(" && (_t = !0);
                      break;
                    }
                  } else if (sn >= 3 && sn < 6) ++X;
                  else if (b.test(Fe)) _t = !0;
                  else if (/["'\/`]/.test(Fe))
                    for (; ; --ce) {
                      if (ce == 0) return;
                      var wo = C.string.charAt(ce - 1);
                      if (wo == Fe && C.string.charAt(ce - 2) != "\\") {
                        ce--;
                        break;
                      }
                    }
                  else if (_t && !X) {
                    ++ce;
                    break;
                  }
                }
                _t && !X && (O.fatArrowAt = ce);
              }
            }
            var nt = {
              atom: !0,
              number: !0,
              variable: !0,
              string: !0,
              regexp: !0,
              this: !0,
              import: !0,
              "jsonld-keyword": !0,
            };
            function at(C, O, et, dt, X, _t) {
              (this.indented = C),
                (this.column = O),
                (this.type = et),
                (this.prev = X),
                (this.info = _t),
                dt != null && (this.align = dt);
            }
            function pt(C, O) {
              if (!g) return !1;
              for (var et = C.localVars; et; et = et.next) if (et.name == O) return !0;
              for (var dt = C.context; dt; dt = dt.prev)
                for (var et = dt.vars; et; et = et.next) if (et.name == O) return !0;
            }
            function gt(C, O, et, dt, X) {
              var _t = C.cc;
              for (
                G.state = C,
                  G.stream = X,
                  G.marked = null,
                  G.cc = _t,
                  G.style = O,
                  C.lexical.hasOwnProperty("align") || (C.lexical.align = !0);
                ;
              ) {
                var ce = _t.length ? _t.pop() : d ? Et : lt;
                if (ce(et, dt)) {
                  for (; _t.length && _t[_t.length - 1].lex; ) _t.pop()();
                  return G.marked ? G.marked : et == "variable" && pt(C, dt) ? "variable-2" : O;
                }
              }
            }
            var G = { state: null, column: null, marked: null, cc: null };
            function z() {
              for (var C = arguments.length - 1; C >= 0; C--) G.cc.push(arguments[C]);
            }
            function k() {
              return z.apply(null, arguments), !0;
            }
            function F(C, O) {
              for (var et = O; et; et = et.next) if (et.name == C) return !0;
              return !1;
            }
            function H(C) {
              var O = G.state;
              if (((G.marked = "def"), !!g)) {
                if (O.context) {
                  if (O.lexical.info == "var" && O.context && O.context.block) {
                    var et = J(C, O.context);
                    if (et != null) {
                      O.context = et;
                      return;
                    }
                  } else if (!F(C, O.localVars)) {
                    O.localVars = new qt(C, O.localVars);
                    return;
                  }
                }
                s.globalVars && !F(C, O.globalVars) && (O.globalVars = new qt(C, O.globalVars));
              }
            }
            function J(C, O) {
              if (O)
                if (O.block) {
                  var et = J(C, O.prev);
                  return et ? (et == O.prev ? O : new At(et, O.vars, !0)) : null;
                } else return F(C, O.vars) ? O : new At(O.prev, new qt(C, O.vars), !1);
              else return null;
            }
            function yt(C) {
              return (
                C == "public" ||
                C == "private" ||
                C == "protected" ||
                C == "abstract" ||
                C == "readonly"
              );
            }
            function At(C, O, et) {
              (this.prev = C), (this.vars = O), (this.block = et);
            }
            function qt(C, O) {
              (this.name = C), (this.next = O);
            }
            var Ht = new qt("this", new qt("arguments", null));
            function Qt() {
              (G.state.context = new At(G.state.context, G.state.localVars, !1)),
                (G.state.localVars = Ht);
            }
            function Jt() {
              (G.state.context = new At(G.state.context, G.state.localVars, !0)),
                (G.state.localVars = null);
            }
            Qt.lex = Jt.lex = !0;
            function Gt() {
              (G.state.localVars = G.state.context.vars), (G.state.context = G.state.context.prev);
            }
            Gt.lex = !0;
            function Tt(C, O) {
              var et = function () {
                var dt = G.state,
                  X = dt.indented;
                if (dt.lexical.type == "stat") X = dt.lexical.indented;
                else
                  for (var _t = dt.lexical; _t && _t.type == ")" && _t.align; _t = _t.prev)
                    X = _t.indented;
                dt.lexical = new at(X, G.stream.column(), C, null, dt.lexical, O);
              };
              return (et.lex = !0), et;
            }
            function j() {
              var C = G.state;
              C.lexical.prev &&
                (C.lexical.type == ")" && (C.indented = C.lexical.indented),
                (C.lexical = C.lexical.prev));
            }
            j.lex = !0;
            function rt(C) {
              function O(et) {
                return et == C ? k() : C == ";" || et == "}" || et == ")" || et == "]" ? z() : k(O);
              }
              return O;
            }
            function lt(C, O) {
              return C == "var"
                ? k(Tt("vardef", O), ho, rt(";"), j)
                : C == "keyword a"
                ? k(Tt("form"), I, lt, j)
                : C == "keyword b"
                ? k(Tt("form"), lt, j)
                : C == "keyword d"
                ? G.stream.match(/^\s*$/, !1)
                  ? k()
                  : k(Tt("stat"), Q, rt(";"), j)
                : C == "debugger"
                ? k(rt(";"))
                : C == "{"
                ? k(Tt("}"), Jt, Pn, j, Gt)
                : C == ";"
                ? k()
                : C == "if"
                ? (G.state.lexical.info == "else" &&
                    G.state.cc[G.state.cc.length - 1] == j &&
                    G.state.cc.pop()(),
                  k(Tt("form"), I, lt, j, po))
                : C == "function"
                ? k(Hn)
                : C == "for"
                ? k(Tt("form"), Jt, Bl, lt, Gt, j)
                : C == "class" || (v && O == "interface")
                ? ((G.marked = "keyword"), k(Tt("form", C == "class" ? C : O), go, j))
                : C == "variable"
                ? v && O == "declare"
                  ? ((G.marked = "keyword"), k(lt))
                  : v &&
                    (O == "module" || O == "enum" || O == "type") &&
                    G.stream.match(/^\s*\w/, !1)
                  ? ((G.marked = "keyword"),
                    O == "enum"
                      ? k(Pt)
                      : O == "type"
                      ? k(Wl, rt("operator"), Yt, rt(";"))
                      : k(Tt("form"), on, rt("{"), Tt("}"), Pn, j, j))
                  : v && O == "namespace"
                  ? ((G.marked = "keyword"), k(Tt("form"), Et, lt, j))
                  : v && O == "abstract"
                  ? ((G.marked = "keyword"), k(lt))
                  : k(Tt("stat"), Bt)
                : C == "switch"
                ? k(Tt("form"), I, rt("{"), Tt("}", "switch"), Jt, Pn, j, j, Gt)
                : C == "case"
                ? k(Et, rt(":"))
                : C == "default"
                ? k(rt(":"))
                : C == "catch"
                ? k(Tt("form"), Qt, Mt, lt, j, Gt)
                : C == "export"
                ? k(Tt("stat"), vo, j)
                : C == "import"
                ? k(Tt("stat"), Qr, j)
                : C == "async"
                ? k(lt)
                : O == "@"
                ? k(Et, lt)
                : z(Tt("stat"), Et, rt(";"), j);
            }
            function Mt(C) {
              if (C == "(") return k(Kn, rt(")"));
            }
            function Et(C, O) {
              return V(C, O, !1);
            }
            function $(C, O) {
              return V(C, O, !0);
            }
            function I(C) {
              return C != "(" ? z() : k(Tt(")"), Q, rt(")"), j);
            }
            function V(C, O, et) {
              if (G.state.fatArrowAt == G.stream.start) {
                var dt = et ? ft : ct;
                if (C == "(") return k(Qt, Tt(")"), se(Kn, ")"), j, rt("=>"), dt, Gt);
                if (C == "variable") return z(Qt, on, rt("=>"), dt, Gt);
              }
              var X = et ? ut : ot;
              return nt.hasOwnProperty(C)
                ? k(X)
                : C == "function"
                ? k(Hn, X)
                : C == "class" || (v && O == "interface")
                ? ((G.marked = "keyword"), k(Tt("form"), Zc, j))
                : C == "keyword c" || C == "async"
                ? k(et ? $ : Et)
                : C == "("
                ? k(Tt(")"), Q, rt(")"), j, X)
                : C == "operator" || C == "spread"
                ? k(et ? $ : Et)
                : C == "["
                ? k(Tt("]"), Te, j, X)
                : C == "{"
                ? rn(re, "}", null, X)
                : C == "quasi"
                ? z(St, X)
                : C == "new"
                ? k($t(et))
                : k();
            }
            function Q(C) {
              return C.match(/[;\}\)\],]/) ? z() : z(Et);
            }
            function ot(C, O) {
              return C == "," ? k(Q) : ut(C, O, !1);
            }
            function ut(C, O, et) {
              var dt = et == !1 ? ot : ut,
                X = et == !1 ? Et : $;
              if (C == "=>") return k(Qt, et ? ft : ct, Gt);
              if (C == "operator")
                return /\+\+|--/.test(O) || (v && O == "!")
                  ? k(dt)
                  : v && O == "<" && G.stream.match(/^([^<>]|<[^<>]*>)*>\s*\(/, !1)
                  ? k(Tt(">"), se(Yt, ">"), j, dt)
                  : O == "?"
                  ? k(Et, rt(":"), X)
                  : k(X);
              if (C == "quasi") return z(St, dt);
              if (C != ";") {
                if (C == "(") return rn($, ")", "call", dt);
                if (C == ".") return k(Kt, dt);
                if (C == "[") return k(Tt("]"), Q, rt("]"), j, dt);
                if (v && O == "as") return (G.marked = "keyword"), k(Yt, dt);
                if (C == "regexp")
                  return (
                    (G.state.lastType = G.marked = "operator"),
                    G.stream.backUp(G.stream.pos - G.stream.start - 1),
                    k(X)
                  );
              }
            }
            function St(C, O) {
              return C != "quasi" ? z() : O.slice(O.length - 2) != "${" ? k(St) : k(Q, mt);
            }
            function mt(C) {
              if (C == "}") return (G.marked = "string-2"), (G.state.tokenize = K), k(St);
            }
            function ct(C) {
              return Y(G.stream, G.state), z(C == "{" ? lt : Et);
            }
            function ft(C) {
              return Y(G.stream, G.state), z(C == "{" ? lt : $);
            }
            function $t(C) {
              return function (O) {
                return O == "."
                  ? k(C ? Dt : Nt)
                  : O == "variable" && v
                  ? k(_n, C ? ut : ot)
                  : z(C ? $ : Et);
              };
            }
            function Nt(C, O) {
              if (O == "target") return (G.marked = "keyword"), k(ot);
            }
            function Dt(C, O) {
              if (O == "target") return (G.marked = "keyword"), k(ut);
            }
            function Bt(C) {
              return C == ":" ? k(j, lt) : z(ot, rt(";"), j);
            }
            function Kt(C) {
              if (C == "variable") return (G.marked = "property"), k();
            }
            function re(C, O) {
              if (C == "async") return (G.marked = "property"), k(re);
              if (C == "variable" || G.style == "keyword") {
                if (((G.marked = "property"), O == "get" || O == "set")) return k(oe);
                var et;
                return (
                  v &&
                    G.state.fatArrowAt == G.stream.start &&
                    (et = G.stream.match(/^\s*:\s*/, !1)) &&
                    (G.state.fatArrowAt = G.stream.pos + et[0].length),
                  k(fe)
                );
              } else {
                if (C == "number" || C == "string")
                  return (G.marked = h ? "property" : G.style + " property"), k(fe);
                if (C == "jsonld-keyword") return k(fe);
                if (v && yt(O)) return (G.marked = "keyword"), k(re);
                if (C == "[") return k(Et, wn, rt("]"), fe);
                if (C == "spread") return k($, fe);
                if (O == "*") return (G.marked = "keyword"), k(re);
                if (C == ":") return z(fe);
              }
            }
            function oe(C) {
              return C != "variable" ? z(fe) : ((G.marked = "property"), k(Hn));
            }
            function fe(C) {
              if (C == ":") return k($);
              if (C == "(") return z(Hn);
            }
            function se(C, O, et) {
              function dt(X, _t) {
                if (et ? et.indexOf(X) > -1 : X == ",") {
                  var ce = G.state.lexical;
                  return (
                    ce.info == "call" && (ce.pos = (ce.pos || 0) + 1),
                    k(function (Fe, sn) {
                      return Fe == O || sn == O ? z() : z(C);
                    }, dt)
                  );
                }
                return X == O || _t == O ? k() : et && et.indexOf(";") > -1 ? z(C) : k(rt(O));
              }
              return function (X, _t) {
                return X == O || _t == O ? k() : z(C, dt);
              };
            }
            function rn(C, O, et) {
              for (var dt = 3; dt < arguments.length; dt++) G.cc.push(arguments[dt]);
              return k(Tt(O, et), se(C, O), j);
            }
            function Pn(C) {
              return C == "}" ? k() : z(lt, Pn);
            }
            function wn(C, O) {
              if (v) {
                if (C == ":") return k(Yt);
                if (O == "?") return k(wn);
              }
            }
            function cr(C, O) {
              if (v && (C == ":" || O == "in")) return k(Yt);
            }
            function Ae(C) {
              if (v && C == ":")
                return G.stream.match(/^\s*\w+\s+is\b/, !1) ? k(Et, xn, Yt) : k(Yt);
            }
            function xn(C, O) {
              if (O == "is") return (G.marked = "keyword"), k();
            }
            function Yt(C, O) {
              if (O == "keyof" || O == "typeof" || O == "infer" || O == "readonly")
                return (G.marked = "keyword"), k(O == "typeof" ? $ : Yt);
              if (C == "variable" || O == "void") return (G.marked = "type"), k(qn);
              if (O == "|" || O == "&") return k(Yt);
              if (C == "string" || C == "number" || C == "atom") return k(qn);
              if (C == "[") return k(Tt("]"), se(Yt, "]", ","), j, qn);
              if (C == "{") return k(Tt("}"), Rt, j, qn);
              if (C == "(") return k(se(Ce, ")"), Hl, qn);
              if (C == "<") return k(se(Yt, ">"), Yt);
              if (C == "quasi") return z(Ke, qn);
            }
            function Hl(C) {
              if (C == "=>") return k(Yt);
            }
            function Rt(C) {
              return C.match(/[\}\)\]]/) ? k() : C == "," || C == ";" ? k(Rt) : z(Cr, Rt);
            }
            function Cr(C, O) {
              if (C == "variable" || G.style == "keyword") return (G.marked = "property"), k(Cr);
              if (O == "?" || C == "number" || C == "string") return k(Cr);
              if (C == ":") return k(Yt);
              if (C == "[") return k(rt("variable"), cr, rt("]"), Cr);
              if (C == "(") return z(Jr, Cr);
              if (!C.match(/[;\}\)\],]/)) return k();
            }
            function Ke(C, O) {
              return C != "quasi" ? z() : O.slice(O.length - 2) != "${" ? k(Ke) : k(Yt, ke);
            }
            function ke(C) {
              if (C == "}") return (G.marked = "string-2"), (G.state.tokenize = K), k(Ke);
            }
            function Ce(C, O) {
              return (C == "variable" && G.stream.match(/^\s*[?:]/, !1)) || O == "?"
                ? k(Ce)
                : C == ":"
                ? k(Yt)
                : C == "spread"
                ? k(Ce)
                : z(Yt);
            }
            function qn(C, O) {
              if (O == "<") return k(Tt(">"), se(Yt, ">"), j, qn);
              if (O == "|" || C == "." || O == "&") return k(Yt);
              if (C == "[") return k(Yt, rt("]"), qn);
              if (O == "extends" || O == "implements") return (G.marked = "keyword"), k(Yt);
              if (O == "?") return k(Yt, rt(":"), Yt);
            }
            function _n(C, O) {
              if (O == "<") return k(Tt(">"), se(Yt, ">"), j, qn);
            }
            function Vn() {
              return z(Yt, Xe);
            }
            function Xe(C, O) {
              if (O == "=") return k(Yt);
            }
            function ho(C, O) {
              return O == "enum" ? ((G.marked = "keyword"), k(Pt)) : z(on, wn, Gn, Yc);
            }
            function on(C, O) {
              if (v && yt(O)) return (G.marked = "keyword"), k(on);
              if (C == "variable") return H(O), k();
              if (C == "spread") return k(on);
              if (C == "[") return rn(ws, "]");
              if (C == "{") return rn(Yr, "}");
            }
            function Yr(C, O) {
              return C == "variable" && !G.stream.match(/^\s*:/, !1)
                ? (H(O), k(Gn))
                : (C == "variable" && (G.marked = "property"),
                  C == "spread"
                    ? k(on)
                    : C == "}"
                    ? z()
                    : C == "["
                    ? k(Et, rt("]"), rt(":"), Yr)
                    : k(rt(":"), on, Gn));
            }
            function ws() {
              return z(on, Gn);
            }
            function Gn(C, O) {
              if (O == "=") return k($);
            }
            function Yc(C) {
              if (C == ",") return k(ho);
            }
            function po(C, O) {
              if (C == "keyword b" && O == "else") return k(Tt("form", "else"), lt, j);
            }
            function Bl(C, O) {
              if (O == "await") return k(Bl);
              if (C == "(") return k(Tt(")"), xs, j);
            }
            function xs(C) {
              return C == "var" ? k(ho, Zr) : C == "variable" ? k(Zr) : z(Zr);
            }
            function Zr(C, O) {
              return C == ")"
                ? k()
                : C == ";"
                ? k(Zr)
                : O == "in" || O == "of"
                ? ((G.marked = "keyword"), k(Et, Zr))
                : z(Et, Zr);
            }
            function Hn(C, O) {
              if (O == "*") return (G.marked = "keyword"), k(Hn);
              if (C == "variable") return H(O), k(Hn);
              if (C == "(") return k(Qt, Tt(")"), se(Kn, ")"), j, Ae, lt, Gt);
              if (v && O == "<") return k(Tt(">"), se(Vn, ">"), j, Hn);
            }
            function Jr(C, O) {
              if (O == "*") return (G.marked = "keyword"), k(Jr);
              if (C == "variable") return H(O), k(Jr);
              if (C == "(") return k(Qt, Tt(")"), se(Kn, ")"), j, Ae, Gt);
              if (v && O == "<") return k(Tt(">"), se(Vn, ">"), j, Jr);
            }
            function Wl(C, O) {
              if (C == "keyword" || C == "variable") return (G.marked = "type"), k(Wl);
              if (O == "<") return k(Tt(">"), se(Vn, ">"), j);
            }
            function Kn(C, O) {
              return (
                O == "@" && k(Et, Kn),
                C == "spread"
                  ? k(Kn)
                  : v && yt(O)
                  ? ((G.marked = "keyword"), k(Kn))
                  : v && C == "this"
                  ? k(wn, Gn)
                  : z(on, wn, Gn)
              );
            }
            function Zc(C, O) {
              return C == "variable" ? go(C, O) : Xn(C, O);
            }
            function go(C, O) {
              if (C == "variable") return H(O), k(Xn);
            }
            function Xn(C, O) {
              if (O == "<") return k(Tt(">"), se(Vn, ">"), j, Xn);
              if (O == "extends" || O == "implements" || (v && C == ","))
                return O == "implements" && (G.marked = "keyword"), k(v ? Yt : Et, Xn);
              if (C == "{") return k(Tt("}"), Yn, j);
            }
            function Yn(C, O) {
              if (
                C == "async" ||
                (C == "variable" &&
                  (O == "static" || O == "get" || O == "set" || (v && yt(O))) &&
                  G.stream.match(/^\s+#?[\w$\xa1-\uffff]/, !1))
              )
                return (G.marked = "keyword"), k(Yn);
              if (C == "variable" || G.style == "keyword")
                return (G.marked = "property"), k(Pi, Yn);
              if (C == "number" || C == "string") return k(Pi, Yn);
              if (C == "[") return k(Et, wn, rt("]"), Pi, Yn);
              if (O == "*") return (G.marked = "keyword"), k(Yn);
              if (v && C == "(") return z(Jr, Yn);
              if (C == ";" || C == ",") return k(Yn);
              if (C == "}") return k();
              if (O == "@") return k(Et, Yn);
            }
            function Pi(C, O) {
              if (O == "!" || O == "?") return k(Pi);
              if (C == ":") return k(Yt, Gn);
              if (O == "=") return k($);
              var et = G.state.lexical.prev,
                dt = et && et.info == "interface";
              return z(dt ? Jr : Hn);
            }
            function vo(C, O) {
              return O == "*"
                ? ((G.marked = "keyword"), k(bo, rt(";")))
                : O == "default"
                ? ((G.marked = "keyword"), k(Et, rt(";")))
                : C == "{"
                ? k(se(mo, "}"), bo, rt(";"))
                : z(lt);
            }
            function mo(C, O) {
              if (O == "as") return (G.marked = "keyword"), k(rt("variable"));
              if (C == "variable") return z($, mo);
            }
            function Qr(C) {
              return C == "string" ? k() : C == "(" ? z(Et) : C == "." ? z(ot) : z(yo, ur, bo);
            }
            function yo(C, O) {
              return C == "{"
                ? rn(yo, "}")
                : (C == "variable" && H(O), O == "*" && (G.marked = "keyword"), k(_s));
            }
            function ur(C) {
              if (C == ",") return k(yo, ur);
            }
            function _s(C, O) {
              if (O == "as") return (G.marked = "keyword"), k(yo);
            }
            function bo(C, O) {
              if (O == "from") return (G.marked = "keyword"), k(Et);
            }
            function Te(C) {
              return C == "]" ? k() : z(se($, "]"));
            }
            function Pt() {
              return z(Tt("form"), on, rt("{"), Tt("}"), se(Tr, "}"), j, j);
            }
            function Tr() {
              return z(on, Gn);
            }
            function Ss(C, O) {
              return (
                C.lastType == "operator" ||
                C.lastType == "," ||
                S.test(O.charAt(0)) ||
                /[,.]/.test(O.charAt(0))
              );
            }
            function On(C, O, et) {
              return (
                (O.tokenize == R &&
                  /^(?:operator|sof|keyword [bcd]|case|new|export|default|spread|[\[{}\(,;:]|=>)$/.test(
                    O.lastType,
                  )) ||
                (O.lastType == "quasi" && /\{\s*$/.test(C.string.slice(0, C.pos - (et || 0))))
              );
            }
            return {
              startState: function (C) {
                var O = {
                  tokenize: R,
                  lastType: "sof",
                  cc: [],
                  lexical: new at((C || 0) - u, 0, "block", !1),
                  localVars: s.localVars,
                  context: s.localVars && new At(null, null, !1),
                  indented: C || 0,
                };
                return (
                  s.globalVars && typeof s.globalVars == "object" && (O.globalVars = s.globalVars),
                  O
                );
              },
              token: function (C, O) {
                if (
                  (C.sol() &&
                    (O.lexical.hasOwnProperty("align") || (O.lexical.align = !1),
                    (O.indented = C.indentation()),
                    Y(C, O)),
                  O.tokenize != B && C.eatSpace())
                )
                  return null;
                var et = O.tokenize(C, O);
                return L == "comment"
                  ? et
                  : ((O.lastType = L == "operator" && (T == "++" || T == "--") ? "incdec" : L),
                    gt(O, et, L, T, C));
              },
              indent: function (C, O) {
                if (C.tokenize == B || C.tokenize == K) return r.Pass;
                if (C.tokenize != R) return 0;
                var et = O && O.charAt(0),
                  dt = C.lexical,
                  X;
                if (!/^\s*else\b/.test(O))
                  for (var _t = C.cc.length - 1; _t >= 0; --_t) {
                    var ce = C.cc[_t];
                    if (ce == j) dt = dt.prev;
                    else if (ce != po && ce != Gt) break;
                  }
                for (
                  ;
                  (dt.type == "stat" || dt.type == "form") &&
                  (et == "}" ||
                    ((X = C.cc[C.cc.length - 1]) &&
                      (X == ot || X == ut) &&
                      !/^[,\.=+\-*:?[\(]/.test(O)));
                )
                  dt = dt.prev;
                f && dt.type == ")" && dt.prev.type == "stat" && (dt = dt.prev);
                var Fe = dt.type,
                  sn = et == Fe;
                return Fe == "vardef"
                  ? dt.indented +
                      (C.lastType == "operator" || C.lastType == "," ? dt.info.length + 1 : 0)
                  : Fe == "form" && et == "{"
                  ? dt.indented
                  : Fe == "form"
                  ? dt.indented + u
                  : Fe == "stat"
                  ? dt.indented + (Ss(C, O) ? f || u : 0)
                  : dt.info == "switch" && !sn && s.doubleIndentSwitch != !1
                  ? dt.indented + (/^(?:case|default)\b/.test(O) ? u : 2 * u)
                  : dt.align
                  ? dt.column + (sn ? 0 : 1)
                  : dt.indented + (sn ? 0 : u);
              },
              electricInput: /^\s*(?:case .*?:|default:|\{|\})$/,
              blockCommentStart: d ? null : "/*",
              blockCommentEnd: d ? null : "*/",
              blockCommentContinue: d ? null : " * ",
              lineComment: d ? null : "//",
              fold: "brace",
              closeBrackets: "()[]{}''\"\"``",
              helperType: d ? "json" : "javascript",
              jsonldMode: h,
              jsonMode: d,
              expressionAllowed: On,
              skipExpression: function (C) {
                gt(C, "atom", "atom", "true", new r.StringStream("", 2, null));
              },
            };
          }),
            r.registerHelper("wordChars", "javascript", /[\w$]/),
            r.defineMIME("text/javascript", "javascript"),
            r.defineMIME("text/ecmascript", "javascript"),
            r.defineMIME("application/javascript", "javascript"),
            r.defineMIME("application/x-javascript", "javascript"),
            r.defineMIME("application/ecmascript", "javascript"),
            r.defineMIME("application/json", { name: "javascript", json: !0 }),
            r.defineMIME("application/x-json", { name: "javascript", json: !0 }),
            r.defineMIME("application/manifest+json", { name: "javascript", json: !0 }),
            r.defineMIME("application/ld+json", { name: "javascript", jsonld: !0 }),
            r.defineMIME("text/typescript", { name: "javascript", typescript: !0 }),
            r.defineMIME("application/typescript", { name: "javascript", typescript: !0 });
        });
      })()),
    Jv.exports
  );
}
ab();
var Iat = { exports: {} };
(function (t, e) {
  (function (r) {
    r(ys());
  })(function (r) {
    var o = {
        autoSelfClosers: {
          area: !0,
          base: !0,
          br: !0,
          col: !0,
          command: !0,
          embed: !0,
          frame: !0,
          hr: !0,
          img: !0,
          input: !0,
          keygen: !0,
          link: !0,
          meta: !0,
          param: !0,
          source: !0,
          track: !0,
          wbr: !0,
          menuitem: !0,
        },
        implicitlyClosed: {
          dd: !0,
          li: !0,
          optgroup: !0,
          option: !0,
          p: !0,
          rp: !0,
          rt: !0,
          tbody: !0,
          td: !0,
          tfoot: !0,
          th: !0,
          tr: !0,
        },
        contextGrabbers: {
          dd: { dd: !0, dt: !0 },
          dt: { dd: !0, dt: !0 },
          li: { li: !0 },
          option: { option: !0, optgroup: !0 },
          optgroup: { optgroup: !0 },
          p: {
            address: !0,
            article: !0,
            aside: !0,
            blockquote: !0,
            dir: !0,
            div: !0,
            dl: !0,
            fieldset: !0,
            footer: !0,
            form: !0,
            h1: !0,
            h2: !0,
            h3: !0,
            h4: !0,
            h5: !0,
            h6: !0,
            header: !0,
            hgroup: !0,
            hr: !0,
            menu: !0,
            nav: !0,
            ol: !0,
            p: !0,
            pre: !0,
            section: !0,
            table: !0,
            ul: !0,
          },
          rp: { rp: !0, rt: !0 },
          rt: { rp: !0, rt: !0 },
          tbody: { tbody: !0, tfoot: !0 },
          td: { td: !0, th: !0 },
          tfoot: { tbody: !0 },
          th: { td: !0, th: !0 },
          thead: { tbody: !0, tfoot: !0 },
          tr: { tr: !0 },
        },
        doNotIndent: { pre: !0 },
        allowUnquoted: !0,
        allowMissing: !0,
        caseFold: !0,
      },
      s = {
        autoSelfClosers: {},
        implicitlyClosed: {},
        contextGrabbers: {},
        doNotIndent: {},
        allowUnquoted: !1,
        allowMissing: !1,
        allowMissingTagName: !1,
        caseFold: !1,
      };
    r.defineMode("xml", function (u, f) {
      var h = u.indentUnit,
        d = {},
        g = f.htmlMode ? o : s;
      for (var v in g) d[v] = g[v];
      for (var v in f) d[v] = f[v];
      var b, w;
      function S(k, F) {
        function H(At) {
          return (F.tokenize = At), At(k, F);
        }
        var J = k.next();
        if (J == "<")
          return k.eat("!")
            ? k.eat("[")
              ? k.match("CDATA[")
                ? H(L("atom", "]]>"))
                : null
              : k.match("--")
              ? H(L("comment", "-->"))
              : k.match("DOCTYPE", !0, !0)
              ? (k.eatWhile(/[\w\._\-]/), H(T(1)))
              : null
            : k.eat("?")
            ? (k.eatWhile(/[\w\._\-]/), (F.tokenize = L("meta", "?>")), "meta")
            : ((b = k.eat("/") ? "closeTag" : "openTag"), (F.tokenize = P), "tag bracket");
        if (J == "&") {
          var yt;
          return (
            k.eat("#")
              ? k.eat("x")
                ? (yt = k.eatWhile(/[a-fA-F\d]/) && k.eat(";"))
                : (yt = k.eatWhile(/[\d]/) && k.eat(";"))
              : (yt = k.eatWhile(/[\w\.\-:]/) && k.eat(";")),
            yt ? "atom" : "error"
          );
        } else return k.eatWhile(/[^&<]/), null;
      }
      S.isInText = !0;
      function P(k, F) {
        var H = k.next();
        if (H == ">" || (H == "/" && k.eat(">")))
          return (F.tokenize = S), (b = H == ">" ? "endTag" : "selfcloseTag"), "tag bracket";
        if (H == "=") return (b = "equals"), null;
        if (H == "<") {
          (F.tokenize = S), (F.state = K), (F.tagName = F.tagStart = null);
          var J = F.tokenize(k, F);
          return J ? J + " tag error" : "tag error";
        } else
          return /[\'\"]/.test(H)
            ? ((F.tokenize = A(H)), (F.stringStartCol = k.column()), F.tokenize(k, F))
            : (k.match(/^[^\s\u00a0=<>\"\']*[^\s\u00a0=<>\"\'\/]/), "word");
      }
      function A(k) {
        var F = function (H, J) {
          for (; !H.eol(); )
            if (H.next() == k) {
              J.tokenize = P;
              break;
            }
          return "string";
        };
        return (F.isInAttribute = !0), F;
      }
      function L(k, F) {
        return function (H, J) {
          for (; !H.eol(); ) {
            if (H.match(F)) {
              J.tokenize = S;
              break;
            }
            H.next();
          }
          return k;
        };
      }
      function T(k) {
        return function (F, H) {
          for (var J; (J = F.next()) != null; ) {
            if (J == "<") return (H.tokenize = T(k + 1)), H.tokenize(F, H);
            if (J == ">")
              if (k == 1) {
                H.tokenize = S;
                break;
              } else return (H.tokenize = T(k - 1)), H.tokenize(F, H);
          }
          return "meta";
        };
      }
      function M(k) {
        return k && k.toLowerCase();
      }
      function R(k, F, H) {
        (this.prev = k.context),
          (this.tagName = F || ""),
          (this.indent = k.indented),
          (this.startOfLine = H),
          (d.doNotIndent.hasOwnProperty(F) || (k.context && k.context.noIndent)) &&
            (this.noIndent = !0);
      }
      function E(k) {
        k.context && (k.context = k.context.prev);
      }
      function B(k, F) {
        for (var H; ; ) {
          if (
            !k.context ||
            ((H = k.context.tagName),
            !d.contextGrabbers.hasOwnProperty(M(H)) ||
              !d.contextGrabbers[M(H)].hasOwnProperty(M(F)))
          )
            return;
          E(k);
        }
      }
      function K(k, F, H) {
        return k == "openTag" ? ((H.tagStart = F.column()), ht) : k == "closeTag" ? Y : K;
      }
      function ht(k, F, H) {
        return k == "word"
          ? ((H.tagName = F.current()), (w = "tag"), pt)
          : d.allowMissingTagName && k == "endTag"
          ? ((w = "tag bracket"), pt(k, F, H))
          : ((w = "error"), ht);
      }
      function Y(k, F, H) {
        if (k == "word") {
          var J = F.current();
          return (
            H.context &&
              H.context.tagName != J &&
              d.implicitlyClosed.hasOwnProperty(M(H.context.tagName)) &&
              E(H),
            (H.context && H.context.tagName == J) || d.matchClosing === !1
              ? ((w = "tag"), nt)
              : ((w = "tag error"), at)
          );
        } else
          return d.allowMissingTagName && k == "endTag"
            ? ((w = "tag bracket"), nt(k, F, H))
            : ((w = "error"), at);
      }
      function nt(k, F, H) {
        return k != "endTag" ? ((w = "error"), nt) : (E(H), K);
      }
      function at(k, F, H) {
        return (w = "error"), nt(k, F, H);
      }
      function pt(k, F, H) {
        if (k == "word") return (w = "attribute"), gt;
        if (k == "endTag" || k == "selfcloseTag") {
          var J = H.tagName,
            yt = H.tagStart;
          return (
            (H.tagName = H.tagStart = null),
            k == "selfcloseTag" || d.autoSelfClosers.hasOwnProperty(M(J))
              ? B(H, J)
              : (B(H, J), (H.context = new R(H, J, yt == H.indented))),
            K
          );
        }
        return (w = "error"), pt;
      }
      function gt(k, F, H) {
        return k == "equals" ? G : (d.allowMissing || (w = "error"), pt(k, F, H));
      }
      function G(k, F, H) {
        return k == "string"
          ? z
          : k == "word" && d.allowUnquoted
          ? ((w = "string"), pt)
          : ((w = "error"), pt(k, F, H));
      }
      function z(k, F, H) {
        return k == "string" ? z : pt(k, F, H);
      }
      return {
        startState: function (k) {
          var F = {
            tokenize: S,
            state: K,
            indented: k || 0,
            tagName: null,
            tagStart: null,
            context: null,
          };
          return k != null && (F.baseIndent = k), F;
        },
        token: function (k, F) {
          if ((!F.tagName && k.sol() && (F.indented = k.indentation()), k.eatSpace())) return null;
          b = null;
          var H = F.tokenize(k, F);
          return (
            (H || b) &&
              H != "comment" &&
              ((w = null),
              (F.state = F.state(b || H, k, F)),
              w && (H = w == "error" ? H + " error" : w)),
            H
          );
        },
        indent: function (k, F, H) {
          var J = k.context;
          if (k.tokenize.isInAttribute)
            return k.tagStart == k.indented ? k.stringStartCol + 1 : k.indented + h;
          if (J && J.noIndent) return r.Pass;
          if (k.tokenize != P && k.tokenize != S) return H ? H.match(/^(\s*)/)[0].length : 0;
          if (k.tagName)
            return d.multilineTagIndentPastTag !== !1
              ? k.tagStart + k.tagName.length + 2
              : k.tagStart + h * (d.multilineTagIndentFactor || 1);
          if (d.alignCDATA && /<!\[CDATA\[/.test(F)) return 0;
          var yt = F && /^<(\/)?([\w_:\.-]*)/.exec(F);
          if (yt && yt[1])
            for (; J; )
              if (J.tagName == yt[2]) {
                J = J.prev;
                break;
              } else if (d.implicitlyClosed.hasOwnProperty(M(J.tagName))) J = J.prev;
              else break;
          else if (yt)
            for (; J; ) {
              var At = d.contextGrabbers[M(J.tagName)];
              if (At && At.hasOwnProperty(M(yt[2]))) J = J.prev;
              else break;
            }
          for (; J && J.prev && !J.startOfLine; ) J = J.prev;
          return J ? J.indent + h : k.baseIndent || 0;
        },
        electricInput: /<\/[\s\w:]+>$/,
        blockCommentStart: "<!--",
        blockCommentEnd: "-->",
        configuration: d.htmlMode ? "html" : "xml",
        helperType: d.htmlMode ? "html" : "xml",
        skipAttribute: function (k) {
          k.state == G && (k.state = pt);
        },
        xmlCurrentTag: function (k) {
          return k.tagName ? { name: k.tagName, close: k.type == "closeTag" } : null;
        },
        xmlCurrentContext: function (k) {
          for (var F = [], H = k.context; H; H = H.prev) F.push(H.tagName);
          return F.reverse();
        },
      };
    }),
      r.defineMIME("text/xml", "xml"),
      r.defineMIME("application/xml", "xml"),
      r.mimeModes.hasOwnProperty("text/html") ||
        r.defineMIME("text/html", { name: "xml", htmlMode: !0 });
  });
})();
var Fat = Iat.exports;
(function (t, e) {
  (function (r) {
    r(ys(), Fat, ab());
  })(function (r) {
    function o(u, f, h, d) {
      (this.state = u), (this.mode = f), (this.depth = h), (this.prev = d);
    }
    function s(u) {
      return new o(r.copyState(u.mode, u.state), u.mode, u.depth, u.prev && s(u.prev));
    }
    r.defineMode(
      "jsx",
      function (u, f) {
        var h = r.getMode(u, {
            name: "xml",
            allowMissing: !0,
            multilineTagIndentPastTag: !1,
            allowMissingTagName: !0,
          }),
          d = r.getMode(u, (f && f.base) || "javascript");
        function g(S) {
          var P = S.tagName;
          S.tagName = null;
          var A = h.indent(S, "", "");
          return (S.tagName = P), A;
        }
        function v(S, P) {
          return P.context.mode == h ? b(S, P, P.context) : w(S, P, P.context);
        }
        function b(S, P, A) {
          if (A.depth == 2) return S.match(/^.*?\*\//) ? (A.depth = 1) : S.skipToEnd(), "comment";
          if (S.peek() == "{") {
            h.skipAttribute(A.state);
            var L = g(A.state),
              T = A.state.context;
            if (T && S.match(/^[^>]*>\s*$/, !1)) {
              for (; T.prev && !T.startOfLine; ) T = T.prev;
              T.startOfLine
                ? (L -= u.indentUnit)
                : A.prev.state.lexical && (L = A.prev.state.lexical.indented);
            } else A.depth == 1 && (L += u.indentUnit);
            return (P.context = new o(r.startState(d, L), d, 0, P.context)), null;
          }
          if (A.depth == 1) {
            if (S.peek() == "<")
              return (
                h.skipAttribute(A.state),
                (P.context = new o(r.startState(h, g(A.state)), h, 0, P.context)),
                null
              );
            if (S.match("//")) return S.skipToEnd(), "comment";
            if (S.match("/*")) return (A.depth = 2), v(S, P);
          }
          var M = h.token(S, A.state),
            R = S.current(),
            E;
          return (
            /\btag\b/.test(M)
              ? />$/.test(R)
                ? A.state.context
                  ? (A.depth = 0)
                  : (P.context = P.context.prev)
                : /^</.test(R) && (A.depth = 1)
              : !M && (E = R.indexOf("{")) > -1 && S.backUp(R.length - E),
            M
          );
        }
        function w(S, P, A) {
          if (
            S.peek() == "<" &&
            !S.match(/^<([^<>]|<[^>]*>)+,\s*>/, !1) &&
            d.expressionAllowed(S, A.state)
          )
            return (
              (P.context = new o(r.startState(h, d.indent(A.state, "", "")), h, 0, P.context)),
              d.skipExpression(A.state),
              null
            );
          var L = d.token(S, A.state);
          if (!L && A.depth != null) {
            var T = S.current();
            T == "{" ? A.depth++ : T == "}" && --A.depth == 0 && (P.context = P.context.prev);
          }
          return L;
        }
        return {
          startState: function () {
            return { context: new o(r.startState(d), d) };
          },
          copyState: function (S) {
            return { context: s(S.context) };
          },
          token: v,
          indent: function (S, P, A) {
            return S.context.mode.indent(S.context.state, P, A);
          },
          innerMode: function (S) {
            return S.context;
          },
        };
      },
      "xml",
      "javascript",
    ),
      r.defineMIME("text/jsx", "jsx"),
      r.defineMIME("text/typescript-jsx", {
        name: "jsx",
        base: { name: "javascript", typescript: !0 },
      });
  });
})();
(function (t, e) {
  (function (r) {
    r(ys());
  })(function (r) {
    r.defineOption("placeholder", "", function (g, v, b) {
      var w = b && b != r.Init;
      if (v && !w)
        g.on("blur", f),
          g.on("change", h),
          g.on("swapDoc", h),
          r.on(
            g.getInputField(),
            "compositionupdate",
            (g.state.placeholderCompose = function () {
              u(g);
            }),
          ),
          h(g);
      else if (!v && w) {
        g.off("blur", f),
          g.off("change", h),
          g.off("swapDoc", h),
          r.off(g.getInputField(), "compositionupdate", g.state.placeholderCompose),
          o(g);
        var S = g.getWrapperElement();
        S.className = S.className.replace(" CodeMirror-empty", "");
      }
      v && !g.hasFocus() && f(g);
    });
    function o(g) {
      g.state.placeholder &&
        (g.state.placeholder.parentNode.removeChild(g.state.placeholder),
        (g.state.placeholder = null));
    }
    function s(g) {
      o(g);
      var v = (g.state.placeholder = document.createElement("pre"));
      (v.style.cssText = "height: 0; overflow: visible"),
        (v.style.direction = g.getOption("direction")),
        (v.className = "CodeMirror-placeholder CodeMirror-line-like");
      var b = g.getOption("placeholder");
      typeof b == "string" && (b = document.createTextNode(b)),
        v.appendChild(b),
        g.display.lineSpace.insertBefore(v, g.display.lineSpace.firstChild);
    }
    function u(g) {
      setTimeout(function () {
        var v = !1;
        if (g.lineCount() == 1) {
          var b = g.getInputField();
          v =
            b.nodeName == "TEXTAREA"
              ? !g.getLine(0).length
              : !/[^\u200b]/.test(b.querySelector(".CodeMirror-line").textContent);
        }
        v ? s(g) : o(g);
      }, 20);
    }
    function f(g) {
      d(g) && s(g);
    }
    function h(g) {
      var v = g.getWrapperElement(),
        b = d(g);
      (v.className = v.className.replace(" CodeMirror-empty", "") + (b ? " CodeMirror-empty" : "")),
        b ? s(g) : o(g);
    }
    function d(g) {
      return g.lineCount() === 1 && g.getLine(0) === "";
    }
  });
})();
(function (t, e) {
  (function (r) {
    r(ys());
  })(function (r) {
    function o(f, h, d) {
      (this.orientation = h),
        (this.scroll = d),
        (this.screen = this.total = this.size = 1),
        (this.pos = 0),
        (this.node = document.createElement("div")),
        (this.node.className = f + "-" + h),
        (this.inner = this.node.appendChild(document.createElement("div")));
      var g = this;
      r.on(this.inner, "mousedown", function (b) {
        if (b.which != 1) return;
        r.e_preventDefault(b);
        var w = g.orientation == "horizontal" ? "pageX" : "pageY",
          S = b[w],
          P = g.pos;
        function A() {
          r.off(document, "mousemove", L), r.off(document, "mouseup", A);
        }
        function L(T) {
          if (T.which != 1) return A();
          g.moveTo(P + (T[w] - S) * (g.total / g.size));
        }
        r.on(document, "mousemove", L), r.on(document, "mouseup", A);
      }),
        r.on(this.node, "click", function (b) {
          r.e_preventDefault(b);
          var w = g.inner.getBoundingClientRect(),
            S;
          g.orientation == "horizontal"
            ? (S = b.clientX < w.left ? -1 : b.clientX > w.right ? 1 : 0)
            : (S = b.clientY < w.top ? -1 : b.clientY > w.bottom ? 1 : 0),
            g.moveTo(g.pos + S * g.screen);
        });
      function v(b) {
        var w = r.wheelEventPixels(b)[g.orientation == "horizontal" ? "x" : "y"],
          S = g.pos;
        g.moveTo(g.pos + w), g.pos != S && r.e_preventDefault(b);
      }
      r.on(this.node, "mousewheel", v), r.on(this.node, "DOMMouseScroll", v);
    }
    (o.prototype.setPos = function (f, h) {
      return (
        f < 0 && (f = 0),
        f > this.total - this.screen && (f = this.total - this.screen),
        !h && f == this.pos
          ? !1
          : ((this.pos = f),
            (this.inner.style[this.orientation == "horizontal" ? "left" : "top"] =
              f * (this.size / this.total) + "px"),
            !0)
      );
    }),
      (o.prototype.moveTo = function (f) {
        this.setPos(f) && this.scroll(f, this.orientation);
      });
    var s = 10;
    o.prototype.update = function (f, h, d) {
      var g = this.screen != h || this.total != f || this.size != d;
      g && ((this.screen = h), (this.total = f), (this.size = d));
      var v = this.screen * (this.size / this.total);
      v < s && ((this.size -= s - v), (v = s)),
        (this.inner.style[this.orientation == "horizontal" ? "width" : "height"] = v + "px"),
        this.setPos(this.pos, g);
    };
    function u(f, h, d) {
      (this.addClass = f),
        (this.horiz = new o(f, "horizontal", d)),
        h(this.horiz.node),
        (this.vert = new o(f, "vertical", d)),
        h(this.vert.node),
        (this.width = null);
    }
    (u.prototype.update = function (f) {
      if (this.width == null) {
        var h = window.getComputedStyle
          ? window.getComputedStyle(this.horiz.node)
          : this.horiz.node.currentStyle;
        h && (this.width = parseInt(h.height));
      }
      var d = this.width || 0,
        g = f.scrollWidth > f.clientWidth + 1,
        v = f.scrollHeight > f.clientHeight + 1;
      return (
        (this.vert.node.style.display = v ? "block" : "none"),
        (this.horiz.node.style.display = g ? "block" : "none"),
        v &&
          (this.vert.update(f.scrollHeight, f.clientHeight, f.viewHeight - (g ? d : 0)),
          (this.vert.node.style.bottom = g ? d + "px" : "0")),
        g &&
          (this.horiz.update(f.scrollWidth, f.clientWidth, f.viewWidth - (v ? d : 0) - f.barLeft),
          (this.horiz.node.style.right = v ? d + "px" : "0"),
          (this.horiz.node.style.left = f.barLeft + "px")),
        { right: v ? d : 0, bottom: g ? d : 0 }
      );
    }),
      (u.prototype.setScrollTop = function (f) {
        this.vert.setPos(f);
      }),
      (u.prototype.setScrollLeft = function (f) {
        this.horiz.setPos(f);
      }),
      (u.prototype.clear = function () {
        var f = this.horiz.node.parentNode;
        f.removeChild(this.horiz.node), f.removeChild(this.vert.node);
      }),
      (r.scrollbarModel.simple = function (f, h) {
        return new u("CodeMirror-simplescroll", f, h);
      }),
      (r.scrollbarModel.overlay = function (f, h) {
        return new u("CodeMirror-overlayscroll", f, h);
      });
  });
})();
function qat(t, e, r = {}) {
  const o = zat.fromTextArea(t.value, { theme: "vars", ...r, scrollbarStyle: "simple" });
  let s = !1;
  return (
    o.on("change", () => {
      if (s) {
        s = !1;
        return;
      }
      e.value = o.getValue();
    }),
    Re(
      e,
      (u) => {
        if (u !== o.getValue()) {
          s = !0;
          const f = o.listSelections();
          o.replaceRange(u, o.posFromIndex(0), o.posFromIndex(Number.POSITIVE_INFINITY)),
            o.setSelections(f);
        }
      },
      { immediate: !0 },
    ),
    mh(o)
  );
}
const Hat = { relative: "", "font-mono": "", "text-sm": "", class: "codemirror-scrolls" },
  cb = ie({
    __name: "CodeMirror",
    props: Tf({ mode: {}, readOnly: { type: Boolean } }, { modelValue: {} }),
    emits: Tf(["save"], ["update:modelValue"]),
    setup(t, { expose: e, emit: r }) {
      const o = r,
        s = u0(t, "modelValue"),
        u = k_(),
        f = {
          js: "javascript",
          mjs: "javascript",
          cjs: "javascript",
          ts: { name: "javascript", typescript: !0 },
          mts: { name: "javascript", typescript: !0 },
          cts: { name: "javascript", typescript: !0 },
          jsx: { name: "javascript", jsx: !0 },
          tsx: { name: "javascript", typescript: !0, jsx: !0 },
        },
        h = Zt(),
        d = vs();
      return (
        e({ cm: d }),
        ms(async () => {
          (d.value = qat(h, s, {
            ...u,
            mode: f[t.mode || ""] || t.mode,
            readOnly: t.readOnly ? !0 : void 0,
            extraKeys: {
              "Cmd-S": function (g) {
                o("save", g.getValue());
              },
              "Ctrl-S": function (g) {
                o("save", g.getValue());
              },
            },
          })),
            d.value.setSize("100%", "100%"),
            d.value.clearHistory(),
            setTimeout(() => d.value.refresh(), 100);
        }),
        (g, v) => (st(), kt("div", Hat, [tt("textarea", { ref_key: "el", ref: h }, null, 512)]))
      );
    },
  }),
  Bat = ie({
    __name: "ViewEditor",
    props: { file: {} },
    emits: ["draft"],
    setup(t, { emit: e }) {
      const r = t,
        o = e,
        s = Zt(""),
        u = vs(void 0),
        f = Zt(!1);
      Re(
        () => r.file,
        async () => {
          var R;
          if (!r.file || !((R = r.file) != null && R.filepath)) {
            (s.value = ""), (u.value = s.value), (f.value = !1);
            return;
          }
          (s.value = (await je.rpc.readTestFile(r.file.filepath)) || ""),
            (u.value = s.value),
            (f.value = !1);
        },
        { immediate: !0 },
      );
      const h = xt(() => {
          var R, E;
          return (
            ((E = (R = r.file) == null ? void 0 : R.filepath) == null
              ? void 0
              : E.split(/\./g).pop()) || "js"
          );
        }),
        d = Zt(),
        g = xt(() => {
          var R;
          return (R = d.value) == null ? void 0 : R.cm;
        }),
        v = xt(() => {
          var R;
          return (
            ((R = r.file) == null
              ? void 0
              : R.tasks.filter((E) => {
                  var B;
                  return ((B = E.result) == null ? void 0 : B.state) === "fail";
                })) || []
          );
        }),
        b = [],
        w = [],
        S = [],
        P = Zt(!1);
      function A() {
        S.forEach(([R, E, B]) => {
          R.removeEventListener("click", E), B();
        }),
          (S.length = 0);
      }
      Rlt(d, () => {
        var R;
        (R = g.value) == null || R.refresh();
      });
      function L() {
        f.value = u.value !== g.value.getValue();
      }
      Re(
        f,
        (R) => {
          o("draft", R);
        },
        { immediate: !0 },
      );
      function T(R) {
        const E = ((R == null ? void 0 : R.stacks) || []).filter((at) => {
            var pt;
            return at.file && at.file === ((pt = r.file) == null ? void 0 : pt.filepath);
          }),
          B = E == null ? void 0 : E[0];
        if (!B) return;
        const K = document.createElement("div");
        K.className = "op80 flex gap-x-2 items-center";
        const ht = document.createElement("pre");
        (ht.className = "c-red-600 dark:c-red-400"),
          (ht.textContent = `${" ".repeat(B.column)}^ ${
            (R == null ? void 0 : R.nameStr) || R.name
          }: ${(R == null ? void 0 : R.message) || ""}`),
          K.appendChild(ht);
        const Y = document.createElement("span");
        (Y.className =
          "i-carbon-launch c-red-600 dark:c-red-400 hover:cursor-pointer min-w-1em min-h-1em"),
          (Y.tabIndex = 0),
          (Y.ariaLabel = "Open in Editor"),
          ty(Y, { content: "Open in Editor", placement: "bottom" }, !1);
        const nt = async () => {
          await Fy(B.file, B.line, B.column);
        };
        K.appendChild(Y),
          S.push([Y, nt, () => Hh(Y)]),
          w.push(g.value.addLineClass(B.line - 1, "wrap", "bg-red-500/10")),
          b.push(g.value.addLineWidget(B.line - 1, K));
      }
      Re(
        [g, v],
        ([R]) => {
          if (!R) {
            A();
            return;
          }
          setTimeout(() => {
            A(),
              b.forEach((E) => E.clear()),
              w.forEach((E) => {
                var B;
                return (B = g.value) == null ? void 0 : B.removeLineClass(E, "wrap");
              }),
              (b.length = 0),
              (w.length = 0),
              R.on("changes", L),
              v.value.forEach((E) => {
                var B, K;
                (K = (B = E.result) == null ? void 0 : B.errors) == null || K.forEach(T);
              }),
              P.value || R.clearHistory();
          }, 100);
        },
        { flush: "post" },
      );
      async function M(R) {
        (P.value = !0),
          await je.rpc.saveTestFile(r.file.filepath, R),
          (u.value = R),
          (f.value = !1);
      }
      return (R, E) => {
        const B = cb;
        return (
          st(),
          te(
            B,
            Ci(
              {
                ref_key: "editor",
                ref: d,
                modelValue: U(s),
                "onUpdate:modelValue": E[0] || (E[0] = (K) => (Le(s) ? (s.value = K) : null)),
                "h-full": "",
              },
              { lineNumbers: !0 },
              { mode: U(h), "data-testid": "code-mirror", onSave: M },
            ),
            null,
            16,
            ["modelValue", "mode"],
          )
        );
      };
    },
  }),
  Wat = ie({
    __name: "Modal",
    props: Tf({ direction: { default: "bottom" } }, { modelValue: { type: Boolean, default: !1 } }),
    emits: ["update:modelValue"],
    setup(t) {
      const e = u0(t, "modelValue"),
        r = xt(() => {
          switch (t.direction) {
            case "bottom":
              return "bottom-0 left-0 right-0 border-t";
            case "top":
              return "top-0 left-0 right-0 border-b";
            case "left":
              return "bottom-0 left-0 top-0 border-r";
            case "right":
              return "bottom-0 top-0 right-0 border-l";
            default:
              return "";
          }
        }),
        o = xt(() => {
          switch (t.direction) {
            case "bottom":
              return "translateY(100%)";
            case "top":
              return "translateY(-100%)";
            case "left":
              return "translateX(-100%)";
            case "right":
              return "translateX(100%)";
            default:
              return "";
          }
        }),
        s = () => (e.value = !1);
      return (u, f) => (
        st(),
        kt(
          "div",
          { class: ve(["fixed inset-0 z-40", e.value ? "" : "pointer-events-none"]) },
          [
            tt(
              "div",
              {
                class: ve([
                  "bg-base inset-0 absolute transition-opacity duration-500 ease-out",
                  e.value ? "opacity-50" : "opacity-0",
                ]),
                onClick: s,
              },
              null,
              2,
            ),
            tt(
              "div",
              {
                class: ve([
                  "bg-base border-base absolute transition-all duration-200 ease-out scrolls",
                  [U(r)],
                ]),
                style: An(e.value ? {} : { transform: U(o) }),
              },
              [sr(u.$slots, "default")],
              6,
            ),
          ],
          2,
        )
      );
    },
  }),
  Uat = ["aria-label", "opacity", "disabled", "hover"],
  bs = ie({
    __name: "IconButton",
    props: { icon: {}, title: {}, disabled: { type: Boolean } },
    setup(t) {
      return (e, r) => (
        st(),
        kt(
          "button",
          {
            "aria-label": e.title,
            role: "button",
            opacity: e.disabled ? 10 : 70,
            rounded: "",
            disabled: e.disabled,
            hover: e.disabled ? "" : "bg-active op100",
            class: "w-1.4em h-1.4em flex",
          },
          [sr(e.$slots, "default", {}, () => [tt("div", { class: ve(e.icon), ma: "" }, null, 2)])],
          8,
          Uat,
        )
      );
    },
  }),
  jat = { "w-350": "", "max-w-screen": "", "h-full": "", flex: "", "flex-col": "" },
  Vat = { "p-4": "", relative: "" },
  Gat = tt("p", null, "Module Info", -1),
  Kat = { op50: "", "font-mono": "", "text-sm": "" },
  Xat = { key: 0, "p-5": "" },
  Yat = { grid: "~ cols-2 rows-[min-content_auto]", "overflow-hidden": "", "flex-auto": "" },
  Zat = tt("div", { p: "x3 y-1", "bg-overlay": "", border: "base b t r" }, " Source ", -1),
  Jat = tt("div", { p: "x3 y-1", "bg-overlay": "", border: "base b t" }, " Transformed ", -1),
  Qat = { key: 0 },
  tct = { p: "x3 y-1", "bg-overlay": "", border: "base b t" },
  ect = ie({
    __name: "ModuleTransformResultView",
    props: { id: {} },
    emits: ["close"],
    setup(t, { emit: e }) {
      const r = t,
        o = e,
        s = klt(() => je.rpc.getTransformResult(r.id)),
        u = xt(() => {
          var g;
          return ((g = r.id) == null ? void 0 : g.split(/\./g).pop()) || "js";
        }),
        f = xt(() => {
          var g, v;
          return (
            ((v = (g = s.value) == null ? void 0 : g.source) == null ? void 0 : v.trim()) || ""
          );
        }),
        h = xt(() => {
          var g, v;
          return (
            ((v = (g = s.value) == null ? void 0 : g.code) == null
              ? void 0
              : v.replace(/\/\/# sourceMappingURL=.*\n/, "").trim()) || ""
          );
        }),
        d = xt(() => {
          var g, v, b, w;
          return {
            mappings:
              ((v = (g = s.value) == null ? void 0 : g.map) == null ? void 0 : v.mappings) ?? "",
            version: (w = (b = s.value) == null ? void 0 : b.map) == null ? void 0 : w.version,
          };
        });
      return (
        Tlt("Escape", () => {
          o("close");
        }),
        (g, v) => {
          const b = bs,
            w = cb;
          return (
            st(),
            kt("div", jat, [
              tt("div", Vat, [
                Gat,
                tt("p", Kat, Ut(g.id), 1),
                Ft(b, {
                  icon: "i-carbon-close",
                  absolute: "",
                  "top-5px": "",
                  "right-5px": "",
                  "text-2xl": "",
                  onClick: v[0] || (v[0] = (S) => o("close")),
                }),
              ]),
              U(s)
                ? (st(),
                  kt(
                    ne,
                    { key: 1 },
                    [
                      tt("div", Yat, [
                        Zat,
                        Jat,
                        Ft(
                          w,
                          Ci(
                            { "h-full": "", "model-value": U(f), "read-only": "" },
                            { lineNumbers: !0 },
                            { mode: U(u) },
                          ),
                          null,
                          16,
                          ["model-value", "mode"],
                        ),
                        Ft(
                          w,
                          Ci(
                            { "h-full": "", "model-value": U(h), "read-only": "" },
                            { lineNumbers: !0 },
                            { mode: U(u) },
                          ),
                          null,
                          16,
                          ["model-value", "mode"],
                        ),
                      ]),
                      U(d).mappings !== ""
                        ? (st(),
                          kt("div", Qat, [
                            tt("div", tct, " Source map (v" + Ut(U(d).version) + ") ", 1),
                            Ft(
                              w,
                              Ci(
                                { "model-value": U(d).mappings, "read-only": "" },
                                { lineNumbers: !0 },
                                { mode: U(u) },
                              ),
                              null,
                              16,
                              ["model-value", "mode"],
                            ),
                          ]))
                        : Vt("", !0),
                    ],
                    64,
                  ))
                : (st(), kt("div", Xat, " No transform result found for this module. ")),
            ])
          );
        }
      );
    },
  });
function nct(t, e) {
  let r;
  return (...o) => {
    r !== void 0 && clearTimeout(r), (r = setTimeout(() => t(...o), e));
  };
}
var Yf = "http://www.w3.org/1999/xhtml";
const tm = {
  svg: "http://www.w3.org/2000/svg",
  xhtml: Yf,
  xlink: "http://www.w3.org/1999/xlink",
  xml: "http://www.w3.org/XML/1998/namespace",
  xmlns: "http://www.w3.org/2000/xmlns/",
};
function Gc(t) {
  var e = (t += ""),
    r = e.indexOf(":");
  return (
    r >= 0 && (e = t.slice(0, r)) !== "xmlns" && (t = t.slice(r + 1)),
    tm.hasOwnProperty(e) ? { space: tm[e], local: t } : t
  );
}
function rct(t) {
  return function () {
    var e = this.ownerDocument,
      r = this.namespaceURI;
    return r === Yf && e.documentElement.namespaceURI === Yf
      ? e.createElement(t)
      : e.createElementNS(r, t);
  };
}
function ict(t) {
  return function () {
    return this.ownerDocument.createElementNS(t.space, t.local);
  };
}
function ub(t) {
  var e = Gc(t);
  return (e.local ? ict : rct)(e);
}
function oct() {}
function Jh(t) {
  return t == null
    ? oct
    : function () {
        return this.querySelector(t);
      };
}
function sct(t) {
  typeof t != "function" && (t = Jh(t));
  for (var e = this._groups, r = e.length, o = new Array(r), s = 0; s < r; ++s)
    for (var u = e[s], f = u.length, h = (o[s] = new Array(f)), d, g, v = 0; v < f; ++v)
      (d = u[v]) &&
        (g = t.call(d, d.__data__, v, u)) &&
        ("__data__" in d && (g.__data__ = d.__data__), (h[v] = g));
  return new Fn(o, this._parents);
}
function lct(t) {
  return t == null ? [] : Array.isArray(t) ? t : Array.from(t);
}
function act() {
  return [];
}
function fb(t) {
  return t == null
    ? act
    : function () {
        return this.querySelectorAll(t);
      };
}
function cct(t) {
  return function () {
    return lct(t.apply(this, arguments));
  };
}
function uct(t) {
  typeof t == "function" ? (t = cct(t)) : (t = fb(t));
  for (var e = this._groups, r = e.length, o = [], s = [], u = 0; u < r; ++u)
    for (var f = e[u], h = f.length, d, g = 0; g < h; ++g)
      (d = f[g]) && (o.push(t.call(d, d.__data__, g, f)), s.push(d));
  return new Fn(o, s);
}
function hb(t) {
  return function () {
    return this.matches(t);
  };
}
function db(t) {
  return function (e) {
    return e.matches(t);
  };
}
var fct = Array.prototype.find;
function hct(t) {
  return function () {
    return fct.call(this.children, t);
  };
}
function dct() {
  return this.firstElementChild;
}
function pct(t) {
  return this.select(t == null ? dct : hct(typeof t == "function" ? t : db(t)));
}
var gct = Array.prototype.filter;
function vct() {
  return Array.from(this.children);
}
function mct(t) {
  return function () {
    return gct.call(this.children, t);
  };
}
function yct(t) {
  return this.selectAll(t == null ? vct : mct(typeof t == "function" ? t : db(t)));
}
function bct(t) {
  typeof t != "function" && (t = hb(t));
  for (var e = this._groups, r = e.length, o = new Array(r), s = 0; s < r; ++s)
    for (var u = e[s], f = u.length, h = (o[s] = []), d, g = 0; g < f; ++g)
      (d = u[g]) && t.call(d, d.__data__, g, u) && h.push(d);
  return new Fn(o, this._parents);
}
function pb(t) {
  return new Array(t.length);
}
function wct() {
  return new Fn(this._enter || this._groups.map(pb), this._parents);
}
function gc(t, e) {
  (this.ownerDocument = t.ownerDocument),
    (this.namespaceURI = t.namespaceURI),
    (this._next = null),
    (this._parent = t),
    (this.__data__ = e);
}
gc.prototype = {
  constructor: gc,
  appendChild: function (t) {
    return this._parent.insertBefore(t, this._next);
  },
  insertBefore: function (t, e) {
    return this._parent.insertBefore(t, e);
  },
  querySelector: function (t) {
    return this._parent.querySelector(t);
  },
  querySelectorAll: function (t) {
    return this._parent.querySelectorAll(t);
  },
};
function xct(t) {
  return function () {
    return t;
  };
}
function _ct(t, e, r, o, s, u) {
  for (var f = 0, h, d = e.length, g = u.length; f < g; ++f)
    (h = e[f]) ? ((h.__data__ = u[f]), (o[f] = h)) : (r[f] = new gc(t, u[f]));
  for (; f < d; ++f) (h = e[f]) && (s[f] = h);
}
function Sct(t, e, r, o, s, u, f) {
  var h,
    d,
    g = new Map(),
    v = e.length,
    b = u.length,
    w = new Array(v),
    S;
  for (h = 0; h < v; ++h)
    (d = e[h]) &&
      ((w[h] = S = f.call(d, d.__data__, h, e) + ""), g.has(S) ? (s[h] = d) : g.set(S, d));
  for (h = 0; h < b; ++h)
    (S = f.call(t, u[h], h, u) + ""),
      (d = g.get(S)) ? ((o[h] = d), (d.__data__ = u[h]), g.delete(S)) : (r[h] = new gc(t, u[h]));
  for (h = 0; h < v; ++h) (d = e[h]) && g.get(w[h]) === d && (s[h] = d);
}
function kct(t) {
  return t.__data__;
}
function Cct(t, e) {
  if (!arguments.length) return Array.from(this, kct);
  var r = e ? Sct : _ct,
    o = this._parents,
    s = this._groups;
  typeof t != "function" && (t = xct(t));
  for (var u = s.length, f = new Array(u), h = new Array(u), d = new Array(u), g = 0; g < u; ++g) {
    var v = o[g],
      b = s[g],
      w = b.length,
      S = Tct(t.call(v, v && v.__data__, g, o)),
      P = S.length,
      A = (h[g] = new Array(P)),
      L = (f[g] = new Array(P)),
      T = (d[g] = new Array(w));
    r(v, b, A, L, T, S, e);
    for (var M = 0, R = 0, E, B; M < P; ++M)
      if ((E = A[M])) {
        for (M >= R && (R = M + 1); !(B = L[R]) && ++R < P; );
        E._next = B || null;
      }
  }
  return (f = new Fn(f, o)), (f._enter = h), (f._exit = d), f;
}
function Tct(t) {
  return typeof t == "object" && "length" in t ? t : Array.from(t);
}
function Ect() {
  return new Fn(this._exit || this._groups.map(pb), this._parents);
}
function Lct(t, e, r) {
  var o = this.enter(),
    s = this,
    u = this.exit();
  return (
    typeof t == "function" ? ((o = t(o)), o && (o = o.selection())) : (o = o.append(t + "")),
    e != null && ((s = e(s)), s && (s = s.selection())),
    r == null ? u.remove() : r(u),
    o && s ? o.merge(s).order() : s
  );
}
function Act(t) {
  for (
    var e = t.selection ? t.selection() : t,
      r = this._groups,
      o = e._groups,
      s = r.length,
      u = o.length,
      f = Math.min(s, u),
      h = new Array(s),
      d = 0;
    d < f;
    ++d
  )
    for (var g = r[d], v = o[d], b = g.length, w = (h[d] = new Array(b)), S, P = 0; P < b; ++P)
      (S = g[P] || v[P]) && (w[P] = S);
  for (; d < s; ++d) h[d] = r[d];
  return new Fn(h, this._parents);
}
function Mct() {
  for (var t = this._groups, e = -1, r = t.length; ++e < r; )
    for (var o = t[e], s = o.length - 1, u = o[s], f; --s >= 0; )
      (f = o[s]) &&
        (u && f.compareDocumentPosition(u) ^ 4 && u.parentNode.insertBefore(f, u), (u = f));
  return this;
}
function Nct(t) {
  t || (t = Pct);
  function e(b, w) {
    return b && w ? t(b.__data__, w.__data__) : !b - !w;
  }
  for (var r = this._groups, o = r.length, s = new Array(o), u = 0; u < o; ++u) {
    for (var f = r[u], h = f.length, d = (s[u] = new Array(h)), g, v = 0; v < h; ++v)
      (g = f[v]) && (d[v] = g);
    d.sort(e);
  }
  return new Fn(s, this._parents).order();
}
function Pct(t, e) {
  return t < e ? -1 : t > e ? 1 : t >= e ? 0 : NaN;
}
function Oct() {
  var t = arguments[0];
  return (arguments[0] = this), t.apply(null, arguments), this;
}
function Dct() {
  return Array.from(this);
}
function $ct() {
  for (var t = this._groups, e = 0, r = t.length; e < r; ++e)
    for (var o = t[e], s = 0, u = o.length; s < u; ++s) {
      var f = o[s];
      if (f) return f;
    }
  return null;
}
function Rct() {
  let t = 0;
  for (const e of this) ++t;
  return t;
}
function zct() {
  return !this.node();
}
function Ict(t) {
  for (var e = this._groups, r = 0, o = e.length; r < o; ++r)
    for (var s = e[r], u = 0, f = s.length, h; u < f; ++u)
      (h = s[u]) && t.call(h, h.__data__, u, s);
  return this;
}
function Fct(t) {
  return function () {
    this.removeAttribute(t);
  };
}
function qct(t) {
  return function () {
    this.removeAttributeNS(t.space, t.local);
  };
}
function Hct(t, e) {
  return function () {
    this.setAttribute(t, e);
  };
}
function Bct(t, e) {
  return function () {
    this.setAttributeNS(t.space, t.local, e);
  };
}
function Wct(t, e) {
  return function () {
    var r = e.apply(this, arguments);
    r == null ? this.removeAttribute(t) : this.setAttribute(t, r);
  };
}
function Uct(t, e) {
  return function () {
    var r = e.apply(this, arguments);
    r == null ? this.removeAttributeNS(t.space, t.local) : this.setAttributeNS(t.space, t.local, r);
  };
}
function jct(t, e) {
  var r = Gc(t);
  if (arguments.length < 2) {
    var o = this.node();
    return r.local ? o.getAttributeNS(r.space, r.local) : o.getAttribute(r);
  }
  return this.each(
    (e == null
      ? r.local
        ? qct
        : Fct
      : typeof e == "function"
      ? r.local
        ? Uct
        : Wct
      : r.local
      ? Bct
      : Hct)(r, e),
  );
}
function gb(t) {
  return (t.ownerDocument && t.ownerDocument.defaultView) || (t.document && t) || t.defaultView;
}
function Vct(t) {
  return function () {
    this.style.removeProperty(t);
  };
}
function Gct(t, e, r) {
  return function () {
    this.style.setProperty(t, e, r);
  };
}
function Kct(t, e, r) {
  return function () {
    var o = e.apply(this, arguments);
    o == null ? this.style.removeProperty(t) : this.style.setProperty(t, o, r);
  };
}
function Xct(t, e, r) {
  return arguments.length > 1
    ? this.each((e == null ? Vct : typeof e == "function" ? Kct : Gct)(t, e, r ?? ""))
    : hs(this.node(), t);
}
function hs(t, e) {
  return t.style.getPropertyValue(e) || gb(t).getComputedStyle(t, null).getPropertyValue(e);
}
function Yct(t) {
  return function () {
    delete this[t];
  };
}
function Zct(t, e) {
  return function () {
    this[t] = e;
  };
}
function Jct(t, e) {
  return function () {
    var r = e.apply(this, arguments);
    r == null ? delete this[t] : (this[t] = r);
  };
}
function Qct(t, e) {
  return arguments.length > 1
    ? this.each((e == null ? Yct : typeof e == "function" ? Jct : Zct)(t, e))
    : this.node()[t];
}
function vb(t) {
  return t.trim().split(/^|\s+/);
}
function Qh(t) {
  return t.classList || new mb(t);
}
function mb(t) {
  (this._node = t), (this._names = vb(t.getAttribute("class") || ""));
}
mb.prototype = {
  add: function (t) {
    var e = this._names.indexOf(t);
    e < 0 && (this._names.push(t), this._node.setAttribute("class", this._names.join(" ")));
  },
  remove: function (t) {
    var e = this._names.indexOf(t);
    e >= 0 && (this._names.splice(e, 1), this._node.setAttribute("class", this._names.join(" ")));
  },
  contains: function (t) {
    return this._names.indexOf(t) >= 0;
  },
};
function yb(t, e) {
  for (var r = Qh(t), o = -1, s = e.length; ++o < s; ) r.add(e[o]);
}
function bb(t, e) {
  for (var r = Qh(t), o = -1, s = e.length; ++o < s; ) r.remove(e[o]);
}
function tut(t) {
  return function () {
    yb(this, t);
  };
}
function eut(t) {
  return function () {
    bb(this, t);
  };
}
function nut(t, e) {
  return function () {
    (e.apply(this, arguments) ? yb : bb)(this, t);
  };
}
function rut(t, e) {
  var r = vb(t + "");
  if (arguments.length < 2) {
    for (var o = Qh(this.node()), s = -1, u = r.length; ++s < u; ) if (!o.contains(r[s])) return !1;
    return !0;
  }
  return this.each((typeof e == "function" ? nut : e ? tut : eut)(r, e));
}
function iut() {
  this.textContent = "";
}
function out(t) {
  return function () {
    this.textContent = t;
  };
}
function sut(t) {
  return function () {
    var e = t.apply(this, arguments);
    this.textContent = e ?? "";
  };
}
function lut(t) {
  return arguments.length
    ? this.each(t == null ? iut : (typeof t == "function" ? sut : out)(t))
    : this.node().textContent;
}
function aut() {
  this.innerHTML = "";
}
function cut(t) {
  return function () {
    this.innerHTML = t;
  };
}
function uut(t) {
  return function () {
    var e = t.apply(this, arguments);
    this.innerHTML = e ?? "";
  };
}
function fut(t) {
  return arguments.length
    ? this.each(t == null ? aut : (typeof t == "function" ? uut : cut)(t))
    : this.node().innerHTML;
}
function hut() {
  this.nextSibling && this.parentNode.appendChild(this);
}
function dut() {
  return this.each(hut);
}
function put() {
  this.previousSibling && this.parentNode.insertBefore(this, this.parentNode.firstChild);
}
function gut() {
  return this.each(put);
}
function vut(t) {
  var e = typeof t == "function" ? t : ub(t);
  return this.select(function () {
    return this.appendChild(e.apply(this, arguments));
  });
}
function mut() {
  return null;
}
function yut(t, e) {
  var r = typeof t == "function" ? t : ub(t),
    o = e == null ? mut : typeof e == "function" ? e : Jh(e);
  return this.select(function () {
    return this.insertBefore(r.apply(this, arguments), o.apply(this, arguments) || null);
  });
}
function but() {
  var t = this.parentNode;
  t && t.removeChild(this);
}
function wut() {
  return this.each(but);
}
function xut() {
  var t = this.cloneNode(!1),
    e = this.parentNode;
  return e ? e.insertBefore(t, this.nextSibling) : t;
}
function _ut() {
  var t = this.cloneNode(!0),
    e = this.parentNode;
  return e ? e.insertBefore(t, this.nextSibling) : t;
}
function Sut(t) {
  return this.select(t ? _ut : xut);
}
function kut(t) {
  return arguments.length ? this.property("__data__", t) : this.node().__data__;
}
function Cut(t) {
  return function (e) {
    t.call(this, e, this.__data__);
  };
}
function Tut(t) {
  return t
    .trim()
    .split(/^|\s+/)
    .map(function (e) {
      var r = "",
        o = e.indexOf(".");
      return o >= 0 && ((r = e.slice(o + 1)), (e = e.slice(0, o))), { type: e, name: r };
    });
}
function Eut(t) {
  return function () {
    var e = this.__on;
    if (e) {
      for (var r = 0, o = -1, s = e.length, u; r < s; ++r)
        (u = e[r]),
          (!t.type || u.type === t.type) && u.name === t.name
            ? this.removeEventListener(u.type, u.listener, u.options)
            : (e[++o] = u);
      ++o ? (e.length = o) : delete this.__on;
    }
  };
}
function Lut(t, e, r) {
  return function () {
    var o = this.__on,
      s,
      u = Cut(e);
    if (o) {
      for (var f = 0, h = o.length; f < h; ++f)
        if ((s = o[f]).type === t.type && s.name === t.name) {
          this.removeEventListener(s.type, s.listener, s.options),
            this.addEventListener(s.type, (s.listener = u), (s.options = r)),
            (s.value = e);
          return;
        }
    }
    this.addEventListener(t.type, u, r),
      (s = { type: t.type, name: t.name, value: e, listener: u, options: r }),
      o ? o.push(s) : (this.__on = [s]);
  };
}
function Aut(t, e, r) {
  var o = Tut(t + ""),
    s,
    u = o.length,
    f;
  if (arguments.length < 2) {
    var h = this.node().__on;
    if (h) {
      for (var d = 0, g = h.length, v; d < g; ++d)
        for (s = 0, v = h[d]; s < u; ++s)
          if ((f = o[s]).type === v.type && f.name === v.name) return v.value;
    }
    return;
  }
  for (h = e ? Lut : Eut, s = 0; s < u; ++s) this.each(h(o[s], e, r));
  return this;
}
function wb(t, e, r) {
  var o = gb(t),
    s = o.CustomEvent;
  typeof s == "function"
    ? (s = new s(e, r))
    : ((s = o.document.createEvent("Event")),
      r
        ? (s.initEvent(e, r.bubbles, r.cancelable), (s.detail = r.detail))
        : s.initEvent(e, !1, !1)),
    t.dispatchEvent(s);
}
function Mut(t, e) {
  return function () {
    return wb(this, t, e);
  };
}
function Nut(t, e) {
  return function () {
    return wb(this, t, e.apply(this, arguments));
  };
}
function Put(t, e) {
  return this.each((typeof e == "function" ? Nut : Mut)(t, e));
}
function* Out() {
  for (var t = this._groups, e = 0, r = t.length; e < r; ++e)
    for (var o = t[e], s = 0, u = o.length, f; s < u; ++s) (f = o[s]) && (yield f);
}
var xb = [null];
function Fn(t, e) {
  (this._groups = t), (this._parents = e);
}
function Il() {
  return new Fn([[document.documentElement]], xb);
}
function Dut() {
  return this;
}
Fn.prototype = Il.prototype = {
  constructor: Fn,
  select: sct,
  selectAll: uct,
  selectChild: pct,
  selectChildren: yct,
  filter: bct,
  data: Cct,
  enter: wct,
  exit: Ect,
  join: Lct,
  merge: Act,
  selection: Dut,
  order: Mct,
  sort: Nct,
  call: Oct,
  nodes: Dct,
  node: $ct,
  size: Rct,
  empty: zct,
  each: Ict,
  attr: jct,
  style: Xct,
  property: Qct,
  classed: rut,
  text: lut,
  html: fut,
  raise: dut,
  lower: gut,
  append: vut,
  insert: yut,
  remove: wut,
  clone: Sut,
  datum: kut,
  on: Aut,
  dispatch: Put,
  [Symbol.iterator]: Out,
};
function En(t) {
  return typeof t == "string"
    ? new Fn([[document.querySelector(t)]], [document.documentElement])
    : new Fn([[t]], xb);
}
function $ut(t) {
  let e;
  for (; (e = t.sourceEvent); ) t = e;
  return t;
}
function Rr(t, e) {
  if (((t = $ut(t)), e === void 0 && (e = t.currentTarget), e)) {
    var r = e.ownerSVGElement || e;
    if (r.createSVGPoint) {
      var o = r.createSVGPoint();
      return (
        (o.x = t.clientX),
        (o.y = t.clientY),
        (o = o.matrixTransform(e.getScreenCTM().inverse())),
        [o.x, o.y]
      );
    }
    if (e.getBoundingClientRect) {
      var s = e.getBoundingClientRect();
      return [t.clientX - s.left - e.clientLeft, t.clientY - s.top - e.clientTop];
    }
  }
  return [t.pageX, t.pageY];
}
class pn {
  constructor(e, r) {
    (this.x = e), (this.y = r);
  }
  static of([e, r]) {
    return new pn(e, r);
  }
  add(e) {
    return new pn(this.x + e.x, this.y + e.y);
  }
  subtract(e) {
    return new pn(this.x - e.x, this.y - e.y);
  }
  multiply(e) {
    return new pn(this.x * e, this.y * e);
  }
  divide(e) {
    return new pn(this.x / e, this.y / e);
  }
  dot(e) {
    return this.x * e.x + this.y * e.y;
  }
  cross(e) {
    return this.x * e.y - e.x * this.y;
  }
  hadamard(e) {
    return new pn(this.x * e.x, this.y * e.y);
  }
  length() {
    return Math.sqrt(this.x ** 2 + this.y ** 2);
  }
  normalize() {
    const e = this.length();
    return new pn(this.x / e, this.y / e);
  }
  rotateByRadians(e) {
    const r = Math.cos(e),
      o = Math.sin(e);
    return new pn(this.x * r - this.y * o, this.x * o + this.y * r);
  }
  rotateByDegrees(e) {
    return this.rotateByRadians((e * Math.PI) / 180);
  }
}
var Rut = { value: () => {} };
function Fl() {
  for (var t = 0, e = arguments.length, r = {}, o; t < e; ++t) {
    if (!(o = arguments[t] + "") || o in r || /[\s.]/.test(o))
      throw new Error("illegal type: " + o);
    r[o] = [];
  }
  return new Ga(r);
}
function Ga(t) {
  this._ = t;
}
function zut(t, e) {
  return t
    .trim()
    .split(/^|\s+/)
    .map(function (r) {
      var o = "",
        s = r.indexOf(".");
      if ((s >= 0 && ((o = r.slice(s + 1)), (r = r.slice(0, s))), r && !e.hasOwnProperty(r)))
        throw new Error("unknown type: " + r);
      return { type: r, name: o };
    });
}
Ga.prototype = Fl.prototype = {
  constructor: Ga,
  on: function (t, e) {
    var r = this._,
      o = zut(t + "", r),
      s,
      u = -1,
      f = o.length;
    if (arguments.length < 2) {
      for (; ++u < f; ) if ((s = (t = o[u]).type) && (s = Iut(r[s], t.name))) return s;
      return;
    }
    if (e != null && typeof e != "function") throw new Error("invalid callback: " + e);
    for (; ++u < f; )
      if ((s = (t = o[u]).type)) r[s] = em(r[s], t.name, e);
      else if (e == null) for (s in r) r[s] = em(r[s], t.name, null);
    return this;
  },
  copy: function () {
    var t = {},
      e = this._;
    for (var r in e) t[r] = e[r].slice();
    return new Ga(t);
  },
  call: function (t, e) {
    if ((s = arguments.length - 2) > 0)
      for (var r = new Array(s), o = 0, s, u; o < s; ++o) r[o] = arguments[o + 2];
    if (!this._.hasOwnProperty(t)) throw new Error("unknown type: " + t);
    for (u = this._[t], o = 0, s = u.length; o < s; ++o) u[o].value.apply(e, r);
  },
  apply: function (t, e, r) {
    if (!this._.hasOwnProperty(t)) throw new Error("unknown type: " + t);
    for (var o = this._[t], s = 0, u = o.length; s < u; ++s) o[s].value.apply(e, r);
  },
};
function Iut(t, e) {
  for (var r = 0, o = t.length, s; r < o; ++r) if ((s = t[r]).name === e) return s.value;
}
function em(t, e, r) {
  for (var o = 0, s = t.length; o < s; ++o)
    if (t[o].name === e) {
      (t[o] = Rut), (t = t.slice(0, o).concat(t.slice(o + 1)));
      break;
    }
  return r != null && t.push({ name: e, value: r }), t;
}
const Fut = { passive: !1 },
  Cl = { capture: !0, passive: !1 };
function ff(t) {
  t.stopImmediatePropagation();
}
function Qo(t) {
  t.preventDefault(), t.stopImmediatePropagation();
}
function _b(t) {
  var e = t.document.documentElement,
    r = En(t).on("dragstart.drag", Qo, Cl);
  "onselectstart" in e
    ? r.on("selectstart.drag", Qo, Cl)
    : ((e.__noselect = e.style.MozUserSelect), (e.style.MozUserSelect = "none"));
}
function Sb(t, e) {
  var r = t.document.documentElement,
    o = En(t).on("dragstart.drag", null);
  e &&
    (o.on("click.drag", Qo, Cl),
    setTimeout(function () {
      o.on("click.drag", null);
    }, 0)),
    "onselectstart" in r
      ? o.on("selectstart.drag", null)
      : ((r.style.MozUserSelect = r.__noselect), delete r.__noselect);
}
const Pa = (t) => () => t;
function Zf(
  t,
  {
    sourceEvent: e,
    subject: r,
    target: o,
    identifier: s,
    active: u,
    x: f,
    y: h,
    dx: d,
    dy: g,
    dispatch: v,
  },
) {
  Object.defineProperties(this, {
    type: { value: t, enumerable: !0, configurable: !0 },
    sourceEvent: { value: e, enumerable: !0, configurable: !0 },
    subject: { value: r, enumerable: !0, configurable: !0 },
    target: { value: o, enumerable: !0, configurable: !0 },
    identifier: { value: s, enumerable: !0, configurable: !0 },
    active: { value: u, enumerable: !0, configurable: !0 },
    x: { value: f, enumerable: !0, configurable: !0 },
    y: { value: h, enumerable: !0, configurable: !0 },
    dx: { value: d, enumerable: !0, configurable: !0 },
    dy: { value: g, enumerable: !0, configurable: !0 },
    _: { value: v },
  });
}
Zf.prototype.on = function () {
  var t = this._.on.apply(this._, arguments);
  return t === this._ ? this : t;
};
function qut(t) {
  return !t.ctrlKey && !t.button;
}
function Hut() {
  return this.parentNode;
}
function But(t, e) {
  return e ?? { x: t.x, y: t.y };
}
function Wut() {
  return navigator.maxTouchPoints || "ontouchstart" in this;
}
function Uut() {
  var t = qut,
    e = Hut,
    r = But,
    o = Wut,
    s = {},
    u = Fl("start", "drag", "end"),
    f = 0,
    h,
    d,
    g,
    v,
    b = 0;
  function w(E) {
    E.on("mousedown.drag", S)
      .filter(o)
      .on("touchstart.drag", L)
      .on("touchmove.drag", T, Fut)
      .on("touchend.drag touchcancel.drag", M)
      .style("touch-action", "none")
      .style("-webkit-tap-highlight-color", "rgba(0,0,0,0)");
  }
  function S(E, B) {
    if (!(v || !t.call(this, E, B))) {
      var K = R(this, e.call(this, E, B), E, B, "mouse");
      K &&
        (En(E.view).on("mousemove.drag", P, Cl).on("mouseup.drag", A, Cl),
        _b(E.view),
        ff(E),
        (g = !1),
        (h = E.clientX),
        (d = E.clientY),
        K("start", E));
    }
  }
  function P(E) {
    if ((Qo(E), !g)) {
      var B = E.clientX - h,
        K = E.clientY - d;
      g = B * B + K * K > b;
    }
    s.mouse("drag", E);
  }
  function A(E) {
    En(E.view).on("mousemove.drag mouseup.drag", null), Sb(E.view, g), Qo(E), s.mouse("end", E);
  }
  function L(E, B) {
    if (t.call(this, E, B)) {
      var K = E.changedTouches,
        ht = e.call(this, E, B),
        Y = K.length,
        nt,
        at;
      for (nt = 0; nt < Y; ++nt)
        (at = R(this, ht, E, B, K[nt].identifier, K[nt])) && (ff(E), at("start", E, K[nt]));
    }
  }
  function T(E) {
    var B = E.changedTouches,
      K = B.length,
      ht,
      Y;
    for (ht = 0; ht < K; ++ht) (Y = s[B[ht].identifier]) && (Qo(E), Y("drag", E, B[ht]));
  }
  function M(E) {
    var B = E.changedTouches,
      K = B.length,
      ht,
      Y;
    for (
      v && clearTimeout(v),
        v = setTimeout(function () {
          v = null;
        }, 500),
        ht = 0;
      ht < K;
      ++ht
    )
      (Y = s[B[ht].identifier]) && (ff(E), Y("end", E, B[ht]));
  }
  function R(E, B, K, ht, Y, nt) {
    var at = u.copy(),
      pt = Rr(nt || K, B),
      gt,
      G,
      z;
    if (
      (z = r.call(
        E,
        new Zf("beforestart", {
          sourceEvent: K,
          target: w,
          identifier: Y,
          active: f,
          x: pt[0],
          y: pt[1],
          dx: 0,
          dy: 0,
          dispatch: at,
        }),
        ht,
      )) != null
    )
      return (
        (gt = z.x - pt[0] || 0),
        (G = z.y - pt[1] || 0),
        function k(F, H, J) {
          var yt = pt,
            At;
          switch (F) {
            case "start":
              (s[Y] = k), (At = f++);
              break;
            case "end":
              delete s[Y], --f;
            case "drag":
              (pt = Rr(J || H, B)), (At = f);
              break;
          }
          at.call(
            F,
            E,
            new Zf(F, {
              sourceEvent: H,
              subject: z,
              target: w,
              identifier: Y,
              active: At,
              x: pt[0] + gt,
              y: pt[1] + G,
              dx: pt[0] - yt[0],
              dy: pt[1] - yt[1],
              dispatch: at,
            }),
            ht,
          );
        }
      );
  }
  return (
    (w.filter = function (E) {
      return arguments.length ? ((t = typeof E == "function" ? E : Pa(!!E)), w) : t;
    }),
    (w.container = function (E) {
      return arguments.length ? ((e = typeof E == "function" ? E : Pa(E)), w) : e;
    }),
    (w.subject = function (E) {
      return arguments.length ? ((r = typeof E == "function" ? E : Pa(E)), w) : r;
    }),
    (w.touchable = function (E) {
      return arguments.length ? ((o = typeof E == "function" ? E : Pa(!!E)), w) : o;
    }),
    (w.on = function () {
      var E = u.on.apply(u, arguments);
      return E === u ? w : E;
    }),
    (w.clickDistance = function (E) {
      return arguments.length ? ((b = (E = +E) * E), w) : Math.sqrt(b);
    }),
    w
  );
}
function td(t, e, r) {
  (t.prototype = e.prototype = r), (r.constructor = t);
}
function kb(t, e) {
  var r = Object.create(t.prototype);
  for (var o in e) r[o] = e[o];
  return r;
}
function ql() {}
var Tl = 0.7,
  vc = 1 / Tl,
  ts = "\\s*([+-]?\\d+)\\s*",
  El = "\\s*([+-]?(?:\\d*\\.)?\\d+(?:[eE][+-]?\\d+)?)\\s*",
  xr = "\\s*([+-]?(?:\\d*\\.)?\\d+(?:[eE][+-]?\\d+)?)%\\s*",
  jut = /^#([0-9a-f]{3,8})$/,
  Vut = new RegExp(`^rgb\\(${ts},${ts},${ts}\\)$`),
  Gut = new RegExp(`^rgb\\(${xr},${xr},${xr}\\)$`),
  Kut = new RegExp(`^rgba\\(${ts},${ts},${ts},${El}\\)$`),
  Xut = new RegExp(`^rgba\\(${xr},${xr},${xr},${El}\\)$`),
  Yut = new RegExp(`^hsl\\(${El},${xr},${xr}\\)$`),
  Zut = new RegExp(`^hsla\\(${El},${xr},${xr},${El}\\)$`),
  nm = {
    aliceblue: 15792383,
    antiquewhite: 16444375,
    aqua: 65535,
    aquamarine: 8388564,
    azure: 15794175,
    beige: 16119260,
    bisque: 16770244,
    black: 0,
    blanchedalmond: 16772045,
    blue: 255,
    blueviolet: 9055202,
    brown: 10824234,
    burlywood: 14596231,
    cadetblue: 6266528,
    chartreuse: 8388352,
    chocolate: 13789470,
    coral: 16744272,
    cornflowerblue: 6591981,
    cornsilk: 16775388,
    crimson: 14423100,
    cyan: 65535,
    darkblue: 139,
    darkcyan: 35723,
    darkgoldenrod: 12092939,
    darkgray: 11119017,
    darkgreen: 25600,
    darkgrey: 11119017,
    darkkhaki: 12433259,
    darkmagenta: 9109643,
    darkolivegreen: 5597999,
    darkorange: 16747520,
    darkorchid: 10040012,
    darkred: 9109504,
    darksalmon: 15308410,
    darkseagreen: 9419919,
    darkslateblue: 4734347,
    darkslategray: 3100495,
    darkslategrey: 3100495,
    darkturquoise: 52945,
    darkviolet: 9699539,
    deeppink: 16716947,
    deepskyblue: 49151,
    dimgray: 6908265,
    dimgrey: 6908265,
    dodgerblue: 2003199,
    firebrick: 11674146,
    floralwhite: 16775920,
    forestgreen: 2263842,
    fuchsia: 16711935,
    gainsboro: 14474460,
    ghostwhite: 16316671,
    gold: 16766720,
    goldenrod: 14329120,
    gray: 8421504,
    green: 32768,
    greenyellow: 11403055,
    grey: 8421504,
    honeydew: 15794160,
    hotpink: 16738740,
    indianred: 13458524,
    indigo: 4915330,
    ivory: 16777200,
    khaki: 15787660,
    lavender: 15132410,
    lavenderblush: 16773365,
    lawngreen: 8190976,
    lemonchiffon: 16775885,
    lightblue: 11393254,
    lightcoral: 15761536,
    lightcyan: 14745599,
    lightgoldenrodyellow: 16448210,
    lightgray: 13882323,
    lightgreen: 9498256,
    lightgrey: 13882323,
    lightpink: 16758465,
    lightsalmon: 16752762,
    lightseagreen: 2142890,
    lightskyblue: 8900346,
    lightslategray: 7833753,
    lightslategrey: 7833753,
    lightsteelblue: 11584734,
    lightyellow: 16777184,
    lime: 65280,
    limegreen: 3329330,
    linen: 16445670,
    magenta: 16711935,
    maroon: 8388608,
    mediumaquamarine: 6737322,
    mediumblue: 205,
    mediumorchid: 12211667,
    mediumpurple: 9662683,
    mediumseagreen: 3978097,
    mediumslateblue: 8087790,
    mediumspringgreen: 64154,
    mediumturquoise: 4772300,
    mediumvioletred: 13047173,
    midnightblue: 1644912,
    mintcream: 16121850,
    mistyrose: 16770273,
    moccasin: 16770229,
    navajowhite: 16768685,
    navy: 128,
    oldlace: 16643558,
    olive: 8421376,
    olivedrab: 7048739,
    orange: 16753920,
    orangered: 16729344,
    orchid: 14315734,
    palegoldenrod: 15657130,
    palegreen: 10025880,
    paleturquoise: 11529966,
    palevioletred: 14381203,
    papayawhip: 16773077,
    peachpuff: 16767673,
    peru: 13468991,
    pink: 16761035,
    plum: 14524637,
    powderblue: 11591910,
    purple: 8388736,
    rebeccapurple: 6697881,
    red: 16711680,
    rosybrown: 12357519,
    royalblue: 4286945,
    saddlebrown: 9127187,
    salmon: 16416882,
    sandybrown: 16032864,
    seagreen: 3050327,
    seashell: 16774638,
    sienna: 10506797,
    silver: 12632256,
    skyblue: 8900331,
    slateblue: 6970061,
    slategray: 7372944,
    slategrey: 7372944,
    snow: 16775930,
    springgreen: 65407,
    steelblue: 4620980,
    tan: 13808780,
    teal: 32896,
    thistle: 14204888,
    tomato: 16737095,
    turquoise: 4251856,
    violet: 15631086,
    wheat: 16113331,
    white: 16777215,
    whitesmoke: 16119285,
    yellow: 16776960,
    yellowgreen: 10145074,
  };
td(ql, Ll, {
  copy(t) {
    return Object.assign(new this.constructor(), this, t);
  },
  displayable() {
    return this.rgb().displayable();
  },
  hex: rm,
  formatHex: rm,
  formatHex8: Jut,
  formatHsl: Qut,
  formatRgb: im,
  toString: im,
});
function rm() {
  return this.rgb().formatHex();
}
function Jut() {
  return this.rgb().formatHex8();
}
function Qut() {
  return Cb(this).formatHsl();
}
function im() {
  return this.rgb().formatRgb();
}
function Ll(t) {
  var e, r;
  return (
    (t = (t + "").trim().toLowerCase()),
    (e = jut.exec(t))
      ? ((r = e[1].length),
        (e = parseInt(e[1], 16)),
        r === 6
          ? om(e)
          : r === 3
          ? new Ln(
              ((e >> 8) & 15) | ((e >> 4) & 240),
              ((e >> 4) & 15) | (e & 240),
              ((e & 15) << 4) | (e & 15),
              1,
            )
          : r === 8
          ? Oa((e >> 24) & 255, (e >> 16) & 255, (e >> 8) & 255, (e & 255) / 255)
          : r === 4
          ? Oa(
              ((e >> 12) & 15) | ((e >> 8) & 240),
              ((e >> 8) & 15) | ((e >> 4) & 240),
              ((e >> 4) & 15) | (e & 240),
              (((e & 15) << 4) | (e & 15)) / 255,
            )
          : null)
      : (e = Vut.exec(t))
      ? new Ln(e[1], e[2], e[3], 1)
      : (e = Gut.exec(t))
      ? new Ln((e[1] * 255) / 100, (e[2] * 255) / 100, (e[3] * 255) / 100, 1)
      : (e = Kut.exec(t))
      ? Oa(e[1], e[2], e[3], e[4])
      : (e = Xut.exec(t))
      ? Oa((e[1] * 255) / 100, (e[2] * 255) / 100, (e[3] * 255) / 100, e[4])
      : (e = Yut.exec(t))
      ? am(e[1], e[2] / 100, e[3] / 100, 1)
      : (e = Zut.exec(t))
      ? am(e[1], e[2] / 100, e[3] / 100, e[4])
      : nm.hasOwnProperty(t)
      ? om(nm[t])
      : t === "transparent"
      ? new Ln(NaN, NaN, NaN, 0)
      : null
  );
}
function om(t) {
  return new Ln((t >> 16) & 255, (t >> 8) & 255, t & 255, 1);
}
function Oa(t, e, r, o) {
  return o <= 0 && (t = e = r = NaN), new Ln(t, e, r, o);
}
function tft(t) {
  return (
    t instanceof ql || (t = Ll(t)), t ? ((t = t.rgb()), new Ln(t.r, t.g, t.b, t.opacity)) : new Ln()
  );
}
function Jf(t, e, r, o) {
  return arguments.length === 1 ? tft(t) : new Ln(t, e, r, o ?? 1);
}
function Ln(t, e, r, o) {
  (this.r = +t), (this.g = +e), (this.b = +r), (this.opacity = +o);
}
td(
  Ln,
  Jf,
  kb(ql, {
    brighter(t) {
      return (
        (t = t == null ? vc : Math.pow(vc, t)),
        new Ln(this.r * t, this.g * t, this.b * t, this.opacity)
      );
    },
    darker(t) {
      return (
        (t = t == null ? Tl : Math.pow(Tl, t)),
        new Ln(this.r * t, this.g * t, this.b * t, this.opacity)
      );
    },
    rgb() {
      return this;
    },
    clamp() {
      return new Ln(no(this.r), no(this.g), no(this.b), mc(this.opacity));
    },
    displayable() {
      return (
        -0.5 <= this.r &&
        this.r < 255.5 &&
        -0.5 <= this.g &&
        this.g < 255.5 &&
        -0.5 <= this.b &&
        this.b < 255.5 &&
        0 <= this.opacity &&
        this.opacity <= 1
      );
    },
    hex: sm,
    formatHex: sm,
    formatHex8: eft,
    formatRgb: lm,
    toString: lm,
  }),
);
function sm() {
  return `#${Ji(this.r)}${Ji(this.g)}${Ji(this.b)}`;
}
function eft() {
  return `#${Ji(this.r)}${Ji(this.g)}${Ji(this.b)}${Ji(
    (isNaN(this.opacity) ? 1 : this.opacity) * 255,
  )}`;
}
function lm() {
  const t = mc(this.opacity);
  return `${t === 1 ? "rgb(" : "rgba("}${no(this.r)}, ${no(this.g)}, ${no(this.b)}${
    t === 1 ? ")" : `, ${t})`
  }`;
}
function mc(t) {
  return isNaN(t) ? 1 : Math.max(0, Math.min(1, t));
}
function no(t) {
  return Math.max(0, Math.min(255, Math.round(t) || 0));
}
function Ji(t) {
  return (t = no(t)), (t < 16 ? "0" : "") + t.toString(16);
}
function am(t, e, r, o) {
  return (
    o <= 0 ? (t = e = r = NaN) : r <= 0 || r >= 1 ? (t = e = NaN) : e <= 0 && (t = NaN),
    new or(t, e, r, o)
  );
}
function Cb(t) {
  if (t instanceof or) return new or(t.h, t.s, t.l, t.opacity);
  if ((t instanceof ql || (t = Ll(t)), !t)) return new or();
  if (t instanceof or) return t;
  t = t.rgb();
  var e = t.r / 255,
    r = t.g / 255,
    o = t.b / 255,
    s = Math.min(e, r, o),
    u = Math.max(e, r, o),
    f = NaN,
    h = u - s,
    d = (u + s) / 2;
  return (
    h
      ? (e === u
          ? (f = (r - o) / h + (r < o) * 6)
          : r === u
          ? (f = (o - e) / h + 2)
          : (f = (e - r) / h + 4),
        (h /= d < 0.5 ? u + s : 2 - u - s),
        (f *= 60))
      : (h = d > 0 && d < 1 ? 0 : f),
    new or(f, h, d, t.opacity)
  );
}
function nft(t, e, r, o) {
  return arguments.length === 1 ? Cb(t) : new or(t, e, r, o ?? 1);
}
function or(t, e, r, o) {
  (this.h = +t), (this.s = +e), (this.l = +r), (this.opacity = +o);
}
td(
  or,
  nft,
  kb(ql, {
    brighter(t) {
      return (
        (t = t == null ? vc : Math.pow(vc, t)), new or(this.h, this.s, this.l * t, this.opacity)
      );
    },
    darker(t) {
      return (
        (t = t == null ? Tl : Math.pow(Tl, t)), new or(this.h, this.s, this.l * t, this.opacity)
      );
    },
    rgb() {
      var t = (this.h % 360) + (this.h < 0) * 360,
        e = isNaN(t) || isNaN(this.s) ? 0 : this.s,
        r = this.l,
        o = r + (r < 0.5 ? r : 1 - r) * e,
        s = 2 * r - o;
      return new Ln(
        hf(t >= 240 ? t - 240 : t + 120, s, o),
        hf(t, s, o),
        hf(t < 120 ? t + 240 : t - 120, s, o),
        this.opacity,
      );
    },
    clamp() {
      return new or(cm(this.h), Da(this.s), Da(this.l), mc(this.opacity));
    },
    displayable() {
      return (
        ((0 <= this.s && this.s <= 1) || isNaN(this.s)) &&
        0 <= this.l &&
        this.l <= 1 &&
        0 <= this.opacity &&
        this.opacity <= 1
      );
    },
    formatHsl() {
      const t = mc(this.opacity);
      return `${t === 1 ? "hsl(" : "hsla("}${cm(this.h)}, ${Da(this.s) * 100}%, ${
        Da(this.l) * 100
      }%${t === 1 ? ")" : `, ${t})`}`;
    },
  }),
);
function cm(t) {
  return (t = (t || 0) % 360), t < 0 ? t + 360 : t;
}
function Da(t) {
  return Math.max(0, Math.min(1, t || 0));
}
function hf(t, e, r) {
  return (
    (t < 60 ? e + ((r - e) * t) / 60 : t < 180 ? r : t < 240 ? e + ((r - e) * (240 - t)) / 60 : e) *
    255
  );
}
const Tb = (t) => () => t;
function rft(t, e) {
  return function (r) {
    return t + r * e;
  };
}
function ift(t, e, r) {
  return (
    (t = Math.pow(t, r)),
    (e = Math.pow(e, r) - t),
    (r = 1 / r),
    function (o) {
      return Math.pow(t + o * e, r);
    }
  );
}
function oft(t) {
  return (t = +t) == 1
    ? Eb
    : function (e, r) {
        return r - e ? ift(e, r, t) : Tb(isNaN(e) ? r : e);
      };
}
function Eb(t, e) {
  var r = e - t;
  return r ? rft(t, r) : Tb(isNaN(t) ? e : t);
}
const um = (function t(e) {
  var r = oft(e);
  function o(s, u) {
    var f = r((s = Jf(s)).r, (u = Jf(u)).r),
      h = r(s.g, u.g),
      d = r(s.b, u.b),
      g = Eb(s.opacity, u.opacity);
    return function (v) {
      return (s.r = f(v)), (s.g = h(v)), (s.b = d(v)), (s.opacity = g(v)), s + "";
    };
  }
  return (o.gamma = t), o;
})(1);
function bi(t, e) {
  return (
    (t = +t),
    (e = +e),
    function (r) {
      return t * (1 - r) + e * r;
    }
  );
}
var Qf = /[-+]?(?:\d+\.?\d*|\.?\d+)(?:[eE][-+]?\d+)?/g,
  df = new RegExp(Qf.source, "g");
function sft(t) {
  return function () {
    return t;
  };
}
function lft(t) {
  return function (e) {
    return t(e) + "";
  };
}
function aft(t, e) {
  var r = (Qf.lastIndex = df.lastIndex = 0),
    o,
    s,
    u,
    f = -1,
    h = [],
    d = [];
  for (t = t + "", e = e + ""; (o = Qf.exec(t)) && (s = df.exec(e)); )
    (u = s.index) > r && ((u = e.slice(r, u)), h[f] ? (h[f] += u) : (h[++f] = u)),
      (o = o[0]) === (s = s[0])
        ? h[f]
          ? (h[f] += s)
          : (h[++f] = s)
        : ((h[++f] = null), d.push({ i: f, x: bi(o, s) })),
      (r = df.lastIndex);
  return (
    r < e.length && ((u = e.slice(r)), h[f] ? (h[f] += u) : (h[++f] = u)),
    h.length < 2
      ? d[0]
        ? lft(d[0].x)
        : sft(e)
      : ((e = d.length),
        function (g) {
          for (var v = 0, b; v < e; ++v) h[(b = d[v]).i] = b.x(g);
          return h.join("");
        })
  );
}
var fm = 180 / Math.PI,
  th = { translateX: 0, translateY: 0, rotate: 0, skewX: 0, scaleX: 1, scaleY: 1 };
function Lb(t, e, r, o, s, u) {
  var f, h, d;
  return (
    (f = Math.sqrt(t * t + e * e)) && ((t /= f), (e /= f)),
    (d = t * r + e * o) && ((r -= t * d), (o -= e * d)),
    (h = Math.sqrt(r * r + o * o)) && ((r /= h), (o /= h), (d /= h)),
    t * o < e * r && ((t = -t), (e = -e), (d = -d), (f = -f)),
    {
      translateX: s,
      translateY: u,
      rotate: Math.atan2(e, t) * fm,
      skewX: Math.atan(d) * fm,
      scaleX: f,
      scaleY: h,
    }
  );
}
var $a;
function cft(t) {
  const e = new (typeof DOMMatrix == "function" ? DOMMatrix : WebKitCSSMatrix)(t + "");
  return e.isIdentity ? th : Lb(e.a, e.b, e.c, e.d, e.e, e.f);
}
function uft(t) {
  return t == null ||
    ($a || ($a = document.createElementNS("http://www.w3.org/2000/svg", "g")),
    $a.setAttribute("transform", t),
    !(t = $a.transform.baseVal.consolidate()))
    ? th
    : ((t = t.matrix), Lb(t.a, t.b, t.c, t.d, t.e, t.f));
}
function Ab(t, e, r, o) {
  function s(g) {
    return g.length ? g.pop() + " " : "";
  }
  function u(g, v, b, w, S, P) {
    if (g !== b || v !== w) {
      var A = S.push("translate(", null, e, null, r);
      P.push({ i: A - 4, x: bi(g, b) }, { i: A - 2, x: bi(v, w) });
    } else (b || w) && S.push("translate(" + b + e + w + r);
  }
  function f(g, v, b, w) {
    g !== v
      ? (g - v > 180 ? (v += 360) : v - g > 180 && (g += 360),
        w.push({ i: b.push(s(b) + "rotate(", null, o) - 2, x: bi(g, v) }))
      : v && b.push(s(b) + "rotate(" + v + o);
  }
  function h(g, v, b, w) {
    g !== v
      ? w.push({ i: b.push(s(b) + "skewX(", null, o) - 2, x: bi(g, v) })
      : v && b.push(s(b) + "skewX(" + v + o);
  }
  function d(g, v, b, w, S, P) {
    if (g !== b || v !== w) {
      var A = S.push(s(S) + "scale(", null, ",", null, ")");
      P.push({ i: A - 4, x: bi(g, b) }, { i: A - 2, x: bi(v, w) });
    } else (b !== 1 || w !== 1) && S.push(s(S) + "scale(" + b + "," + w + ")");
  }
  return function (g, v) {
    var b = [],
      w = [];
    return (
      (g = t(g)),
      (v = t(v)),
      u(g.translateX, g.translateY, v.translateX, v.translateY, b, w),
      f(g.rotate, v.rotate, b, w),
      h(g.skewX, v.skewX, b, w),
      d(g.scaleX, g.scaleY, v.scaleX, v.scaleY, b, w),
      (g = v = null),
      function (S) {
        for (var P = -1, A = w.length, L; ++P < A; ) b[(L = w[P]).i] = L.x(S);
        return b.join("");
      }
    );
  };
}
var fft = Ab(cft, "px, ", "px)", "deg)"),
  hft = Ab(uft, ", ", ")", ")"),
  dft = 1e-12;
function hm(t) {
  return ((t = Math.exp(t)) + 1 / t) / 2;
}
function pft(t) {
  return ((t = Math.exp(t)) - 1 / t) / 2;
}
function gft(t) {
  return ((t = Math.exp(2 * t)) - 1) / (t + 1);
}
const vft = (function t(e, r, o) {
  function s(u, f) {
    var h = u[0],
      d = u[1],
      g = u[2],
      v = f[0],
      b = f[1],
      w = f[2],
      S = v - h,
      P = b - d,
      A = S * S + P * P,
      L,
      T;
    if (A < dft)
      (T = Math.log(w / g) / e),
        (L = function (ht) {
          return [h + ht * S, d + ht * P, g * Math.exp(e * ht * T)];
        });
    else {
      var M = Math.sqrt(A),
        R = (w * w - g * g + o * A) / (2 * g * r * M),
        E = (w * w - g * g - o * A) / (2 * w * r * M),
        B = Math.log(Math.sqrt(R * R + 1) - R),
        K = Math.log(Math.sqrt(E * E + 1) - E);
      (T = (K - B) / e),
        (L = function (ht) {
          var Y = ht * T,
            nt = hm(B),
            at = (g / (r * M)) * (nt * gft(e * Y + B) - pft(B));
          return [h + at * S, d + at * P, (g * nt) / hm(e * Y + B)];
        });
    }
    return (L.duration = (T * 1e3 * e) / Math.SQRT2), L;
  }
  return (
    (s.rho = function (u) {
      var f = Math.max(0.001, +u),
        h = f * f,
        d = h * h;
      return t(f, h, d);
    }),
    s
  );
})(Math.SQRT2, 2, 4);
var ds = 0,
  ol = 0,
  el = 0,
  Mb = 1e3,
  yc,
  sl,
  bc = 0,
  lo = 0,
  Kc = 0,
  Al = typeof performance == "object" && performance.now ? performance : Date,
  Nb =
    typeof window == "object" && window.requestAnimationFrame
      ? window.requestAnimationFrame.bind(window)
      : function (t) {
          setTimeout(t, 17);
        };
function ed() {
  return lo || (Nb(mft), (lo = Al.now() + Kc));
}
function mft() {
  lo = 0;
}
function wc() {
  this._call = this._time = this._next = null;
}
wc.prototype = nd.prototype = {
  constructor: wc,
  restart: function (t, e, r) {
    if (typeof t != "function") throw new TypeError("callback is not a function");
    (r = (r == null ? ed() : +r) + (e == null ? 0 : +e)),
      !this._next && sl !== this && (sl ? (sl._next = this) : (yc = this), (sl = this)),
      (this._call = t),
      (this._time = r),
      eh();
  },
  stop: function () {
    this._call && ((this._call = null), (this._time = 1 / 0), eh());
  },
};
function nd(t, e, r) {
  var o = new wc();
  return o.restart(t, e, r), o;
}
function yft() {
  ed(), ++ds;
  for (var t = yc, e; t; ) (e = lo - t._time) >= 0 && t._call.call(void 0, e), (t = t._next);
  --ds;
}
function dm() {
  (lo = (bc = Al.now()) + Kc), (ds = ol = 0);
  try {
    yft();
  } finally {
    (ds = 0), wft(), (lo = 0);
  }
}
function bft() {
  var t = Al.now(),
    e = t - bc;
  e > Mb && ((Kc -= e), (bc = t));
}
function wft() {
  for (var t, e = yc, r, o = 1 / 0; e; )
    e._call
      ? (o > e._time && (o = e._time), (t = e), (e = e._next))
      : ((r = e._next), (e._next = null), (e = t ? (t._next = r) : (yc = r)));
  (sl = t), eh(o);
}
function eh(t) {
  if (!ds) {
    ol && (ol = clearTimeout(ol));
    var e = t - lo;
    e > 24
      ? (t < 1 / 0 && (ol = setTimeout(dm, t - Al.now() - Kc)), el && (el = clearInterval(el)))
      : (el || ((bc = Al.now()), (el = setInterval(bft, Mb))), (ds = 1), Nb(dm));
  }
}
function pm(t, e, r) {
  var o = new wc();
  return (
    (e = e == null ? 0 : +e),
    o.restart(
      (s) => {
        o.stop(), t(s + e);
      },
      e,
      r,
    ),
    o
  );
}
var xft = Fl("start", "end", "cancel", "interrupt"),
  _ft = [],
  Pb = 0,
  gm = 1,
  nh = 2,
  Ka = 3,
  vm = 4,
  rh = 5,
  Xa = 6;
function Xc(t, e, r, o, s, u) {
  var f = t.__transition;
  if (!f) t.__transition = {};
  else if (r in f) return;
  Sft(t, r, {
    name: e,
    index: o,
    group: s,
    on: xft,
    tween: _ft,
    time: u.time,
    delay: u.delay,
    duration: u.duration,
    ease: u.ease,
    timer: null,
    state: Pb,
  });
}
function rd(t, e) {
  var r = ar(t, e);
  if (r.state > Pb) throw new Error("too late; already scheduled");
  return r;
}
function kr(t, e) {
  var r = ar(t, e);
  if (r.state > Ka) throw new Error("too late; already running");
  return r;
}
function ar(t, e) {
  var r = t.__transition;
  if (!r || !(r = r[e])) throw new Error("transition not found");
  return r;
}
function Sft(t, e, r) {
  var o = t.__transition,
    s;
  (o[e] = r), (r.timer = nd(u, 0, r.time));
  function u(g) {
    (r.state = gm), r.timer.restart(f, r.delay, r.time), r.delay <= g && f(g - r.delay);
  }
  function f(g) {
    var v, b, w, S;
    if (r.state !== gm) return d();
    for (v in o)
      if (((S = o[v]), S.name === r.name)) {
        if (S.state === Ka) return pm(f);
        S.state === vm
          ? ((S.state = Xa),
            S.timer.stop(),
            S.on.call("interrupt", t, t.__data__, S.index, S.group),
            delete o[v])
          : +v < e &&
            ((S.state = Xa),
            S.timer.stop(),
            S.on.call("cancel", t, t.__data__, S.index, S.group),
            delete o[v]);
      }
    if (
      (pm(function () {
        r.state === Ka && ((r.state = vm), r.timer.restart(h, r.delay, r.time), h(g));
      }),
      (r.state = nh),
      r.on.call("start", t, t.__data__, r.index, r.group),
      r.state === nh)
    ) {
      for (r.state = Ka, s = new Array((w = r.tween.length)), v = 0, b = -1; v < w; ++v)
        (S = r.tween[v].value.call(t, t.__data__, r.index, r.group)) && (s[++b] = S);
      s.length = b + 1;
    }
  }
  function h(g) {
    for (
      var v =
          g < r.duration
            ? r.ease.call(null, g / r.duration)
            : (r.timer.restart(d), (r.state = rh), 1),
        b = -1,
        w = s.length;
      ++b < w;
    )
      s[b].call(t, v);
    r.state === rh && (r.on.call("end", t, t.__data__, r.index, r.group), d());
  }
  function d() {
    (r.state = Xa), r.timer.stop(), delete o[e];
    for (var g in o) return;
    delete t.__transition;
  }
}
function Ya(t, e) {
  var r = t.__transition,
    o,
    s,
    u = !0,
    f;
  if (r) {
    e = e == null ? null : e + "";
    for (f in r) {
      if ((o = r[f]).name !== e) {
        u = !1;
        continue;
      }
      (s = o.state > nh && o.state < rh),
        (o.state = Xa),
        o.timer.stop(),
        o.on.call(s ? "interrupt" : "cancel", t, t.__data__, o.index, o.group),
        delete r[f];
    }
    u && delete t.__transition;
  }
}
function kft(t) {
  return this.each(function () {
    Ya(this, t);
  });
}
function Cft(t, e) {
  var r, o;
  return function () {
    var s = kr(this, t),
      u = s.tween;
    if (u !== r) {
      o = r = u;
      for (var f = 0, h = o.length; f < h; ++f)
        if (o[f].name === e) {
          (o = o.slice()), o.splice(f, 1);
          break;
        }
    }
    s.tween = o;
  };
}
function Tft(t, e, r) {
  var o, s;
  if (typeof r != "function") throw new Error();
  return function () {
    var u = kr(this, t),
      f = u.tween;
    if (f !== o) {
      s = (o = f).slice();
      for (var h = { name: e, value: r }, d = 0, g = s.length; d < g; ++d)
        if (s[d].name === e) {
          s[d] = h;
          break;
        }
      d === g && s.push(h);
    }
    u.tween = s;
  };
}
function Eft(t, e) {
  var r = this._id;
  if (((t += ""), arguments.length < 2)) {
    for (var o = ar(this.node(), r).tween, s = 0, u = o.length, f; s < u; ++s)
      if ((f = o[s]).name === t) return f.value;
    return null;
  }
  return this.each((e == null ? Cft : Tft)(r, t, e));
}
function id(t, e, r) {
  var o = t._id;
  return (
    t.each(function () {
      var s = kr(this, o);
      (s.value || (s.value = {}))[e] = r.apply(this, arguments);
    }),
    function (s) {
      return ar(s, o).value[e];
    }
  );
}
function Ob(t, e) {
  var r;
  return (typeof e == "number" ? bi : e instanceof Ll ? um : (r = Ll(e)) ? ((e = r), um) : aft)(
    t,
    e,
  );
}
function Lft(t) {
  return function () {
    this.removeAttribute(t);
  };
}
function Aft(t) {
  return function () {
    this.removeAttributeNS(t.space, t.local);
  };
}
function Mft(t, e, r) {
  var o,
    s = r + "",
    u;
  return function () {
    var f = this.getAttribute(t);
    return f === s ? null : f === o ? u : (u = e((o = f), r));
  };
}
function Nft(t, e, r) {
  var o,
    s = r + "",
    u;
  return function () {
    var f = this.getAttributeNS(t.space, t.local);
    return f === s ? null : f === o ? u : (u = e((o = f), r));
  };
}
function Pft(t, e, r) {
  var o, s, u;
  return function () {
    var f,
      h = r(this),
      d;
    return h == null
      ? void this.removeAttribute(t)
      : ((f = this.getAttribute(t)),
        (d = h + ""),
        f === d ? null : f === o && d === s ? u : ((s = d), (u = e((o = f), h))));
  };
}
function Oft(t, e, r) {
  var o, s, u;
  return function () {
    var f,
      h = r(this),
      d;
    return h == null
      ? void this.removeAttributeNS(t.space, t.local)
      : ((f = this.getAttributeNS(t.space, t.local)),
        (d = h + ""),
        f === d ? null : f === o && d === s ? u : ((s = d), (u = e((o = f), h))));
  };
}
function Dft(t, e) {
  var r = Gc(t),
    o = r === "transform" ? hft : Ob;
  return this.attrTween(
    t,
    typeof e == "function"
      ? (r.local ? Oft : Pft)(r, o, id(this, "attr." + t, e))
      : e == null
      ? (r.local ? Aft : Lft)(r)
      : (r.local ? Nft : Mft)(r, o, e),
  );
}
function $ft(t, e) {
  return function (r) {
    this.setAttribute(t, e.call(this, r));
  };
}
function Rft(t, e) {
  return function (r) {
    this.setAttributeNS(t.space, t.local, e.call(this, r));
  };
}
function zft(t, e) {
  var r, o;
  function s() {
    var u = e.apply(this, arguments);
    return u !== o && (r = (o = u) && Rft(t, u)), r;
  }
  return (s._value = e), s;
}
function Ift(t, e) {
  var r, o;
  function s() {
    var u = e.apply(this, arguments);
    return u !== o && (r = (o = u) && $ft(t, u)), r;
  }
  return (s._value = e), s;
}
function Fft(t, e) {
  var r = "attr." + t;
  if (arguments.length < 2) return (r = this.tween(r)) && r._value;
  if (e == null) return this.tween(r, null);
  if (typeof e != "function") throw new Error();
  var o = Gc(t);
  return this.tween(r, (o.local ? zft : Ift)(o, e));
}
function qft(t, e) {
  return function () {
    rd(this, t).delay = +e.apply(this, arguments);
  };
}
function Hft(t, e) {
  return (
    (e = +e),
    function () {
      rd(this, t).delay = e;
    }
  );
}
function Bft(t) {
  var e = this._id;
  return arguments.length
    ? this.each((typeof t == "function" ? qft : Hft)(e, t))
    : ar(this.node(), e).delay;
}
function Wft(t, e) {
  return function () {
    kr(this, t).duration = +e.apply(this, arguments);
  };
}
function Uft(t, e) {
  return (
    (e = +e),
    function () {
      kr(this, t).duration = e;
    }
  );
}
function jft(t) {
  var e = this._id;
  return arguments.length
    ? this.each((typeof t == "function" ? Wft : Uft)(e, t))
    : ar(this.node(), e).duration;
}
function Vft(t, e) {
  if (typeof e != "function") throw new Error();
  return function () {
    kr(this, t).ease = e;
  };
}
function Gft(t) {
  var e = this._id;
  return arguments.length ? this.each(Vft(e, t)) : ar(this.node(), e).ease;
}
function Kft(t, e) {
  return function () {
    var r = e.apply(this, arguments);
    if (typeof r != "function") throw new Error();
    kr(this, t).ease = r;
  };
}
function Xft(t) {
  if (typeof t != "function") throw new Error();
  return this.each(Kft(this._id, t));
}
function Yft(t) {
  typeof t != "function" && (t = hb(t));
  for (var e = this._groups, r = e.length, o = new Array(r), s = 0; s < r; ++s)
    for (var u = e[s], f = u.length, h = (o[s] = []), d, g = 0; g < f; ++g)
      (d = u[g]) && t.call(d, d.__data__, g, u) && h.push(d);
  return new Gr(o, this._parents, this._name, this._id);
}
function Zft(t) {
  if (t._id !== this._id) throw new Error();
  for (
    var e = this._groups,
      r = t._groups,
      o = e.length,
      s = r.length,
      u = Math.min(o, s),
      f = new Array(o),
      h = 0;
    h < u;
    ++h
  )
    for (var d = e[h], g = r[h], v = d.length, b = (f[h] = new Array(v)), w, S = 0; S < v; ++S)
      (w = d[S] || g[S]) && (b[S] = w);
  for (; h < o; ++h) f[h] = e[h];
  return new Gr(f, this._parents, this._name, this._id);
}
function Jft(t) {
  return (t + "")
    .trim()
    .split(/^|\s+/)
    .every(function (e) {
      var r = e.indexOf(".");
      return r >= 0 && (e = e.slice(0, r)), !e || e === "start";
    });
}
function Qft(t, e, r) {
  var o,
    s,
    u = Jft(e) ? rd : kr;
  return function () {
    var f = u(this, t),
      h = f.on;
    h !== o && (s = (o = h).copy()).on(e, r), (f.on = s);
  };
}
function tht(t, e) {
  var r = this._id;
  return arguments.length < 2 ? ar(this.node(), r).on.on(t) : this.each(Qft(r, t, e));
}
function eht(t) {
  return function () {
    var e = this.parentNode;
    for (var r in this.__transition) if (+r !== t) return;
    e && e.removeChild(this);
  };
}
function nht() {
  return this.on("end.remove", eht(this._id));
}
function rht(t) {
  var e = this._name,
    r = this._id;
  typeof t != "function" && (t = Jh(t));
  for (var o = this._groups, s = o.length, u = new Array(s), f = 0; f < s; ++f)
    for (var h = o[f], d = h.length, g = (u[f] = new Array(d)), v, b, w = 0; w < d; ++w)
      (v = h[w]) &&
        (b = t.call(v, v.__data__, w, h)) &&
        ("__data__" in v && (b.__data__ = v.__data__), (g[w] = b), Xc(g[w], e, r, w, g, ar(v, r)));
  return new Gr(u, this._parents, e, r);
}
function iht(t) {
  var e = this._name,
    r = this._id;
  typeof t != "function" && (t = fb(t));
  for (var o = this._groups, s = o.length, u = [], f = [], h = 0; h < s; ++h)
    for (var d = o[h], g = d.length, v, b = 0; b < g; ++b)
      if ((v = d[b])) {
        for (var w = t.call(v, v.__data__, b, d), S, P = ar(v, r), A = 0, L = w.length; A < L; ++A)
          (S = w[A]) && Xc(S, e, r, A, w, P);
        u.push(w), f.push(v);
      }
  return new Gr(u, f, e, r);
}
var oht = Il.prototype.constructor;
function sht() {
  return new oht(this._groups, this._parents);
}
function lht(t, e) {
  var r, o, s;
  return function () {
    var u = hs(this, t),
      f = (this.style.removeProperty(t), hs(this, t));
    return u === f ? null : u === r && f === o ? s : (s = e((r = u), (o = f)));
  };
}
function Db(t) {
  return function () {
    this.style.removeProperty(t);
  };
}
function aht(t, e, r) {
  var o,
    s = r + "",
    u;
  return function () {
    var f = hs(this, t);
    return f === s ? null : f === o ? u : (u = e((o = f), r));
  };
}
function cht(t, e, r) {
  var o, s, u;
  return function () {
    var f = hs(this, t),
      h = r(this),
      d = h + "";
    return (
      h == null && (d = h = (this.style.removeProperty(t), hs(this, t))),
      f === d ? null : f === o && d === s ? u : ((s = d), (u = e((o = f), h)))
    );
  };
}
function uht(t, e) {
  var r,
    o,
    s,
    u = "style." + e,
    f = "end." + u,
    h;
  return function () {
    var d = kr(this, t),
      g = d.on,
      v = d.value[u] == null ? h || (h = Db(e)) : void 0;
    (g !== r || s !== v) && (o = (r = g).copy()).on(f, (s = v)), (d.on = o);
  };
}
function fht(t, e, r) {
  var o = (t += "") == "transform" ? fft : Ob;
  return e == null
    ? this.styleTween(t, lht(t, o)).on("end.style." + t, Db(t))
    : typeof e == "function"
    ? this.styleTween(t, cht(t, o, id(this, "style." + t, e))).each(uht(this._id, t))
    : this.styleTween(t, aht(t, o, e), r).on("end.style." + t, null);
}
function hht(t, e, r) {
  return function (o) {
    this.style.setProperty(t, e.call(this, o), r);
  };
}
function dht(t, e, r) {
  var o, s;
  function u() {
    var f = e.apply(this, arguments);
    return f !== s && (o = (s = f) && hht(t, f, r)), o;
  }
  return (u._value = e), u;
}
function pht(t, e, r) {
  var o = "style." + (t += "");
  if (arguments.length < 2) return (o = this.tween(o)) && o._value;
  if (e == null) return this.tween(o, null);
  if (typeof e != "function") throw new Error();
  return this.tween(o, dht(t, e, r ?? ""));
}
function ght(t) {
  return function () {
    this.textContent = t;
  };
}
function vht(t) {
  return function () {
    var e = t(this);
    this.textContent = e ?? "";
  };
}
function mht(t) {
  return this.tween(
    "text",
    typeof t == "function" ? vht(id(this, "text", t)) : ght(t == null ? "" : t + ""),
  );
}
function yht(t) {
  return function (e) {
    this.textContent = t.call(this, e);
  };
}
function bht(t) {
  var e, r;
  function o() {
    var s = t.apply(this, arguments);
    return s !== r && (e = (r = s) && yht(s)), e;
  }
  return (o._value = t), o;
}
function wht(t) {
  var e = "text";
  if (arguments.length < 1) return (e = this.tween(e)) && e._value;
  if (t == null) return this.tween(e, null);
  if (typeof t != "function") throw new Error();
  return this.tween(e, bht(t));
}
function xht() {
  for (
    var t = this._name, e = this._id, r = $b(), o = this._groups, s = o.length, u = 0;
    u < s;
    ++u
  )
    for (var f = o[u], h = f.length, d, g = 0; g < h; ++g)
      if ((d = f[g])) {
        var v = ar(d, e);
        Xc(d, t, r, g, f, {
          time: v.time + v.delay + v.duration,
          delay: 0,
          duration: v.duration,
          ease: v.ease,
        });
      }
  return new Gr(o, this._parents, t, r);
}
function _ht() {
  var t,
    e,
    r = this,
    o = r._id,
    s = r.size();
  return new Promise(function (u, f) {
    var h = { value: f },
      d = {
        value: function () {
          --s === 0 && u();
        },
      };
    r.each(function () {
      var g = kr(this, o),
        v = g.on;
      v !== t && ((e = (t = v).copy()), e._.cancel.push(h), e._.interrupt.push(h), e._.end.push(d)),
        (g.on = e);
    }),
      s === 0 && u();
  });
}
var Sht = 0;
function Gr(t, e, r, o) {
  (this._groups = t), (this._parents = e), (this._name = r), (this._id = o);
}
function $b() {
  return ++Sht;
}
var Dr = Il.prototype;
Gr.prototype = {
  constructor: Gr,
  select: rht,
  selectAll: iht,
  selectChild: Dr.selectChild,
  selectChildren: Dr.selectChildren,
  filter: Yft,
  merge: Zft,
  selection: sht,
  transition: xht,
  call: Dr.call,
  nodes: Dr.nodes,
  node: Dr.node,
  size: Dr.size,
  empty: Dr.empty,
  each: Dr.each,
  on: tht,
  attr: Dft,
  attrTween: Fft,
  style: fht,
  styleTween: pht,
  text: mht,
  textTween: wht,
  remove: nht,
  tween: Eft,
  delay: Bft,
  duration: jft,
  ease: Gft,
  easeVarying: Xft,
  end: _ht,
  [Symbol.iterator]: Dr[Symbol.iterator],
};
function kht(t) {
  return ((t *= 2) <= 1 ? t * t * t : (t -= 2) * t * t + 2) / 2;
}
var Cht = { time: null, delay: 0, duration: 250, ease: kht };
function Tht(t, e) {
  for (var r; !(r = t.__transition) || !(r = r[e]); )
    if (!(t = t.parentNode)) throw new Error(`transition ${e} not found`);
  return r;
}
function Eht(t) {
  var e, r;
  t instanceof Gr
    ? ((e = t._id), (t = t._name))
    : ((e = $b()), ((r = Cht).time = ed()), (t = t == null ? null : t + ""));
  for (var o = this._groups, s = o.length, u = 0; u < s; ++u)
    for (var f = o[u], h = f.length, d, g = 0; g < h; ++g)
      (d = f[g]) && Xc(d, t, e, g, f, r || Tht(d, e));
  return new Gr(o, this._parents, t, e);
}
Il.prototype.interrupt = kft;
Il.prototype.transition = Eht;
const Ra = (t) => () => t;
function Lht(t, { sourceEvent: e, target: r, transform: o, dispatch: s }) {
  Object.defineProperties(this, {
    type: { value: t, enumerable: !0, configurable: !0 },
    sourceEvent: { value: e, enumerable: !0, configurable: !0 },
    target: { value: r, enumerable: !0, configurable: !0 },
    transform: { value: o, enumerable: !0, configurable: !0 },
    _: { value: s },
  });
}
function Ir(t, e, r) {
  (this.k = t), (this.x = e), (this.y = r);
}
Ir.prototype = {
  constructor: Ir,
  scale: function (t) {
    return t === 1 ? this : new Ir(this.k * t, this.x, this.y);
  },
  translate: function (t, e) {
    return (t === 0) & (e === 0) ? this : new Ir(this.k, this.x + this.k * t, this.y + this.k * e);
  },
  apply: function (t) {
    return [t[0] * this.k + this.x, t[1] * this.k + this.y];
  },
  applyX: function (t) {
    return t * this.k + this.x;
  },
  applyY: function (t) {
    return t * this.k + this.y;
  },
  invert: function (t) {
    return [(t[0] - this.x) / this.k, (t[1] - this.y) / this.k];
  },
  invertX: function (t) {
    return (t - this.x) / this.k;
  },
  invertY: function (t) {
    return (t - this.y) / this.k;
  },
  rescaleX: function (t) {
    return t.copy().domain(t.range().map(this.invertX, this).map(t.invert, t));
  },
  rescaleY: function (t) {
    return t.copy().domain(t.range().map(this.invertY, this).map(t.invert, t));
  },
  toString: function () {
    return "translate(" + this.x + "," + this.y + ") scale(" + this.k + ")";
  },
};
var od = new Ir(1, 0, 0);
Ir.prototype;
function pf(t) {
  t.stopImmediatePropagation();
}
function nl(t) {
  t.preventDefault(), t.stopImmediatePropagation();
}
function Aht(t) {
  return (!t.ctrlKey || t.type === "wheel") && !t.button;
}
function Mht() {
  var t = this;
  return t instanceof SVGElement
    ? ((t = t.ownerSVGElement || t),
      t.hasAttribute("viewBox")
        ? ((t = t.viewBox.baseVal),
          [
            [t.x, t.y],
            [t.x + t.width, t.y + t.height],
          ])
        : [
            [0, 0],
            [t.width.baseVal.value, t.height.baseVal.value],
          ])
    : [
        [0, 0],
        [t.clientWidth, t.clientHeight],
      ];
}
function mm() {
  return this.__zoom || od;
}
function Nht(t) {
  return -t.deltaY * (t.deltaMode === 1 ? 0.05 : t.deltaMode ? 1 : 0.002) * (t.ctrlKey ? 10 : 1);
}
function Pht() {
  return navigator.maxTouchPoints || "ontouchstart" in this;
}
function Oht(t, e, r) {
  var o = t.invertX(e[0][0]) - r[0][0],
    s = t.invertX(e[1][0]) - r[1][0],
    u = t.invertY(e[0][1]) - r[0][1],
    f = t.invertY(e[1][1]) - r[1][1];
  return t.translate(
    s > o ? (o + s) / 2 : Math.min(0, o) || Math.max(0, s),
    f > u ? (u + f) / 2 : Math.min(0, u) || Math.max(0, f),
  );
}
function Dht() {
  var t = Aht,
    e = Mht,
    r = Oht,
    o = Nht,
    s = Pht,
    u = [0, 1 / 0],
    f = [
      [-1 / 0, -1 / 0],
      [1 / 0, 1 / 0],
    ],
    h = 250,
    d = vft,
    g = Fl("start", "zoom", "end"),
    v,
    b,
    w,
    S = 500,
    P = 150,
    A = 0,
    L = 10;
  function T(z) {
    z.property("__zoom", mm)
      .on("wheel.zoom", Y, { passive: !1 })
      .on("mousedown.zoom", nt)
      .on("dblclick.zoom", at)
      .filter(s)
      .on("touchstart.zoom", pt)
      .on("touchmove.zoom", gt)
      .on("touchend.zoom touchcancel.zoom", G)
      .style("-webkit-tap-highlight-color", "rgba(0,0,0,0)");
  }
  (T.transform = function (z, k, F, H) {
    var J = z.selection ? z.selection() : z;
    J.property("__zoom", mm),
      z !== J
        ? B(z, k, F, H)
        : J.interrupt().each(function () {
            K(this, arguments)
              .event(H)
              .start()
              .zoom(null, typeof k == "function" ? k.apply(this, arguments) : k)
              .end();
          });
  }),
    (T.scaleBy = function (z, k, F, H) {
      T.scaleTo(
        z,
        function () {
          var J = this.__zoom.k,
            yt = typeof k == "function" ? k.apply(this, arguments) : k;
          return J * yt;
        },
        F,
        H,
      );
    }),
    (T.scaleTo = function (z, k, F, H) {
      T.transform(
        z,
        function () {
          var J = e.apply(this, arguments),
            yt = this.__zoom,
            At = F == null ? E(J) : typeof F == "function" ? F.apply(this, arguments) : F,
            qt = yt.invert(At),
            Ht = typeof k == "function" ? k.apply(this, arguments) : k;
          return r(R(M(yt, Ht), At, qt), J, f);
        },
        F,
        H,
      );
    }),
    (T.translateBy = function (z, k, F, H) {
      T.transform(
        z,
        function () {
          return r(
            this.__zoom.translate(
              typeof k == "function" ? k.apply(this, arguments) : k,
              typeof F == "function" ? F.apply(this, arguments) : F,
            ),
            e.apply(this, arguments),
            f,
          );
        },
        null,
        H,
      );
    }),
    (T.translateTo = function (z, k, F, H, J) {
      T.transform(
        z,
        function () {
          var yt = e.apply(this, arguments),
            At = this.__zoom,
            qt = H == null ? E(yt) : typeof H == "function" ? H.apply(this, arguments) : H;
          return r(
            od
              .translate(qt[0], qt[1])
              .scale(At.k)
              .translate(
                typeof k == "function" ? -k.apply(this, arguments) : -k,
                typeof F == "function" ? -F.apply(this, arguments) : -F,
              ),
            yt,
            f,
          );
        },
        H,
        J,
      );
    });
  function M(z, k) {
    return (k = Math.max(u[0], Math.min(u[1], k))), k === z.k ? z : new Ir(k, z.x, z.y);
  }
  function R(z, k, F) {
    var H = k[0] - F[0] * z.k,
      J = k[1] - F[1] * z.k;
    return H === z.x && J === z.y ? z : new Ir(z.k, H, J);
  }
  function E(z) {
    return [(+z[0][0] + +z[1][0]) / 2, (+z[0][1] + +z[1][1]) / 2];
  }
  function B(z, k, F, H) {
    z.on("start.zoom", function () {
      K(this, arguments).event(H).start();
    })
      .on("interrupt.zoom end.zoom", function () {
        K(this, arguments).event(H).end();
      })
      .tween("zoom", function () {
        var J = this,
          yt = arguments,
          At = K(J, yt).event(H),
          qt = e.apply(J, yt),
          Ht = F == null ? E(qt) : typeof F == "function" ? F.apply(J, yt) : F,
          Qt = Math.max(qt[1][0] - qt[0][0], qt[1][1] - qt[0][1]),
          Jt = J.__zoom,
          Gt = typeof k == "function" ? k.apply(J, yt) : k,
          Tt = d(Jt.invert(Ht).concat(Qt / Jt.k), Gt.invert(Ht).concat(Qt / Gt.k));
        return function (j) {
          if (j === 1) j = Gt;
          else {
            var rt = Tt(j),
              lt = Qt / rt[2];
            j = new Ir(lt, Ht[0] - rt[0] * lt, Ht[1] - rt[1] * lt);
          }
          At.zoom(null, j);
        };
      });
  }
  function K(z, k, F) {
    return (!F && z.__zooming) || new ht(z, k);
  }
  function ht(z, k) {
    (this.that = z),
      (this.args = k),
      (this.active = 0),
      (this.sourceEvent = null),
      (this.extent = e.apply(z, k)),
      (this.taps = 0);
  }
  ht.prototype = {
    event: function (z) {
      return z && (this.sourceEvent = z), this;
    },
    start: function () {
      return ++this.active === 1 && ((this.that.__zooming = this), this.emit("start")), this;
    },
    zoom: function (z, k) {
      return (
        this.mouse && z !== "mouse" && (this.mouse[1] = k.invert(this.mouse[0])),
        this.touch0 && z !== "touch" && (this.touch0[1] = k.invert(this.touch0[0])),
        this.touch1 && z !== "touch" && (this.touch1[1] = k.invert(this.touch1[0])),
        (this.that.__zoom = k),
        this.emit("zoom"),
        this
      );
    },
    end: function () {
      return --this.active === 0 && (delete this.that.__zooming, this.emit("end")), this;
    },
    emit: function (z) {
      var k = En(this.that).datum();
      g.call(
        z,
        this.that,
        new Lht(z, {
          sourceEvent: this.sourceEvent,
          target: T,
          type: z,
          transform: this.that.__zoom,
          dispatch: g,
        }),
        k,
      );
    },
  };
  function Y(z, ...k) {
    if (!t.apply(this, arguments)) return;
    var F = K(this, k).event(z),
      H = this.__zoom,
      J = Math.max(u[0], Math.min(u[1], H.k * Math.pow(2, o.apply(this, arguments)))),
      yt = Rr(z);
    if (F.wheel)
      (F.mouse[0][0] !== yt[0] || F.mouse[0][1] !== yt[1]) &&
        (F.mouse[1] = H.invert((F.mouse[0] = yt))),
        clearTimeout(F.wheel);
    else {
      if (H.k === J) return;
      (F.mouse = [yt, H.invert(yt)]), Ya(this), F.start();
    }
    nl(z),
      (F.wheel = setTimeout(At, P)),
      F.zoom("mouse", r(R(M(H, J), F.mouse[0], F.mouse[1]), F.extent, f));
    function At() {
      (F.wheel = null), F.end();
    }
  }
  function nt(z, ...k) {
    if (w || !t.apply(this, arguments)) return;
    var F = z.currentTarget,
      H = K(this, k, !0).event(z),
      J = En(z.view).on("mousemove.zoom", Ht, !0).on("mouseup.zoom", Qt, !0),
      yt = Rr(z, F),
      At = z.clientX,
      qt = z.clientY;
    _b(z.view), pf(z), (H.mouse = [yt, this.__zoom.invert(yt)]), Ya(this), H.start();
    function Ht(Jt) {
      if ((nl(Jt), !H.moved)) {
        var Gt = Jt.clientX - At,
          Tt = Jt.clientY - qt;
        H.moved = Gt * Gt + Tt * Tt > A;
      }
      H.event(Jt).zoom(
        "mouse",
        r(R(H.that.__zoom, (H.mouse[0] = Rr(Jt, F)), H.mouse[1]), H.extent, f),
      );
    }
    function Qt(Jt) {
      J.on("mousemove.zoom mouseup.zoom", null), Sb(Jt.view, H.moved), nl(Jt), H.event(Jt).end();
    }
  }
  function at(z, ...k) {
    if (t.apply(this, arguments)) {
      var F = this.__zoom,
        H = Rr(z.changedTouches ? z.changedTouches[0] : z, this),
        J = F.invert(H),
        yt = F.k * (z.shiftKey ? 0.5 : 2),
        At = r(R(M(F, yt), H, J), e.apply(this, k), f);
      nl(z),
        h > 0
          ? En(this).transition().duration(h).call(B, At, H, z)
          : En(this).call(T.transform, At, H, z);
    }
  }
  function pt(z, ...k) {
    if (t.apply(this, arguments)) {
      var F = z.touches,
        H = F.length,
        J = K(this, k, z.changedTouches.length === H).event(z),
        yt,
        At,
        qt,
        Ht;
      for (pf(z), At = 0; At < H; ++At)
        (qt = F[At]),
          (Ht = Rr(qt, this)),
          (Ht = [Ht, this.__zoom.invert(Ht), qt.identifier]),
          J.touch0
            ? !J.touch1 && J.touch0[2] !== Ht[2] && ((J.touch1 = Ht), (J.taps = 0))
            : ((J.touch0 = Ht), (yt = !0), (J.taps = 1 + !!v));
      v && (v = clearTimeout(v)),
        yt &&
          (J.taps < 2 &&
            ((b = Ht[0]),
            (v = setTimeout(function () {
              v = null;
            }, S))),
          Ya(this),
          J.start());
    }
  }
  function gt(z, ...k) {
    if (this.__zooming) {
      var F = K(this, k).event(z),
        H = z.changedTouches,
        J = H.length,
        yt,
        At,
        qt,
        Ht;
      for (nl(z), yt = 0; yt < J; ++yt)
        (At = H[yt]),
          (qt = Rr(At, this)),
          F.touch0 && F.touch0[2] === At.identifier
            ? (F.touch0[0] = qt)
            : F.touch1 && F.touch1[2] === At.identifier && (F.touch1[0] = qt);
      if (((At = F.that.__zoom), F.touch1)) {
        var Qt = F.touch0[0],
          Jt = F.touch0[1],
          Gt = F.touch1[0],
          Tt = F.touch1[1],
          j = (j = Gt[0] - Qt[0]) * j + (j = Gt[1] - Qt[1]) * j,
          rt = (rt = Tt[0] - Jt[0]) * rt + (rt = Tt[1] - Jt[1]) * rt;
        (At = M(At, Math.sqrt(j / rt))),
          (qt = [(Qt[0] + Gt[0]) / 2, (Qt[1] + Gt[1]) / 2]),
          (Ht = [(Jt[0] + Tt[0]) / 2, (Jt[1] + Tt[1]) / 2]);
      } else if (F.touch0) (qt = F.touch0[0]), (Ht = F.touch0[1]);
      else return;
      F.zoom("touch", r(R(At, qt, Ht), F.extent, f));
    }
  }
  function G(z, ...k) {
    if (this.__zooming) {
      var F = K(this, k).event(z),
        H = z.changedTouches,
        J = H.length,
        yt,
        At;
      for (
        pf(z),
          w && clearTimeout(w),
          w = setTimeout(function () {
            w = null;
          }, S),
          yt = 0;
        yt < J;
        ++yt
      )
        (At = H[yt]),
          F.touch0 && F.touch0[2] === At.identifier
            ? delete F.touch0
            : F.touch1 && F.touch1[2] === At.identifier && delete F.touch1;
      if ((F.touch1 && !F.touch0 && ((F.touch0 = F.touch1), delete F.touch1), F.touch0))
        F.touch0[1] = this.__zoom.invert(F.touch0[0]);
      else if (
        (F.end(), F.taps === 2 && ((At = Rr(At, this)), Math.hypot(b[0] - At[0], b[1] - At[1]) < L))
      ) {
        var qt = En(this).on("dblclick.zoom");
        qt && qt.apply(this, arguments);
      }
    }
  }
  return (
    (T.wheelDelta = function (z) {
      return arguments.length ? ((o = typeof z == "function" ? z : Ra(+z)), T) : o;
    }),
    (T.filter = function (z) {
      return arguments.length ? ((t = typeof z == "function" ? z : Ra(!!z)), T) : t;
    }),
    (T.touchable = function (z) {
      return arguments.length ? ((s = typeof z == "function" ? z : Ra(!!z)), T) : s;
    }),
    (T.extent = function (z) {
      return arguments.length
        ? ((e =
            typeof z == "function"
              ? z
              : Ra([
                  [+z[0][0], +z[0][1]],
                  [+z[1][0], +z[1][1]],
                ])),
          T)
        : e;
    }),
    (T.scaleExtent = function (z) {
      return arguments.length ? ((u[0] = +z[0]), (u[1] = +z[1]), T) : [u[0], u[1]];
    }),
    (T.translateExtent = function (z) {
      return arguments.length
        ? ((f[0][0] = +z[0][0]),
          (f[1][0] = +z[1][0]),
          (f[0][1] = +z[0][1]),
          (f[1][1] = +z[1][1]),
          T)
        : [
            [f[0][0], f[0][1]],
            [f[1][0], f[1][1]],
          ];
    }),
    (T.constrain = function (z) {
      return arguments.length ? ((r = z), T) : r;
    }),
    (T.duration = function (z) {
      return arguments.length ? ((h = +z), T) : h;
    }),
    (T.interpolate = function (z) {
      return arguments.length ? ((d = z), T) : d;
    }),
    (T.on = function () {
      var z = g.on.apply(g, arguments);
      return z === g ? T : z;
    }),
    (T.clickDistance = function (z) {
      return arguments.length ? ((A = (z = +z) * z), T) : Math.sqrt(A);
    }),
    (T.tapDistance = function (z) {
      return arguments.length ? ((L = +z), T) : L;
    }),
    T
  );
}
function $ht(t) {
  const e = +this._x.call(null, t),
    r = +this._y.call(null, t);
  return Rb(this.cover(e, r), e, r, t);
}
function Rb(t, e, r, o) {
  if (isNaN(e) || isNaN(r)) return t;
  var s,
    u = t._root,
    f = { data: o },
    h = t._x0,
    d = t._y0,
    g = t._x1,
    v = t._y1,
    b,
    w,
    S,
    P,
    A,
    L,
    T,
    M;
  if (!u) return (t._root = f), t;
  for (; u.length; )
    if (
      ((A = e >= (b = (h + g) / 2)) ? (h = b) : (g = b),
      (L = r >= (w = (d + v) / 2)) ? (d = w) : (v = w),
      (s = u),
      !(u = u[(T = (L << 1) | A)]))
    )
      return (s[T] = f), t;
  if (((S = +t._x.call(null, u.data)), (P = +t._y.call(null, u.data)), e === S && r === P))
    return (f.next = u), s ? (s[T] = f) : (t._root = f), t;
  do
    (s = s ? (s[T] = new Array(4)) : (t._root = new Array(4))),
      (A = e >= (b = (h + g) / 2)) ? (h = b) : (g = b),
      (L = r >= (w = (d + v) / 2)) ? (d = w) : (v = w);
  while ((T = (L << 1) | A) === (M = ((P >= w) << 1) | (S >= b)));
  return (s[M] = u), (s[T] = f), t;
}
function Rht(t) {
  var e,
    r,
    o = t.length,
    s,
    u,
    f = new Array(o),
    h = new Array(o),
    d = 1 / 0,
    g = 1 / 0,
    v = -1 / 0,
    b = -1 / 0;
  for (r = 0; r < o; ++r)
    isNaN((s = +this._x.call(null, (e = t[r])))) ||
      isNaN((u = +this._y.call(null, e))) ||
      ((f[r] = s),
      (h[r] = u),
      s < d && (d = s),
      s > v && (v = s),
      u < g && (g = u),
      u > b && (b = u));
  if (d > v || g > b) return this;
  for (this.cover(d, g).cover(v, b), r = 0; r < o; ++r) Rb(this, f[r], h[r], t[r]);
  return this;
}
function zht(t, e) {
  if (isNaN((t = +t)) || isNaN((e = +e))) return this;
  var r = this._x0,
    o = this._y0,
    s = this._x1,
    u = this._y1;
  if (isNaN(r)) (s = (r = Math.floor(t)) + 1), (u = (o = Math.floor(e)) + 1);
  else {
    for (var f = s - r || 1, h = this._root, d, g; r > t || t >= s || o > e || e >= u; )
      switch (
        ((g = ((e < o) << 1) | (t < r)), (d = new Array(4)), (d[g] = h), (h = d), (f *= 2), g)
      ) {
        case 0:
          (s = r + f), (u = o + f);
          break;
        case 1:
          (r = s - f), (u = o + f);
          break;
        case 2:
          (s = r + f), (o = u - f);
          break;
        case 3:
          (r = s - f), (o = u - f);
          break;
      }
    this._root && this._root.length && (this._root = h);
  }
  return (this._x0 = r), (this._y0 = o), (this._x1 = s), (this._y1 = u), this;
}
function Iht() {
  var t = [];
  return (
    this.visit(function (e) {
      if (!e.length)
        do t.push(e.data);
        while ((e = e.next));
    }),
    t
  );
}
function Fht(t) {
  return arguments.length
    ? this.cover(+t[0][0], +t[0][1]).cover(+t[1][0], +t[1][1])
    : isNaN(this._x0)
    ? void 0
    : [
        [this._x0, this._y0],
        [this._x1, this._y1],
      ];
}
function gn(t, e, r, o, s) {
  (this.node = t), (this.x0 = e), (this.y0 = r), (this.x1 = o), (this.y1 = s);
}
function qht(t, e, r) {
  var o,
    s = this._x0,
    u = this._y0,
    f,
    h,
    d,
    g,
    v = this._x1,
    b = this._y1,
    w = [],
    S = this._root,
    P,
    A;
  for (
    S && w.push(new gn(S, s, u, v, b)),
      r == null ? (r = 1 / 0) : ((s = t - r), (u = e - r), (v = t + r), (b = e + r), (r *= r));
    (P = w.pop());
  )
    if (!(!(S = P.node) || (f = P.x0) > v || (h = P.y0) > b || (d = P.x1) < s || (g = P.y1) < u))
      if (S.length) {
        var L = (f + d) / 2,
          T = (h + g) / 2;
        w.push(
          new gn(S[3], L, T, d, g),
          new gn(S[2], f, T, L, g),
          new gn(S[1], L, h, d, T),
          new gn(S[0], f, h, L, T),
        ),
          (A = ((e >= T) << 1) | (t >= L)) &&
            ((P = w[w.length - 1]),
            (w[w.length - 1] = w[w.length - 1 - A]),
            (w[w.length - 1 - A] = P));
      } else {
        var M = t - +this._x.call(null, S.data),
          R = e - +this._y.call(null, S.data),
          E = M * M + R * R;
        if (E < r) {
          var B = Math.sqrt((r = E));
          (s = t - B), (u = e - B), (v = t + B), (b = e + B), (o = S.data);
        }
      }
  return o;
}
function Hht(t) {
  if (isNaN((v = +this._x.call(null, t))) || isNaN((b = +this._y.call(null, t)))) return this;
  var e,
    r = this._root,
    o,
    s,
    u,
    f = this._x0,
    h = this._y0,
    d = this._x1,
    g = this._y1,
    v,
    b,
    w,
    S,
    P,
    A,
    L,
    T;
  if (!r) return this;
  if (r.length)
    for (;;) {
      if (
        ((P = v >= (w = (f + d) / 2)) ? (f = w) : (d = w),
        (A = b >= (S = (h + g) / 2)) ? (h = S) : (g = S),
        (e = r),
        !(r = r[(L = (A << 1) | P)]))
      )
        return this;
      if (!r.length) break;
      (e[(L + 1) & 3] || e[(L + 2) & 3] || e[(L + 3) & 3]) && ((o = e), (T = L));
    }
  for (; r.data !== t; ) if (((s = r), !(r = r.next))) return this;
  return (
    (u = r.next) && delete r.next,
    s
      ? (u ? (s.next = u) : delete s.next, this)
      : e
      ? (u ? (e[L] = u) : delete e[L],
        (r = e[0] || e[1] || e[2] || e[3]) &&
          r === (e[3] || e[2] || e[1] || e[0]) &&
          !r.length &&
          (o ? (o[T] = r) : (this._root = r)),
        this)
      : ((this._root = u), this)
  );
}
function Bht(t) {
  for (var e = 0, r = t.length; e < r; ++e) this.remove(t[e]);
  return this;
}
function Wht() {
  return this._root;
}
function Uht() {
  var t = 0;
  return (
    this.visit(function (e) {
      if (!e.length)
        do ++t;
        while ((e = e.next));
    }),
    t
  );
}
function jht(t) {
  var e = [],
    r,
    o = this._root,
    s,
    u,
    f,
    h,
    d;
  for (o && e.push(new gn(o, this._x0, this._y0, this._x1, this._y1)); (r = e.pop()); )
    if (!t((o = r.node), (u = r.x0), (f = r.y0), (h = r.x1), (d = r.y1)) && o.length) {
      var g = (u + h) / 2,
        v = (f + d) / 2;
      (s = o[3]) && e.push(new gn(s, g, v, h, d)),
        (s = o[2]) && e.push(new gn(s, u, v, g, d)),
        (s = o[1]) && e.push(new gn(s, g, f, h, v)),
        (s = o[0]) && e.push(new gn(s, u, f, g, v));
    }
  return this;
}
function Vht(t) {
  var e = [],
    r = [],
    o;
  for (
    this._root && e.push(new gn(this._root, this._x0, this._y0, this._x1, this._y1));
    (o = e.pop());
  ) {
    var s = o.node;
    if (s.length) {
      var u,
        f = o.x0,
        h = o.y0,
        d = o.x1,
        g = o.y1,
        v = (f + d) / 2,
        b = (h + g) / 2;
      (u = s[0]) && e.push(new gn(u, f, h, v, b)),
        (u = s[1]) && e.push(new gn(u, v, h, d, b)),
        (u = s[2]) && e.push(new gn(u, f, b, v, g)),
        (u = s[3]) && e.push(new gn(u, v, b, d, g));
    }
    r.push(o);
  }
  for (; (o = r.pop()); ) t(o.node, o.x0, o.y0, o.x1, o.y1);
  return this;
}
function Ght(t) {
  return t[0];
}
function Kht(t) {
  return arguments.length ? ((this._x = t), this) : this._x;
}
function Xht(t) {
  return t[1];
}
function Yht(t) {
  return arguments.length ? ((this._y = t), this) : this._y;
}
function sd(t, e, r) {
  var o = new ld(e ?? Ght, r ?? Xht, NaN, NaN, NaN, NaN);
  return t == null ? o : o.addAll(t);
}
function ld(t, e, r, o, s, u) {
  (this._x = t),
    (this._y = e),
    (this._x0 = r),
    (this._y0 = o),
    (this._x1 = s),
    (this._y1 = u),
    (this._root = void 0);
}
function ym(t) {
  for (var e = { data: t.data }, r = e; (t = t.next); ) r = r.next = { data: t.data };
  return e;
}
var yn = (sd.prototype = ld.prototype);
yn.copy = function () {
  var t = new ld(this._x, this._y, this._x0, this._y0, this._x1, this._y1),
    e = this._root,
    r,
    o;
  if (!e) return t;
  if (!e.length) return (t._root = ym(e)), t;
  for (r = [{ source: e, target: (t._root = new Array(4)) }]; (e = r.pop()); )
    for (var s = 0; s < 4; ++s)
      (o = e.source[s]) &&
        (o.length
          ? r.push({ source: o, target: (e.target[s] = new Array(4)) })
          : (e.target[s] = ym(o)));
  return t;
};
yn.add = $ht;
yn.addAll = Rht;
yn.cover = zht;
yn.data = Iht;
yn.extent = Fht;
yn.find = qht;
yn.remove = Hht;
yn.removeAll = Bht;
yn.root = Wht;
yn.size = Uht;
yn.visit = jht;
yn.visitAfter = Vht;
yn.x = Kht;
yn.y = Yht;
function vn(t) {
  return function () {
    return t;
  };
}
function xi(t) {
  return (t() - 0.5) * 1e-6;
}
function Zht(t) {
  return t.x + t.vx;
}
function Jht(t) {
  return t.y + t.vy;
}
function Qht(t) {
  var e,
    r,
    o,
    s = 1,
    u = 1;
  typeof t != "function" && (t = vn(t == null ? 1 : +t));
  function f() {
    for (var g, v = e.length, b, w, S, P, A, L, T = 0; T < u; ++T)
      for (b = sd(e, Zht, Jht).visitAfter(h), g = 0; g < v; ++g)
        (w = e[g]), (A = r[w.index]), (L = A * A), (S = w.x + w.vx), (P = w.y + w.vy), b.visit(M);
    function M(R, E, B, K, ht) {
      var Y = R.data,
        nt = R.r,
        at = A + nt;
      if (Y) {
        if (Y.index > w.index) {
          var pt = S - Y.x - Y.vx,
            gt = P - Y.y - Y.vy,
            G = pt * pt + gt * gt;
          G < at * at &&
            (pt === 0 && ((pt = xi(o)), (G += pt * pt)),
            gt === 0 && ((gt = xi(o)), (G += gt * gt)),
            (G = ((at - (G = Math.sqrt(G))) / G) * s),
            (w.vx += (pt *= G) * (at = (nt *= nt) / (L + nt))),
            (w.vy += (gt *= G) * at),
            (Y.vx -= pt * (at = 1 - at)),
            (Y.vy -= gt * at));
        }
        return;
      }
      return E > S + at || K < S - at || B > P + at || ht < P - at;
    }
  }
  function h(g) {
    if (g.data) return (g.r = r[g.data.index]);
    for (var v = (g.r = 0); v < 4; ++v) g[v] && g[v].r > g.r && (g.r = g[v].r);
  }
  function d() {
    if (e) {
      var g,
        v = e.length,
        b;
      for (r = new Array(v), g = 0; g < v; ++g) (b = e[g]), (r[b.index] = +t(b, g, e));
    }
  }
  return (
    (f.initialize = function (g, v) {
      (e = g), (o = v), d();
    }),
    (f.iterations = function (g) {
      return arguments.length ? ((u = +g), f) : u;
    }),
    (f.strength = function (g) {
      return arguments.length ? ((s = +g), f) : s;
    }),
    (f.radius = function (g) {
      return arguments.length ? ((t = typeof g == "function" ? g : vn(+g)), d(), f) : t;
    }),
    f
  );
}
function tdt(t) {
  return t.index;
}
function bm(t, e) {
  var r = t.get(e);
  if (!r) throw new Error("node not found: " + e);
  return r;
}
function edt(t) {
  var e = tdt,
    r = b,
    o,
    s = vn(30),
    u,
    f,
    h,
    d,
    g,
    v = 1;
  t == null && (t = []);
  function b(L) {
    return 1 / Math.min(h[L.source.index], h[L.target.index]);
  }
  function w(L) {
    for (var T = 0, M = t.length; T < v; ++T)
      for (var R = 0, E, B, K, ht, Y, nt, at; R < M; ++R)
        (E = t[R]),
          (B = E.source),
          (K = E.target),
          (ht = K.x + K.vx - B.x - B.vx || xi(g)),
          (Y = K.y + K.vy - B.y - B.vy || xi(g)),
          (nt = Math.sqrt(ht * ht + Y * Y)),
          (nt = ((nt - u[R]) / nt) * L * o[R]),
          (ht *= nt),
          (Y *= nt),
          (K.vx -= ht * (at = d[R])),
          (K.vy -= Y * at),
          (B.vx += ht * (at = 1 - at)),
          (B.vy += Y * at);
  }
  function S() {
    if (f) {
      var L,
        T = f.length,
        M = t.length,
        R = new Map(f.map((B, K) => [e(B, K, f), B])),
        E;
      for (L = 0, h = new Array(T); L < M; ++L)
        (E = t[L]),
          (E.index = L),
          typeof E.source != "object" && (E.source = bm(R, E.source)),
          typeof E.target != "object" && (E.target = bm(R, E.target)),
          (h[E.source.index] = (h[E.source.index] || 0) + 1),
          (h[E.target.index] = (h[E.target.index] || 0) + 1);
      for (L = 0, d = new Array(M); L < M; ++L)
        (E = t[L]), (d[L] = h[E.source.index] / (h[E.source.index] + h[E.target.index]));
      (o = new Array(M)), P(), (u = new Array(M)), A();
    }
  }
  function P() {
    if (f) for (var L = 0, T = t.length; L < T; ++L) o[L] = +r(t[L], L, t);
  }
  function A() {
    if (f) for (var L = 0, T = t.length; L < T; ++L) u[L] = +s(t[L], L, t);
  }
  return (
    (w.initialize = function (L, T) {
      (f = L), (g = T), S();
    }),
    (w.links = function (L) {
      return arguments.length ? ((t = L), S(), w) : t;
    }),
    (w.id = function (L) {
      return arguments.length ? ((e = L), w) : e;
    }),
    (w.iterations = function (L) {
      return arguments.length ? ((v = +L), w) : v;
    }),
    (w.strength = function (L) {
      return arguments.length ? ((r = typeof L == "function" ? L : vn(+L)), P(), w) : r;
    }),
    (w.distance = function (L) {
      return arguments.length ? ((s = typeof L == "function" ? L : vn(+L)), A(), w) : s;
    }),
    w
  );
}
const ndt = 1664525,
  rdt = 1013904223,
  wm = 4294967296;
function idt() {
  let t = 1;
  return () => (t = (ndt * t + rdt) % wm) / wm;
}
function odt(t) {
  return t.x;
}
function sdt(t) {
  return t.y;
}
var ldt = 10,
  adt = Math.PI * (3 - Math.sqrt(5));
function cdt(t) {
  var e,
    r = 1,
    o = 0.001,
    s = 1 - Math.pow(o, 1 / 300),
    u = 0,
    f = 0.6,
    h = new Map(),
    d = nd(b),
    g = Fl("tick", "end"),
    v = idt();
  t == null && (t = []);
  function b() {
    w(), g.call("tick", e), r < o && (d.stop(), g.call("end", e));
  }
  function w(A) {
    var L,
      T = t.length,
      M;
    A === void 0 && (A = 1);
    for (var R = 0; R < A; ++R)
      for (
        r += (u - r) * s,
          h.forEach(function (E) {
            E(r);
          }),
          L = 0;
        L < T;
        ++L
      )
        (M = t[L]),
          M.fx == null ? (M.x += M.vx *= f) : ((M.x = M.fx), (M.vx = 0)),
          M.fy == null ? (M.y += M.vy *= f) : ((M.y = M.fy), (M.vy = 0));
    return e;
  }
  function S() {
    for (var A = 0, L = t.length, T; A < L; ++A) {
      if (
        ((T = t[A]),
        (T.index = A),
        T.fx != null && (T.x = T.fx),
        T.fy != null && (T.y = T.fy),
        isNaN(T.x) || isNaN(T.y))
      ) {
        var M = ldt * Math.sqrt(0.5 + A),
          R = A * adt;
        (T.x = M * Math.cos(R)), (T.y = M * Math.sin(R));
      }
      (isNaN(T.vx) || isNaN(T.vy)) && (T.vx = T.vy = 0);
    }
  }
  function P(A) {
    return A.initialize && A.initialize(t, v), A;
  }
  return (
    S(),
    (e = {
      tick: w,
      restart: function () {
        return d.restart(b), e;
      },
      stop: function () {
        return d.stop(), e;
      },
      nodes: function (A) {
        return arguments.length ? ((t = A), S(), h.forEach(P), e) : t;
      },
      alpha: function (A) {
        return arguments.length ? ((r = +A), e) : r;
      },
      alphaMin: function (A) {
        return arguments.length ? ((o = +A), e) : o;
      },
      alphaDecay: function (A) {
        return arguments.length ? ((s = +A), e) : +s;
      },
      alphaTarget: function (A) {
        return arguments.length ? ((u = +A), e) : u;
      },
      velocityDecay: function (A) {
        return arguments.length ? ((f = 1 - A), e) : 1 - f;
      },
      randomSource: function (A) {
        return arguments.length ? ((v = A), h.forEach(P), e) : v;
      },
      force: function (A, L) {
        return arguments.length > 1 ? (L == null ? h.delete(A) : h.set(A, P(L)), e) : h.get(A);
      },
      find: function (A, L, T) {
        var M = 0,
          R = t.length,
          E,
          B,
          K,
          ht,
          Y;
        for (T == null ? (T = 1 / 0) : (T *= T), M = 0; M < R; ++M)
          (ht = t[M]),
            (E = A - ht.x),
            (B = L - ht.y),
            (K = E * E + B * B),
            K < T && ((Y = ht), (T = K));
        return Y;
      },
      on: function (A, L) {
        return arguments.length > 1 ? (g.on(A, L), e) : g.on(A);
      },
    })
  );
}
function udt() {
  var t,
    e,
    r,
    o,
    s = vn(-30),
    u,
    f = 1,
    h = 1 / 0,
    d = 0.81;
  function g(S) {
    var P,
      A = t.length,
      L = sd(t, odt, sdt).visitAfter(b);
    for (o = S, P = 0; P < A; ++P) (e = t[P]), L.visit(w);
  }
  function v() {
    if (t) {
      var S,
        P = t.length,
        A;
      for (u = new Array(P), S = 0; S < P; ++S) (A = t[S]), (u[A.index] = +s(A, S, t));
    }
  }
  function b(S) {
    var P = 0,
      A,
      L,
      T = 0,
      M,
      R,
      E;
    if (S.length) {
      for (M = R = E = 0; E < 4; ++E)
        (A = S[E]) &&
          (L = Math.abs(A.value)) &&
          ((P += A.value), (T += L), (M += L * A.x), (R += L * A.y));
      (S.x = M / T), (S.y = R / T);
    } else {
      (A = S), (A.x = A.data.x), (A.y = A.data.y);
      do P += u[A.data.index];
      while ((A = A.next));
    }
    S.value = P;
  }
  function w(S, P, A, L) {
    if (!S.value) return !0;
    var T = S.x - e.x,
      M = S.y - e.y,
      R = L - P,
      E = T * T + M * M;
    if ((R * R) / d < E)
      return (
        E < h &&
          (T === 0 && ((T = xi(r)), (E += T * T)),
          M === 0 && ((M = xi(r)), (E += M * M)),
          E < f && (E = Math.sqrt(f * E)),
          (e.vx += (T * S.value * o) / E),
          (e.vy += (M * S.value * o) / E)),
        !0
      );
    if (S.length || E >= h) return;
    (S.data !== e || S.next) &&
      (T === 0 && ((T = xi(r)), (E += T * T)),
      M === 0 && ((M = xi(r)), (E += M * M)),
      E < f && (E = Math.sqrt(f * E)));
    do S.data !== e && ((R = (u[S.data.index] * o) / E), (e.vx += T * R), (e.vy += M * R));
    while ((S = S.next));
  }
  return (
    (g.initialize = function (S, P) {
      (t = S), (r = P), v();
    }),
    (g.strength = function (S) {
      return arguments.length ? ((s = typeof S == "function" ? S : vn(+S)), v(), g) : s;
    }),
    (g.distanceMin = function (S) {
      return arguments.length ? ((f = S * S), g) : Math.sqrt(f);
    }),
    (g.distanceMax = function (S) {
      return arguments.length ? ((h = S * S), g) : Math.sqrt(h);
    }),
    (g.theta = function (S) {
      return arguments.length ? ((d = S * S), g) : Math.sqrt(d);
    }),
    g
  );
}
function fdt(t) {
  var e = vn(0.1),
    r,
    o,
    s;
  typeof t != "function" && (t = vn(t == null ? 0 : +t));
  function u(h) {
    for (var d = 0, g = r.length, v; d < g; ++d) (v = r[d]), (v.vx += (s[d] - v.x) * o[d] * h);
  }
  function f() {
    if (r) {
      var h,
        d = r.length;
      for (o = new Array(d), s = new Array(d), h = 0; h < d; ++h)
        o[h] = isNaN((s[h] = +t(r[h], h, r))) ? 0 : +e(r[h], h, r);
    }
  }
  return (
    (u.initialize = function (h) {
      (r = h), f();
    }),
    (u.strength = function (h) {
      return arguments.length ? ((e = typeof h == "function" ? h : vn(+h)), f(), u) : e;
    }),
    (u.x = function (h) {
      return arguments.length ? ((t = typeof h == "function" ? h : vn(+h)), f(), u) : t;
    }),
    u
  );
}
function hdt(t) {
  var e = vn(0.1),
    r,
    o,
    s;
  typeof t != "function" && (t = vn(t == null ? 0 : +t));
  function u(h) {
    for (var d = 0, g = r.length, v; d < g; ++d) (v = r[d]), (v.vy += (s[d] - v.y) * o[d] * h);
  }
  function f() {
    if (r) {
      var h,
        d = r.length;
      for (o = new Array(d), s = new Array(d), h = 0; h < d; ++h)
        o[h] = isNaN((s[h] = +t(r[h], h, r))) ? 0 : +e(r[h], h, r);
    }
  }
  return (
    (u.initialize = function (h) {
      (r = h), f();
    }),
    (u.strength = function (h) {
      return arguments.length ? ((e = typeof h == "function" ? h : vn(+h)), f(), u) : e;
    }),
    (u.y = function (h) {
      return arguments.length ? ((t = typeof h == "function" ? h : vn(+h)), f(), u) : t;
    }),
    u
  );
}
var ddt = Object.defineProperty,
  pdt = (t, e, r) =>
    e in t ? ddt(t, e, { enumerable: !0, configurable: !0, writable: !0, value: r }) : (t[e] = r),
  Oe = (t, e, r) => (pdt(t, typeof e != "symbol" ? e + "" : e, r), r);
function gdt() {
  return {
    drag: { end: 0, start: 0.1 },
    filter: { link: 1, type: 0.1, unlinked: { include: 0.1, exclude: 0.1 } },
    focus: { acquire: () => 0.1, release: () => 0.1 },
    initialize: 1,
    labels: { links: { hide: 0, show: 0 }, nodes: { hide: 0, show: 0 } },
    resize: 0.5,
  };
}
function xm(t) {
  if (typeof t == "object" && t !== null) {
    if (typeof Object.getPrototypeOf == "function") {
      const e = Object.getPrototypeOf(t);
      return e === Object.prototype || e === null;
    }
    return Object.prototype.toString.call(t) === "[object Object]";
  }
  return !1;
}
function _i(...t) {
  return t.reduce((e, r) => {
    if (Array.isArray(r))
      throw new TypeError("Arguments provided to deepmerge must be objects, not arrays.");
    return (
      Object.keys(r).forEach((o) => {
        ["__proto__", "constructor", "prototype"].includes(o) ||
          (Array.isArray(e[o]) && Array.isArray(r[o])
            ? (e[o] = _i.options.mergeArrays ? Array.from(new Set(e[o].concat(r[o]))) : r[o])
            : xm(e[o]) && xm(r[o])
            ? (e[o] = _i(e[o], r[o]))
            : (e[o] = r[o]));
      }),
      e
    );
  }, {});
}
const zb = { mergeArrays: !0 };
_i.options = zb;
_i.withOptions = (t, ...e) => {
  _i.options = { mergeArrays: !0, ...t };
  const r = _i(...e);
  return (_i.options = zb), r;
};
function vdt() {
  return {
    centering: { enabled: !0, strength: 0.1 },
    charge: { enabled: !0, strength: -1 },
    collision: { enabled: !0, strength: 1, radiusMultiplier: 2 },
    link: { enabled: !0, strength: 1, length: 128 },
  };
}
function mdt() {
  return {
    includeUnlinked: !0,
    linkFilter: () => !0,
    nodeTypeFilter: void 0,
    showLinkLabels: !0,
    showNodeLabels: !0,
  };
}
function Ib(t) {
  t.preventDefault(), t.stopPropagation();
}
function Fb(t) {
  return typeof t == "number";
}
function Ai(t, e) {
  return Fb(t.nodeRadius) ? t.nodeRadius : t.nodeRadius(e);
}
function ydt(t) {
  return `${t.source.id}-${t.target.id}`;
}
function qb(t) {
  return `link-arrow-${t}`.replace(/[()]/g, "~");
}
function bdt(t) {
  return `url(#${qb(t.color)})`;
}
function wdt(t) {
  return {
    size: t,
    padding: (e, r) => Ai(r, e) + 2 * t,
    ref: [t / 2, t / 2],
    path: [
      [0, 0],
      [0, t],
      [t, t / 2],
    ],
    viewBox: [0, 0, t, t].join(","),
  };
}
const Hb = { Arrow: (t) => wdt(t) },
  xdt = (t, e, r) => [e / 2, r / 2],
  Bb = (t, e, r) => [_m(0, e), _m(0, r)];
function _m(t, e) {
  return Math.random() * (e - t) + t;
}
function _dt(t) {
  const e = Object.fromEntries(t.nodes.map((r) => [r.id, [r.x, r.y]]));
  return (r, o, s) => {
    const [u, f] = e[r.id] ?? [];
    return !u || !f ? Bb(r, o, s) : [u, f];
  };
}
const ih = { Centered: xdt, Randomized: Bb, Stable: _dt };
function Sdt() {
  return {
    autoResize: !1,
    callbacks: {},
    hooks: {},
    initial: mdt(),
    nodeRadius: 16,
    marker: Hb.Arrow(4),
    modifiers: {},
    positionInitializer: ih.Centered,
    simulation: { alphas: gdt(), forces: vdt() },
    zoom: { initial: 1, min: 0.1, max: 2 },
  };
}
function kdt(t = {}) {
  return _i.withOptions({ mergeArrays: !1 }, Sdt(), t);
}
function Cdt({
  applyZoom: t,
  container: e,
  onDoubleClick: r,
  onPointerMoved: o,
  onPointerUp: s,
  offset: [u, f],
  scale: h,
  zoom: d,
}) {
  const g = e
    .classed("graph", !0)
    .append("svg")
    .attr("height", "100%")
    .attr("width", "100%")
    .call(d)
    .on("contextmenu", (v) => Ib(v))
    .on("dblclick", (v) => (r == null ? void 0 : r(v)))
    .on("dblclick.zoom", null)
    .on("pointermove", (v) => (o == null ? void 0 : o(v)))
    .on("pointerup", (v) => (s == null ? void 0 : s(v)))
    .style("cursor", "grab");
  return t && g.call(d.transform, od.translate(u, f).scale(h)), g.append("g");
}
function Tdt({ canvas: t, scale: e, xOffset: r, yOffset: o }) {
  t == null || t.attr("transform", `translate(${r},${o})scale(${e})`);
}
function Edt({ config: t, onDragStart: e, onDragEnd: r }) {
  var o, s;
  const u = Uut()
    .filter((f) =>
      f.type === "mousedown"
        ? f.button === 0
        : f.type === "touchstart"
        ? f.touches.length === 1
        : !1,
    )
    .on("start", (f, h) => {
      f.active === 0 && e(f, h),
        En(f.sourceEvent.target).classed("grabbed", !0),
        (h.fx = h.x),
        (h.fy = h.y);
    })
    .on("drag", (f, h) => {
      (h.fx = f.x), (h.fy = f.y);
    })
    .on("end", (f, h) => {
      f.active === 0 && r(f, h),
        En(f.sourceEvent.target).classed("grabbed", !1),
        (h.fx = void 0),
        (h.fy = void 0);
    });
  return (s = (o = t.modifiers).drag) == null || s.call(o, u), u;
}
function Ldt({ graph: t, filter: e, focusedNode: r, includeUnlinked: o, linkFilter: s }) {
  const u = t.links.filter((d) => e.includes(d.source.type) && e.includes(d.target.type) && s(d)),
    f = (d) => u.find((g) => g.source.id === d.id || g.target.id === d.id) !== void 0,
    h = t.nodes.filter((d) => e.includes(d.type) && (o || f(d)));
  return r === void 0 || !e.includes(r.type)
    ? { nodes: h, links: u }
    : Adt({ nodes: h, links: u }, r);
}
function Adt(t, e) {
  const r = [...Mdt(t, e), ...Ndt(t, e)],
    o = r.flatMap((s) => [s.source, s.target]);
  return { nodes: [...new Set([...o, e])], links: [...new Set(r)] };
}
function Mdt(t, e) {
  return Wb(t, e, (r, o) => r.target.id === o.id);
}
function Ndt(t, e) {
  return Wb(t, e, (r, o) => r.source.id === o.id);
}
function Wb(t, e, r) {
  const o = new Set(t.links),
    s = new Set([e]),
    u = [];
  for (; o.size > 0; ) {
    const f = [...o].filter((h) => [...s].some((d) => r(h, d)));
    if (f.length === 0) return u;
    f.forEach((h) => {
      s.add(h.source), s.add(h.target), u.push(h), o.delete(h);
    });
  }
  return u;
}
function oh(t) {
  return t.x ?? 0;
}
function sh(t) {
  return t.y ?? 0;
}
function ad({ source: t, target: e }) {
  const r = new pn(oh(t), sh(t)),
    o = new pn(oh(e), sh(e)),
    s = o.subtract(r),
    u = s.length(),
    f = s.normalize(),
    h = f.multiply(-1);
  return { s: r, t: o, dist: u, norm: f, endNorm: h };
}
function Ub({ center: t, node: e }) {
  const r = new pn(oh(e), sh(e));
  let o = t;
  return r.x === o.x && r.y === o.y && (o = o.add(new pn(0, 1))), { n: r, c: o };
}
function jb({ config: t, source: e, target: r }) {
  const { s: o, t: s, norm: u } = ad({ config: t, source: e, target: r }),
    f = o.add(u.multiply(Ai(t, e) - 1)),
    h = s.subtract(u.multiply(t.marker.padding(r, t)));
  return { start: f, end: h };
}
function Pdt(t) {
  const { start: e, end: r } = jb(t);
  return `M${e.x},${e.y}
          L${r.x},${r.y}`;
}
function Odt(t) {
  const { start: e, end: r } = jb(t),
    o = r.subtract(e).multiply(0.5),
    s = e.add(o);
  return `translate(${s.x - 8},${s.y - 4})`;
}
function Ddt({ config: t, source: e, target: r }) {
  const { s: o, t: s, dist: u, norm: f, endNorm: h } = ad({ config: t, source: e, target: r }),
    d = 10,
    g = f
      .rotateByDegrees(-d)
      .multiply(Ai(t, e) - 1)
      .add(o),
    v = h
      .rotateByDegrees(d)
      .multiply(Ai(t, r))
      .add(s)
      .add(h.rotateByDegrees(d).multiply(2 * t.marker.size)),
    b = 1.2 * u;
  return `M${g.x},${g.y}
          A${b},${b},0,0,1,${v.x},${v.y}`;
}
function $dt({ center: t, config: e, node: r }) {
  const { n: o, c: s } = Ub({ center: t, config: e, node: r }),
    u = Ai(e, r),
    f = o.subtract(s),
    h = f.multiply(1 / f.length()),
    d = 40,
    g = h
      .rotateByDegrees(d)
      .multiply(u - 1)
      .add(o),
    v = h
      .rotateByDegrees(-d)
      .multiply(u)
      .add(o)
      .add(h.rotateByDegrees(-d).multiply(2 * e.marker.size));
  return `M${g.x},${g.y}
          A${u},${u},0,1,0,${v.x},${v.y}`;
}
function Rdt({ config: t, source: e, target: r }) {
  const { t: o, dist: s, endNorm: u } = ad({ config: t, source: e, target: r }),
    f = 10,
    h = u
      .rotateByDegrees(f)
      .multiply(0.5 * s)
      .add(o);
  return `translate(${h.x},${h.y})`;
}
function zdt({ center: t, config: e, node: r }) {
  const { n: o, c: s } = Ub({ center: t, config: e, node: r }),
    u = o.subtract(s),
    f = u
      .multiply(1 / u.length())
      .multiply(3 * Ai(e, r) + 8)
      .add(o);
  return `translate(${f.x},${f.y})`;
}
const es = {
  line: { labelTransform: Odt, path: Pdt },
  arc: { labelTransform: Rdt, path: Ddt },
  reflexive: { labelTransform: zdt, path: $dt },
};
function Idt(t) {
  return t.append("g").classed("links", !0).selectAll("path");
}
function Fdt({ config: t, graph: e, selection: r, showLabels: o }) {
  const s =
    r == null
      ? void 0
      : r
          .data(e.links, (u) => ydt(u))
          .join((u) => {
            var f, h, d, g;
            const v = u.append("g"),
              b = v
                .append("path")
                .classed("link", !0)
                .style("marker-end", (S) => bdt(S))
                .style("stroke", (S) => S.color);
            (h = (f = t.modifiers).link) == null || h.call(f, b);
            const w = v
              .append("text")
              .classed("link__label", !0)
              .style("fill", (S) => (S.label ? S.label.color : null))
              .style("font-size", (S) => (S.label ? S.label.fontSize : null))
              .text((S) => (S.label ? S.label.text : null));
            return (g = (d = t.modifiers).linkLabel) == null || g.call(d, w), v;
          });
  return s == null || s.select(".link__label").attr("opacity", (u) => (u.label && o ? 1 : 0)), s;
}
function qdt(t) {
  Hdt(t), Bdt(t);
}
function Hdt({ center: t, config: e, graph: r, selection: o }) {
  o == null ||
    o
      .selectAll("path")
      .attr("d", (s) =>
        s.source.x === void 0 ||
        s.source.y === void 0 ||
        s.target.x === void 0 ||
        s.target.y === void 0
          ? ""
          : s.source.id === s.target.id
          ? es.reflexive.path({ config: e, node: s.source, center: t })
          : Vb(r, s.source, s.target)
          ? es.arc.path({ config: e, source: s.source, target: s.target })
          : es.line.path({ config: e, source: s.source, target: s.target }),
      );
}
function Bdt({ config: t, center: e, graph: r, selection: o }) {
  o == null ||
    o
      .select(".link__label")
      .attr("transform", (s) =>
        s.source.x === void 0 ||
        s.source.y === void 0 ||
        s.target.x === void 0 ||
        s.target.y === void 0
          ? "translate(0, 0)"
          : s.source.id === s.target.id
          ? es.reflexive.labelTransform({ config: t, node: s.source, center: e })
          : Vb(r, s.source, s.target)
          ? es.arc.labelTransform({ config: t, source: s.source, target: s.target })
          : es.line.labelTransform({ config: t, source: s.source, target: s.target }),
      );
}
function Vb(t, e, r) {
  return (
    e.id !== r.id &&
    t.links.some((o) => o.target.id === e.id && o.source.id === r.id) &&
    t.links.some((o) => o.target.id === r.id && o.source.id === e.id)
  );
}
function Wdt(t) {
  return t.append("defs").selectAll("marker");
}
function Udt({ config: t, graph: e, selection: r }) {
  return r == null
    ? void 0
    : r
        .data(jdt(e), (o) => o)
        .join((o) => {
          const s = o
            .append("marker")
            .attr("id", (u) => qb(u))
            .attr("markerHeight", 4 * t.marker.size)
            .attr("markerWidth", 4 * t.marker.size)
            .attr("markerUnits", "userSpaceOnUse")
            .attr("orient", "auto")
            .attr("refX", t.marker.ref[0])
            .attr("refY", t.marker.ref[1])
            .attr("viewBox", t.marker.viewBox)
            .style("fill", (u) => u);
          return s.append("path").attr("d", Vdt(t.marker.path)), s;
        });
}
function jdt(t) {
  return [...new Set(t.links.map((e) => e.color))];
}
function Vdt(t) {
  const [e, ...r] = t;
  if (!e) return "M0,0";
  const [o, s] = e;
  return r.reduce((u, [f, h]) => `${u}L${f},${h}`, `M${o},${s}`);
}
function Gdt(t) {
  return t.append("g").classed("nodes", !0).selectAll("circle");
}
function Kdt({
  config: t,
  drag: e,
  graph: r,
  onNodeContext: o,
  onNodeSelected: s,
  selection: u,
  showLabels: f,
}) {
  const h =
    u == null
      ? void 0
      : u
          .data(r.nodes, (d) => d.id)
          .join((d) => {
            var g, v, b, w;
            const S = d.append("g");
            e !== void 0 && S.call(e);
            const P = S.append("circle")
              .classed("node", !0)
              .attr("r", (L) => Ai(t, L))
              .on("contextmenu", (L, T) => {
                Ib(L), o(T);
              })
              .on("pointerdown", (L, T) => Ydt(L, T, s ?? o))
              .style("fill", (L) => L.color);
            (v = (g = t.modifiers).node) == null || v.call(g, P);
            const A = S.append("text")
              .classed("node__label", !0)
              .attr("dy", "0.33em")
              .style("fill", (L) => (L.label ? L.label.color : null))
              .style("font-size", (L) => (L.label ? L.label.fontSize : null))
              .style("stroke", "none")
              .text((L) => (L.label ? L.label.text : null));
            return (w = (b = t.modifiers).nodeLabel) == null || w.call(b, A), S;
          });
  return (
    h == null || h.select(".node").classed("focused", (d) => d.isFocused),
    h == null || h.select(".node__label").attr("opacity", f ? 1 : 0),
    h
  );
}
const Xdt = 500;
function Ydt(t, e, r) {
  if (t.button !== void 0 && t.button !== 0) return;
  const o = e.lastInteractionTimestamp,
    s = Date.now();
  if (o === void 0 || s - o > Xdt) {
    e.lastInteractionTimestamp = s;
    return;
  }
  (e.lastInteractionTimestamp = void 0), r(e);
}
function Zdt(t) {
  t == null || t.attr("transform", (e) => `translate(${e.x ?? 0},${e.y ?? 0})`);
}
function Jdt({ center: t, config: e, graph: r, onTick: o }) {
  var s, u;
  const f = cdt(r.nodes),
    h = e.simulation.forces.centering;
  if (h && h.enabled) {
    const b = h.strength;
    f.force("x", fdt(() => t().x).strength(b)).force("y", hdt(() => t().y).strength(b));
  }
  const d = e.simulation.forces.charge;
  d && d.enabled && f.force("charge", udt().strength(d.strength));
  const g = e.simulation.forces.collision;
  g &&
    g.enabled &&
    f.force(
      "collision",
      Qht().radius((b) => g.radiusMultiplier * Ai(e, b)),
    );
  const v = e.simulation.forces.link;
  return (
    v &&
      v.enabled &&
      f.force(
        "link",
        edt(r.links)
          .id((b) => b.id)
          .distance(e.simulation.forces.link.length)
          .strength(v.strength),
      ),
    f.on("tick", () => o()),
    (u = (s = e.modifiers).simulation) == null || u.call(s, f),
    f
  );
}
function Qdt({ canvasContainer: t, config: e, min: r, max: o, onZoom: s }) {
  var u, f;
  const h = Dht()
    .scaleExtent([r, o])
    .filter((d) => {
      var g;
      return d.button === 0 || ((g = d.touches) == null ? void 0 : g.length) >= 2;
    })
    .on("start", () => t().classed("grabbed", !0))
    .on("zoom", (d) => s(d))
    .on("end", () => t().classed("grabbed", !1));
  return (f = (u = e.modifiers).zoom) == null || f.call(u, h), h;
}
class tpt {
  constructor(e, r, o) {
    if (
      (Oe(this, "nodeTypes"),
      Oe(this, "_nodeTypeFilter"),
      Oe(this, "_includeUnlinked", !0),
      Oe(this, "_linkFilter", () => !0),
      Oe(this, "_showLinkLabels", !0),
      Oe(this, "_showNodeLabels", !0),
      Oe(this, "filteredGraph"),
      Oe(this, "width", 0),
      Oe(this, "height", 0),
      Oe(this, "simulation"),
      Oe(this, "canvas"),
      Oe(this, "linkSelection"),
      Oe(this, "nodeSelection"),
      Oe(this, "markerSelection"),
      Oe(this, "zoom"),
      Oe(this, "drag"),
      Oe(this, "xOffset", 0),
      Oe(this, "yOffset", 0),
      Oe(this, "scale"),
      Oe(this, "focusedNode"),
      Oe(this, "resizeObserver"),
      (this.container = e),
      (this.graph = r),
      (this.config = o),
      (this.scale = o.zoom.initial),
      this.resetView(),
      this.graph.nodes.forEach((s) => {
        const [u, f] = o.positionInitializer(s, this.effectiveWidth, this.effectiveHeight);
        (s.x = s.x ?? u), (s.y = s.y ?? f);
      }),
      (this.nodeTypes = [...new Set(r.nodes.map((s) => s.type))]),
      (this._nodeTypeFilter = [...this.nodeTypes]),
      o.initial)
    ) {
      const {
        includeUnlinked: s,
        nodeTypeFilter: u,
        linkFilter: f,
        showLinkLabels: h,
        showNodeLabels: d,
      } = o.initial;
      (this._includeUnlinked = s ?? this._includeUnlinked),
        (this._showLinkLabels = h ?? this._showLinkLabels),
        (this._showNodeLabels = d ?? this._showNodeLabels),
        (this._nodeTypeFilter = u ?? this._nodeTypeFilter),
        (this._linkFilter = f ?? this._linkFilter);
    }
    this.filterGraph(void 0),
      this.initGraph(),
      this.restart(o.simulation.alphas.initialize),
      o.autoResize &&
        ((this.resizeObserver = new ResizeObserver(nct(() => this.resize()))),
        this.resizeObserver.observe(this.container));
  }
  get nodeTypeFilter() {
    return this._nodeTypeFilter;
  }
  get includeUnlinked() {
    return this._includeUnlinked;
  }
  set includeUnlinked(e) {
    (this._includeUnlinked = e), this.filterGraph(this.focusedNode);
    const { include: r, exclude: o } = this.config.simulation.alphas.filter.unlinked,
      s = e ? r : o;
    this.restart(s);
  }
  set linkFilter(e) {
    (this._linkFilter = e),
      this.filterGraph(this.focusedNode),
      this.restart(this.config.simulation.alphas.filter.link);
  }
  get linkFilter() {
    return this._linkFilter;
  }
  get showNodeLabels() {
    return this._showNodeLabels;
  }
  set showNodeLabels(e) {
    this._showNodeLabels = e;
    const { hide: r, show: o } = this.config.simulation.alphas.labels.nodes,
      s = e ? o : r;
    this.restart(s);
  }
  get showLinkLabels() {
    return this._showLinkLabels;
  }
  set showLinkLabels(e) {
    this._showLinkLabels = e;
    const { hide: r, show: o } = this.config.simulation.alphas.labels.links,
      s = e ? o : r;
    this.restart(s);
  }
  get effectiveWidth() {
    return this.width / this.scale;
  }
  get effectiveHeight() {
    return this.height / this.scale;
  }
  get effectiveCenter() {
    return pn
      .of([this.width, this.height])
      .divide(2)
      .subtract(pn.of([this.xOffset, this.yOffset]))
      .divide(this.scale);
  }
  resize() {
    const e = this.width,
      r = this.height,
      o = this.container.getBoundingClientRect().width,
      s = this.container.getBoundingClientRect().height,
      u = e.toFixed() !== o.toFixed(),
      f = r.toFixed() !== s.toFixed();
    if (!u && !f) return;
    (this.width = this.container.getBoundingClientRect().width),
      (this.height = this.container.getBoundingClientRect().height);
    const h = this.config.simulation.alphas.resize;
    this.restart(Fb(h) ? h : h({ oldWidth: e, oldHeight: r, newWidth: o, newHeight: s }));
  }
  restart(e) {
    var r;
    (this.markerSelection = Udt({
      config: this.config,
      graph: this.filteredGraph,
      selection: this.markerSelection,
    })),
      (this.linkSelection = Fdt({
        config: this.config,
        graph: this.filteredGraph,
        selection: this.linkSelection,
        showLabels: this._showLinkLabels,
      })),
      (this.nodeSelection = Kdt({
        config: this.config,
        drag: this.drag,
        graph: this.filteredGraph,
        onNodeContext: (o) => this.toggleNodeFocus(o),
        onNodeSelected: this.config.callbacks.nodeClicked,
        selection: this.nodeSelection,
        showLabels: this._showNodeLabels,
      })),
      (r = this.simulation) == null || r.stop(),
      (this.simulation = Jdt({
        center: () => this.effectiveCenter,
        config: this.config,
        graph: this.filteredGraph,
        onTick: () => this.onTick(),
      })
        .alpha(e)
        .restart());
  }
  filterNodesByType(e, r) {
    e
      ? this._nodeTypeFilter.push(r)
      : (this._nodeTypeFilter = this._nodeTypeFilter.filter((o) => o !== r)),
      this.filterGraph(this.focusedNode),
      this.restart(this.config.simulation.alphas.filter.type);
  }
  shutdown() {
    var e, r;
    this.focusedNode !== void 0 && ((this.focusedNode.isFocused = !1), (this.focusedNode = void 0)),
      (e = this.resizeObserver) == null || e.unobserve(this.container),
      (r = this.simulation) == null || r.stop();
  }
  initGraph() {
    (this.zoom = Qdt({
      config: this.config,
      canvasContainer: () => En(this.container).select("svg"),
      min: this.config.zoom.min,
      max: this.config.zoom.max,
      onZoom: (e) => this.onZoom(e),
    })),
      (this.canvas = Cdt({
        applyZoom: this.scale !== 1,
        container: En(this.container),
        offset: [this.xOffset, this.yOffset],
        scale: this.scale,
        zoom: this.zoom,
      })),
      this.applyZoom(),
      (this.linkSelection = Idt(this.canvas)),
      (this.nodeSelection = Gdt(this.canvas)),
      (this.markerSelection = Wdt(this.canvas)),
      (this.drag = Edt({
        config: this.config,
        onDragStart: () => {
          var e;
          return (e = this.simulation) == null
            ? void 0
            : e.alphaTarget(this.config.simulation.alphas.drag.start).restart();
        },
        onDragEnd: () => {
          var e;
          return (e = this.simulation) == null
            ? void 0
            : e.alphaTarget(this.config.simulation.alphas.drag.end).restart();
        },
      }));
  }
  onTick() {
    Zdt(this.nodeSelection),
      qdt({
        config: this.config,
        center: this.effectiveCenter,
        graph: this.filteredGraph,
        selection: this.linkSelection,
      });
  }
  resetView() {
    var e;
    (e = this.simulation) == null || e.stop(),
      En(this.container).selectChildren().remove(),
      (this.zoom = void 0),
      (this.canvas = void 0),
      (this.linkSelection = void 0),
      (this.nodeSelection = void 0),
      (this.markerSelection = void 0),
      (this.simulation = void 0),
      (this.width = this.container.getBoundingClientRect().width),
      (this.height = this.container.getBoundingClientRect().height);
  }
  onZoom(e) {
    var r, o, s;
    (this.xOffset = e.transform.x),
      (this.yOffset = e.transform.y),
      (this.scale = e.transform.k),
      this.applyZoom(),
      (o = (r = this.config.hooks).afterZoom) == null ||
        o.call(r, this.scale, this.xOffset, this.yOffset),
      (s = this.simulation) == null || s.restart();
  }
  applyZoom() {
    Tdt({ canvas: this.canvas, scale: this.scale, xOffset: this.xOffset, yOffset: this.yOffset });
  }
  toggleNodeFocus(e) {
    e.isFocused
      ? (this.filterGraph(void 0), this.restart(this.config.simulation.alphas.focus.release(e)))
      : this.focusNode(e);
  }
  focusNode(e) {
    this.filterGraph(e), this.restart(this.config.simulation.alphas.focus.acquire(e));
  }
  filterGraph(e) {
    this.focusedNode !== void 0 && ((this.focusedNode.isFocused = !1), (this.focusedNode = void 0)),
      e !== void 0 &&
        this._nodeTypeFilter.includes(e.type) &&
        ((e.isFocused = !0), (this.focusedNode = e)),
      (this.filteredGraph = Ldt({
        graph: this.graph,
        filter: this._nodeTypeFilter,
        focusedNode: this.focusedNode,
        includeUnlinked: this._includeUnlinked,
        linkFilter: this._linkFilter,
      }));
  }
}
function Sm({ nodes: t, links: e }) {
  return { nodes: t ?? [], links: e ?? [] };
}
function ept(t) {
  return { ...t };
}
function Gb(t) {
  return { ...t, isFocused: !1, lastInteractionTimestamp: void 0 };
}
const npt = { "h-full": "", "min-h-75": "", "flex-1": "", overflow: "hidden" },
  rpt = { flex: "", "items-center": "", "gap-4": "", "px-3": "", "py-2": "" },
  ipt = ["id", "checked", "onChange"],
  opt = ["for"],
  spt = tt("div", { "flex-auto": "" }, null, -1),
  lpt = ie({
    __name: "ViewModuleGraph",
    props: { graph: {} },
    setup(t) {
      const e = t,
        { graph: r } = z1(e),
        o = Zt(),
        s = Zt(!1),
        u = Zt(),
        f = Zt();
      Th(
        () => {
          s.value === !1 && setTimeout(() => (u.value = void 0), 300);
        },
        { flush: "post" },
      ),
        ms(() => {
          g();
        }),
        Lh(() => {
          var b;
          (b = f.value) == null || b.shutdown();
        }),
        Re(r, g);
      function h(b, w) {
        var S;
        (S = f.value) == null || S.filterNodesByType(w, b);
      }
      function d(b) {
        (u.value = b), (s.value = !0);
      }
      function g() {
        var b;
        (b = f.value) == null || b.shutdown(),
          !(!r.value || !o.value) &&
            (f.value = new tpt(
              o.value,
              r.value,
              kdt({
                nodeRadius: 10,
                autoResize: !0,
                simulation: {
                  alphas: {
                    initialize: 1,
                    resize: ({ newHeight: w, newWidth: S }) => (w === 0 && S === 0 ? 0 : 0.25),
                  },
                  forces: { collision: { radiusMultiplier: 10 }, link: { length: 240 } },
                },
                marker: Hb.Arrow(2),
                modifiers: { node: v },
                positionInitializer: r.value.nodes.length > 1 ? ih.Randomized : ih.Centered,
                zoom: { min: 0.5, max: 2 },
              }),
            ));
      }
      function v(b) {
        if (jr) return;
        const w = (L) => L.button === 0;
        let S = 0,
          P = 0,
          A = 0;
        b.on("pointerdown", (L, T) => {
          T.type !== "external" &&
            (!T.x || !T.y || !w(L) || ((S = T.x), (P = T.y), (A = Date.now())));
        }).on("pointerup", (L, T) => {
          if (T.type === "external" || !T.x || !T.y || !w(L) || Date.now() - A > 500) return;
          const M = T.x - S,
            R = T.y - P;
          M ** 2 + R ** 2 < 100 && d(T.id);
        });
      }
      return (b, w) => {
        var T;
        const S = bs,
          P = ect,
          A = Wat,
          L = uo("tooltip");
        return (
          st(),
          kt("div", npt, [
            tt("div", null, [
              tt("div", rpt, [
                (st(!0),
                kt(
                  ne,
                  null,
                  Rn((T = U(f)) == null ? void 0 : T.nodeTypes.sort(), (M) => {
                    var R;
                    return (
                      st(),
                      kt(
                        "div",
                        { key: M, flex: "~ gap-1", "items-center": "", "select-none": "" },
                        [
                          tt(
                            "input",
                            {
                              id: `type-${M}`,
                              type: "checkbox",
                              checked: (R = U(f)) == null ? void 0 : R.nodeTypeFilter.includes(M),
                              onChange: (E) => h(M, E.target.checked),
                            },
                            null,
                            40,
                            ipt,
                          ),
                          tt(
                            "label",
                            {
                              "font-light": "",
                              "text-sm": "",
                              "ws-nowrap": "",
                              "overflow-hidden": "",
                              capitalize: "",
                              truncate: "",
                              for: `type-${M}`,
                              "border-b-2": "",
                              style: An({ "border-color": `var(--color-node-${M})` }),
                            },
                            Ut(M) + " Modules",
                            13,
                            opt,
                          ),
                        ],
                      )
                    );
                  }),
                  128,
                )),
                spt,
                tt("div", null, [
                  nn(Ft(S, { icon: "i-carbon-reset", onClick: g }, null, 512), [
                    [L, "Reset", void 0, { bottom: !0 }],
                  ]),
                ]),
              ]),
            ]),
            tt("div", { ref_key: "el", ref: o }, null, 512),
            Ft(
              A,
              {
                modelValue: U(s),
                "onUpdate:modelValue": w[1] || (w[1] = (M) => (Le(s) ? (s.value = M) : null)),
                direction: "right",
              },
              {
                default: ee(() => [
                  U(u)
                    ? (st(),
                      te(
                        n_,
                        { key: 0 },
                        {
                          default: ee(() => [
                            Ft(
                              P,
                              { id: U(u), onClose: w[0] || (w[0] = (M) => (s.value = !1)) },
                              null,
                              8,
                              ["id"],
                            ),
                          ]),
                          _: 1,
                        },
                      ))
                    : Vt("", !0),
                ]),
                _: 1,
              },
              8,
              ["modelValue"],
            ),
          ])
        );
      };
    },
  }),
  apt = { key: 0, "text-green-500": "", "flex-shrink-0": "", "i-carbon:checkmark": "" },
  cpt = { key: 1, "text-red-500": "", "flex-shrink-0": "", "i-carbon:compare": "" },
  upt = { key: 2, "text-red-500": "", "flex-shrink-0": "", "i-carbon:close": "" },
  fpt = { key: 3, "text-gray-500": "", "flex-shrink-0": "", "i-carbon:document-blank": "" },
  hpt = { key: 4, "text-gray-500": "", "flex-shrink-0": "", "i-carbon:redo": "", "rotate-90": "" },
  dpt = {
    key: 5,
    "text-yellow-500": "",
    "flex-shrink-0": "",
    "i-carbon:circle-dash": "",
    "animate-spin": "",
  },
  cd = ie({
    __name: "StatusIcon",
    props: { task: {} },
    setup(t) {
      return (e, r) => {
        var s, u, f;
        const o = uo("tooltip");
        return ((s = e.task.result) == null ? void 0 : s.state) === "pass"
          ? (st(), kt("div", apt))
          : U(Wc)(e.task)
          ? nn((st(), kt("div", cpt, null, 512)), [
              [o, "Contains failed snapshot", void 0, { right: !0 }],
            ])
          : ((u = e.task.result) == null ? void 0 : u.state) === "fail"
          ? (st(), kt("div", upt))
          : e.task.mode === "todo"
          ? nn((st(), kt("div", fpt, null, 512)), [[o, "Todo", void 0, { right: !0 }]])
          : e.task.mode === "skip" || ((f = e.task.result) == null ? void 0 : f.state) === "skip"
          ? nn((st(), kt("div", hpt, null, 512)), [[o, "Skipped", void 0, { right: !0 }]])
          : (st(), kt("div", dpt));
      };
    },
  });
function ppt(t) {
  const e = new Map(),
    r = new Map(),
    o = [];
  for (;;) {
    let s = 0;
    if (
      (t.forEach((u, f) => {
        var v;
        const { splits: h, finished: d } = u;
        if (d) {
          s++;
          const { raw: b, candidate: w } = u;
          e.set(b, w);
          return;
        }
        if (h.length === 0) {
          u.finished = !0;
          return;
        }
        const g = h[0];
        r.has(g)
          ? ((u.candidate += u.candidate === "" ? g : `/${g}`),
            (v = r.get(g)) == null || v.push(f),
            h.shift())
          : (r.set(g, [f]), o.push(f));
      }),
      o.forEach((u) => {
        const f = t[u],
          h = f.splits.shift();
        f.candidate += f.candidate === "" ? h : `/${h}`;
      }),
      r.forEach((u) => {
        if (u.length === 1) {
          const f = u[0];
          t[f].finished = !0;
        }
      }),
      r.clear(),
      (o.length = 0),
      s === t.length)
    )
      break;
  }
  return e;
}
function gpt(t) {
  let e = t;
  e.includes("/node_modules/") && (e = t.split(/\/node_modules\//g).pop());
  const r = e.split(/\//g);
  return { raw: e, splits: r, candidate: "", finished: !1, id: t };
}
function vpt(t) {
  const e = t.map((o) => gpt(o)),
    r = ppt(e);
  return e.map(({ raw: o, id: s }) =>
    Gb({
      color: "var(--color-node-external)",
      label: { color: "var(--color-node-external)", fontSize: "0.875rem", text: r.get(o) ?? "" },
      isFocused: !1,
      id: s,
      type: "external",
    }),
  );
}
function mpt(t, e) {
  return Gb({
    color: e ? "var(--color-node-root)" : "var(--color-node-inline)",
    label: {
      color: e ? "var(--color-node-root)" : "var(--color-node-inline)",
      fontSize: "0.875rem",
      text: t.split(/\//g).pop(),
    },
    isFocused: !1,
    id: t,
    type: "inline",
  });
}
function ypt(t, e) {
  if (!t) return Sm({});
  const r = vpt(t.externalized),
    o = t.inlined.map((h) => mpt(h, h === e)) ?? [],
    s = [...r, ...o],
    u = Object.fromEntries(s.map((h) => [h.id, h])),
    f = Object.entries(t.graph).flatMap(([h, d]) =>
      d
        .map((g) => {
          const v = u[h],
            b = u[g];
          if (!(v === void 0 || b === void 0))
            return ept({ source: v, target: b, color: "var(--color-link)", label: !1 });
        })
        .filter((g) => g !== void 0),
    );
  return Sm({ nodes: s, links: f });
}
const bpt = {
    key: 0,
    flex: "",
    "flex-col": "",
    "h-full": "",
    "max-h-full": "",
    "overflow-hidden": "",
    "data-testid": "file-detail",
  },
  wpt = {
    p: "2",
    "h-10": "",
    flex: "~ gap-2",
    "items-center": "",
    "bg-header": "",
    border: "b base",
  },
  xpt = {
    "flex-1": "",
    "font-light": "",
    "op-50": "",
    "ws-nowrap": "",
    truncate: "",
    "text-sm": "",
  },
  _pt = { class: "flex text-lg" },
  Spt = {
    flex: "~",
    "items-center": "",
    "bg-header": "",
    border: "b-2 base",
    "text-sm": "",
    "h-41px": "",
  },
  kpt = { flex: "", "flex-col": "", "flex-1": "", overflow: "hidden" },
  Cpt = ["flex-1"],
  Tpt = ie({
    __name: "FileDetails",
    setup(t) {
      const e = Zt({ externalized: [], graph: {}, inlined: [] }),
        r = Zt({ nodes: [], links: [] }),
        o = Zt(!1),
        s = Zt(!1);
      _lt(
        Se,
        async (g, v) => {
          g &&
            g.filepath !== (v == null ? void 0 : v.filepath) &&
            ((e.value = await je.rpc.getModuleGraph(g.filepath)),
            (r.value = ypt(e.value, g.filepath)));
        },
        { debounce: 100, immediate: !0 },
      );
      function u() {
        var v;
        const g = (v = Se.value) == null ? void 0 : v.filepath;
        g && fetch(`/__open-in-editor?file=${encodeURIComponent(g)}`);
      }
      function f(g) {
        g === "graph" && (s.value = !0), (er.value = g);
      }
      const h = xt(() => {
        var g;
        return ((g = nb.value) == null ? void 0 : g.reduce((v, { size: b }) => v + b, 0)) ?? 0;
      });
      function d(g) {
        o.value = g;
      }
      return (g, v) => {
        var M, R;
        const b = cd,
          w = bs,
          S = lpt,
          P = Bat,
          A = $at,
          L = Sat,
          T = uo("tooltip");
        return U(Se)
          ? (st(),
            kt("div", bpt, [
              tt("div", null, [
                tt("div", wpt, [
                  Ft(b, { task: U(Se) }, null, 8, ["task"]),
                  tt("div", xpt, Ut((M = U(Se)) == null ? void 0 : M.filepath), 1),
                  tt("div", _pt, [
                    U(jr)
                      ? Vt("", !0)
                      : nn(
                          (st(),
                          te(
                            w,
                            {
                              key: 0,
                              title: "Open in editor",
                              icon: "i-carbon-launch",
                              disabled: !((R = U(Se)) != null && R.filepath),
                              onClick: u,
                            },
                            null,
                            8,
                            ["disabled"],
                          )),
                          [[T, "Open in editor", void 0, { bottom: !0 }]],
                        ),
                  ]),
                ]),
                tt("div", Spt, [
                  tt(
                    "button",
                    {
                      "tab-button": "",
                      class: ve({ "tab-button-active": U(er) == null }),
                      "data-testid": "btn-report",
                      onClick: v[0] || (v[0] = (E) => f(null)),
                    },
                    " Report ",
                    2,
                  ),
                  tt(
                    "button",
                    {
                      "tab-button": "",
                      "data-testid": "btn-graph",
                      class: ve({ "tab-button-active": U(er) === "graph" }),
                      onClick: v[1] || (v[1] = (E) => f("graph")),
                    },
                    " Module Graph ",
                    2,
                  ),
                  U(jr)
                    ? Vt("", !0)
                    : (st(),
                      kt(
                        "button",
                        {
                          key: 0,
                          "tab-button": "",
                          "data-testid": "btn-code",
                          class: ve({ "tab-button-active": U(er) === "editor" }),
                          onClick: v[2] || (v[2] = (E) => f("editor")),
                        },
                        Ut(U(o) ? "* " : "") + "Code ",
                        3,
                      )),
                  tt(
                    "button",
                    {
                      "tab-button": "",
                      "data-testid": "btn-console",
                      class: ve({
                        "tab-button-active": U(er) === "console",
                        op20: U(er) !== "console" && U(h) === 0,
                      }),
                      onClick: v[3] || (v[3] = (E) => f("console")),
                    },
                    " Console (" + Ut(U(h)) + ") ",
                    3,
                  ),
                ]),
              ]),
              tt("div", kpt, [
                U(s)
                  ? (st(),
                    kt(
                      "div",
                      { key: 0, "flex-1": U(er) === "graph" && "" },
                      [
                        nn(Ft(S, { graph: U(r), "data-testid": "graph" }, null, 8, ["graph"]), [
                          [Pf, U(er) === "graph"],
                        ]),
                      ],
                      8,
                      Cpt,
                    ))
                  : Vt("", !0),
                U(er) === "editor"
                  ? (st(),
                    te(
                      P,
                      { key: U(Se).filepath, file: U(Se), "data-testid": "editor", onDraft: d },
                      null,
                      8,
                      ["file"],
                    ))
                  : U(er) === "console"
                  ? (st(),
                    te(A, { key: 2, file: U(Se), "data-testid": "console" }, null, 8, ["file"]))
                  : U(er)
                  ? Vt("", !0)
                  : (st(),
                    te(L, { key: 3, file: U(Se), "data-testid": "report" }, null, 8, ["file"])),
              ]),
            ]))
          : Vt("", !0);
      };
    },
  }),
  Ept = ["open"],
  Lpt = tt("div", { "flex-1": "", "h-1px": "", border: "base b", op80: "" }, null, -1),
  Apt = tt("div", { "flex-1": "", "h-1px": "", border: "base b", op80: "" }, null, -1),
  Mpt = ie({
    __name: "DetailsPanel",
    props: { color: {} },
    setup(t) {
      const e = Zt(!0);
      return (r, o) => (
        st(),
        kt(
          "div",
          {
            open: U(e),
            class: "details-panel",
            "data-testid": "details-panel",
            onToggle: o[0] || (o[0] = (s) => (e.value = s.target.open)),
          },
          [
            tt(
              "div",
              {
                p: "y1",
                "text-sm": "",
                "bg-base": "",
                "items-center": "",
                "z-5": "",
                "gap-2": "",
                class: ve(r.color),
                "w-full": "",
                flex: "",
                "select-none": "",
                sticky: "",
                top: "-1",
              },
              [Lpt, sr(r.$slots, "summary", { open: U(e) }), Apt],
              2,
            ),
            sr(r.$slots, "default"),
          ],
          40,
          Ept,
        )
      );
    },
  }),
  Npt = {
    key: 0,
    flex: "~ row",
    "items-center": "",
    p: "x-2 y-1",
    "border-rounded": "",
    "cursor-pointer": "",
    hover: "bg-active",
  },
  Ppt = { key: 0, "i-logos:typescript-icon": "", "flex-shrink-0": "", "mr-2": "" },
  Opt = ["text"],
  Dpt = { "text-sm": "", truncate: "", "font-light": "" },
  $pt = { key: 0, text: "xs", op20: "", style: { "white-space": "nowrap" } },
  Rpt = ie({
    __name: "TaskItem",
    props: { task: {} },
    setup(t) {
      const e = t,
        r = xt(() => {
          const { result: o } = e.task;
          return o && Math.round(o.duration || 0);
        });
      return (o, s) => {
        var f, h;
        const u = cd;
        return o.task
          ? (st(),
            kt("div", Npt, [
              Ft(u, { task: o.task, "mr-2": "" }, null, 8, ["task"]),
              o.task.type === "suite" && o.task.meta.typecheck
                ? (st(), kt("div", Ppt))
                : Vt("", !0),
              tt(
                "div",
                {
                  flex: "",
                  "items-end": "",
                  "gap-2": "",
                  text:
                    ((h = (f = o.task) == null ? void 0 : f.result) == null ? void 0 : h.state) ===
                    "fail"
                      ? "red-500"
                      : "",
                },
                [
                  tt("span", Dpt, Ut(o.task.name), 1),
                  typeof U(r) == "number"
                    ? (st(), kt("span", $pt, Ut(U(r) > 0 ? U(r) : "< 1") + "ms ", 1))
                    : Vt("", !0),
                ],
                8,
                Opt,
              ),
            ]))
          : Vt("", !0);
      };
    },
  });
function zpt(t) {
  return Object.hasOwnProperty.call(t, "tasks");
}
function Kb(t, e) {
  return typeof t != "string" || typeof e != "string"
    ? !1
    : t.toLowerCase().includes(e.toLowerCase());
}
const Ipt = { key: 1 },
  Fpt = ie({
    inheritAttrs: !1,
    __name: "TaskTree",
    props: {
      task: {},
      indent: { default: 0 },
      nested: { type: Boolean, default: !1 },
      search: {},
      onItemClick: { type: Function },
    },
    setup(t) {
      return (e, r) => {
        const o = Rpt,
          s = io("TaskTree", !0);
        return (
          st(),
          kt(
            ne,
            null,
            [
              !e.nested || !e.search || U(Kb)(e.task.name, e.search)
                ? (st(),
                  te(
                    o,
                    Ci({ key: 0 }, e.$attrs, {
                      task: e.task,
                      style: { paddingLeft: `${e.indent * 0.75 + 1}rem` },
                      onClick: r[0] || (r[0] = (u) => e.onItemClick && e.onItemClick(e.task)),
                    }),
                    null,
                    16,
                    ["task", "style"],
                  ))
                : Vt("", !0),
              e.nested && e.task.type === "suite" && e.task.tasks.length
                ? (st(),
                  kt("div", Ipt, [
                    (st(!0),
                    kt(
                      ne,
                      null,
                      Rn(
                        e.task.tasks,
                        (u) => (
                          st(),
                          te(
                            s,
                            {
                              key: u.id,
                              task: u,
                              nested: e.nested,
                              indent: e.indent + 1,
                              search: e.search,
                              "on-item-click": e.onItemClick,
                            },
                            null,
                            8,
                            ["task", "nested", "indent", "search", "on-item-click"],
                          )
                        ),
                      ),
                      128,
                    )),
                  ]))
                : Vt("", !0),
            ],
            64,
          )
        );
      };
    },
  }),
  qpt = { h: "full", flex: "~ col" },
  Hpt = {
    p: "2",
    "h-10": "",
    flex: "~ gap-2",
    "items-center": "",
    "bg-header": "",
    border: "b base",
  },
  Bpt = { p: "l3 y2 r2", flex: "~ gap-2", "items-center": "", "bg-header": "", border: "b-2 base" },
  Wpt = tt("div", { class: "i-carbon:search", "flex-shrink-0": "" }, null, -1),
  Upt = ["op"],
  jpt = { class: "scrolls", "flex-auto": "", "py-1": "" },
  Vpt = { "text-red5": "" },
  Gpt = { "text-yellow5": "" },
  Kpt = { "text-green5": "" },
  Xpt = { class: "text-purple5:50" },
  Ypt = { key: 2, flex: "~ col", "items-center": "", p: "x4 y4", "font-light": "" },
  Zpt = tt("div", { op30: "" }, " No matched test ", -1),
  Xb = ie({
    inheritAttrs: !1,
    __name: "TasksList",
    props: {
      tasks: {},
      indent: { default: 0 },
      nested: { type: Boolean, default: !1 },
      groupByType: { type: Boolean, default: !1 },
      onItemClick: { type: Function },
    },
    emits: ["run"],
    setup(t, { emit: e }) {
      const r = e,
        o = Zt(""),
        s = Zt(),
        u = xt(() => o.value.trim() !== ""),
        f = xt(() => (o.value.trim() ? t.tasks.filter((L) => A([L], o.value)) : t.tasks)),
        h = xt(() => (u.value ? f.value.map((L) => hc(L.id)).filter(Boolean) : [])),
        d = xt(() =>
          f.value.filter((L) => {
            var T;
            return ((T = L.result) == null ? void 0 : T.state) === "fail";
          }),
        ),
        g = xt(() =>
          f.value.filter((L) => {
            var T;
            return ((T = L.result) == null ? void 0 : T.state) === "pass";
          }),
        ),
        v = xt(() => f.value.filter((L) => L.mode === "skip" || L.mode === "todo")),
        b = xt(() =>
          f.value.filter(
            (L) => !d.value.includes(L) && !g.value.includes(L) && !v.value.includes(L),
          ),
        ),
        w = xt(() => o.value === ""),
        S = wlt(b, 250);
      function P(L) {
        var T;
        (o.value = ""), L && ((T = s.value) == null || T.focus());
      }
      function A(L, T) {
        let M = !1;
        for (let R = 0; R < L.length; R++) {
          const E = L[R];
          if (Kb(E.name, T)) {
            M = !0;
            break;
          }
          if (zpt(E) && E.tasks && ((M = A(E.tasks, T)), M)) break;
        }
        return M;
      }
      return (L, T) => {
        const M = bs,
          R = Fpt,
          E = Mpt,
          B = uo("tooltip");
        return (
          st(),
          kt("div", qpt, [
            tt("div", null, [
              tt("div", Hpt, [sr(L.$slots, "header", { filteredTests: U(u) ? U(h) : void 0 })]),
              tt("div", Bpt, [
                Wpt,
                nn(
                  tt(
                    "input",
                    {
                      ref_key: "searchBox",
                      ref: s,
                      "onUpdate:modelValue": T[0] || (T[0] = (K) => (Le(o) ? (o.value = K) : null)),
                      placeholder: "Search...",
                      outline: "none",
                      bg: "transparent",
                      font: "light",
                      text: "sm",
                      "flex-1": "",
                      "pl-1": "",
                      op: U(o).length ? "100" : "50",
                      onKeydown: [
                        T[1] || (T[1] = Df((K) => P(!1), ["esc"])),
                        T[2] || (T[2] = Df((K) => r("run", U(u) ? U(h) : void 0), ["enter"])),
                      ],
                    },
                    null,
                    40,
                    Upt,
                  ),
                  [[kS, U(o)]],
                ),
                nn(
                  Ft(
                    M,
                    {
                      disabled: U(w),
                      title: "Clear search",
                      icon: "i-carbon:filter-remove",
                      onClickPassive: T[3] || (T[3] = (K) => P(!0)),
                    },
                    null,
                    8,
                    ["disabled"],
                  ),
                  [[B, "Clear search", void 0, { bottom: !0 }]],
                ),
              ]),
            ]),
            tt("div", jpt, [
              L.groupByType
                ? (st(),
                  kt(
                    ne,
                    { key: 0 },
                    [
                      U(d).length
                        ? (st(),
                          te(
                            E,
                            { key: 0 },
                            {
                              summary: ee(() => [
                                tt("div", Vpt, " FAIL (" + Ut(U(d).length) + ") ", 1),
                              ]),
                              default: ee(() => [
                                (st(!0),
                                kt(
                                  ne,
                                  null,
                                  Rn(
                                    U(d),
                                    (K) => (
                                      st(),
                                      te(
                                        R,
                                        {
                                          key: K.id,
                                          task: K,
                                          nested: L.nested,
                                          search: U(o),
                                          class: ve(U(yr) === K.id ? "bg-active" : ""),
                                          "on-item-click": L.onItemClick,
                                        },
                                        null,
                                        8,
                                        ["task", "nested", "search", "class", "on-item-click"],
                                      )
                                    ),
                                  ),
                                  128,
                                )),
                              ]),
                              _: 1,
                            },
                          ))
                        : Vt("", !0),
                      U(b).length || U(kl) === "running"
                        ? (st(),
                          te(
                            E,
                            { key: 1 },
                            {
                              summary: ee(() => [
                                tt("div", Gpt, " RUNNING (" + Ut(U(S).length) + ") ", 1),
                              ]),
                              default: ee(() => [
                                (st(!0),
                                kt(
                                  ne,
                                  null,
                                  Rn(
                                    U(S),
                                    (K) => (
                                      st(),
                                      te(
                                        R,
                                        {
                                          key: K.id,
                                          task: K,
                                          nested: L.nested,
                                          search: U(o),
                                          class: ve(U(yr) === K.id ? "bg-active" : ""),
                                          "on-item-click": L.onItemClick,
                                        },
                                        null,
                                        8,
                                        ["task", "nested", "search", "class", "on-item-click"],
                                      )
                                    ),
                                  ),
                                  128,
                                )),
                              ]),
                              _: 1,
                            },
                          ))
                        : Vt("", !0),
                      U(g).length
                        ? (st(),
                          te(
                            E,
                            { key: 2 },
                            {
                              summary: ee(() => [
                                tt("div", Kpt, " PASS (" + Ut(U(g).length) + ") ", 1),
                              ]),
                              default: ee(() => [
                                (st(!0),
                                kt(
                                  ne,
                                  null,
                                  Rn(
                                    U(g),
                                    (K) => (
                                      st(),
                                      te(
                                        R,
                                        {
                                          key: K.id,
                                          task: K,
                                          nested: L.nested,
                                          search: U(o),
                                          class: ve(U(yr) === K.id ? "bg-active" : ""),
                                          "on-item-click": L.onItemClick,
                                        },
                                        null,
                                        8,
                                        ["task", "nested", "search", "class", "on-item-click"],
                                      )
                                    ),
                                  ),
                                  128,
                                )),
                              ]),
                              _: 1,
                            },
                          ))
                        : Vt("", !0),
                      U(v).length
                        ? (st(),
                          te(
                            E,
                            { key: 3 },
                            {
                              summary: ee(() => [
                                tt("div", Xpt, " SKIP (" + Ut(U(v).length) + ") ", 1),
                              ]),
                              default: ee(() => [
                                (st(!0),
                                kt(
                                  ne,
                                  null,
                                  Rn(
                                    U(v),
                                    (K) => (
                                      st(),
                                      te(
                                        R,
                                        {
                                          key: K.id,
                                          task: K,
                                          nested: L.nested,
                                          search: U(o),
                                          class: ve(U(yr) === K.id ? "bg-active" : ""),
                                          "on-item-click": L.onItemClick,
                                        },
                                        null,
                                        8,
                                        ["task", "nested", "search", "class", "on-item-click"],
                                      )
                                    ),
                                  ),
                                  128,
                                )),
                              ]),
                              _: 1,
                            },
                          ))
                        : Vt("", !0),
                    ],
                    64,
                  ))
                : (st(!0),
                  kt(
                    ne,
                    { key: 1 },
                    Rn(
                      U(f),
                      (K) => (
                        st(),
                        te(
                          R,
                          {
                            key: K.id,
                            task: K,
                            nested: L.nested,
                            search: U(o),
                            class: ve(U(yr) === K.id ? "bg-active" : ""),
                            "on-item-click": L.onItemClick,
                          },
                          null,
                          8,
                          ["task", "nested", "search", "class", "on-item-click"],
                        )
                      ),
                    ),
                    128,
                  )),
              U(u) && U(f).length === 0
                ? (st(),
                  kt("div", Ypt, [
                    Zpt,
                    tt(
                      "button",
                      {
                        "font-light": "",
                        op: "50 hover:100",
                        "text-sm": "",
                        border: "~ gray-400/50 rounded",
                        p: "x2 y0.5",
                        m: "t2",
                        onClickPassive: T[4] || (T[4] = (K) => P(!0)),
                      },
                      " Clear ",
                      32,
                    ),
                  ]))
                : Vt("", !0),
            ]),
          ])
        );
      };
    },
  }),
  Ml = Zt(),
  ns = Zt(!0),
  ao = Zt(!1),
  xc = Zt(!0),
  jo = xt(() => {
    var t;
    return (t = Yh.value) == null ? void 0 : t.coverage;
  }),
  lh = xt(() => {
    var t;
    return (t = jo.value) == null ? void 0 : t.enabled;
  }),
  Vo = xt(() => lh.value && jo.value.reporter.map(([t]) => t).includes("html")),
  Jpt = xt(() => {
    if (Vo.value) {
      const t = jo.value.reportsDirectory.lastIndexOf("/"),
        e = jo.value.reporter.find((r) => {
          if (r[0] === "html") return r;
        });
      return e && "subdir" in e[1]
        ? `/${jo.value.reportsDirectory.slice(t + 1)}/${e[1].subdir}/index.html`
        : `/${jo.value.reportsDirectory.slice(t + 1)}/index.html`;
    }
  });
Re(
  kl,
  (t) => {
    xc.value = t === "running";
  },
  { immediate: !0 },
);
function Qpt() {
  const t = yr.value;
  if (t && t.length > 0) {
    const e = hc(t);
    e
      ? ((Ml.value = e), (ns.value = !1), (ao.value = !1))
      : Slt(
          () => je.state.getFiles(),
          () => {
            (Ml.value = hc(t)), (ns.value = !1), (ao.value = !1);
          },
        );
  }
  return ns;
}
function gf(t) {
  (ns.value = t), (ao.value = !1), t && ((Ml.value = void 0), (yr.value = ""));
}
function tgt() {
  (ao.value = !0), (ns.value = !1), (Ml.value = void 0), (yr.value = "");
}
const egt = { key: 0, "h-full": "" },
  ngt = { key: 0, "i-logos:typescript-icon": "", "flex-shrink-0": "", "mr-1": "" },
  rgt = {
    "data-testid": "filenames",
    "font-bold": "",
    "text-sm": "",
    "flex-auto": "",
    "ws-nowrap": "",
    "overflow-hidden": "",
    truncate: "",
  },
  igt = { class: "flex text-lg" },
  ogt = ie({
    __name: "Suites",
    setup(t) {
      const e = xt(() => {
          var u;
          return (u = Se.value) == null ? void 0 : u.name.split(/\//g).pop();
        }),
        r = xt(() => {
          var u, f;
          return (
            ((u = Se.value) == null ? void 0 : u.tasks) &&
            Wc((f = Se.value) == null ? void 0 : f.tasks)
          );
        });
      function o() {
        return Se.value && je.rpc.updateSnapshot(Se.value);
      }
      async function s() {
        Vo.value && ((xc.value = !0), await Br()), await aat();
      }
      return (u, f) => {
        const h = cd,
          d = bs,
          g = Xb,
          v = uo("tooltip");
        return U(Se)
          ? (st(),
            kt("div", egt, [
              Ft(
                g,
                { tasks: U(Se).tasks, nested: !0 },
                {
                  header: ee(() => [
                    Ft(h, { "mx-1": "", task: U(Se) }, null, 8, ["task"]),
                    U(Se).type === "suite" && U(Se).meta.typecheck
                      ? (st(), kt("div", ngt))
                      : Vt("", !0),
                    tt("span", rgt, Ut(U(e)), 1),
                    tt("div", igt, [
                      U(r) && !U(jr)
                        ? nn(
                            (st(),
                            te(
                              d,
                              {
                                key: 0,
                                icon: "i-carbon-result-old",
                                onClick: f[0] || (f[0] = (b) => o()),
                              },
                              null,
                              512,
                            )),
                            [
                              [
                                v,
                                `Update failed snapshot(s) of ${U(Se).name}`,
                                void 0,
                                { bottom: !0 },
                              ],
                            ],
                          )
                        : Vt("", !0),
                      U(jr)
                        ? Vt("", !0)
                        : nn(
                            (st(),
                            te(
                              d,
                              {
                                key: 1,
                                icon: "i-carbon-play",
                                onClick: f[1] || (f[1] = (b) => s()),
                              },
                              null,
                              512,
                            )),
                            [[v, "Rerun file", void 0, { bottom: !0 }]],
                          ),
                    ]),
                  ]),
                  _: 1,
                },
                8,
                ["tasks"],
              ),
            ]))
          : Vt("", !0);
      };
    },
  }),
  sgt = { h: "full", flex: "~ col" },
  lgt = tt(
    "div",
    { p: "3", "h-10": "", flex: "~ gap-2", "items-center": "", "bg-header": "", border: "b base" },
    [
      tt("div", { class: "i-carbon:folder-details-reference" }),
      tt(
        "span",
        {
          "pl-1": "",
          "font-bold": "",
          "text-sm": "",
          "flex-auto": "",
          "ws-nowrap": "",
          "overflow-hidden": "",
          truncate: "",
        },
        "Coverage",
      ),
    ],
    -1,
  ),
  agt = { "flex-auto": "", "py-1": "", "bg-white": "" },
  cgt = ["src"],
  ugt = ie({
    __name: "Coverage",
    props: { src: {} },
    setup(t) {
      return (e, r) => (
        st(),
        kt("div", sgt, [
          lgt,
          tt("div", agt, [tt("iframe", { id: "vitest-ui-coverage", src: e.src }, null, 8, cgt)]),
        ])
      );
    },
  }),
  fgt = { bg: "red500/10", "p-1": "", "mb-1": "", "mt-2": "", rounded: "" },
  hgt = { "font-bold": "" },
  dgt = {
    key: 0,
    class: "scrolls",
    text: "xs",
    "font-mono": "",
    "mx-1": "",
    "my-2": "",
    "pb-2": "",
    "overflow-auto": "",
  },
  pgt = ["font-bold"],
  ggt = { text: "red500/70" },
  vgt = tt("br", null, null, -1),
  mgt = { key: 1, text: "sm", "mb-2": "" },
  ygt = { "font-bold": "" },
  bgt = { key: 2, text: "sm", "mb-2": "" },
  wgt = { "font-bold": "" },
  xgt = tt("br", null, null, -1),
  _gt = tt(
    "ul",
    null,
    [
      tt("li", null, " The error was thrown, while Vitest was running this test. "),
      tt(
        "li",
        null,
        " If the error occurred after the test had been completed, this was the last documented test before it was thrown. ",
      ),
    ],
    -1,
  ),
  Sgt = { key: 3, text: "sm", "font-thin": "" },
  kgt = tt("br", null, null, -1),
  Cgt = tt(
    "ul",
    null,
    [
      tt("li", null, " Cancel timeouts using clearTimeout and clearInterval. "),
      tt("li", null, " Wait for promises to resolve using the await keyword. "),
    ],
    -1,
  ),
  Tgt = ie({
    __name: "ErrorEntry",
    props: { error: {} },
    setup(t) {
      return (e, r) => {
        var o;
        return (
          st(),
          kt(
            ne,
            null,
            [
              tt("h4", fgt, [
                tt("span", hgt, [
                  me(Ut(e.error.name || e.error.nameStr || "Unknown Error"), 1),
                  e.error.message ? (st(), kt(ne, { key: 0 }, [me(":")], 64)) : Vt("", !0),
                ]),
                me(" " + Ut(e.error.message), 1),
              ]),
              (o = e.error.stacks) != null && o.length
                ? (st(),
                  kt("p", dgt, [
                    (st(!0),
                    kt(
                      ne,
                      null,
                      Rn(
                        e.error.stacks,
                        (s, u) => (
                          st(),
                          kt(
                            "span",
                            { "whitespace-pre": "", "font-bold": u === 0 ? "" : null },
                            [
                              me("❯ " + Ut(s.method) + " " + Ut(s.file) + ":", 1),
                              tt("span", ggt, Ut(s.line) + ":" + Ut(s.column), 1),
                              vgt,
                            ],
                            8,
                            pgt,
                          )
                        ),
                      ),
                      256,
                    )),
                  ]))
                : Vt("", !0),
              e.error.VITEST_TEST_PATH
                ? (st(),
                  kt("p", mgt, [
                    me(" This error originated in "),
                    tt("span", ygt, Ut(e.error.VITEST_TEST_PATH), 1),
                    me(
                      " test file. It doesn't mean the error was thrown inside the file itself, but while it was running. ",
                    ),
                  ]))
                : Vt("", !0),
              e.error.VITEST_TEST_NAME
                ? (st(),
                  kt("p", bgt, [
                    me(" The latest test that might've caused the error is "),
                    tt("span", wgt, Ut(e.error.VITEST_TEST_NAME), 1),
                    me(". It might mean one of the following:"),
                    xgt,
                    _gt,
                  ]))
                : Vt("", !0),
              e.error.VITEST_AFTER_ENV_TEARDOWN
                ? (st(),
                  kt("p", Sgt, [
                    me(
                      " This error was caught after test environment was torn down. Make sure to cancel any running tasks before test finishes:",
                    ),
                    kgt,
                    Cgt,
                  ]))
                : Vt("", !0),
            ],
            64,
          )
        );
      };
    },
  }),
  bn = (t) => (Qm("data-v-09d153f7"), (t = t()), t0(), t),
  Egt = {
    "data-testid": "test-files-entry",
    grid: "~ cols-[min-content_1fr_min-content]",
    "items-center": "",
    gap: "x-2 y-3",
    p: "x4",
    relative: "",
    "font-light": "",
    "w-80": "",
    op80: "",
  },
  Lgt = bn(() => tt("div", { "i-carbon-document": "" }, null, -1)),
  Agt = bn(() => tt("div", null, "Files", -1)),
  Mgt = { class: "number", "data-testid": "num-files" },
  Ngt = bn(() => tt("div", { "i-carbon-checkmark": "" }, null, -1)),
  Pgt = bn(() => tt("div", null, "Pass", -1)),
  Ogt = { class: "number" },
  Dgt = bn(() => tt("div", { "i-carbon-close": "" }, null, -1)),
  $gt = bn(() => tt("div", null, " Fail ", -1)),
  Rgt = { class: "number", "text-red5": "" },
  zgt = bn(() => tt("div", { "i-carbon-compare": "" }, null, -1)),
  Igt = bn(() => tt("div", null, " Snapshot Fail ", -1)),
  Fgt = { class: "number", "text-red5": "" },
  qgt = bn(() => tt("div", { "i-carbon-checkmark-outline-error": "" }, null, -1)),
  Hgt = bn(() => tt("div", null, " Errors ", -1)),
  Bgt = { class: "number", "text-red5": "" },
  Wgt = bn(() => tt("div", { "i-carbon-timer": "" }, null, -1)),
  Ugt = bn(() => tt("div", null, "Time", -1)),
  jgt = { class: "number", "data-testid": "run-time" },
  Vgt = {
    key: 0,
    bg: "red500/10",
    text: "red500",
    p: "x3 y2",
    "max-w-xl": "",
    "m-2": "",
    rounded: "",
  },
  Ggt = bn(() => tt("h3", { "text-center": "", "mb-2": "" }, " Unhandled Errors ", -1)),
  Kgt = { text: "sm", "font-thin": "", "mb-2": "", "data-testid": "unhandled-errors" },
  Xgt = bn(() => tt("br", null, null, -1)),
  Ygt = {
    "data-testid": "unhandled-errors-details",
    class: "scrolls unhandled-errors",
    text: "sm",
    "font-thin": "",
    "pe-2.5": "",
    "open:max-h-52": "",
    "overflow-auto": "",
  },
  Zgt = bn(() => tt("summary", { "font-bold": "", "cursor-pointer": "" }, "Errors", -1)),
  Jgt = ie({
    __name: "TestFilesEntry",
    setup(t) {
      return (e, r) => {
        const o = Tgt;
        return (
          st(),
          kt(
            ne,
            null,
            [
              tt("div", Egt, [
                Lgt,
                Agt,
                tt("div", Mgt, Ut(U(mn).length), 1),
                U(pc).length
                  ? (st(), kt(ne, { key: 0 }, [Ngt, Pgt, tt("div", Ogt, Ut(U(pc).length), 1)], 64))
                  : Vt("", !0),
                U(dc).length
                  ? (st(), kt(ne, { key: 1 }, [Dgt, $gt, tt("div", Rgt, Ut(U(dc).length), 1)], 64))
                  : Vt("", !0),
                U(Xv).length
                  ? (st(), kt(ne, { key: 2 }, [zgt, Igt, tt("div", Fgt, Ut(U(Xv).length), 1)], 64))
                  : Vt("", !0),
                U(yi).length
                  ? (st(), kt(ne, { key: 3 }, [qgt, Hgt, tt("div", Bgt, Ut(U(yi).length), 1)], 64))
                  : Vt("", !0),
                Wgt,
                Ugt,
                tt("div", jgt, Ut(U(Mat)), 1),
              ]),
              U(yi).length
                ? (st(),
                  kt("div", Vgt, [
                    Ggt,
                    tt("p", Kgt, [
                      me(
                        " Vitest caught " +
                          Ut(U(yi).length) +
                          " error" +
                          Ut(U(yi).length > 1 ? "s" : "") +
                          " during the test run.",
                        1,
                      ),
                      Xgt,
                      me(
                        " This might cause false positive tests. Resolve unhandled errors to make sure your tests are not affected. ",
                      ),
                    ]),
                    tt("details", Ygt, [
                      Zgt,
                      (st(!0),
                      kt(
                        ne,
                        null,
                        Rn(U(yi), (s) => (st(), te(o, { error: s }, null, 8, ["error"]))),
                        256,
                      )),
                    ]),
                  ]))
                : Vt("", !0),
            ],
            64,
          )
        );
      };
    },
  }),
  Qgt = fo(Jgt, [["__scopeId", "data-v-09d153f7"]]),
  tvt = { "p-2": "", "text-center": "", flex: "" },
  evt = { "text-4xl": "", "min-w-2em": "" },
  nvt = { "text-md": "" },
  rvt = ie({
    __name: "DashboardEntry",
    props: { tail: { type: Boolean, default: !1 } },
    setup(t) {
      return (e, r) => (
        st(),
        kt("div", tvt, [
          tt("div", null, [
            tt("div", evt, [sr(e.$slots, "body")]),
            tt("div", nvt, [sr(e.$slots, "header")]),
          ]),
        ])
      );
    },
  }),
  ivt = { flex: "~ wrap", "justify-evenly": "", "gap-2": "", p: "x-4", relative: "" },
  ovt = ie({
    __name: "TestsEntry",
    setup(t) {
      const e = xt(() => Vc.value.length),
        r = xt(() => ob.value.length),
        o = xt(() => ib.value.length),
        s = xt(() => Lat.value.length),
        u = xt(() => Aat.value.length);
      return (f, h) => {
        const d = rvt;
        return (
          st(),
          kt("div", ivt, [
            Ft(
              d,
              { "text-green5": "", "data-testid": "pass-entry" },
              { header: ee(() => [me(" Pass ")]), body: ee(() => [me(Ut(U(r)), 1)]), _: 1 },
            ),
            Ft(
              d,
              { class: ve({ "text-red5": U(o), op50: !U(o) }), "data-testid": "fail-entry" },
              { header: ee(() => [me(" Fail ")]), body: ee(() => [me(Ut(U(o)), 1)]), _: 1 },
              8,
              ["class"],
            ),
            U(s)
              ? (st(),
                te(
                  d,
                  { key: 0, op50: "", "data-testid": "skipped-entry" },
                  { header: ee(() => [me(" Skip ")]), body: ee(() => [me(Ut(U(s)), 1)]), _: 1 },
                ))
              : Vt("", !0),
            U(u)
              ? (st(),
                te(
                  d,
                  { key: 1, op50: "", "data-testid": "todo-entry" },
                  { header: ee(() => [me(" Todo ")]), body: ee(() => [me(Ut(U(u)), 1)]), _: 1 },
                ))
              : Vt("", !0),
            Ft(
              d,
              { tail: !0, "data-testid": "total-entry" },
              { header: ee(() => [me(" Total ")]), body: ee(() => [me(Ut(U(e)), 1)]), _: 1 },
            ),
          ])
        );
      };
    },
  }),
  svt = {},
  lvt = {
    "gap-0": "",
    flex: "~ col gap-4",
    "h-full": "",
    "justify-center": "",
    "items-center": "",
  },
  avt = { "aria-labelledby": "tests", m: "y-4 x-2" };
function cvt(t, e) {
  const r = ovt,
    o = Qgt;
  return st(), kt("div", lvt, [tt("section", avt, [Ft(r)]), Ft(o)]);
}
const uvt = fo(svt, [["render", cvt]]),
  fvt = {},
  hvt = { h: "full", flex: "~ col" },
  dvt = tt(
    "div",
    { p: "3", "h-10": "", flex: "~ gap-2", "items-center": "", "bg-header": "", border: "b base" },
    [
      tt("div", { class: "i-carbon-dashboard" }),
      tt(
        "span",
        {
          "pl-1": "",
          "font-bold": "",
          "text-sm": "",
          "flex-auto": "",
          "ws-nowrap": "",
          "overflow-hidden": "",
          truncate: "",
        },
        "Dashboard",
      ),
    ],
    -1,
  ),
  pvt = { class: "scrolls", "flex-auto": "", "py-1": "" };
function gvt(t, e) {
  const r = uvt;
  return st(), kt("div", hvt, [dvt, tt("div", pvt, [Ft(r)])]);
}
const vvt = fo(fvt, [["render", gvt]]),
  mvt = "" + new URL("../favicon.svg", import.meta.url).href,
  yvt = tt("img", { "w-6": "", "h-6": "", src: mvt, alt: "Vitest logo" }, null, -1),
  bvt = tt("span", { "font-light": "", "text-sm": "", "flex-1": "" }, "Vitest", -1),
  wvt = { class: "flex text-lg" },
  xvt = tt("div", { class: "i-carbon:folder-off ma" }, null, -1),
  _vt = tt(
    "div",
    { class: "op100 gap-1 p-y-1", grid: "~ items-center cols-[1.5em_1fr]" },
    [
      tt("div", { class: "i-carbon:information-square w-1.5em h-1.5em" }),
      tt("div", null, "Coverage enabled but missing html reporter."),
      tt(
        "div",
        { style: { "grid-column": "2" } },
        " Add html reporter to your configuration to see coverage here. ",
      ),
    ],
    -1,
  ),
  Svt = ie({
    __name: "Navigation",
    setup(t) {
      const e = xt(() => mn.value && Wc(mn.value));
      function r() {
        return je.rpc.updateSnapshot();
      }
      const o = xt(() => (zl.value ? "light" : "dark"));
      function s(f) {
        (yr.value = f.id), (Ml.value = hc(f.id)), gf(!1);
      }
      async function u(f) {
        Vo.value && ((xc.value = !0), await Br(), ao.value && (gf(!0), await Br())), await lat(f);
      }
      return (f, h) => {
        const d = bs,
          g = Xb,
          v = uo("tooltip");
        return (
          st(),
          te(
            g,
            { border: "r base", tasks: U(mn), "on-item-click": s, "group-by-type": !0, onRun: u },
            {
              header: ee(({ filteredTests: b }) => [
                yvt,
                bvt,
                tt("div", wvt, [
                  nn(
                    Ft(
                      d,
                      {
                        title: "Show dashboard",
                        class: "!animate-100ms",
                        "animate-count-1": "",
                        icon: "i-carbon:dashboard",
                        onClick: h[0] || (h[0] = (w) => U(gf)(!0)),
                      },
                      null,
                      512,
                    ),
                    [
                      [Pf, (U(lh) && !U(Vo)) || !U(ns)],
                      [v, "Dashboard", void 0, { bottom: !0 }],
                    ],
                  ),
                  U(lh) && !U(Vo)
                    ? (st(),
                      te(
                        U(VC),
                        {
                          key: 0,
                          title: "Coverage enabled but missing html reporter",
                          class:
                            "w-1.4em h-1.4em op100 rounded flex color-red5 dark:color-#f43f5e cursor-help",
                        },
                        { popper: ee(() => [_vt]), default: ee(() => [xvt]), _: 1 },
                      ))
                    : Vt("", !0),
                  U(Vo)
                    ? nn(
                        (st(),
                        te(
                          d,
                          {
                            key: 1,
                            disabled: U(xc),
                            title: "Show coverage",
                            class: "!animate-100ms",
                            "animate-count-1": "",
                            icon: "i-carbon:folder-details-reference",
                            onClick: h[1] || (h[1] = (w) => U(tgt)()),
                          },
                          null,
                          8,
                          ["disabled"],
                        )),
                        [
                          [Pf, !U(ao)],
                          [v, "Coverage", void 0, { bottom: !0 }],
                        ],
                      )
                    : Vt("", !0),
                  U(e) && !U(jr)
                    ? nn(
                        (st(),
                        te(
                          d,
                          {
                            key: 2,
                            icon: "i-carbon:result-old",
                            onClick: h[2] || (h[2] = (w) => r()),
                          },
                          null,
                          512,
                        )),
                        [[v, "Update all failed snapshot(s)", void 0, { bottom: !0 }]],
                      )
                    : Vt("", !0),
                  U(jr)
                    ? Vt("", !0)
                    : nn(
                        (st(),
                        te(
                          d,
                          {
                            key: 3,
                            disabled: (b == null ? void 0 : b.length) === 0,
                            icon: "i-carbon:play",
                            onClick: (w) => u(b),
                          },
                          null,
                          8,
                          ["disabled", "onClick"],
                        )),
                        [
                          [
                            v,
                            b
                              ? b.length === 0
                                ? "No test to run (clear filter)"
                                : "Rerun filtered"
                              : "Rerun all",
                            void 0,
                            { bottom: !0 },
                          ],
                        ],
                      ),
                  nn(
                    Ft(
                      d,
                      {
                        icon: "dark:i-carbon-moon i-carbon:sun",
                        onClick: h[3] || (h[3] = (w) => U(hat)()),
                      },
                      null,
                      512,
                    ),
                    [[v, `Toggle to ${U(o)} mode`, void 0, { bottom: !0 }]],
                  ),
                ]),
              ]),
              _: 1,
            },
            8,
            ["tasks"],
          )
        );
      };
    },
  }),
  kvt = { "h-3px": "", relative: "", "overflow-hidden": "", class: "px-0", "w-screen": "" },
  Cvt = ie({
    __name: "ProgressBar",
    setup(t) {
      const { width: e } = Ilt(),
        r = xt(() =>
          mn.value.length === 0
            ? "!bg-gray-4 !dark:bg-gray-7 in-progress"
            : Eat.value
            ? null
            : "in-progress",
        ),
        o = xt(() => mn.value.length),
        s = xt(() => pc.value.length),
        u = xt(() => dc.value.length),
        f = xt(() => {
          const v = U(o);
          return v > 0 ? (e.value * s.value) / v : 0;
        }),
        h = xt(() => {
          const v = U(o);
          return v > 0 ? (e.value * u.value) / v : 0;
        }),
        d = xt(() => U(o) - u.value - s.value),
        g = xt(() => {
          const v = U(o);
          return v > 0 ? (e.value * d.value) / v : 0;
        });
      return (v, b) => (
        st(),
        kt(
          "div",
          {
            absolute: "",
            "t-0": "",
            "l-0": "",
            "r-0": "",
            "z-index-1031": "",
            "pointer-events-none": "",
            "p-0": "",
            "h-3px": "",
            grid: "~ auto-cols-max",
            "justify-items-center": "",
            "w-screen": "",
            class: ve(U(r)),
          },
          [
            tt("div", kvt, [
              tt(
                "div",
                {
                  absolute: "",
                  "l-0": "",
                  "t-0": "",
                  "bg-red5": "",
                  "h-3px": "",
                  class: ve(U(r)),
                  style: An(`width: ${U(h)}px;`),
                },
                "   ",
                6,
              ),
              tt(
                "div",
                {
                  absolute: "",
                  "l-0": "",
                  "t-0": "",
                  "bg-green5": "",
                  "h-3px": "",
                  class: ve(U(r)),
                  style: An(`left: ${U(h)}px; width: ${U(f)}px;`),
                },
                "   ",
                6,
              ),
              tt(
                "div",
                {
                  absolute: "",
                  "l-0": "",
                  "t-0": "",
                  "bg-yellow5": "",
                  "h-3px": "",
                  class: ve(U(r)),
                  style: An(`left: ${U(f) + U(h)}px; width: ${U(g)}px;`),
                },
                "   ",
                6,
              ),
            ]),
          ],
          2,
        )
      );
    },
  }),
  Tvt = fo(Cvt, [["__scopeId", "data-v-f967c1fe"]]),
  km = {
    name: "splitpanes",
    emits: [
      "ready",
      "resize",
      "resized",
      "pane-click",
      "pane-maximize",
      "pane-add",
      "pane-remove",
      "splitter-click",
    ],
    props: {
      horizontal: { type: Boolean },
      pushOtherPanes: { type: Boolean, default: !0 },
      dblClickSplitter: { type: Boolean, default: !0 },
      rtl: { type: Boolean, default: !1 },
      firstSplitter: { type: Boolean },
    },
    provide() {
      return {
        requestUpdate: this.requestUpdate,
        onPaneAdd: this.onPaneAdd,
        onPaneRemove: this.onPaneRemove,
        onPaneClick: this.onPaneClick,
      };
    },
    data: () => ({
      container: null,
      ready: !1,
      panes: [],
      touch: { mouseDown: !1, dragging: !1, activeSplitter: null },
      splitterTaps: { splitter: null, timeoutId: null },
    }),
    computed: {
      panesCount() {
        return this.panes.length;
      },
      indexedPanes() {
        return this.panes.reduce((t, e) => (t[e.id] = e) && t, {});
      },
    },
    methods: {
      updatePaneComponents() {
        this.panes.forEach((t) => {
          t.update &&
            t.update({
              [this.horizontal ? "height" : "width"]: `${this.indexedPanes[t.id].size}%`,
            });
        });
      },
      bindEvents() {
        document.addEventListener("mousemove", this.onMouseMove, { passive: !1 }),
          document.addEventListener("mouseup", this.onMouseUp),
          "ontouchstart" in window &&
            (document.addEventListener("touchmove", this.onMouseMove, { passive: !1 }),
            document.addEventListener("touchend", this.onMouseUp));
      },
      unbindEvents() {
        document.removeEventListener("mousemove", this.onMouseMove, { passive: !1 }),
          document.removeEventListener("mouseup", this.onMouseUp),
          "ontouchstart" in window &&
            (document.removeEventListener("touchmove", this.onMouseMove, { passive: !1 }),
            document.removeEventListener("touchend", this.onMouseUp));
      },
      onMouseDown(t, e) {
        this.bindEvents(), (this.touch.mouseDown = !0), (this.touch.activeSplitter = e);
      },
      onMouseMove(t) {
        this.touch.mouseDown &&
          (t.preventDefault(),
          (this.touch.dragging = !0),
          this.calculatePanesSize(this.getCurrentMouseDrag(t)),
          this.$emit(
            "resize",
            this.panes.map((e) => ({ min: e.min, max: e.max, size: e.size })),
          ));
      },
      onMouseUp() {
        this.touch.dragging &&
          this.$emit(
            "resized",
            this.panes.map((t) => ({ min: t.min, max: t.max, size: t.size })),
          ),
          (this.touch.mouseDown = !1),
          setTimeout(() => {
            (this.touch.dragging = !1), this.unbindEvents();
          }, 100);
      },
      onSplitterClick(t, e) {
        "ontouchstart" in window &&
          (t.preventDefault(),
          this.dblClickSplitter &&
            (this.splitterTaps.splitter === e
              ? (clearTimeout(this.splitterTaps.timeoutId),
                (this.splitterTaps.timeoutId = null),
                this.onSplitterDblClick(t, e),
                (this.splitterTaps.splitter = null))
              : ((this.splitterTaps.splitter = e),
                (this.splitterTaps.timeoutId = setTimeout(() => {
                  this.splitterTaps.splitter = null;
                }, 500))))),
          this.touch.dragging || this.$emit("splitter-click", this.panes[e]);
      },
      onSplitterDblClick(t, e) {
        let r = 0;
        (this.panes = this.panes.map(
          (o, s) => ((o.size = s === e ? o.max : o.min), s !== e && (r += o.min), o),
        )),
          (this.panes[e].size -= r),
          this.$emit("pane-maximize", this.panes[e]),
          this.$emit(
            "resized",
            this.panes.map((o) => ({ min: o.min, max: o.max, size: o.size })),
          );
      },
      onPaneClick(t, e) {
        this.$emit("pane-click", this.indexedPanes[e]);
      },
      getCurrentMouseDrag(t) {
        const e = this.container.getBoundingClientRect(),
          { clientX: r, clientY: o } = "ontouchstart" in window && t.touches ? t.touches[0] : t;
        return { x: r - e.left, y: o - e.top };
      },
      getCurrentDragPercentage(t) {
        t = t[this.horizontal ? "y" : "x"];
        const e = this.container[this.horizontal ? "clientHeight" : "clientWidth"];
        return this.rtl && !this.horizontal && (t = e - t), (t * 100) / e;
      },
      calculatePanesSize(t) {
        const e = this.touch.activeSplitter;
        let r = {
          prevPanesSize: this.sumPrevPanesSize(e),
          nextPanesSize: this.sumNextPanesSize(e),
          prevReachedMinPanes: 0,
          nextReachedMinPanes: 0,
        };
        const o = 0 + (this.pushOtherPanes ? 0 : r.prevPanesSize),
          s = 100 - (this.pushOtherPanes ? 0 : r.nextPanesSize),
          u = Math.max(Math.min(this.getCurrentDragPercentage(t), s), o);
        let f = [e, e + 1],
          h = this.panes[f[0]] || null,
          d = this.panes[f[1]] || null;
        const g = h.max < 100 && u >= h.max + r.prevPanesSize,
          v = d.max < 100 && u <= 100 - (d.max + this.sumNextPanesSize(e + 1));
        if (g || v) {
          g
            ? ((h.size = h.max),
              (d.size = Math.max(100 - h.max - r.prevPanesSize - r.nextPanesSize, 0)))
            : ((h.size = Math.max(100 - d.max - r.prevPanesSize - this.sumNextPanesSize(e + 1), 0)),
              (d.size = d.max));
          return;
        }
        if (this.pushOtherPanes) {
          const b = this.doPushOtherPanes(r, u);
          if (!b) return;
          ({ sums: r, panesToResize: f } = b),
            (h = this.panes[f[0]] || null),
            (d = this.panes[f[1]] || null);
        }
        h !== null &&
          (h.size = Math.min(Math.max(u - r.prevPanesSize - r.prevReachedMinPanes, h.min), h.max)),
          d !== null &&
            (d.size = Math.min(
              Math.max(100 - u - r.nextPanesSize - r.nextReachedMinPanes, d.min),
              d.max,
            ));
      },
      doPushOtherPanes(t, e) {
        const r = this.touch.activeSplitter,
          o = [r, r + 1];
        return e < t.prevPanesSize + this.panes[o[0]].min &&
          ((o[0] = this.findPrevExpandedPane(r).index),
          (t.prevReachedMinPanes = 0),
          o[0] < r &&
            this.panes.forEach((s, u) => {
              u > o[0] && u <= r && ((s.size = s.min), (t.prevReachedMinPanes += s.min));
            }),
          (t.prevPanesSize = this.sumPrevPanesSize(o[0])),
          o[0] === void 0)
          ? ((t.prevReachedMinPanes = 0),
            (this.panes[0].size = this.panes[0].min),
            this.panes.forEach((s, u) => {
              u > 0 && u <= r && ((s.size = s.min), (t.prevReachedMinPanes += s.min));
            }),
            (this.panes[o[1]].size =
              100 - t.prevReachedMinPanes - this.panes[0].min - t.prevPanesSize - t.nextPanesSize),
            null)
          : e > 100 - t.nextPanesSize - this.panes[o[1]].min &&
            ((o[1] = this.findNextExpandedPane(r).index),
            (t.nextReachedMinPanes = 0),
            o[1] > r + 1 &&
              this.panes.forEach((s, u) => {
                u > r && u < o[1] && ((s.size = s.min), (t.nextReachedMinPanes += s.min));
              }),
            (t.nextPanesSize = this.sumNextPanesSize(o[1] - 1)),
            o[1] === void 0)
          ? ((t.nextReachedMinPanes = 0),
            (this.panes[this.panesCount - 1].size = this.panes[this.panesCount - 1].min),
            this.panes.forEach((s, u) => {
              u < this.panesCount - 1 &&
                u >= r + 1 &&
                ((s.size = s.min), (t.nextReachedMinPanes += s.min));
            }),
            (this.panes[o[0]].size =
              100 -
              t.prevPanesSize -
              t.nextReachedMinPanes -
              this.panes[this.panesCount - 1].min -
              t.nextPanesSize),
            null)
          : { sums: t, panesToResize: o };
      },
      sumPrevPanesSize(t) {
        return this.panes.reduce((e, r, o) => e + (o < t ? r.size : 0), 0);
      },
      sumNextPanesSize(t) {
        return this.panes.reduce((e, r, o) => e + (o > t + 1 ? r.size : 0), 0);
      },
      findPrevExpandedPane(t) {
        return [...this.panes].reverse().find((e) => e.index < t && e.size > e.min) || {};
      },
      findNextExpandedPane(t) {
        return this.panes.find((e) => e.index > t + 1 && e.size > e.min) || {};
      },
      checkSplitpanesNodes() {
        Array.from(this.container.children).forEach((t) => {
          const e = t.classList.contains("splitpanes__pane"),
            r = t.classList.contains("splitpanes__splitter");
          !e &&
            !r &&
            (t.parentNode.removeChild(t),
            console.warn(
              "Splitpanes: Only <pane> elements are allowed at the root of <splitpanes>. One of your DOM nodes was removed.",
            ));
        });
      },
      addSplitter(t, e, r = !1) {
        const o = t - 1,
          s = document.createElement("div");
        s.classList.add("splitpanes__splitter"),
          r ||
            ((s.onmousedown = (u) => this.onMouseDown(u, o)),
            typeof window < "u" &&
              "ontouchstart" in window &&
              (s.ontouchstart = (u) => this.onMouseDown(u, o)),
            (s.onclick = (u) => this.onSplitterClick(u, o + 1))),
          this.dblClickSplitter && (s.ondblclick = (u) => this.onSplitterDblClick(u, o + 1)),
          e.parentNode.insertBefore(s, e);
      },
      removeSplitter(t) {
        (t.onmousedown = void 0),
          (t.onclick = void 0),
          (t.ondblclick = void 0),
          t.parentNode.removeChild(t);
      },
      redoSplitters() {
        const t = Array.from(this.container.children);
        t.forEach((r) => {
          r.className.includes("splitpanes__splitter") && this.removeSplitter(r);
        });
        let e = 0;
        t.forEach((r) => {
          r.className.includes("splitpanes__pane") &&
            (!e && this.firstSplitter ? this.addSplitter(e, r, !0) : e && this.addSplitter(e, r),
            e++);
        });
      },
      requestUpdate({ target: t, ...e }) {
        const r = this.indexedPanes[t._.uid];
        Object.entries(e).forEach(([o, s]) => (r[o] = s));
      },
      onPaneAdd(t) {
        let e = -1;
        Array.from(t.$el.parentNode.children).some(
          (s) => (s.className.includes("splitpanes__pane") && e++, s === t.$el),
        );
        const r = parseFloat(t.minSize),
          o = parseFloat(t.maxSize);
        this.panes.splice(e, 0, {
          id: t._.uid,
          index: e,
          min: isNaN(r) ? 0 : r,
          max: isNaN(o) ? 100 : o,
          size: t.size === null ? null : parseFloat(t.size),
          givenSize: t.size,
          update: t.update,
        }),
          this.panes.forEach((s, u) => (s.index = u)),
          this.ready &&
            this.$nextTick(() => {
              this.redoSplitters(),
                this.resetPaneSizes({ addedPane: this.panes[e] }),
                this.$emit("pane-add", {
                  index: e,
                  panes: this.panes.map((s) => ({ min: s.min, max: s.max, size: s.size })),
                });
            });
      },
      onPaneRemove(t) {
        const e = this.panes.findIndex((o) => o.id === t._.uid),
          r = this.panes.splice(e, 1)[0];
        this.panes.forEach((o, s) => (o.index = s)),
          this.$nextTick(() => {
            this.redoSplitters(),
              this.resetPaneSizes({ removedPane: { ...r, index: e } }),
              this.$emit("pane-remove", {
                removed: r,
                panes: this.panes.map((o) => ({ min: o.min, max: o.max, size: o.size })),
              });
          });
      },
      resetPaneSizes(t = {}) {
        !t.addedPane && !t.removedPane
          ? this.initialPanesSizing()
          : this.panes.some((e) => e.givenSize !== null || e.min || e.max < 100)
          ? this.equalizeAfterAddOrRemove(t)
          : this.equalize(),
          this.ready &&
            this.$emit(
              "resized",
              this.panes.map((e) => ({ min: e.min, max: e.max, size: e.size })),
            );
      },
      equalize() {
        const t = 100 / this.panesCount;
        let e = 0;
        const r = [],
          o = [];
        this.panes.forEach((s) => {
          (s.size = Math.max(Math.min(t, s.max), s.min)),
            (e -= s.size),
            s.size >= s.max && r.push(s.id),
            s.size <= s.min && o.push(s.id);
        }),
          e > 0.1 && this.readjustSizes(e, r, o);
      },
      initialPanesSizing() {
        let t = 100;
        const e = [],
          r = [];
        let o = 0;
        this.panes.forEach((u) => {
          (t -= u.size),
            u.size !== null && o++,
            u.size >= u.max && e.push(u.id),
            u.size <= u.min && r.push(u.id);
        });
        let s = 100;
        t > 0.1 &&
          (this.panes.forEach((u) => {
            u.size === null &&
              (u.size = Math.max(Math.min(t / (this.panesCount - o), u.max), u.min)),
              (s -= u.size);
          }),
          s > 0.1 && this.readjustSizes(t, e, r));
      },
      equalizeAfterAddOrRemove({ addedPane: t, removedPane: e } = {}) {
        let r = 100 / this.panesCount,
          o = 0;
        const s = [],
          u = [];
        t && t.givenSize !== null && (r = (100 - t.givenSize) / (this.panesCount - 1)),
          this.panes.forEach((f) => {
            (o -= f.size), f.size >= f.max && s.push(f.id), f.size <= f.min && u.push(f.id);
          }),
          !(Math.abs(o) < 0.1) &&
            (this.panes.forEach((f) => {
              (t && t.givenSize !== null && t.id === f.id) ||
                (f.size = Math.max(Math.min(r, f.max), f.min)),
                (o -= f.size),
                f.size >= f.max && s.push(f.id),
                f.size <= f.min && u.push(f.id);
            }),
            o > 0.1 && this.readjustSizes(o, s, u));
      },
      readjustSizes(t, e, r) {
        let o;
        t > 0 ? (o = t / (this.panesCount - e.length)) : (o = t / (this.panesCount - r.length)),
          this.panes.forEach((s, u) => {
            if (t > 0 && !e.includes(s.id)) {
              const f = Math.max(Math.min(s.size + o, s.max), s.min),
                h = f - s.size;
              (t -= h), (s.size = f);
            } else if (!r.includes(s.id)) {
              const f = Math.max(Math.min(s.size + o, s.max), s.min),
                h = f - s.size;
              (t -= h), (s.size = f);
            }
            s.update({
              [this.horizontal ? "height" : "width"]: `${this.indexedPanes[s.id].size}%`,
            });
          }),
          Math.abs(t) > 0.1 &&
            this.$nextTick(() => {
              this.ready &&
                console.warn(
                  "Splitpanes: Could not resize panes correctly due to their constraints.",
                );
            });
      },
    },
    watch: {
      panes: {
        deep: !0,
        immediate: !1,
        handler() {
          this.updatePaneComponents();
        },
      },
      horizontal() {
        this.updatePaneComponents();
      },
      firstSplitter() {
        this.redoSplitters();
      },
      dblClickSplitter(t) {
        [...this.container.querySelectorAll(".splitpanes__splitter")].forEach((e, r) => {
          e.ondblclick = t ? (o) => this.onSplitterDblClick(o, r) : void 0;
        });
      },
    },
    beforeUnmount() {
      this.ready = !1;
    },
    mounted() {
      (this.container = this.$refs.container),
        this.checkSplitpanesNodes(),
        this.redoSplitters(),
        this.resetPaneSizes(),
        this.$emit("ready"),
        (this.ready = !0);
    },
    render() {
      return Ol(
        "div",
        {
          ref: "container",
          class: [
            "splitpanes",
            `splitpanes--${this.horizontal ? "horizontal" : "vertical"}`,
            { "splitpanes--dragging": this.touch.dragging },
          ],
        },
        this.$slots.default(),
      );
    },
  },
  Evt = (t, e) => {
    const r = t.__vccOpts || t;
    for (const [o, s] of e) r[o] = s;
    return r;
  },
  Lvt = {
    name: "pane",
    inject: ["requestUpdate", "onPaneAdd", "onPaneRemove", "onPaneClick"],
    props: {
      size: { type: [Number, String], default: null },
      minSize: { type: [Number, String], default: 0 },
      maxSize: { type: [Number, String], default: 100 },
    },
    data: () => ({ style: {} }),
    mounted() {
      this.onPaneAdd(this);
    },
    beforeUnmount() {
      this.onPaneRemove(this);
    },
    methods: {
      update(t) {
        this.style = t;
      },
    },
    computed: {
      sizeNumber() {
        return this.size || this.size === 0 ? parseFloat(this.size) : null;
      },
      minSizeNumber() {
        return parseFloat(this.minSize);
      },
      maxSizeNumber() {
        return parseFloat(this.maxSize);
      },
    },
    watch: {
      sizeNumber(t) {
        this.requestUpdate({ target: this, size: t });
      },
      minSizeNumber(t) {
        this.requestUpdate({ target: this, min: t });
      },
      maxSizeNumber(t) {
        this.requestUpdate({ target: this, max: t });
      },
    },
  };
function Avt(t, e, r, o, s, u) {
  return (
    st(),
    kt(
      "div",
      {
        class: "splitpanes__pane",
        onClick: e[0] || (e[0] = (f) => u.onPaneClick(f, t._.uid)),
        style: An(t.style),
      },
      [sr(t.$slots, "default")],
      4,
    )
  );
}
const za = Evt(Lvt, [["render", Avt]]),
  Mvt = { "h-screen": "", "w-screen": "", overflow: "hidden" },
  Nvt = ie({
    __name: "index",
    setup(t) {
      const e = Qpt(),
        r = Un([33, 67]),
        o = Un([33, 67]),
        s = Gv((h) => {
          h.forEach((d, g) => {
            r[g] = d.size;
          });
        }, 0),
        u = Gv((h) => {
          h.forEach((d, g) => {
            o[g] = d.size;
          });
        }, 0);
      function f() {
        const h = window.innerWidth,
          d = Math.min(h / 3, 300);
        (r[0] = (100 * d) / h),
          (r[1] = 100 - r[0]),
          (o[0] = (100 * d) / (h - d)),
          (o[1] = 100 - o[0]);
      }
      return (h, d) => {
        const g = Tvt,
          v = Svt,
          b = vvt,
          w = ugt,
          S = ogt,
          P = Tpt,
          A = fat;
        return (
          st(),
          kt(
            ne,
            null,
            [
              Ft(g),
              tt("div", Mvt, [
                Ft(
                  U(km),
                  { class: "pt-4px", onResized: U(s), onReady: f },
                  {
                    default: ee(() => [
                      Ft(U(za), { size: U(r)[0] }, { default: ee(() => [Ft(v)]), _: 1 }, 8, [
                        "size",
                      ]),
                      Ft(
                        U(za),
                        { size: U(r)[1] },
                        {
                          default: ee(() => [
                            Ft(Oh, null, {
                              default: ee(() => [
                                U(e)
                                  ? (st(), te(b, { key: "summary" }))
                                  : U(ao)
                                  ? (st(),
                                    te(w, { key: "coverage", src: U(Jpt) }, null, 8, ["src"]))
                                  : (st(),
                                    te(
                                      U(km),
                                      { key: "detail", onResized: U(u) },
                                      {
                                        default: ee(() => [
                                          Ft(
                                            U(za),
                                            { size: U(o)[0] },
                                            { default: ee(() => [Ft(S)]), _: 1 },
                                            8,
                                            ["size"],
                                          ),
                                          Ft(
                                            U(za),
                                            { size: U(o)[1] },
                                            { default: ee(() => [Ft(P)]), _: 1 },
                                            8,
                                            ["size"],
                                          ),
                                        ]),
                                        _: 1,
                                      },
                                      8,
                                      ["onResized"],
                                    )),
                              ]),
                              _: 1,
                            }),
                          ]),
                          _: 1,
                        },
                        8,
                        ["size"],
                      ),
                    ]),
                    _: 1,
                  },
                  8,
                  ["onResized"],
                ),
              ]),
              Ft(A),
            ],
            64,
          )
        );
      };
    },
  }),
  Pvt = [{ name: "index", path: "/", component: Nvt, props: !0 }],
  Ovt = { tooltip: jC };
sy.options.instantMove = !0;
sy.options.distance = 10;
function Dvt() {
  return Dk({ history: YS(), routes: Pvt });
}
const $vt = [Dvt],
  ud = T0(NS);
$vt.forEach((t) => {
  ud.use(t());
});
Object.entries(Ovt).forEach(([t, e]) => {
  ud.directive(t, e);
});
ud.mount("#app");
