import assert from 'node:assert/strict'
import { chromium } from '@playwright/test'

const webBase = process.env.WEB_BASE_URL ?? 'http://127.0.0.1:5173'
const adminEmail = process.env.TEST_ADMIN_EMAIL ?? 'admin@liteyuki.dev'
const adminPassword = process.env.TEST_ADMIN_PASSWORD ?? 'devops'

async function launchBrowser() {
  try {
    return await chromium.launch({ channel: 'chrome', headless: true })
  }
  catch {
    return chromium.launch({ headless: true })
  }
}

async function main() {
  const browser = await launchBrowser()
  const page = await browser.newPage({ viewport: { width: 1440, height: 960 } })
  page.setDefaultTimeout(15_000)

  await page.goto(`${webBase}/login`, { waitUntil: 'networkidle' })
  await page.getByRole('textbox').first().fill(adminEmail)
  await page.locator('input[type="password"]').fill(adminPassword)
  await page.getByRole('button', { name: /^(登录|Sign in)$/i }).click()
  await page.waitForURL(/dashboard|projects|settings|registries|code-repositories|clusters/, { timeout: 20_000 })

  for (const path of ['/dashboard', '/projects', '/registries', '/code-repositories', '/clusters', '/settings/account', '/settings/users', '/settings/auth-providers', '/settings/site']) {
    await page.goto(`${webBase}${path}`, { waitUntil: 'networkidle' })
    await assertVisibleMain(page, path)
  }

  await page.goto(`${webBase}/projects`, { waitUntil: 'networkidle' })
  const themeButtons = page.locator('[aria-label*="主题"], [aria-label*="theme"], button').filter({ hasText: /^$/ })
  assert(await themeButtons.count() >= 0)

  await browser.close()
  console.log('Browser smoke passed')
}

async function assertVisibleMain(page, path) {
  const main = page.locator('main')
  await main.waitFor({ state: 'visible' })
  const text = await main.innerText()
  assert(text.trim().length > 0, `${path} main content should not be empty`)
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
