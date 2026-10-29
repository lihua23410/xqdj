const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const source = fs.readFileSync(__dirname + '/app.js', 'utf8');
const sandbox = {};
vm.createContext(sandbox);
vm.runInContext(source.slice(source.indexOf('function screenPos('), source.indexOf('function dmgPop(')), sandbox);

test('shared health drawing rounds current life without showing zero for a living unit', () => {
  for (const [hp, expected] of [[73.4, '73'], [73.5, '74'], [0.4, '1'], [0, '0'], [-2, '0']]) {
    const el = { classList: new Set(), style: {} };
    sandbox.drawHealthNumber(el, { x: 12, y: -8, radius: 9, hp }, { scale: 2, cx: 100, cy: 200 });
    assert.equal(el.textContent, expected);
    assert.equal(el.classList.has('hp-float'), true);
    assert.equal(el.style.left, '124px');
    assert.equal(el.style.top, '194px');
  }
});

test('a private display can choose the life source while keeping the common position and format', () => {
  const el = { classList: new Set(['guide']), style: {} };
  sandbox.drawHealthNumber(el, { x: -10, y: 20, radius: 18, hp: 9999 }, { scale: 1, cx: 300, cy: 200 }, 41.4);
  assert.equal(el.textContent, '41');
  assert.deepEqual([...el.classList], ['guide', 'hp-float']);
  assert.equal(el.style.left, '290px');
  assert.equal(el.style.top, '158px');
});
