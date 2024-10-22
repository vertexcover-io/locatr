import asyncio

import json
import typing as ty
from dataclasses import asdict, dataclass

type MethodType = ty.Literal["init", "get_locator", "evaluate_js"]
type ParamsType = ty.Dict[str, ty.Any]
type ConnType = ty.Literal[
    "server_request", "server_response", "client_request", "client_response"
]


@dataclass
class IpcResponse:
    status_code: int
    data: ty.Any
    error: str | None
    conn_type: ConnType = "client_response"


class IpcHelper:
    def __init__(
        self,
        evaluate_js_func: ty.Callable[[str], ty.Awaitable[str]],
        ipc_host: str = "localhost",
        ipc_port: int = 8080,
    ) -> None:
        self._evaluate_js_func = evaluate_js_func
        self._ipc_host = ipc_host
        self._ipc_port = ipc_port
        self._reader: asyncio.StreamReader | None = None
        self._writer: asyncio.StreamWriter | None = None
        self._connected = False
        self._response_queue: asyncio.Queue[IpcResponse] = asyncio.Queue()

    async def connect(self):
        if not self._connected:
            self._reader, self._writer = await asyncio.open_connection(self._ipc_host, self._ipc_port)
            self._connected = True
            asyncio.create_task(self._handle_incoming_messages())

    async def close(self):
        if self._connected and self._writer:
            self._writer.close()
            await self._writer.wait_closed()
            self._connected = False

    async def _write_response(
        self, response: IpcResponse
    ) -> None:
        assert self._writer is not None, "Writer is not initialized"
        print("Writing to stream:", response)
        self._writer.write(json.dumps(asdict(response)).encode("utf-8") + b"\n")
        await self._writer.drain()

    async def _handle_request(self, data: str) -> None:
        try:
            request_data = json.loads(data)
        except json.JSONDecodeError:
            print("Json decode error")
            await self._write_response(
                IpcResponse(status_code=422, data=None, error="JSON decode error")
            )
            return

        conn_type = request_data.get("conn_type")
        if conn_type != "server_request":
            return

        try:
            method: MethodType = request_data["method"]
            params: ParamsType = request_data["params"]
        except KeyError as e:
            await self._write_response(
                IpcResponse(status_code=400, data=None, error=f"Missing key: {e}")
            )
            return

        try:
            match method:
                case "evaluate_js":
                    js_response = await self._evaluate_js_func(params["js_str"])
                    await self._write_response(
                        IpcResponse(
                            status_code=200,
                            data={"js_response": js_response},
                            error=None,
                        )
                    )
                case _:
                    await self._write_response(
                        IpcResponse(status_code=400, data=None, error="Unknown method")
                    )
        except KeyError as e:
            await self._write_response(
                IpcResponse(status_code=400, data=None, error=f"Missing key: {e}")
            )
        except Exception as e:
            await self._write_response(
                IpcResponse(status_code=500, data=None, error=f"Internal server error: {e}")
            )

    async def _handle_incoming_messages(self):
        assert self._reader is not None, "Reader is not initialized"
        while self._connected:
            try:
                data = await self._reader.readline()
                if not data:
                    break
                message = data.decode("utf-8").strip()
                if not message:
                    continue

                print(f"Received message: {message}")

                try:
                    parsed_data = json.loads(message)
                    conn_type = parsed_data.get("conn_type")
                    if conn_type == "server_response":
                        await self._response_queue.put(IpcResponse(**parsed_data))
                    elif conn_type == "server_request":
                        await self._handle_request(message)
                except json.JSONDecodeError:
                    print("JSON decode error")
            except Exception as e:
                print(f"Error handling incoming message: {e}")
                break
        self._connected = False

    async def send_request(
        self, method: MethodType, params: ParamsType
    ) -> IpcResponse | None:
        if not self._connected:
            await self.connect()

        assert self._writer is not None, "Writer is not initialized"
        request = (
            json.dumps(
                {"method": method, "params": params, "conn_type": "client_request"}
            )
            + "\n"
        )

        self._writer.write(request.encode("utf-8"))
        await self._writer.drain()

        return await self._response_queue.get()
