from pathlib import Path
import typing as ty
from playwright.sync_api import Page
from wasmer import (
    engine,  # type: ignore
    Store,  # type: ignore
    Module,  # type: ignore
    Instance,  # type: ignore
    Function,  # type: ignore
    FunctionType,  # type: ignore
    Type,  # type: ignore
    wasi,  # type: ignore
    Memory,  # type: ignore
    MemoryType,  # type: ignore
)
from wasmer_compiler_cranelift import Compiler  # type: ignore
import requests
import json

from logging import getLogger

logger = getLogger(__name__)

MASK = (2**32) - 1


class LlmCreds(ty.TypedDict):
    provider: ty.Literal["openai", "anthropic"]
    model: str
    api_key: str


class GetLocatorResult(ty.NamedTuple):
    error: str
    locator: str


class LocatorService:
    def __init__(
        self, llm_creds: LlmCreds, page: Page, wasm_file: Path = Path("main.wasm")
    ):
        self.llm_creds = llm_creds
        self.page = page

        self._init_wasm(wasm_file)
        self._init_locatr()

    def _init_wasm(self, wasm_fp: Path) -> None:
        self.store = Store(engine.Universal(Compiler))
        module = Module(self.store, wasm_fp.read_bytes())

        wasi_version = wasi.get_version(module, strict=False)
        wasi_env = (
            wasi.StateBuilder("LocatorService").environment("COLOR", "true").finalize()
        )
        import_object = wasi_env.generate_import_object(self.store, wasi_version)
        import_object.register(
            "env",
            {
                "wasiEvaluateJs": Function(
                    self.store,
                    self._evaluate_func,
                    FunctionType(params=[Type.I64], results=[Type.I64]),
                ),
                "wasiHttpPost": Function(
                    self.store,
                    self._http_post,
                    FunctionType(
                        params=[Type.I64, Type.I64, Type.I64], results=[Type.I64]
                    ),
                ),
                "memory": Memory(self.store, MemoryType(minimum=256)),
            },
        )

        self.instance = Instance(module, import_object)

        self._wasm_memory = self.instance.exports.memory
        self._wasm_malloc = self.instance.exports.malloc
        self._wasm_free = self.instance.exports.free
        self._wasm_init_locatr = self.instance.exports.InitLocatr
        self._wasm_get_locator_str = self.instance.exports.GetLocatorStr

    def _to_wasm_string(self, string: str) -> int:
        string_bytes = string.encode("utf-8")
        length = len(string_bytes)
        ptr = self._wasm_malloc(length)

        memory = self._wasm_memory.uint8_view(ptr)
        if len(memory) < length:
            raise RuntimeError(
                f"Allocated memory is too small: {len(memory)} < {length}"
            )

        memory[:length] = string_bytes
        return (ptr << 32) | length

    def _get_ptr_and_size_from_ptr_size(self, ptr_size: int) -> ty.Tuple[int, int]:
        ptr = ptr_size >> 32
        size = ptr_size & MASK
        return ptr, size

    def _read_str_from_wasm(self, ptr_size: int) -> str:
        ptr, length = self._get_ptr_and_size_from_ptr_size(ptr_size)
        memory = self._wasm_memory.uint8_view(ptr)
        output = memory[:length]

        string = bytes(output).decode("utf-8")
        return string

    def _free_wasm_memory(self, ptr_size: int) -> None:
        ptr, _ = self._get_ptr_and_size_from_ptr_size(ptr_size)
        self._wasm_free(ptr)

    def _init_locatr(self):
        provider_ptr_size = self._to_wasm_string(self.llm_creds["provider"])
        model_ptr_size = self._to_wasm_string(self.llm_creds["model"])
        api_key_ptr_size = self._to_wasm_string(self.llm_creds["api_key"])

        self.locator_ptr = self._wasm_init_locatr(
            provider_ptr_size,
            model_ptr_size,
            api_key_ptr_size,
        )

        self._free_wasm_memory(provider_ptr_size)
        self._free_wasm_memory(model_ptr_size)
        self._free_wasm_memory(api_key_ptr_size)

        logger.debug("LocatorService initialized")

    def _evaluate_func(self, script_ptr_size: int) -> int:
        js_str = self._read_str_from_wasm(script_ptr_size)
        result = self.page.evaluate(js_str)
        result_ptr = self._to_wasm_string(result)
        return result_ptr

    def _http_post(
        self, url_ptr_size: int, headers_ptr_size: int, body_ptr_size: int
    ) -> int:
        url_str = self._read_str_from_wasm(url_ptr_size)
        headers_str = self._read_str_from_wasm(headers_ptr_size)
        body_str = self._read_str_from_wasm(body_ptr_size)

        response = requests.post(
            url_str, headers=json.loads(headers_str), json=json.loads(body_str)
        )
        result_ptr = self._to_wasm_string(response.text)
        return result_ptr

    def get_locator(self, user_req: str) -> str:
        user_req_ptr = self._to_wasm_string(user_req)
        result_ptr = self._wasm_get_locator_str(self.locator_ptr, user_req_ptr)

        locator_str = self._read_str_from_wasm(result_ptr)
        self._free_wasm_memory(user_req_ptr)

        locator_obj = json.loads(locator_str)
        if error := (locator_obj.get("error")):
            raise Exception(error)

        return locator_obj["locator"]

    def close(self) -> None:
        self._wasm_free(self.locator_ptr)
