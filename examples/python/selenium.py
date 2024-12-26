import os
import time

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By

from locatr import LlmProvider, LlmSettings, Locatr, LocatrSeleniumSettings


def setup_locatr(selenium_url: str, selenium_session_id: str) -> Locatr:
    llm_settings = LlmSettings(
        llm_provider=LlmProvider.OPENAI,
        llm_api_key=os.environ.get("LLM_API_KEY"),
        model_name=os.environ.get("LLM_MODEL"),
        reranker_api_key=os.environ.get("COHERE_API_KEY"),
    )
    locatr_settings_cdp = LocatrSeleniumSettings(
        llm_settings=llm_settings,
        selenium_url=selenium_url,
        selenium_session_id=selenium_session_id,
    )
    return Locatr(locatr_settings_cdp)


def main():
    opts = Options()
    dr = webdriver.Remote(
        command_executor="http://127.0.0.1:4444/wd/hub", options=opts
    )
    try:
        dr.get("https://hub.docker.com/")
        time.sleep(5)
        locatr = setup_locatr(
            "http://127.0.0.1:4444/wd/hub", str(dr.session_id)
        )
        search_bar_path = locatr.get_locatr("Search Docker Hub input field")
        element = dr.find_element(By.CSS_SELECTOR, search_bar_path)
        element.send_keys("Python")
        element.send_keys("Enter")
        time.sleep(2)
        print(dr.title)
    finally:
        dr.quit()


if __name__ == "__main__":
    main()
