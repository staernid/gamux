#!/usr/bin/env node

/**
 * Unified GUI Diagnostic, Tracing & Evaluation CLI
 *
 * Commands:
 *   dump                Dump current store snapshots, timeline, and audit status as JSON
 *   trace [scenario]    Run a lifecycle scenario and display formatted event trace & store diffs
 *   audit [scenario]    Run invariant checks and exit with code 0 (clean) or 1 (violations)
 *   eval <expr>         Evaluate arbitrary JavaScript expression against window.__GAMUX_DEV__
 *   snap                Capture full headless Chromium UI view snapshots into .gui-snapshots/
 */

import { spawn, execSync } from 'child_process';
import http from 'http';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ROOT_DIR = path.resolve(__dirname, '..');
const FRONTEND_DIR = path.resolve(ROOT_DIR, 'frontend');
const SNAPSHOT_DIR = path.resolve(ROOT_DIR, '.gui-snapshots');

const PORT = 5174;
const BASE_URL = `http://127.0.0.1:${PORT}`;

function findChromium() {
  const binaries = ['chromium-browser', 'chromium', 'google-chrome', 'google-chrome-stable'];
  for (const b of binaries) {
    try {
      const out = execSync(`which ${b} 2>/dev/null`, { stdio: ['pipe', 'pipe', 'ignore'] }).toString().trim();
      if (out && fs.existsSync(out)) return out;
    } catch {}
  }
  return null;
}

function checkServerReady() {
  return new Promise((resolve) => {
    const req = http.get(BASE_URL, (res) => {
      resolve(res.statusCode >= 200 && res.statusCode < 400);
    });
    req.on('error', () => resolve(false));
    req.setTimeout(500, () => {
      req.destroy();
      resolve(false);
    });
  });
}

async function ensureViteServer() {
  const isUp = await checkServerReady();
  if (isUp) {
    return { process: null, pid: null };
  }

  const serverProc = spawn('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', String(PORT), '--strictPort'], {
    cwd: FRONTEND_DIR,
    stdio: 'ignore'
  });

  for (let i = 0; i < 50; i++) {
    await new Promise(r => setTimeout(r, 100));
    if (await checkServerReady()) {
      return { process: serverProc, pid: serverProc.pid };
    }
  }

  throw new Error(`Failed to start Vite preview server on ${BASE_URL} after 5s`);
}

async function executeViaCDP(chromeBin, targetUrl, jsExpression) {
  const cdpPort = 9222 + Math.floor(Math.random() * 500);
  const tmpProfile = fs.mkdtempSync('/tmp/gamux-chrome-');
  const chromeProc = spawn(chromeBin, [
    '--headless=new',
    '--disable-gpu',
    '--no-sandbox',
    '--disable-extensions',
    `--user-data-dir=${tmpProfile}`,
    `--remote-debugging-port=${cdpPort}`,
    'about:blank'
  ], { stdio: 'ignore' });

  const cleanup = () => {
    try {
      chromeProc.kill('SIGKILL');
      setTimeout(() => {
        try { fs.rmSync(tmpProfile, { recursive: true, force: true }); } catch {}
      }, 500);
    } catch {}
  };

  try {
    // Wait for CDP endpoint
    let versionData = null;
    for (let i = 0; i < 40; i++) {
      await new Promise(r => setTimeout(r, 100));
      try {
        const res = await fetch(`http://127.0.0.1:${cdpPort}/json/list`);
        if (res.ok) {
          const list = await res.json();
          if (Array.isArray(list) && list.some(t => t.type === 'page' && t.webSocketDebuggerUrl)) {
            versionData = list;
            break;
          }
        }
      } catch {}
    }

    if (!versionData || versionData.length === 0) {
      throw new Error(`CDP endpoint not ready on port ${cdpPort}`);
    }

    const pageTarget = versionData.find(t => t.type === 'page' && t.url.includes(String(PORT))) ||
                       versionData.find(t => t.type === 'page') ||
                       versionData[0];
    const wsUrl = pageTarget.webSocketDebuggerUrl;

    const result = await evaluateWebSocket(wsUrl, targetUrl, jsExpression);
    cleanup();
    return result;
  } catch (err) {
    cleanup();
    throw err;
  }
}

