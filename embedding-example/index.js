import http from 'http'
import fs from 'fs'

const BASE_URL = 'http://localhost:3000'
const PORT = 3001
const VARIABLES = process.env.VARIABLES || '{}'
const API_KEY = process.env.API_KEY
const DASHBOARD_ID = process.env.DASHBOARD_ID

if (!API_KEY) {
  console.error('API_KEY env var is required')
  process.exit(1)
}
if (!DASHBOARD_ID) {
  console.error('DASHBOARD_ID env var is required')
  process.exit(1)
}

const server = http.createServer(async (req, res) => {
  if (req.url === '/api/jwt' && req.method === 'POST') {
    // verify coookie and define customer here
    let body = ''
    req.on('data', chunk => {
      body += chunk.toString()
    })
    req.on('end', async () => {
      try {
        const r = await fetch(`${BASE_URL}/api/auth/token`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            token: API_KEY,
            dashboardId: DASHBOARD_ID,
            variables: JSON.parse(VARIABLES),
          }),
        })
        if (r.status !== 200) {
          console.error('failed fetching token:', await r.text())
          res.writeHead(500, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ error: 'Fail to get JWT' }))
          return
        }
        const { jwt } = await r.json()
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify(jwt))
      } catch (error) {
        console.error(error)
        res.writeHead(400, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ error: 'Invalid JSON or missing baseUrl' }))
      }
    })
    return
  }

  fs.readFile('index.html', (err, content) => {
    if (err) {
      if (err.code === 'ENOENT') {
        res.writeHead(404)
        res.end('File not found')
        return
      }
      res.writeHead(500)
      res.end('Sorry, there was an error loading the page')
      return
    }
    res.writeHead(200, { 'Content-Type': 'text/html' })
    content = content.toString().replace('$BASE_URL', BASE_URL)
    content = content.toString().replace('$DASHBOARD_ID', DASHBOARD_ID)
    res.end(content)
  })
})

server.listen(PORT, () => {
  console.log(`Server running at http://localhost:${PORT}/`)
})

