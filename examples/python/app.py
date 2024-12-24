import os

from python_client import (
    LlmProvider,
    LlmSettings,
    Locatr,
    LocatrCdpSettings,
    LocatrSeleniumSettings,
    PluginType,
)

llm_settings = LlmSettings(
    llm_provider=LlmProvider.OPENAI,
    llm_api_key=os.environ.get("LLM_API_KEY"),
    model_name=os.environ.get("LLM_MODEL_NAME"),
    reranker_api_key=os.environ.get("RERANKER_API_KEY"),
)
locatr_settings_selenium = LocatrSeleniumSettings(
    plugin_type=PluginType.SELENIUM,
    llm_settings=llm_settings,
    selenium_url="http://localhost:4444/wd/hub",
    selenium_session_id="e4c543363b9000a66073db7a39152719",
)

locatr_settins_cdp = LocatrCdpSettings(
    llm_settings=llm_settings,
    cdp_url="http://localhost:9222",
)

lib = Locatr(locatr_settings_selenium, debug=True)

print("finished sleeping")
print(lib.get_locatr("H1 element with text Example Domain"))
