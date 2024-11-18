import enum
from typing import Literal
from pydantic import BaseModel, Field

from locatr_py.constants import DEFAULT_LOCATR_RESULTS_PATH

LlmProvider = Literal["OpenAi", "Antropic"]


class LogLevel(enum.Enum):
    SILENT = 1
    ERROR = 2
    WARN = 3
    INFO = 4
    DEBUG = 5


class LlmOptions(BaseModel):
    provider: LlmProvider
    api_key: str
    model: str


class CohereReRankClinetOptions(BaseModel):
    api_key: str


class LogConfig(BaseModel):
    level: LogLevel = LogLevel.ERROR


class LocatrOptions(BaseModel):
    cache_path: str
    use_cache: bool = Field(default=True)
    results_file_path: str = Field(default=DEFAULT_LOCATR_RESULTS_PATH)
    llm_options: LlmOptions
    rerank_client_options: CohereReRankClinetOptions
    log_config: LogConfig
    connection_id: int
