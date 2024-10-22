import asyncio
from dataclasses import asdict, dataclass
from abc import ABC, abstractmethod
import typing as ty
from playwright.async_api import Page

from .ipc import IpcHelper


@dataclass
class LlmConfig:
    api_key: str
    model: str
    provider: ty.Literal["anthropic", "openai"]


@dataclass
class _IpcLocatrConfig:
    llm_config: LlmConfig
    cache_path: str


class _BaseLocatr(ABC):
    def __init__(self, llm_conf: LlmConfig) -> None:
        self._ipc_helper = IpcHelper(evaluate_js_func=self._evaluate_js)
        self._config = _IpcLocatrConfig(
            llm_config=llm_conf,
            cache_path=".locator.cache",
        )
        self._start_future = asyncio.create_task(self._start())

    async def _start(self) -> None:
        resp = await self._ipc_helper.send_request(
            "init", {"config": asdict(self._config)}
        )
        if not resp:
            raise Exception("Failed to initalize locator")
        if resp.status_code != 200:
            raise Exception(f"Failed to initialize locatr: {resp.error}")

    async def get_locator(self, user_req: str) -> str:
        if not self._start_future.done():
            await self._start_future

        resp = await self._ipc_helper.send_request(
            "get_locator", {"user_req": user_req}
        )
        if not resp:
            raise Exception("Failed to get locator")
        elif resp.status_code != 200:
            raise Exception(f"Failed to get locator: {resp.error}")
        return resp.data

    @abstractmethod
    async def _evaluate_js(self, js_str: str) -> str: ...


class PlaywrightLocatr(_BaseLocatr):
    def __init__(self, llm_conf: LlmConfig, page: Page) -> None:
        self._page = page
        super().__init__(llm_conf)

    async def _evaluate_js(self, js_str: str) -> str:
        print("Evaluating:", js_str)
        js_res = await self._page.evaluate(js_str)
        return str(js_res)
