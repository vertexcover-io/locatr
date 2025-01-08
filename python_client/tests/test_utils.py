import os
import socket
import struct
import tempfile
import unittest
from unittest.mock import MagicMock, patch

from locatr._constants import VERSION, SocketFilePath
from locatr._utils import (
    change_socket_file,
    check_socket_in_use,
    compare_version,
    convert_version,
    create_packed_message,
    locatr_go_cleanup,
    read_data_over_socket,
    send_data_over_socket,
    wait_for_socket,
)
from locatr.exceptions import (
    LocatrClientServerVersionMisMatch,
    LocatrSocketError,
    LocatrSocketNotAvailable,
)


class TestUtils(unittest.TestCase):
    def test_check_socket_in_use(self):
        with tempfile.NamedTemporaryFile() as tmp:
            self.assertTrue(check_socket_in_use(tmp.name))
        self.assertFalse(check_socket_in_use("/non/existent/path"))

    def test_change_socket_file(self):
        with patch("locatr._utils.check_socket_in_use", return_value=False):
            socket_file = change_socket_file()
            self.assertTrue(socket_file.startswith("/tmp/locatr"))
            self.assertTrue(socket_file.endswith(".sock"))

    def test_locatr_go_cleanup(self):
        with tempfile.NamedTemporaryFile(delete=False) as tmp:
            SocketFilePath.path = tmp.name
            process = MagicMock()
            locatr_go_cleanup(process)
            process.kill.assert_called_once()
            self.assertFalse(os.path.exists(SocketFilePath.path))

    def test_create_packed_message(self):
        message = "test message"
        packed_message = create_packed_message(message)
        expected_message = struct.pack(">3B", *bytes(VERSION)) + struct.pack(
            f">I{len(message)}s", len(message), message.encode()
        )
        self.assertEqual(packed_message, expected_message)

    @patch("socket.socket.connect")
    def test_wait_for_socket(self, mock_connect):
        mock_connect.side_effect = [socket.error] * 5 + [None]
        sock = MagicMock()
        sock.connect = mock_connect
        wait_for_socket(sock)
        self.assertEqual(mock_connect.call_count, 6)

        mock_connect.side_effect = socket.error
        with self.assertRaises(LocatrSocketNotAvailable):
            wait_for_socket(sock)

    @patch("socket.socket.send")
    def test_send_data_over_socket(self, mock_send):
        sock = MagicMock()
        sock.send = mock_send
        packed_data = b"test data"
        send_data_over_socket(sock, packed_data)
        mock_send.assert_called_once_with(packed_data)

        mock_send.side_effect = socket.error("Connection reset by peer")
        with self.assertRaises(LocatrSocketError):
            send_data_over_socket(sock, packed_data)

    def test_compare_version(self):
        self.assertTrue(compare_version(bytes(VERSION)))
        self.assertFalse(compare_version(b"\x00\x00\x00"))

    def test_convert_version(self):
        version = [1, 2, 3]
        self.assertEqual(convert_version(version), "1.2.3")
        version = [10, 0, 1]
        self.assertEqual(convert_version(version), "10.0.1")

    @patch("socket.socket")
    def test_read_data_over_socket(self, mock_socket):
        mock_socket_instance = mock_socket.return_value
        mock_socket_instance.recv.side_effect = [
            bytes(VERSION),
            struct.pack(">I", 4),
            b"data",
        ]
        data = read_data_over_socket(mock_socket_instance)
        self.assertEqual(data, b"data")

        mock_socket_instance.recv.side_effect = [
            b"\x00\x00\x00",
            struct.pack(">I", 4),
            b"data",
        ]
        with self.assertRaises(LocatrClientServerVersionMisMatch):
            read_data_over_socket(mock_socket_instance)

        mock_socket_instance.recv.side_effect = socket.timeout
        with self.assertRaises(LocatrSocketError):
            read_data_over_socket(mock_socket_instance)

        mock_socket_instance.recv.side_effect = Exception("Unknown error")
        with self.assertRaises(LocatrSocketError):
            read_data_over_socket(mock_socket_instance)
