import ctypes

from locatr_py import constants
from locatr_py.options import LocatrOptions


SharedLocatr = ctypes.CDLL(constants.SHARED_OBJECT_C_FILE)

# types for SharedCreateRpccConnection
SharedLocatr.SharedCreateRpccConnection.argtypes = [
    ctypes.c_int,  # port
    ctypes.c_char_p,  # targetId
]
SharedLocatr.SharedCreateRpccConnection.restype = ctypes.c_int

# types for SharedCreateBaseLocatr
SharedLocatr.SharedCreateBaseLocatr.argtypes = [
    ctypes.c_int,  # connectionId
    ctypes.c_char_p,  # cachePath
    ctypes.c_bool,  # useCache
    ctypes.c_int,  # logLevel
    ctypes.c_char_p,  # resultsFilePath
    ctypes.c_char_p,  # llmApiKey
    ctypes.c_char_p,  # llmProvider
    ctypes.c_char_p,  # llmModel
    ctypes.c_char_p,  # cohereReRankApiKey
]
SharedLocatr.SharedCreateBaseLocatr.restype = ctypes.c_char_p


def create_rpc_connection(cdp_port: int, target_id: str) -> int:
    port = ctypes.c_int(cdp_port)
    page_target_id = target_id.encode()
    return int(SharedLocatr.SharedCreateRpccConnection(port, page_target_id))


def create_base_locatr(
    locatr_options: LocatrOptions,
) -> str:
    # function params
    connection_id = ctypes.c_int(locatr_options.connection_id)
    cache_path = locatr_options.cache_path.encode()
    use_cache = ctypes.c_bool(locatr_options.use_cache)
    log_level = ctypes.c_int(locatr_options.log_config.level.value)
    results_file_path = locatr_options.results_file_path.encode()
    llm_api_key = locatr_options.llm_options.api_key.encode()
    llm_provider = locatr_options.llm_options.provider.encode()
    llm_model = locatr_options.llm_options.model.encode()
    cohere_re_rank_api_key = locatr_options.rerank_client_options.api_key.encode()

    return str(
        SharedLocatr.SharedCreateBaseLocatr(
            connection_id,
            cache_path,
            use_cache,
            log_level,
            results_file_path,
            llm_api_key,
            llm_provider,
            llm_model,
            cohere_re_rank_api_key,
        )
    )
