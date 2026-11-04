const { readFileSync } = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')
const ts = require('typescript')
function deferred() {
  let resolve, reject
  const promise = new Promise((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}
function loadSource(file, globals = {}, importModule = require) {
  const source = readFileSync(path.join(__dirname, '..', file), 'utf8').replaceAll('import.meta.client', 'true').replaceAll('import.meta.dev', 'true')
  const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
  const module = { exports: {} }
  vm.runInNewContext(compiled, { ...globals, module, exports: module.exports, require: importModule, console, URL, setTimeout, clearTimeout }, { filename: file })
  return module.exports
}
module.exports = { deferred, loadSource }
