from dataclasses import dataclass
from typing import Optional

@dataclass
class A200Data:
    cache: dict[str, dict]
    config: Optional[dict]
