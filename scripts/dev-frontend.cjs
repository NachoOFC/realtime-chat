const { existsSync } = require('node:fs')
const { spawn } = require('node:child_process')
const path = require('node:path')
const frontend = path.join(__dirname, '..', 'frontend')
const ca = path.join(__dirname, '..', '.certs', 'ca.crt')
const cli = path.join(path.dirname(require.resolve('nuxt/package.json', { paths: [frontend] })), 'bin', 'nuxt.mjs')
const env = { ...process.env }
if (!env.NODE_EXTRA_CA_CERTS && existsSync(ca)) env.NODE_EXTRA_CA_CERTS = ca
const child = spawn(process.execPath, [cli, 'dev', ...process.argv.slice(2)], { cwd: frontend, env, stdio: 'inherit' })
child.on('error', error => { console.error(error.message); process.exitCode = 1 })
child.on('exit', code => process.exit(code ?? 1))
