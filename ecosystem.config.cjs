const path = require('node:path')

module.exports = {
  apps: [{
    name: 'realtime-chat-backend',
    cwd: path.join(__dirname, 'backend'),
    script: path.join(__dirname, 'backend', 'bin', process.platform === 'win32' ? 'server.exe' : 'server'),
    interpreter: 'none',
    exec_mode: 'fork',
    instances: 1,
    autorestart: true,
    watch: false,
    restart_delay: 3000,
    min_uptime: '10s',
    max_restarts: 10,
    kill_timeout: 12000,
    time: true,
  }],
}
