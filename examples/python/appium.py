import os

from appium import webdriver
from appium.options.android import UiAutomator2Options

from locatr import LlmProvider, LlmSettings, Locatr
from python_client.locatr.schema import LocatrAppiumSettings

capabilities = dict(
    platformName="Android",
    automationName="uiautomator2",
    deviceName=os.environ.get("DEVICE_NAME"),
    appPackage="com.google.android.deskclock",
    appActivity="com.android.deskclock.DeskClock",
    language="en",
    locale="US",
)

appium_server_url = "http://localhost:4723"


def setup_locatr(appium_url: str, appium_session_id: str) -> Locatr:
    llm_settings = LlmSettings(
        llm_provider=LlmProvider.OPENAI,
        llm_api_key=os.environ.get("LLM_API_KEY"),
        model_name=os.environ.get("LLM_MODEL"),
        reranker_api_key=os.environ.get("COHERE_API_KEY"),
    )
    locatr_settings_appium = LocatrAppiumSettings(
        llm_settings=llm_settings,
        appium_url=appium_url,
        appium_session_id=appium_session_id,
    )
    return Locatr(locatr_settings_appium)


def main():
    options = UiAutomator2Options().load_capabilities(capabilities)
    driver = webdriver.Remote(appium_server_url, options=options)
    appium_locatr = setup_locatr(appium_server_url, driver.session_id)
    print(appium_locatr.get_locatr("Give xpath of the current time."))


if __name__ == "__main__":
    main()
