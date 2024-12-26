import asyncio
import os

from locatr import LlmProvider, LlmSettings, Locatr, LocatrCdpSettings

llm_settings = LlmSettings(
    llm_provider=LlmProvider.OPENAI,
    llm_api_key=os.environ.get("LLM_API_KEY"),
    model_name=os.environ.get("LLM_MODEL"),
    reranker_api_key=os.environ.get("COHERE_API_KEY"),
)

# there is already a page opened for url: https://example.com/
locatr_settings_cdp = LocatrCdpSettings(
    llm_settings=llm_settings,
    cdp_url="http://localhost:9222",
)


async def main():
    selenium_locatr = Locatr(locatr_settings_cdp)
    print(
        await selenium_locatr.get_locatr_async("Link to 'More information...'")
    )


asyncio.run(main())