function evaluateWebSocket(wsUrl, targetUrl, expr) {
  return new Promise((resolve, reject) => {
    const WS = globalThis.WebSocket;
    if (!WS) {
      return reject(new Error('global WebSocket not available in Node runtime'));
    }

    const ws = new WS(wsUrl);
    let msgId = 1;

    const sendCmd = (method, params = {}) => {
      const id = msgId++;
      ws.send(JSON.stringify({ id, method, params }));
      return id;
    };

    const pendingRequests = new Map();

    const request = (method, params = {}) => {
      return new Promise((res, rej) => {
        const id = sendCmd(method, params);
        pendingRequests.set(id, { resolve: res, reject: rej });
      });
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.id && pendingRequests.has(msg.id)) {
          const { resolve, reject } = pendingRequests.get(msg.id);
          pendingRequests.delete(msg.id);
          if (msg.result?.exceptionDetails) {
            const desc = msg.result.exceptionDetails.exception?.description || msg.result.exceptionDetails.text;
            reject(new Error(desc));
          } else {
            resolve(msg.result?.result?.value);
          }
        }
      } catch (e) {
        reject(e);
      }
    };

    ws.onopen = async () => {
      try {
        await request('Page.enable');
        await request('Runtime.enable');
        await request('Page.navigate', { url: targetUrl });

        // Poll for window.__GAMUX_DEV__ readiness via Node-side intervals
        let ready = false;
        for (let i = 0; i < 50; i++) {
          await new Promise(r => setTimeout(r, 100));
          try {
            const isReady = await request('Runtime.evaluate', {
              expression: 'Boolean(window.__GAMUX_DEV__)',
              returnByValue: true
            });
            if (isReady === true) {
              ready = true;
              break;
            }
          } catch {}
        }

        if (!ready) {
          throw new Error('GAMUX_DEV bridge not ready after waiting');
        }

        const evalResult = await request('Runtime.evaluate', {
          expression: `(async () => { return await (${expr}); })()`,
          awaitPromise: true,
          returnByValue: true
        });

        resolve(evalResult);
      } catch (err) {
        reject(err);
      } finally {
        try { ws.close(); } catch {}
      }
    };

    ws.onerror = (err) => {
      reject(err);
      try { ws.close(); } catch {}
    };

    setTimeout(() => {
      reject(new Error('CDP evaluation timed out after 15000ms'));
      try { ws.close(); } catch {}
    }, 15000);
  });
}

// Subcommands
async function handleDump(chromeBin) {
  const result = await executeViaCDP(chromeBin, `${BASE_URL}/?view=library`, `window.__GAMUX_DEV__.dumpState()`);
  console.log(JSON.stringify(result, null, 2));
}

async function handleTrace(chromeBin, scenario = 'apply-game-lifecycle') {
  console.log(`🔍 Running diagnostic trace scenario: '${scenario}'...`);
  const result = await executeViaCDP(
    chromeBin,
    `${BASE_URL}/?view=library`,
    `
      (async () => {
        const res = await window.__GAMUX_DEV__.runScenario('${scenario}');
        const audit = window.__GAMUX_DEV__.audit();
        const timeline = window.__GAMUX_DEV__.timeline.getEntries();
        const stores = window.__GAMUX_DEV__.getStoreSnapshots();
        return { scenario: res, audit, timeline, stores };
      })()
    `
  );

  console.log(`\n📋 Event Timeline (${result.timeline.length} events recorded):`);
  console.table(
    result.timeline.map((e) => ({
      Index: e.index,
      Time: new Date(e.timestamp).toLocaleTimeString(),
      Event: e.eventName || e.type,
      Details: e.payload?.stage || e.payload?.details || e.scenarioName || ''
    }))
  );

  console.log(`\n📊 Store Summary:`);
  console.log(`  • Active Task:`, result.stores.taskQueue.activeTask ? `${result.stores.taskQueue.activeTask.title} (${result.stores.taskQueue.activeTask.percent}%)` : 'None');
  console.log(`  • Total Active Tasks:`, result.stores.taskQueue.totalCount);
  console.log(`  • Active Transfers:`, result.stores.transfers.activeCount);

  console.log(`\n🩺 Invariant Audit Report:`);
  if (result.audit.passed) {
    console.log(`  ✅ All invariants passed cleanly (0 errors, ${result.audit.warningsCount} warnings)`);
  } else {
    console.error(`  ❌ INVARIANT FAILURES (${result.audit.errorsCount} errors):`);
    result.audit.issues.forEach((iss) => {
      console.error(`    [${iss.severity.toUpperCase()}] [${iss.component}] ${iss.rule}: ${iss.message}`);
    });
  }
}

