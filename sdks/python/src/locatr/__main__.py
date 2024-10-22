from .locatr import PlaywrightLocatr, LlmConfig
from playwright.async_api import async_playwright

API_KEY = "OPENAI_API_KEY"
API_MODEL = "gpt-4o"
API_PROVIDER = "openai"


async def main():
    async with async_playwright() as pw:
        browser = await pw.chromium.launch()
        page = await browser.new_page()
        await page.goto("https://youtube.com")

        llm_conf = LlmConfig(api_key=API_KEY, model=API_MODEL, provider=API_PROVIDER)
        locatr = PlaywrightLocatr(llm_conf, page)
        selector = await locatr.get_locator("search bar")
        print(selector)


if __name__ == "__main__":
    import asyncio

    asyncio.run(main())
