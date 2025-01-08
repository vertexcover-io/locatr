from unittest.mock import MagicMock, patch

import pytest

from locatr._locatr import Locatr
from locatr.exceptions import (
    FailedToRetrieveLocatr,
)
from locatr.schema import (
    LocatrCdpSettings,
)


@pytest.fixture
def locatr_instance():
    settings = LocatrCdpSettings(
        cdp_url="http://localhost:9222",
        llm_settings={},
    )
    return Locatr(locatr_settings=settings, debug=True)


class TestLocatr:
    @patch("locatr._locatr.spawn_locatr_process")
    @patch("locatr._locatr.check_socket_in_use", return_value=False)
    @patch("locatr._locatr.change_socket_file", return_value="/tmp/locatr.sock")
    def test_initialize_process(
        self,
        mock_change_socket_file,
        mock_check_socket_in_use,
        mock_spawn_locatr_process,
        locatr_instance,
    ):
        mock_process = MagicMock()
        mock_process.stdout.readline.side_effect = [b"", b""]
        mock_process.stderr.readline.side_effect = [b"", b""]
        mock_spawn_locatr_process.return_value = mock_process

        locatr_instance._initialize_process()
        mock_spawn_locatr_process.assert_called_once()

    @patch("locatr._locatr.socket.socket")
    def test_initialize_socket(self, mock_socket, locatr_instance):
        mock_socket_instance = mock_socket.return_value
        locatr_instance._initialize_socket()
        mock_socket_instance.connect.assert_called_once()

    @patch("locatr._locatr.wait_for_socket")
    def test_wait_for_server(self, mock_wait_for_socket, locatr_instance):
        locatr_instance._socket = MagicMock()
        locatr_instance._wait_for_server()
        mock_wait_for_socket.assert_called_once_with(locatr_instance._socket)

    @patch("locatr._locatr.Locatr.get_locatr", return_value="locatr output")
    @pytest.mark.asyncio
    async def test_get_locatr_async(self, mock_get_locatr, locatr_instance):
        output = await locatr_instance.get_locatr_async("user request")
        assert output == "locatr output"
        mock_get_locatr.assert_called_once_with("user request")

    @patch("locatr._locatr.Locatr._initialize_process_and_socket")
    @patch("locatr._locatr.Locatr._send_message")
    @patch(
        "locatr._locatr.Locatr._recv_message",
        return_value=b'{"status": "error", "error": "some error"}',
    )
    def test_get_locatr_failure(
        self,
        mock_recv_message,
        mock_send_message,
        mock_initialize_process_and_socket,
        locatr_instance,
    ):
        with pytest.raises(FailedToRetrieveLocatr):
            locatr_instance.get_locatr("user request")
        mock_initialize_process_and_socket.assert_called_once()
        mock_send_message.assert_called_once()
        mock_recv_message.assert_called_once()

    @patch("locatr._locatr.Locatr._initialize_process_and_socket")
    @patch("locatr._locatr.Locatr._send_message")
    @patch("locatr._locatr.Locatr._recv_message", return_value=b"invalid json")
    def test_get_locatr_invalid_json(
        self,
        mock_recv_message,
        mock_send_message,
        mock_initialize_process_and_socket,
        locatr_instance,
    ):
        with pytest.raises(FailedToRetrieveLocatr):
            locatr_instance.get_locatr("user request")
        mock_initialize_process_and_socket.assert_called_once()
        mock_send_message.assert_called_once()
        mock_recv_message.assert_called_once()

    @patch("locatr._locatr.Locatr._initialize_process_and_socket")
    @patch("locatr._locatr.Locatr._send_message")
    @patch(
        "locatr._locatr.Locatr._recv_message",
        return_value=b'{"status": "ok", "id": "123e4567-e89b-12d3-a456-426614174000", "type": "initial_handshake", "error": "", "selectors": ["hello"], "selector_type": "xpath"}',
    )
    def test_get_locatr(
        self,
        mock_recv_message,
        mock_send_message,
        mock_initialize_process_and_socket,
        locatr_instance,
    ):
        result = locatr_instance.get_locatr("user request")
        assert result.selectors == ["hello"]
        assert result.selector_type == "xpath"
        mock_initialize_process_and_socket.assert_called_once()
        mock_send_message.assert_called_once()
        mock_recv_message.assert_called_once()

    @patch("locatr._locatr.Locatr._send_message")
    @patch(
        "locatr._locatr.Locatr._recv_message",
        return_value=b'{"status": "ok", "id": "123e4567-e89b-12d3-a456-426614174000", "type": "initial_handshake", "error": "", "output": ""}',
    )
    def test_perform_initial_handshake(
        self, mock_recv_message, mock_send_message, locatr_instance
    ):
        locatr_instance._perform_initial_handshake()
        mock_send_message.assert_called_once()
        mock_recv_message.assert_called_once()

    @patch("locatr._locatr.send_data_over_socket")
    def test_send_message(self, mock_send_data_over_socket, locatr_instance):
        locatr_instance._socket = MagicMock()
        locatr_instance._send_message(b"test data")
        mock_send_data_over_socket.assert_called_once_with(
            locatr_instance._socket, b"test data"
        )

    @patch("locatr._locatr.read_data_over_socket", return_value=b"test data")
    def test_recv_message(self, mock_read_data_over_socket, locatr_instance):
        locatr_instance._socket = MagicMock()
        data = locatr_instance._recv_message()
        assert data == b"test data"
        mock_read_data_over_socket.assert_called_once_with(
            locatr_instance._socket
        )
