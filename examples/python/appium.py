# pyright: reportPrivateImportUsage=false
import os

from appium import webdriver
from appium.options.android import UiAutomator2Options

from locatr import Locatr
from locatr.schema import LocatrAppiumSettings, LlmProvider, LlmSettings

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
        llm_provider=LlmProvider.ANTHROPIC,
        llm_api_key=os.environ.get("ANTHROPIC_API_KEY"),
        model_name="claude-3-5-sonnet-latest",
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
    print(appium_locatr.get_locatr("Give xpath of the current time.").selectors)


if __name__ == "__main__":
    main()
