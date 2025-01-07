import atexit
import os
import random
import socket
import struct
import subprocess
import sys
import time
from subprocess import CalledProcessError, Popen
from typing import List

from locatr._constants import (
    SOCKET_RETRY_DELAY,
    SOCKET_SEND_DATA_MAX_RETRIES,
    VERSION,
    WAIT_FOR_SOCKET_MAXIMUM_RETRIES,
    SocketFilePath,
)
from locatr.exceptions import (
    LocatrBinaryNotFound,
    LocatrClientServerVersionMisMatch,
    LocatrExecutionError,
    LocatrSocketError,
    LocatrSocketNotAvailable,
)

NAME = "/tmp/locatr{}.sock"


def check_socket_in_use(path: str):
    if os.path.exists(path):
        return True
    return False


def change_socket_file() -> str:
    name = NAME.format(random.randint(0, 100000))
    while check_socket_in_use(name):
        name = NAME.format(random.randint(0, 100000))
    return name


def locatr_go_cleanup(process: Popen[bytes]):
    process.kill()
    if check_socket_in_use(SocketFilePath.path):
        os.remove(SocketFilePath.path)


def spawn_locatr_process(args: List[str]) -> Popen[bytes]:
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
        print("exception while reading process output", e, file=sys.stderr)


def create_packed_message(message_str: str) -> bytes:
    message_length = len(message_str)
    version_bytes = bytes(VERSION)
    packed_data = struct.pack(">3B", *version_bytes) + struct.pack(
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
    raise LocatrSocketNotAvailable(
        f"Locatr socket not available after "
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


def compare_version(server_version: bytes) -> bool:
    return list(server_version) == VERSION


def convert_version(version: List[int]) -> str:
    version_string = ""
    for index, ver in enumerate(version):
        if index > 0:
            version_string = f"{version_string}.{ver}"
        else:
            version_string = f"{version_string}{ver}"

    return version_string


def read_data_over_socket(sock: socket.socket) -> bytes:
    try:
        version = sock.recv(3)
        if not compare_version(version):
            raise LocatrClientServerVersionMisMatch(
                convert_version(VERSION), convert_version(list(version))
            )
        length_data = sock.recv(4)
        actual_length = int.from_bytes(length_data, byteorder="big")
        output_data = sock.recv(actual_length)
        return output_data
    except socket.timeout:
        raise LocatrSocketError("Socket connection timed out while reading")
    except LocatrClientServerVersionMisMatch as e:
        raise e
    except Exception as e:
        raise LocatrSocketError(
            f"Unexpected error occured while receving data: {e}"
        )
