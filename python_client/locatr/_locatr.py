import asyncio
import socket
import threading
import uuid
from subprocess import Popen
from typing import Optional, Union

from pydantic import ValidationError

from locatr._constants import SOCKET_TIMEOUT, SocketFilePath
from locatr._utils import (
    change_socket_file,
    check_socket_in_use,
    create_packed_message,
    log_output,
    read_data_over_socket,
    send_data_over_socket,
    spawn_locatr_process,
    wait_for_socket,
)
from locatr.exceptions import (
    FailedToRetrieveLocatr,
    LocatrInitialHandshakeFailed,
    LocatrSocketIsNone,
    SocketInitializationError,
)
from locatr.schema import (
    InitialHandshakeMessage,
    LocatrAppiumSettings,
    LocatrCdpSettings,
    LocatrOutput,
    LocatrSeleniumSettings,
    MessageType,
    InitialHandShakeOutputMessage,
    OutputStatus,
    UserRequestMessage,
)


class Locatr:
    _process: Optional[Popen[bytes]] = None
    _socket: Optional[socket.socket]

    def __init__(
        self,
        locatr_settings: Union[
            LocatrCdpSettings, LocatrSeleniumSettings, LocatrAppiumSettings
        ],
        debug: bool = False,
    ) -> None:
        self._settings = locatr_settings
        self._id = uuid.uuid4()
        self._debug: bool = debug
        self._socket = None

    def _initialize_process_and_socket(self):
        self._initialize_process()
        if not self._socket:
            self._initialize_socket()
            self._perform_initial_handshake()

    def _initialize_process(self):
        if not Locatr._process:
            if check_socket_in_use(SocketFilePath.path):
                SocketFilePath.path = change_socket_file()
            Locatr._process = spawn_locatr_process(
                [f"-socketFilePath={SocketFilePath.path}"]
            )
        if self._debug:
            self._start_locatr_log()

    def _start_locatr_log(self):
        thread = threading.Thread(target=log_output, args=(self._process,))
        thread.start()

    def _initialize_socket(self):
        try:
            self._socket = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            self._socket.settimeout(SOCKET_TIMEOUT)
            self._wait_for_server()
        except Exception as e:
            raise SocketInitializationError(f"Failed to initialize socket {e}")

    def _wait_for_server(self):
        if not self._socket:
            raise LocatrSocketIsNone()
        wait_for_socket(self._socket)

    def _perform_initial_handshake(self):
        message = InitialHandshakeMessage(
            locatr_settings=self._settings,
            id=self._id,
            type=MessageType.INITIAL_HANDSHAKE,
        )
        message_str = message.model_dump_json(exclude_none=True)
        packed_data = create_packed_message(message_str)
        self._send_message(packed_data)

        data = self._recv_message()
        try:
            output = InitialHandShakeOutputMessage.model_validate_json(data)
            if not output.status == OutputStatus.OK:
                raise LocatrInitialHandshakeFailed(output.error)
        except ValidationError as e:
            raise LocatrInitialHandshakeFailed(str(e.errors()))

    def _send_message(self, data: bytes):
        if not self._socket:
            raise LocatrSocketIsNone()
        try:
            send_data_over_socket(self._socket, data)
        except Exception as e:
            self._socket.close()
            raise e

    def _recv_message(self) -> bytes:
        if not self._socket:
            raise LocatrSocketIsNone()
        try:
            return read_data_over_socket(self._socket)
        except Exception as e:
            self._socket.close()
            raise e

    def get_locatr(self, user_req: str) -> LocatrOutput:
        self._initialize_process_and_socket()
        message = UserRequestMessage(
            user_request=user_req, id=self._id, type=MessageType.LOCATR_REQUEST
        )
        message_str = message.model_dump_json(exclude_none=True)
        packed_data = create_packed_message(message_str)
        self._send_message(packed_data)
        output_data = self._recv_message()
        try:
            print(str(output_data))
            output_msg = LocatrOutput.model_validate_json(output_data)
            if not output_msg.status == OutputStatus.OK:
                raise FailedToRetrieveLocatr(output_msg.error)
            return output_msg
        except ValidationError as e:
            raise FailedToRetrieveLocatr(str(e.errors()))

    async def get_locatr_async(self, user_req: str) -> LocatrOutput:
        return await asyncio.to_thread(self.get_locatr, user_req)

    def __del__(self):
        if self._socket:
            self._socket.close()
