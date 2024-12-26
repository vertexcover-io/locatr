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


class LocatrOutputMessageValidationFailed(Exception):
    def __init__(self, msg: str) -> None:
        super().__init__(f"Failed to validate message locatr output: {msg}")


class LocatrSocketError(Exception):
    def __init__(self, msg: str) -> None:
        super().__init__(msg)


class LocatrSocketNotAvialable(Exception):
    def __init__(self, msg: str) -> None:
        super().__init__(msg)