async function handleAudit(chromeBin, scenario = 'apply-game-lifecycle') {
  const result = await executeViaCDP(
    chromeBin,
    `${BASE_URL}/?view=library`,
    `
      (async () => {
        if ('${scenario}') {
          try { await window.__GAMUX_DEV__.runScenario('${scenario}'); } catch {}
        }
        return window.__GAMUX_DEV__.audit();
      })()
    `
  );

  if (result.passed) {
    console.log(`✅ Invariant audit passed: 0 errors (${result.warningsCount} warnings)`);
    if (result.warningsCount > 0) {
      result.issues.forEach(i => console.warn(`   ⚠️ [${i.component}] ${i.rule}: ${i.message}`));
    }
    process.exit(0);
  } else {
    console.error(`❌ Invariant audit FAILED with ${result.errorsCount} errors:`);
    result.issues.filter(i => i.severity === 'error').forEach(i => {
      console.error(`   ❌ [${i.component}] ${i.rule}: ${i.message}`);
    });
    process.exit(1);
  }
}

async function handleEval(chromeBin, expression) {
  if (!expression) {
    console.error('Error: expression required for eval');
    process.exit(1);
  }
  const result = await executeViaCDP(chromeBin, `${BASE_URL}/?view=library`, expression);
  console.log(typeof result === 'object' ? JSON.stringify(result, null, 2) : result);
}

async function handleSnapshots(chromeBin) {
  fs.mkdirSync(SNAPSHOT_DIR, { recursive: true });
  console.log(`📸 Capturing full suite of UI views into ${SNAPSHOT_DIR}...`);

  const views = [
    { name: '01_library_hero.png', path: '?view=library' },
    { name: '02_downloads_view.png', path: '?view=downloads' },
    { name: '03_settings_view.png', path: '?view=settings' },
    { name: '04_game_drawer.png', path: '?view=library&drawer=1' },
    { name: '05_settings_emu.png', path: '?view=settings&tab=emu' },
    { name: '06_search_grid.png', path: '?view=library&search=1' },
  ];

  for (const v of views) {
    const targetFile = path.join(SNAPSHOT_DIR, v.name);
    const cmd = `"${chromeBin}" --headless=new --hide-scrollbars --window-size=1280,800 --virtual-time-budget=2000 --run-all-compositor-stages-before-draw --screenshot="${targetFile}" "${BASE_URL}/${v.path}"`;
    execSync(cmd, { stdio: 'ignore' });
    console.log(`  ✓ Captured ${v.name}`);
  }

  console.log(`✨ All snapshots captured successfully in: ${SNAPSHOT_DIR}`);
}

async function main() {
  const args = process.argv.slice(2);
  const command = args[0] || 'audit';

  const chromeBin = findChromium();
  if (!chromeBin) {
    console.error('⚠️ Chromium / Google Chrome binary not found in PATH.');
    process.exit(command === 'audit' ? 0 : 1);
  }

  let server = null;
  try {
    server = await ensureViteServer();

    switch (command) {
      case 'dump':
        await handleDump(chromeBin);
        break;
      case 'trace':
        await handleTrace(chromeBin, args[1] || 'apply-game-lifecycle');
        break;
      case 'audit':
        await handleAudit(chromeBin, args[1] || 'apply-game-lifecycle');
        break;
      case 'eval':
        await handleEval(chromeBin, args.slice(1).join(' '));
        break;
      case 'snap':
        await handleSnapshots(chromeBin);
        break;
      default:
        console.error(`Unknown command: ${command}`);
        console.log('Available commands: dump, trace, audit, eval, snap');
        process.exit(1);
    }
  } finally {
    if (server && server.process) {
      try {
        server.process.kill('SIGKILL');
      } catch {}
    }
  }
}

main().catch((err) => {
  console.error('Fatal CLI Error:', err.message || err);
  process.exit(1);
});
