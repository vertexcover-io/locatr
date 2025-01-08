import os

from locatr import (
    LlmProvider,
    LlmSettings,
    Locatr,
    LocatrCdpSettings,
    LocatrSeleniumSettings,
)

llm_settings = LlmSettings(
    llm_provider=LlmProvider.OPENAI,
    llm_api_key=os.environ.get("LLM_API_KEY"),
    model_name=os.environ.get("LLM_MODEL_NAME"),
    reranker_api_key=os.environ.get("COHERE_API_KEY"),
)
locatr_settings_selenium = LocatrSeleniumSettings(
    llm_settings=llm_settings,
    selenium_url=os.environ.get("SELENIUM_URL"),
    selenium_session_id="e4c543363b9000a66073db7a39152719",
)

locatr_settings_cdp = LocatrCdpSettings(
    llm_settings=llm_settings,
    cdp_url="http://localhost:9222",
)

selenium_locatr = Locatr(locatr_settings_selenium, debug=True)

print(selenium_locatr.get_locatr("H1 element with text Example Domain"))
