const { existsSync, readFileSync, writeFileSync } = require('node:fs')
const { spawnSync } = require('node:child_process')
const path = require('node:path')
const root = path.join(__dirname, '..')
const certs = path.join(root, '.certs')
if (!existsSync(path.join(certs, 'localhost.crt'))) {
 const result = spawnSync('go', ['run', './cmd/devcert', '-out', certs], { cwd: path.join(root, 'backend'), stdio: 'inherit' })
 if (result.error || result.status !== 0) process.exit(result.status || 1)
}
function updateEnv(file, settings) {
 let content = existsSync(file) ? readFileSync(file, 'utf8') : ''
 for (const [key, value] of Object.entries(settings)) {
  const line = new RegExp(`^${key}=.*$`, 'm')
  content = line.test(content) ? content.replace(line, () => `${key}=${value}`) : content.trimEnd() + `\n${key}=${value}\n`
 }
 writeFileSync(file, content)
}
updateEnv(path.join(root, '.env'), {
 TLS_CERT_FILE: '../.certs/localhost.crt', TLS_KEY_FILE: '../.certs/localhost.key',
 CORS_ORIGIN: 'https://localhost:3000', COOKIE_SECURE: 'true', COOKIE_SAME_SITE: 'lax',
})
updateEnv(path.join(root, 'frontend/.env'), {
 NUXT_DEV_TLS: 'true', NUXT_PUBLIC_API_BASE: 'https://localhost:8080', NUXT_PUBLIC_WS_BASE: 'wss://localhost:8080',
})
console.log('HTTPS/WSS configurados en localhost. Reinicia el backend y el frontend.')
console.log('Para confiar en el certificado en Windows (solo si tú generaste esta CA):')
console.log('Import-Certificate -FilePath .certs/ca.crt -CertStoreLocation Cert:\CurrentUser\Root')
