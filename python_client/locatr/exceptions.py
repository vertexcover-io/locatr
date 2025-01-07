class LocatrBinaryNotFound(Exception):
    def __init__(self, path):
        super().__init__(f"Locatr binary not found at path: {path}")


class LocatrExecutionError(Exception):
    def __init__(self, message):
        super().__init__(message)


class SocketInitializationError(Exception):
    def __init__(self, message):
        super().__init__(message)


class LocatrSocketIsNone(Exception):
    ...


class LocatrInitialHandshakeFailed(Exception):
    def __init__(self, message: str) -> None:
        super().__init__(f"Locatr Initial handshake failed: {message}")


class FailedToRetrieveLocatr(Exception):
    def __init__(self, msg) -> None:
        super().__init__(msg)


class LocatrSocketError(Exception):
    def __init__(self, msg: str) -> None:
        super().__init__(msg)


class LocatrSocketNotAvailable(Exception):
    def __init__(self, msg: str) -> None:
        super().__init__(msg)


class LocatrClientServerVersionMisMatch(Exception):
    def __init__(self, client_version: str, server_version: str) -> None:
        super().__init__(
            f"Client version: {client_version}, "
            f"server version: {server_version}"
        )
