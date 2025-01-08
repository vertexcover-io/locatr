import os
import time

from playwright.sync_api import sync_playwright

from locatr import LlmProvider, LlmSettings, Locatr, LocatrCdpSettings


def setup_locatr() -> Locatr:
    llm_settings = LlmSettings(
        llm_provider=LlmProvider.OPENAI,
        llm_api_key=os.environ.get("LLM_API_KEY"),
        model_name=os.environ.get("LLM_MODEL"),
        reranker_api_key=os.environ.get("COHERE_API_KEY"),
    )
    locatr_settings_playwright = LocatrCdpSettings(
        llm_settings=llm_settings,
        cdp_url="http://localhost:9222",
    )
    return Locatr(locatr_settings_playwright)


def main(cdp_locatr: Locatr):
    with sync_playwright() as p:
        browser = p.chromium.launch(
            headless=False, args=["--remote-debugging-port=9222"]
        )
        page = browser.new_page()
        page.goto("https://store.steampowered.com/")
        time.sleep(5)

        search_bar_selector = cdp_locatr.get_locatr(
            "Search input bar on the steam store."
        ).selectors[0]

        search_barch_locatr = page.locator(search_bar_selector)
        search_barch_locatr.first.fill("Counter Strike 2")
        search_barch_locatr.first.press("Enter")

        time.sleep(5)


if __name__ == "__main__":
    main(setup_locatr())
