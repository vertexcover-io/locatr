# pyright: reportPrivateImportUsage=false
import asyncio
from enum import Enum
from typing import Literal
from appium import webdriver
from selenium.webdriver.common.by import By
import yaml
from pydantic import BaseModel, Field, HttpUrl
from typing import List, Union
from typer import Typer
from appium.options.android import UiAutomator2Options
from locatr import LlmSettings, LlmProvider, Locatr, LocatrAppiumSettings
import os


app = Typer()


class EvalActions(str, Enum):
    CLICK = "click"
    PRESS = "press"
    FILL = "fill"
    HOVER = "hover"


class AndroidConfig(BaseModel):
    app_package: str = Field(alias="appPackage")
    app_activity: str = Field(alias="appActivity")


class IosConfig(BaseModel):
    bundle_id: str = Field(alias="bundleId")


class LocatrConfig(BaseModel):
    use_cache: bool = Field(default=False, alias="useCache")
    cache_path: str = Field(alias="cachePath")
    results_file_path: str = Field(alias="resultsFilePath")
    use_rerank: bool = Field(alias="useReRank")


class Step(BaseModel):
    name: str
    timeout: int = Field(default=0)


class ActionStep(Step):
    action: EvalActions
    key: str | None = Field(default=None)
    fill_text: str | None = Field(alias="fillText", default=None)


class LocateStep(Step):
    user_request: str = Field(alias="userRequest")
    expected_locatrs: List[str] = Field(alias="expectedLocatrs")


class EvalConfigYaml(BaseModel):
    name: str
    automation_name: Literal["uiautomator2", "XCUITest"] = Field(
        alias="automationName"
    )
    platform: Literal["android", "ios"] = Field(alias="platform")
    server_url: HttpUrl = Field(alias="serverUrl")
    app_config: Union[AndroidConfig, IosConfig] = Field(alias="appConfig")
    steps: List[Union[ActionStep, LocateStep]]
    config: LocatrConfig
    device_name: str = Field(alias="deviceName")


def parse_yaml(file_path: str) -> Union[EvalConfigYaml, None]:
    with open(file_path, "r") as cf:
        loaded_yaml = yaml.safe_load(cf)
        return EvalConfigYaml.model_validate(loaded_yaml)


def start_appium_server(config: EvalConfigYaml) -> webdriver.Remote:
    cap = dict()
    if config.platform == "android" and isinstance(
        config.app_config, AndroidConfig
    ):
        cap = dict(
            platformName="Android",
            automationName=config.automation_name,
            deviceName=config.device_name,
            appPackage=config.app_config.app_package,
            appActivity=config.app_config.app_activity,
            language="en",
            locale="US",
        )
    print(config.server_url)
    opts = UiAutomator2Options().load_capabilities(cap)
    driver = webdriver.Remote("http://172.30.192.1:4723", options=opts)
    return driver


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


async def run_eval(config: EvalConfigYaml):
    driver = start_appium_server(config)
    locatr = setup_locatr(
        appium_url="http://172.30.192.1:4723",
        appium_session_id=driver.session_id,
    )
    try:
        selector = ""
        for step in config.steps:
            if isinstance(step, LocateStep):
                selector = locatr.get_locatr(step.user_request).selectors[0]
            elif isinstance(step, ActionStep):
                element = driver.find_element(By.XPATH, selector)
                match step.action:
                    case EvalActions.CLICK:
                        element.click()
            if step.timeout:
                await asyncio.sleep(step.timeout)

    finally:
        driver.quit()


@app.command()
def main():
    config = EvalConfigYaml.model_validate(parse_yaml("eval/appium/clock.yaml"))
    asyncio.run(run_eval(config))


if __name__ == "__main__":
    app()
