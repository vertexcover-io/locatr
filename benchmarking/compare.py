from pathlib import Path
import yaml
import sys
from playwright.sync_api import sync_playwright

if len(sys.argv) == 2:
    compare_with = sys.argv[1]
    if compare_with not in ["original", "anthropic", "os_atlas"]:
        raise ValueError(f"Invalid value: {compare_with}. Must be one of: original, anthropic, os_atlas")
else:
    compare_with = "original"

if compare_with == "original":
    result_folder_name = "original_locatr"
elif compare_with == "anthropic":
    result_folder_name = "anthropic_grounding"
elif compare_with == "os_atlas":
    result_folder_name = "os_atlas_grounding"

eval_schema_folder = Path("schema") / "eval"

with sync_playwright() as p:
    browser = p.chromium.launch(headless=False)
    context = browser.new_context(viewport={"width": 1920, "height": 991})

    for schema_file in eval_schema_folder.glob("*.yaml"):
        result_schema_file = Path("schema") / "results" / result_folder_name / schema_file.name
        if not result_schema_file.exists():
            continue

        eval_content = yaml.safe_load(schema_file.open("r"))
        result_content = yaml.safe_load(result_schema_file.open("r"))
        if eval_content["url"] != result_content["url"]:
            continue

        page = context.new_page()
        page.goto(eval_content["url"])
        page.add_script_tag(path="compareInject.js")
        page.wait_for_function("typeof compareLocators === 'function'")

        eval_steps = eval_content["steps"]
        result_steps = result_content["steps"]
        for eval_step, result_step in zip(eval_steps, result_steps):
            if eval_step["userRequest"] != result_step["userRequest"]:
                continue

            eval_locators = eval_step["locatrs"]
            result_locators = result_step["locatrs"]
            print(f"Eval locators: {eval_locators}")
            print(f"Result locators: {result_locators}")

            passed = page.evaluate(
                """
                ([evalLocators, resultLocators]) => {
                    console.log("Eval Locators:", evalLocators);
                    console.log("Result Locators:", resultLocators);
                    let result = compareLocators(evalLocators, resultLocators);
                    console.log("Comparison Result:", result);
                    return result;
                }
                """,
                [eval_locators, result_locators]
            )
            print(f"Comparison difference: {passed}")
