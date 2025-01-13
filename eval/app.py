# pyright: reportPrivateImportUsage=false
import asyncio
import csv
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
import typer


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


class EvalResult(BaseModel):
    user_request: str
    passed: bool
    generated_locatrs: list[str]
    expected_locatrs: list[str]
    error: str | None = Field(default=None)


def parse_yaml(file_path: str) -> EvalConfigYaml:
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
    opts = UiAutomator2Options().load_capabilities(cap)
    driver = webdriver.Remote(str(config.server_url), options=opts)
    return driver


def setup_locatr(
    appium_url: str, appium_session_id: str, cache_path: str, use_cache: bool
) -> Locatr:
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
        cache_path=cache_path,
        use_cache=use_cache,
    )
    return Locatr(locatr_settings_appium)


async def run_eval(eval_config: EvalConfigYaml) -> List[EvalResult]:
    results: List[EvalResult] = []
    driver = start_appium_server(eval_config)
    locatr = setup_locatr(
        appium_url=str(eval_config.server_url),
        appium_session_id=driver.session_id,
        cache_path=eval_config.config.cache_path,
        use_cache=eval_config.config.use_cache,
    )
    try:
        selector = ""
        for step in eval_config.steps:
            if isinstance(step, LocateStep):
                typer.echo(
                    f"Locating element with user request: {step.user_request}"
                )
                try:
                    selectors = locatr.get_locatr(step.user_request)
                    selector = selectors.selectors[0]
                    results.append(
                        EvalResult(
                            user_request=step.user_request,
                            passed=True,
                            generated_locatrs=selectors.selectors,
                            expected_locatrs=step.expected_locatrs,
                        )
                    )
                except Exception as e:
                    results.append(
                        EvalResult(
                            user_request=step.user_request,
                            passed=False,
                            generated_locatrs=[],
                            expected_locatrs=step.expected_locatrs,
                            error=str(e),
                        )
                    )
                    typer.echo(f"Exception when fetching locatr {e}", err=True)
                    return results
            elif isinstance(step, ActionStep):
                typer.echo(
                    f"Performing action: {step.action} on selector: {selector}"
                )
                element = driver.find_element(By.XPATH, selector)
                match step.action:
                    case EvalActions.CLICK:
                        element.click()
            if step.timeout:
                typer.echo(f"Sleeping for {step.timeout} seconds.")
                await asyncio.sleep(step.timeout)
    except Exception as e:
        typer.echo(f"Error during evaluation: {e}", err=True)
    finally:
        typer.echo("Quitting the Appium driver.")
        driver.quit()
        return results


def write_resulsts_to_csv(file_name: str, results: List[EvalResult]):
    field_names = list(EvalResult.model_json_schema()["properties"].keys())
    with open(file_name, "w") as fp:
        writer = csv.DictWriter(fp, fieldnames=field_names)
        writer.writeheader()
        for result in results:
            writer.writerow(result.model_dump())


@app.command()
def run_one(file_path: str):
    typer.echo(f"Running evaluation for YAML file: {file_path}")
    try:
        config = parse_yaml(file_path)
        results = asyncio.run(run_eval(config))
        write_resulsts_to_csv(file_path.replace("yaml", "csv"), results)
        typer.echo("Evaluation completed successfully.")
    except Exception as e:
        typer.echo(f"Error during run_one execution: {e}", err=True)


if __name__ == "__main__":
    app()
