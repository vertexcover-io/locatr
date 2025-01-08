# Locatr 
Locatr package helps you to find HTML locators on a webpage using prompts and llms.

## Overview 
- LLM based HTML element css path finder.
- Re-rank support for improved accuracy.
- Supports playwright, selenium, cdp.  
- Uses cache to reduce calls to llm apis.

Example: 

```python
print(locatr.get_locatr("Search input bar in the page"))
# output: 'html > div > input'
```
For more examples check the `examples/python` folder.

### Install locatr with 

```
pip install locatr
```

## Table of Contents

- [ Quick Example ](#quick-example)
- [ Locatr Settings ](#locatr-options)
- [ Get Locatr ](#get-locatr)

### Quick Example

```python
# example assumes that there is already a page opened in the selenium session.
import os

from locatr import (
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
    reranker_api_key=os.environ.get("COHERE_API_KEY"),
)

locatr_settings_selenium = LocatrSeleniumSettings(
    llm_settings=llm_settings,
    selenium_url=os.environ.get("SELENIUM_URL"), # url must end with `/wd/hub`
    selenium_session_id="e4c543363b9000a66073db7a39152719",
)

selenium_locatr = Locatr(locatr_settings_selenium, debug=True)

print(selenium_locatr.get_locatr("H1 element with text Example Domain"))

```

### Locatr Settings 

Locatr settings expects the following fields:

#### Cache Settings.

- `cache_path` -> The path to the cache file. This file will be reused later to save llm api requests. If nothing is provided then it is stored in `.locatr.cache`.
- `use_cache` -> Weather to use cache or not.

For cache schema see: [link](../README.md#cache-schema)

#### Llm settings. 

Contains all the settings required for llm and re-ranking.

```
from locatr import LlmSettings
```
It expects the following values: 
1. `llm_provider` -> The llm provider you want to use. Options are `locatr.LlmProvider.OPENAI` and `locatr.LlmProvider.ANTHROPIC`
2. `llm_api_key` -> The provider's api key value.
3. `model_name` -> Specify which llm model you want to use.
4. `reranker_api_key` -> Api key for cohere reranker. It is optional if not provided reranking will not be used.

Example:
```python
from locatr import LlmSettings

llm_settings = LlmSettings(
    llm_provider=LlmProvider.OPENAI,
    llm_api_key=os.environ.get("LLM_API_KEY"),
    model_name=os.environ.get("LLM_MODEL_NAME"),
    reranker_api_key=os.environ.get("COHERE_API_KEY"),
)
```
**Note: If values are not provided in the settings then they will be read from the following env variables.**
- `LLM_PROVIDER`
- `LLM_MODEL`
- `LLM_API_KEY`
- `COHERE_API_KEY`

Locatr settings is bound with the type of plugin you want to use (cdp/selenium).

To create settings for cdp use (use with playwright):

```python
from locatr import LocatrCdpSettings

# .... create llm settings

locatr_setting_cdp = LocatrCdpSettings(
    llm_settings=llm_settings,
    cdp_url="http://localhost:9222", # You can get this port by passing the following argument to chromium based browsers: `--remote-debugging-port=9222`
)

```

To create settings for selenium use:

```python
from locatr import LocatrSeleniumSettings

locatr_settings_selenium = LocatrSeleniumSettings(
    llm_settings=llm_settings,
    selenium_url=os.environ.get("SELENIUM_URL"), # url must end with `/wd/hub`
    selenium_session_id="e4c543363b9000a66073db7a39152719",
)
```

### Get locatr

To get locatr string we need to import the Locatr class and pass the settings to it.

```python
from locatr import Locatr

# ... create settings 

l = Locatr(locatr_settings_selenium)
```

By default, the `Locatr` class operates without logging. However, if you'd like to view the Locatr server logs for debugging purposes, you can enable the `debug` parameter by passing `True` as the second argument during initialization.

```python
l.get_locatr("red 'yes button' in the form")
```

You can also get locatrs asynchronously. Just call `get_locatr_async`.

```
    await l.get_locatr_async("red 'yes button' in the form")
```
