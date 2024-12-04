import http from 'http'
import fs from 'fs'
import path from 'path'

const PORT = 3001
const TOKEN = 'test'

const server = http.createServer(async (req, res) => {
  if (req.url === '/api/jwt' && req.method === 'POST') {
    //verify coookie here
    let body = ''
    req.on('data', chunk => {
      body += chunk.toString()
    })
    req.on('end', async () => {
      try {
        const { baseUrl, dashboardId } = JSON.parse(body)
        const r = await fetch(`${baseUrl}/api/auth/token`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            token: TOKEN,
            dashboardId
            // customer vars
          }),
        })
        if (r.status !== 200) {
          console.error('failed fetching token:', await r.text())
          res.writeHead(500)
          res.end('Fail to get JWT')
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
    res.end(content)
  })
})

server.listen(PORT, () => {
  console.log(`Server running at http://localhost:${PORT}/`)
})

