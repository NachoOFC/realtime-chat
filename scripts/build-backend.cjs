const { mkdirSync } = require('node:fs')
const path = require('node:path')
const { spawnSync } = require('node:child_process')

const backendPath = path.join(__dirname, '..', 'backend')
const outputPath = path.join(backendPath, 'bin', process.platform === 'win32' ? 'server.exe' : 'server')
mkdirSync(path.dirname(outputPath), { recursive: true })

const result = spawnSync('go', ['build', '-o', outputPath, './cmd/server'], {
  cwd: backendPath,
  stdio: 'inherit',
})

if (result.error) {
  console.error('No se pudo ejecutar Go. Comprueba que esté instalado y disponible en PATH.')
  console.error(result.error.message)
}
process.exit(result.status ?? 1)
