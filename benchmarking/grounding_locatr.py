from playwright.sync_api import sync_playwright
from pathlib import Path
import re
import sys
import yaml
from dataclasses import dataclass, asdict
from typing import Literal
from pathlib import Path
from base64 import standard_b64encode
from textwrap import dedent
from json_repair import repair_json
import logging

from anthropic import Anthropic

from dotenv import load_dotenv

from gradio_client import Client, handle_file
from PIL import Image, ImageDraw

load_dotenv()

# Add after imports
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def draw_point(image_path, coordinates, color="red", radius=12):
    image = Image.open(image_path)
    draw = ImageDraw.Draw(image)
    x, y = coordinates
    bounding_box = [x - radius, y - radius, x + radius, y + radius]
    draw.ellipse(bounding_box, fill=color, outline=color)
    image.show()


class AnthropicGrounding:
    INSTRUCTION_TEMPLATE = dedent("""
        Given the screen resolution of {screen_width}x{screen_height} and a description of a specific area, element, or object on a Browser GUI screen, \
        identify and provide the exact (x, y) coordinates of the interest. \
        Give the output in JSON format. The JSON schema should be: {{"x": int, "y": int}}.
        For example, if the description is "top-right corner of the search bar," respond with coordinates such as "{{"x": 1850, "y": 50}}".
    """)

    def __init__(self, screen_width: int, screen_height: int):
        self.client = Anthropic()
        self.screen_width = screen_width
        self.screen_height = screen_height

    def get_coordinates(self, query: str, image_file: str | Path):
        logger.info(f"Getting coordinates for query: {query}")
        response = self.client.beta.messages.create(
            model="claude-3-5-sonnet-20241022",
            max_tokens=1024,
            messages=[{
                "role": "user", 
                "content": [
                    {
                        "type": "text",
                        "text": self.INSTRUCTION_TEMPLATE.format(
                            screen_width=self.screen_width, screen_height=self.screen_height
                        )
                    },
                    { "type": "text", "text": f"Description: {query}" },
                    {
                        "type": "image",
                        "source": {
                            "type": "base64",
                            "media_type": "image/png",
                            "data": standard_b64encode(Path(image_file).read_bytes()).decode()
                        }
                    }
                ]
            }],
            betas=["computer-use-2024-10-22"],
        )
        logger.info(f"Raw response: {response.content[0].text}")
        json_response = repair_json(response.content[0].text, return_objects=True)
        if not json_response:
            logger.error(f"No coordinates found for query: {query}")
            return None
        coords = (json_response["x"], json_response["y"])
        logger.info(f"Coordinates found: {coords}")
        return coords


class OSAtlasGrounding:

    def __init__(self):
        self.client = Client("maxiw/OS-ATLAS")

    def get_coordinates(self, query: str, image_file: str | Path):
        result = self.client.predict(
            image=handle_file(image_file),
            text_input=f"{query}\nReturn the response in the form of a bbox",
            model_id="OS-Copilot/OS-Atlas-Base-7B",
            api_name="/run_example",
        )
        numbers = re.findall(r'-?\d+(?:\.\d+)?', result[1])
        if not numbers:
            return None
        x1, y1, x2, y2 = map(float, numbers[:4])
        return int((x1 + x2) / 2), int((y1 + y2) / 2)


class Locator:
    def __init__(
        self, 
        headless_browser: bool = True,
        viewport_size: tuple[int, int] = (1920, 991),
        grounding_provider: Literal["anthropic", "os_atlas"] = "anthropic",
    ) -> None:
        logger.info(f"Initializing Locator with provider: {grounding_provider}")
        self._playwright = sync_playwright().start()
        self._browser = self._playwright.chromium.launch(headless=headless_browser)
        self._context = self._browser.new_context(
            viewport={"width": viewport_size[0], "height": viewport_size[1]}
        )
        if grounding_provider == "anthropic":
            self._grounding_model = AnthropicGrounding(
                screen_width=viewport_size[0], screen_height=viewport_size[1],
            )
        elif grounding_provider == "os_atlas":
            self._grounding_model = OSAtlasGrounding()
        else:
            raise ValueError(f"Invalid grounding provider: {grounding_provider}")

    def generate_selectors(self, url: str, query: str, *, show: bool = False) -> list[str]:
        logger.info(f"Generating selectors for URL: {url}")
        logger.info(f"Query: {query}")
        page = self._context.new_page()
        page.goto(url, wait_until="networkidle")

        screenshot_path = Path("screenshot.png")
        page.screenshot(path=str(screenshot_path))

        coords = self._grounding_model.get_coordinates(query, screenshot_path)
        if not coords:
            logger.error(f"No coordinates found for query: {query}")
            raise ValueError(f"No relevant coordinates found for the query: {query}.")
        
        logger.info(f"Found coordinates: {coords}")

        if show:
            draw_point(screenshot_path, coords)

        page.add_script_tag(path="locatrInject.js")

        selectors = page.evaluate(
            "([x, y]) => generateUniqueLocatorsFromCoords(x, y)", coords
        )
        logger.info(f"Generated selectors: {selectors}")
        page.close()
        return selectors

@dataclass
class Step:
    userRequest: str
    locatrs: list[str]

@dataclass
class Schema:
    url: str
    steps: list[Step]

def main():
    logger.info("Starting main execution")
    if len(sys.argv) == 2:
        provider = sys.argv[1]
        if provider not in ["anthropic", "os_atlas"]:
            raise ValueError(f"Invalid provider: {provider}. Must be one of: anthropic, os_atlas")
    else:
        provider = "anthropic"

    schema_folder = Path("schema")
    eval_folder = schema_folder / "eval"
    result_folder = schema_folder / "results" / f"{provider}_grounding"
    print(result_folder)
    result_folder.mkdir(parents=True, exist_ok=True)

    locator = Locator(grounding_provider=provider)
    logger.info(f"Processing eval files from: {eval_folder}")

    for eval_file in eval_folder.glob("*.yaml"):
        logger.info(f"Processing file: {eval_file}")
        eval_dict = yaml.safe_load(eval_file.open("r"))
        eval_schema = Schema(url=eval_dict["url"], steps=[Step(**step) for step in eval_dict["steps"]])
        eval_url, eval_steps = eval_schema.url, eval_schema.steps

        result_steps = []
        for step in eval_steps:
            try:
                logger.info(f"Processing step: {step.userRequest}")
                locators = locator.generate_selectors(eval_url, step.userRequest)
            except ValueError as e:
                logger.error(f"Error generating locators for {step.userRequest}: {e}")
                locators = []
            result_steps.append(Step(userRequest=step.userRequest, locatrs=locators))
        
        result_schema = Schema(url=eval_url, steps=result_steps)
        
        logger.info(f"Writing results to: {result_folder / eval_file.name}")
        yaml.safe_dump(asdict(result_schema), (result_folder / eval_file.name).open("w"))

if __name__ == "__main__":
    main()

