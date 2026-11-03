// 只跑 character/fujihime/fx/fujihime.js 里的 lookFX 挂件，用最小 DOM 桩。
// 文件名带下划线：包文件列表会跳过它，不会发到页面上。
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

class FakeEl {
  constructor(tag) {
    this.tag = tag;
    this.className = '';
    this.children = [];
    this.textContent = '';
    this.style = { props: {}, setProperty(k, v) { this.props[k] = v; } };
  }
  get firstElementChild() { return this.children[0] || null; }
  get lastElementChild() { return this.children[this.children.length - 1] || null; }
  append(...kids) { for (const k of kids) this.appendChild(k); }
  appendChild(kid) { kid.parent = this; this.children.push(kid); return kid; }
  remove() {
    this.removed = true;
    if (this.parent) {
      const i = this.parent.children.indexOf(this);
      if (i >= 0) this.parent.children.splice(i, 1);
      this.parent = null;
    }
  }
  querySelector(sel) {
    if (!sel.startsWith(".")) return null;
    const want = sel.slice(1);
    return this.children.find((c) => c.className.split(/\s+/).includes(want)) || null;
  }
}

function load() {
  const sandbox = {
    window: {},
    arena: { registerShot() {}, spawnFx() { return null; } },
    document: { createElement: (tag) => new FakeEl(tag) },
  };
  vm.createContext(sandbox);
  vm.runInContext(fs.readFileSync(__dirname + '/fujihime.js', 'utf8'), sandbox);
  return sandbox;
}

test('藤色池子画成球下面的一条，本体和幻役共用同一套挂件', () => {
  const sandbox = load();
  const hook = sandbox.window.lookFX.fujihime;
  assert.ok(hook && typeof hook.tick === 'function');
  assert.equal(sandbox.window.lookFX['fujihime-doll'], hook);

  const el = new FakeEl('div');
  hook.tick(el, { absorb: 40 });
  const bar = el.children[0];
  assert.ok(bar, '没有建出额外血条');
  assert.equal(bar.className, 'fujihime-shield');
  assert.equal(bar.children[0].style.props['--fill'], '40.0%');
  assert.equal(bar.children[1].textContent, '40');

  hook.tick(el, { absorb: 240 });
  assert.equal(el.children.length, 1, '同一只单位复用同一个节点');
  assert.equal(el.children[0].children[0].style.props['--fill'], '100.0%');
  assert.equal(el.children[0].children[1].textContent, '240');

  hook.tick(el, { absorb: 0 });
  assert.equal(el.children.length, 0, '池子空了该把条撤掉');

  hook.tick(el, { absorb: 9 });
  assert.equal(el.children[0].children[0].style.props['--fill'], '9.0%');
  hook.unmount(el);
  assert.equal(el.children.length, 0, '单位拆掉时该清干净挂件');
});

test('池子剩一点点也画得出来，读数最低 1', () => {
  const { window: w } = load();
  const el = new FakeEl('div');
  w.lookFX.fujihime.tick(el, { absorb: 0.4 });
  assert.equal(el.children[0].children[0].style.props['--fill'], '6.0%');
  assert.equal(el.children[0].children[1].textContent, '1');
});
