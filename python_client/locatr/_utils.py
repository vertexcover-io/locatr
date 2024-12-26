import atexit
import os
import random
import socket
import struct
import subprocess
import time
from subprocess import CalledProcessError, Popen

from locatr._constants import (
    SOCKET_RETRY_DELAY,
    SOCKET_SEND_DATA_MAX_RETRIES,
    WAIT_FOR_SOCKET_MAXIMUM_RETRIES,
    SocketFilePath,
)
from locatr.exceptions import (
    LocatrBinaryNotFound,
    LocatrExecutionError,
    LocatrSocketError,
    LocatrSocketNotAvialable,
)


def check_socket_in_use(path: str):
    if os.path.exists(path):
        return True
    return False


def change_socket_file() -> str:
    name = "/tmp/locatr{}.sock"
    name = name.format(name, random.randint(0, 100))

    while check_socket_in_use(name):
        name = name.format(name, random.randint(0, 100))

    return name


def locatr_go_cleanup(process: Popen[bytes]):
    process.kill()
    if check_socket_in_use(SocketFilePath.path):
        os.remove(SocketFilePath.path)


def spawn_locatr_process(args: list[str]) -> Popen[bytes]:
    locatr_path = os.path.join(os.path.dirname(__file__), "bin/locatr.bin")
    args = [locatr_path, *args]
    if not os.path.isfile(locatr_path):
        raise LocatrBinaryNotFound(locatr_path)
    try:
        process = Popen(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        atexit.register(locatr_go_cleanup, process)
        return process
    except CalledProcessError as e:
        raise LocatrExecutionError(f"Error during execution: {e}")
    except Exception as e:
        raise LocatrExecutionError(
            f"An uknown error occured while executing locatr binary: {e}"
        )


def log_output(process: Popen[bytes]):
    if not process.stdout or not process.stderr:
        return
    try:
        while True:
            stdout_line = process.stdout.readline().decode()
            stderr_line = process.stderr.readline().decode()
            if stdout_line:
                print(stdout_line)
            if stderr_line:
                print(stderr_line)

    except Exception as e:
        print("exception while reading process output", e)


def create_packed_message(message_str: str) -> bytes:
    message_length = len(message_str)
    packed_data = struct.pack(
        f">I{message_length}s", message_length, message_str.encode()
    )
    return packed_data


def wait_for_socket(sock: socket.socket):
    index = 0
    while index <= WAIT_FOR_SOCKET_MAXIMUM_RETRIES:
        try:
            sock.connect(SocketFilePath.path)
            return
        except socket.error:
            index += 1
            time.sleep(1)
    raise LocatrSocketNotAvialable(
        f"Locatr socket not avialable after "
        f"{WAIT_FOR_SOCKET_MAXIMUM_RETRIES} retries"
    )


def send_data_over_socket(sock: socket.socket, packed_data: bytes):
    retries = 0

    while retries < SOCKET_SEND_DATA_MAX_RETRIES:
        try:
            sock.send(packed_data)
            return
        except BrokenPipeError as e:
            raise e
        except socket.error as e:
            if "Connection reset by peer" in str(e):
                raise LocatrSocketError("Connection was closed unexpectedly")
            raise LocatrSocketError(f"Socket error occurred: {e}")
        except Exception as e:
            if retries == SOCKET_SEND_DATA_MAX_RETRIES:
                raise LocatrSocketError(
                    f"Unexpected error occurred when sending data: {e}"
                )

        retries += 1
        time.sleep(SOCKET_RETRY_DELAY)

    raise LocatrSocketError(
        f"Failed to send data after {SOCKET_SEND_DATA_MAX_RETRIES} retries."
    )


def read_data_over_socket(sock: socket.socket) -> bytes:
    try:
        length_data = sock.recv(4)
        actual_length = int.from_bytes(length_data, byteorder="big")
        output_data = sock.recv(actual_length)
        return output_data
    except socket.timeout:
        raise LocatrSocketError("Socket connection timed out while reading")
    except Exception as e:
        raise LocatrSocketError(
            f"Unexpected error occured while receving data: {e}"
        )
