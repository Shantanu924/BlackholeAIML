import os
import json

base_dir = r"c:\Users\user\OneDrive\Desktop\Desktop\BlackholeAIML\vajra-intelligence\dashboard"
os.makedirs(os.path.join(base_dir, "src", "components"), exist_ok=True)
os.makedirs(os.path.join(base_dir, "src", "pages"), exist_ok=True)

package_json = {
  "name": "dashboard",
  "private": True,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.26.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.3.1",
    "vite": "^5.4.1"
  }
}

files = {
    "package.json": json.dumps(package_json, indent=2),
    "vite.config.js": """import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
export default defineConfig({
  plugins: [react()],
  base: '/dashboard/',
})""",
    "index.html": """<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>VAJRA Dashboard</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>""",
    "src/main.jsx": """import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'
import './index.css'
createRoot(document.getElementById('root')).render(<StrictMode><App /></StrictMode>)""",
    "src/index.css": """body { margin: 0; background: #0f0f1a; color: #e0e0e0; font-family: sans-serif; } a { color: #00ff88; }""",
    "src/App.jsx": """import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
function App() {
  return (
    <BrowserRouter basename="/dashboard">
      <nav style={{padding: '1rem', background: '#1a1a2e', display: 'flex', gap: '1rem'}}>
        <Link to="/">Events</Link>
        <Link to="/assess">Assess</Link>
      </nav>
      <div style={{padding: '1rem'}}>
        <Routes>
          <Route path="/" element={<div><h2>Live Events <span style={{color:'#00ff88'}}>LIVE</span></h2></div>} />
          <Route path="/assess" element={<div><h2>Risk Assessment <span style={{color:'#ffaa00'}}>SIMULATED</span></h2></div>} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}
export default App;"""
}

for path, content in files.items():
    full_path = os.path.join(base_dir, path)
    with open(full_path, "w") as f:
        f.write(content)
print("Files created.")
