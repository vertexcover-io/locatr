from locatr_py._c_functions import create_base_locatr
from locatr_py.options import LocatrOptions


class Locatr:
    def __init__(self, base_locatr_id: int) -> None:
        self.locatr_id = base_locatr_id

    @classmethod
    def create_locatr(cls, locatr_options: LocatrOptions) -> "Locatr":
        base_locatr = create_base_locatr(locatr_options)
        if base_locatr == "":
            return
